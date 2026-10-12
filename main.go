// MacBill System - marketing website + subscription admin.
// One small Go program: public site, enquiry form (with thank-you e-mail), and an admin panel
// to manage enquiries, customers' subscriptions, plans and e-mail settings.
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed web
var webFS embed.FS

type Server struct {
	admins adminStore
	ps     *pageSite
	db     *sql.DB
	lim    *limiter
	site   string
}

func env(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}

func main() {
	loadDotEnv()
	_ = os.MkdirAll("data", 0o755)
	if d := filepath.Dir(env("DB_PATH", "data/macbill.db")); d != "." {
		_ = os.MkdirAll(d, 0o755)
	}
	logEnvStatus()
	db, err := openDB(env("DB_PATH", "data/macbill.db"))
	if err != nil {
		log.Fatal(err)
	}
	s := &Server{db: db, lim: newLimiter(), site: env("SITE_URL", "http://localhost:4000")}
	if err := s.migrate(); err != nil {
		log.Fatal(err)
	}
	s.admins = s.openAdminStore()
	log.Printf("admin accounts are stored in %s", s.admins.Name())
	if err := s.seedAdmin(); err != nil {
		log.Printf("WARNING: could not prepare the admin account yet (%v) - the public website still works; it will retry on the first admin login", err)
	}
	go s.reminderLoop()

	handler := s.routes()
	addr := env("ADDR", ":"+env("PORT", "4000")) // hosting platforms usually provide PORT
	log.Printf("MacBill System site: http://localhost%s   admin: /admin", addr)
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // one connection = writes are naturally serialised; traffic here is tiny
	return db, nil
}

func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(b)
}

func (s *Server) secure(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hd := w.Header()
		hd.Set("X-Content-Type-Options", "nosniff")
		hd.Set("X-Frame-Options", "DENY")
		hd.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		hd.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		script, connect, img := "'self'", "'self'", "'self' data:"
		if s.setting("ga_id") != "" { // Google Analytics only when the owner has turned it on
			script += " https://www.googletagmanager.com"
			connect += " https://*.google-analytics.com https://*.analytics.google.com https://www.googletagmanager.com"
			img += " https://*.google-analytics.com https://www.googletagmanager.com"
		}
		hd.Set("Content-Security-Policy", "default-src 'self'; img-src "+img+"; style-src 'self' 'unsafe-inline'; script-src "+script+"; connect-src "+connect+"; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			hd.Set("Strict-Transport-Security", "max-age=31536000")
		}
		h.ServeHTTP(w, r)
	})
}

// canonical sends http->https and www/other hostnames to the one real address (set CANONICAL_REDIRECT=1 in production).
func (s *Server) canonical(h http.Handler) http.Handler {
	if env("CANONICAL_REDIRECT", "") != "1" {
		return h
	}
	u, err := url.Parse(s.site)
	if err != nil || u.Host == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			h.ServeHTTP(w, r)
			return
		}
		insecure := u.Scheme == "https" && r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https"
		if strings.EqualFold(r.Host, u.Host) && !insecure {
			h.ServeHTTP(w, r)
			return
		}
		http.Redirect(w, r, strings.TrimRight(s.site, "/")+r.URL.RequestURI(), http.StatusMovedPermanently)
	})
}

func (s *Server) gaScript(w http.ResponseWriter, r *http.Request) {
	id := s.setting("ga_id")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprintf(w, "window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments)}gtag('js',new Date());gtag('config','%s',{anonymize_ip:true});", id)
}

// ---------- small helpers ----------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, 400, "invalid request")
		return false
	}
	return true
}
func token() string       { b := make([]byte, 32); rand.Read(b); return hex.EncodeToString(b) }
func sha(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func now() int64          { return time.Now().Unix() }
func clientIP(r *http.Request) string {
	if env("TRUST_PROXY", "") == "1" {
		if x := r.Header.Get("X-Forwarded-For"); x != "" {
			return strings.TrimSpace(strings.Split(x, ",")[0])
		}
	}
	h, _, _ := net.SplitHostPort(r.RemoteAddr)
	return h
}
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
func eq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

// tiny sliding-window rate limiter
type limiter struct {
	mu sync.Mutex
	m  map[string][]int64
}

func newLimiter() *limiter { return &limiter{m: map[string][]int64{}} }
func (l *limiter) allow(key string, n int, per time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cut := time.Now().Add(-per).Unix()
	v := l.m[key][:0]
	for _, t := range l.m[key] {
		if t > cut {
			v = append(v, t)
		}
	}
	if len(v) >= n {
		l.m[key] = v
		return false
	}
	l.m[key] = append(v, time.Now().Unix())
	if len(l.m) > 20000 {
		l.m = map[string][]int64{}
	}
	return true
}

func (s *Server) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS settings(k TEXT PRIMARY KEY, v TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS admins(id INTEGER PRIMARY KEY, email TEXT UNIQUE NOT NULL, hash TEXT NOT NULL, name TEXT NOT NULL DEFAULT 'Admin', fails INTEGER NOT NULL DEFAULT 0, locked_until INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS sess(token_hash TEXT PRIMARY KEY, admin_key TEXT NOT NULL, expires INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS plans(id INTEGER PRIMARY KEY, name TEXT NOT NULL, months INTEGER NOT NULL, price INTEGER NOT NULL, tagline TEXT NOT NULL DEFAULT '', features TEXT NOT NULL DEFAULT '', popular INTEGER NOT NULL DEFAULT 0, active INTEGER NOT NULL DEFAULT 1, sort INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS enquiries(id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT NOT NULL, phone TEXT NOT NULL DEFAULT '', business TEXT NOT NULL DEFAULT '', city TEXT NOT NULL DEFAULT '', plan TEXT NOT NULL DEFAULT '', message TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'new', note TEXT NOT NULL DEFAULT '', mail TEXT NOT NULL DEFAULT 'pending', ip TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS ix_enq ON enquiries(created_at);
CREATE TABLE IF NOT EXISTS customers(id INTEGER PRIMARY KEY, business TEXT NOT NULL, contact TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL DEFAULT '', city TEXT NOT NULL DEFAULT '', plan_id INTEGER NOT NULL DEFAULT 0, plan_name TEXT NOT NULL DEFAULT '', amount INTEGER NOT NULL DEFAULT 0, starts TEXT NOT NULL DEFAULT '', ends TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'active', notes TEXT NOT NULL DEFAULT '', reminded_for TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS payments(id INTEGER PRIMARY KEY, customer_id INTEGER NOT NULL, amount INTEGER NOT NULL, plan_name TEXT NOT NULL DEFAULT '', months INTEGER NOT NULL DEFAULT 0, method TEXT NOT NULL DEFAULT '', ref TEXT NOT NULL DEFAULT '', paid_on TEXT NOT NULL, created_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS seo_overrides(path TEXT PRIMARY KEY, title TEXT NOT NULL DEFAULT '', descr TEXT NOT NULL DEFAULT '', noindex INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS outbox(id INTEGER PRIMARY KEY, to_addr TEXT NOT NULL, subject TEXT NOT NULL, kind TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'queued', error TEXT NOT NULL DEFAULT '', html TEXT NOT NULL DEFAULT '', text TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL);
`)
	if err != nil {
		return err
	}
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM plans`).Scan(&n)
	if n == 0 {
		type p struct {
			name          string
			months, price int
			tag, feat     string
			pop           int
		}
		for i, x := range []p{
			{"Starter · 6 Months", 6, 3499, "Try it for half a year", "POS & billing with GST|Kitchen display (KDS)|Table QR ordering|1 outlet · up to 5 staff logins|Daily sales reports|Email support", 0},
			{"Business · 1 Year", 12, 5999, "Best value for one restaurant", "Everything in Starter|Inventory, purchasing & suppliers|Customers, loyalty & gift cards|Accounting (double-entry) & GST reports|Up to 15 staff logins|Free setup & training call|Priority phone + WhatsApp support", 1},
			{"Chain · 1 Year", 12, 9999, "For 2 to 5 outlets", "Everything in Business|Up to 5 branches, one dashboard|HRMS - attendance, leave & payroll|Delivery module & driver app|Unlimited staff logins|Dedicated onboarding manager", 0},
		} {
			s.db.Exec(`INSERT INTO plans(name,months,price,tagline,features,popular,sort) VALUES(?,?,?,?,?,?,?)`, x.name, x.months, x.price, x.tag, x.feat, x.pop, i)
		}
	}
	for k, v := range map[string]string{"company": "MacBill System", "support_email": "info@maccoxe.com", "phone": "8541039703", "address": "Ward No. 29, Near Dr. Nutun Singh Hospital, Shanti Nagar, Mehsaual Road, Mehsaual Goat, Sitamarhi 843301, Bihar, India", "legal_name": "Macco xe", "notify_email": "", "from_name": "MacBill System", "reply_to": "",
		"smtp_host": "", "smtp_port": "587", "smtp_user": "", "smtp_pass": "", "smtp_from": "", "reminder_days": "15", "reply_hours": "24", "google_verify": "", "bing_verify": "", "ga_id": ""} {
		s.db.Exec(`INSERT OR IGNORE INTO settings(k,v) VALUES(?,?)`, k, v)
	}
	return nil
}

func (s *Server) setting(k string) string {
	if strings.HasPrefix(k, "smtp_") { // environment overrides the saved value (keeps the password out of the database)
		if e := os.Getenv("MAIL_" + strings.ToUpper(strings.TrimPrefix(k, "smtp_"))); e != "" {
			return e
		}
	}
	var v string
	s.db.QueryRow(`SELECT v FROM settings WHERE k=?`, k).Scan(&v)
	return v
}

func (s *Server) routes() http.Handler {
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))
	pages := s.newPageSite()
	s.ps = pages
	pages.loadOverrides()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/public/plans", s.handlePlans)
	mux.HandleFunc("GET /api/public/info", s.handleInfo)
	mux.HandleFunc("POST /api/public/enquiry", s.handleEnquiry)
	s.adminRoutes(mux)
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) { serveFile(w, r, sub, "admin.html") })
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /sitemap.xml", pages.sitemap)
	mux.HandleFunc("GET /robots.txt", pages.robots)
	mux.HandleFunc("GET /feed.xml", pages.feed)
	s.brandRoutes(mux)
	mux.HandleFunc("GET /ga.js", s.gaScript)
	mux.HandleFunc("GET /{path...}", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p != "/" {
			p = strings.TrimRight(p, "/")
		}
		if pages.serve(w, r, p) {
			return
		}
		if _, err := fs.Stat(sub, strings.TrimPrefix(p, "/")); err == nil && p != "/" && !strings.HasSuffix(p, ".html") {
			w.Header().Set("Cache-Control", "public, max-age=3600")
			files.ServeHTTP(w, r)
			return
		}
		pages.notFound(w, r)
	})
	return gz(s.canonical(s.secure(mux)))
}

// logEnvStatus prints which important settings were found (names only - never the values), to make deployment problems obvious.
func logEnvStatus() {
	var set, missing []string
	for _, k := range []string{"SITE_URL", "MONGO_URI", "MONGO_DB", "ADMIN_EMAIL", "ADMIN_PASSWORD", "TRUST_PROXY", "CANONICAL_REDIRECT", "DB_PATH", "MAIL_HOST", "MAIL_USER", "MAIL_PASS", "MAIL_FROM", "PORT"} {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			set = append(set, k)
		} else {
			missing = append(missing, k)
		}
	}
	log.Printf("environment: set=[%s] not set=[%s]", strings.Join(set, " "), strings.Join(missing, " "))
}
