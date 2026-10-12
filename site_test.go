package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSMTP is a tiny SMTP server that records every message it receives.
type fakeSMTP struct {
	ln   net.Listener
	mu   sync.Mutex
	msgs []string
}

func newFakeSMTP(t *testing.T) *fakeSMTP {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{ln: ln}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				r := bufio.NewReader(c)
				fmt.Fprint(c, "220 fake ESMTP\r\n")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd := strings.ToUpper(strings.TrimSpace(l))
					switch {
					case strings.HasPrefix(cmd, "EHLO"):
						fmt.Fprint(c, "250 fake\r\n")
					case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
						fmt.Fprint(c, "250 OK\r\n")
					case cmd == "DATA":
						fmt.Fprint(c, "354 go\r\n")
						var b bytes.Buffer
						for {
							d, _ := r.ReadString('\n')
							if d == ".\r\n" {
								break
							}
							b.WriteString(d)
						}
						f.mu.Lock()
						f.msgs = append(f.msgs, b.String())
						f.mu.Unlock()
						fmt.Fprint(c, "250 queued\r\n")
					case cmd == "QUIT":
						fmt.Fprint(c, "221 bye\r\n")
						return
					default:
						fmt.Fprint(c, "250 OK\r\n")
					}
				}
			}()
		}
	}()
	return f
}

func (f *fakeSMTP) count() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.msgs) }

func testServer(t *testing.T) (*Server, http.Handler) {
	t.Setenv("ADMIN_PASSWORD", "Test-Password-123")
	t.Setenv("ADMIN_EMAIL", "admin@test.in")
	dir := t.TempDir()
	db, err := openDB(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := &Server{db: db, lim: newLimiter(), site: "http://localhost:4000"}
	s.admins = &sqliteAdmins{db: db}
	if err := s.migrate(); err != nil {
		t.Fatal(err)
	}
	s.seedAdmin()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/public/plans", s.handlePlans)
	mux.HandleFunc("POST /api/public/enquiry", s.handleEnquiry)
	s.adminRoutes(mux)
	return s, mux
}

func call(h http.Handler, method, path string, body any, cookie *http.Cookie, hdr ...string) (int, map[string]any, []*http.Cookie) {
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
		req.Header.Set("X-Requested-With", "macbill")
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var m map[string]any
	json.Unmarshal(w.Body.Bytes(), &m)
	return w.Code, m, w.Result().Cookies()
}

func TestEnquiryEmailAndAdmin(t *testing.T) {
	smtpSrv := newFakeSMTP(t)
	defer smtpSrv.ln.Close()
	host, port, _ := net.SplitHostPort(smtpSrv.ln.Addr().String())
	t.Setenv("MAIL_HOST", host)
	t.Setenv("MAIL_PORT", port)
	t.Setenv("MAIL_FROM", "hello@macbill.in")
	s, h := testServer(t)

	// 1. bad input is rejected
	if c, _, _ := call(h, "POST", "/api/public/enquiry", map[string]any{"name": "A", "email": "bad"}, nil); c != 400 {
		t.Fatalf("bad enquiry accepted: %d", c)
	}
	// header-injection style e-mail is rejected
	if c, _, _ := call(h, "POST", "/api/public/enquiry", map[string]any{"name": "Ravi Kumar", "email": "x@y.com\r\nBcc: z@z.com"}, nil); c != 400 {
		t.Fatalf("injection accepted: %d", c)
	}
	// 2. a real enquiry sends a thank-you to the visitor and an alert to the team
	if c, _, _ := call(h, "POST", "/api/public/enquiry", map[string]any{"name": "Ravi Kumar", "email": "ravi@example.com", "phone": "+91 98765 43210", "business": "Spice Hub", "city": "Pune", "plan": "Business · 1 Year", "message": "Need demo <b>soon</b>", "took": 9000}, nil); c != 200 {
		t.Fatalf("enquiry failed: %d", c)
	}
	for i := 0; i < 50 && smtpSrv.count() < 2; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	if smtpSrv.count() != 2 {
		t.Fatalf("expected 2 e-mails (thank-you + team alert), got %d", smtpSrv.count())
	}
	all := strings.Join(smtpSrv.msgs, "\n")
	if !strings.Contains(all, "To: <ravi@example.com>") || !strings.Contains(all, "MacBill System") {
		t.Fatalf("thank-you mail missing or not branded:\n%s", all)
	}
	// decode the HTML part of the first message and check wording + escaping
	body := ""
	for _, m := range smtpSrv.msgs {
		if strings.Contains(m, "To: <ravi@example.com>") {
			body = decodeParts(m)
		}
	}
	if !strings.Contains(body, "contact you very soon") || strings.Contains(body, "<b>soon</b>") || !strings.Contains(body, "&lt;b&gt;soon&lt;/b&gt;") {
		t.Fatalf("mail body wrong / message not escaped: %d bytes; a=%v b=%v c=%v", len(body), strings.Contains(body, "contact you very soon"), strings.Contains(body, "<b>soon</b>"), strings.Contains(body, "&lt;b&gt;soon&lt;/b&gt;"))
	}
	// honeypot + too-fast submissions are silently dropped (no mail)
	before := smtpSrv.count()
	call(h, "POST", "/api/public/enquiry", map[string]any{"name": "Bot Bot", "email": "bot@example.com", "website": "http://spam"}, nil)
	call(h, "POST", "/api/public/enquiry", map[string]any{"name": "Fast Bot", "email": "fast@example.com", "took": 100}, nil)
	time.Sleep(300 * time.Millisecond)
	if smtpSrv.count() != before {
		t.Fatal("bot submission sent mail")
	}

	// 3. admin is protected
	if c, _, _ := call(h, "GET", "/api/admin/customers", nil, nil); c != 401 {
		t.Fatalf("admin open without login: %d", c)
	}
	if c, _, _ := call(h, "POST", "/api/admin/login", map[string]any{"email": "admin@test.in", "password": "wrong"}, nil); c != 401 {
		t.Fatalf("wrong password accepted: %d", c)
	}
	c, _, ck := call(h, "POST", "/api/admin/login", map[string]any{"email": "admin@test.in", "password": "Test-Password-123"}, nil)
	if c != 200 || len(ck) == 0 || !ck[0].HttpOnly {
		t.Fatalf("login failed or cookie not HttpOnly: %d", c)
	}
	sess := ck[0]
	// CSRF: a state-changing call without the custom header is refused
	req := httptest.NewRequest("POST", "/api/admin/plans", strings.NewReader(`{}`))
	req.AddCookie(sess)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("CSRF guard missing: %d", w.Code)
	}

	// 4. enquiry is visible and mail status is tracked
	_, _, _ = call(h, "GET", "/api/admin/me", nil, sess)
	req = httptest.NewRequest("GET", "/api/admin/enquiries", nil)
	req.AddCookie(sess)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var list []map[string]any
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0]["mail"] != "sent" {
		t.Fatalf("enquiry list wrong: %s", w.Body.String())
	}

	// 5. convert to customer -> subscription mail; renew extends from the end date
	plans := s.qmaps(`SELECT id FROM plans WHERE months=6`)
	pid6 := fmt.Sprint(plans[0]["id"])
	_, r1, _ := call(h, "POST", "/api/admin/enquiries/1/convert", map[string]any{"plan_id": toInt(plans[0]["id"])}, sess)
	if r1["ok"] != true {
		t.Fatalf("convert failed: %v", r1)
	}
	cust := s.one(`SELECT * FROM customers`)
	if cust["plan_name"] == nil || fmt.Sprint(cust["amount"]) != "3499" {
		t.Fatalf("customer wrong: %v (plan %s)", cust, pid6)
	}
	wantEnd := time.Now().AddDate(0, 6, 0).Format("2006-01-02")
	if cust["ends"] != wantEnd {
		t.Fatalf("6-month plan should end %s, got %v", wantEnd, cust["ends"])
	}
	_, r2, _ := call(h, "POST", fmt.Sprintf("/api/admin/customers/%v/renew", cust["id"]), map[string]any{"plan_id": toInt(plans[0]["id"]), "notify": true}, sess)
	if r2["ends"] != time.Now().AddDate(0, 12, 0).Format("2006-01-02") {
		t.Fatalf("renew did not extend from the old end date: %v", r2)
	}
	if n := toInt(s.one(`SELECT COUNT(*) AS n FROM payments`)["n"]); n != 2 {
		t.Fatalf("expected 2 payments, got %d", n)
	}
	for i := 0; i < 50 && smtpSrv.count() < 4; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	if smtpSrv.count() != 4 {
		t.Fatalf("expected subscription + renewal mails, total 4, got %d", smtpSrv.count())
	}

	// 6. the SMTP password is never returned to the browser
	req = httptest.NewRequest("GET", "/api/admin/settings", nil)
	req.AddCookie(sess)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if strings.Contains(w.Body.String(), "smtp_pass\":\"x") {
		t.Fatal("smtp password leaked")
	}
}

func TestNoSMTPKeepsMailInOutbox(t *testing.T) {
	os.Unsetenv("MAIL_HOST")
	s, h := testServer(t)
	call(h, "POST", "/api/public/enquiry", map[string]any{"name": "Asha Rao", "email": "asha@example.com", "took": 9000}, nil)
	var st string
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		s.db.QueryRow(`SELECT status FROM outbox WHERE kind='enquiry_thanks'`).Scan(&st)
		if st == "not_configured" {
			break
		}
	}
	if st != "not_configured" {
		t.Fatalf("mail should wait in the outbox when SMTP is missing, status=%q", st)
	}
	if n := toInt(s.one(`SELECT COUNT(*) AS n FROM enquiries`)["n"]); n != 1 {
		t.Fatal("enquiry must be saved even if mail cannot be sent")
	}
}

func TestLoginLockout(t *testing.T) {
	_, h := testServer(t)
	for i := 0; i < 5; i++ {
		call(h, "POST", "/api/admin/login", map[string]any{"email": "admin@test.in", "password": "nope"}, nil)
	}
	if c, _, _ := call(h, "POST", "/api/admin/login", map[string]any{"email": "admin@test.in", "password": "Test-Password-123"}, nil); c != 429 {
		t.Fatalf("account should be locked after 5 wrong passwords, got %d", c)
	}
}

// decodeParts returns the readable (base64-decoded) text of every part of a MIME message.
func decodeParts(msg string) string {
	var out strings.Builder
	for _, part := range strings.Split(msg, "\r\n\r\n")[1:] {
		chunk := part
		if i := strings.Index(chunk, "\r\n--"); i >= 0 {
			chunk = chunk[:i]
		}
		if b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(chunk), "\r\n", "")); err == nil {
			out.Write(b)
		}
	}
	return out.String()
}
