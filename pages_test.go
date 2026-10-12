package main

import (
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func pagesServer(t *testing.T) (*Server, http.Handler) {
	t.Setenv("ADMIN_PASSWORD", "Test-Password-123")
	db, err := openDB(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := &Server{db: db, lim: newLimiter(), site: "https://macbill.example"}
	if err := s.migrate(); err != nil {
		t.Fatal(err)
	}
	return s, s.routes()
}

func get(h http.Handler, path string) (int, string, http.Header) {
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	b, _ := io.ReadAll(w.Result().Body)
	return w.Code, string(b), w.Header()
}

var (
	titleRe = regexp.MustCompile(`<title>([^<]+)</title>`)
	descRe  = regexp.MustCompile(`<meta name="description" content="([^"]+)"`)
	hrefRe  = regexp.MustCompile(`href="(/[^"#?]*)`)
	h1Re    = regexp.MustCompile(`<h1`)
)

func TestAllPagesAreSEOReady(t *testing.T) {
	_, h := pagesServer(t)
	code, sm, _ := get(h, "/sitemap.xml")
	if code != 200 {
		t.Fatalf("sitemap: %d", code)
	}
	locs := regexp.MustCompile(`<loc>https://macbill\.example([^<]*)</loc>`).FindAllStringSubmatch(sm, -1)
	if len(locs) < 25 {
		t.Fatalf("expected 25+ pages in the sitemap, got %d", len(locs))
	}
	titles, descs := map[string]string{}, map[string]string{}
	links := map[string]bool{}
	for _, m := range locs {
		p := m[1]
		if p == "" {
			p = "/"
		}
		code, body, _ := get(h, p)
		if code != 200 {
			t.Errorf("%s -> %d", p, code)
			continue
		}
		ti, de := titleRe.FindStringSubmatch(body), descRe.FindStringSubmatch(body)
		if ti == nil || de == nil {
			t.Errorf("%s: missing title or description", p)
			continue
		}
		if len(ti[1]) > 75 || len(ti[1]) < 20 {
			t.Errorf("%s: title length %d (%q)", p, len(ti[1]), ti[1])
		}
		if len(de[1]) > 170 || len(de[1]) < 60 {
			t.Errorf("%s: description length %d", p, len(de[1]))
		}
		if o, dup := titles[ti[1]]; dup {
			t.Errorf("duplicate title on %s and %s", o, p)
		}
		if o, dup := descs[de[1]]; dup {
			t.Errorf("duplicate description on %s and %s", o, p)
		}
		titles[ti[1]], descs[de[1]] = p, p
		if len(h1Re.FindAllString(body, -1)) != 1 {
			t.Errorf("%s: should have exactly one <h1>", p)
		}
		if !strings.Contains(body, `<link rel="canonical" href="https://macbill.example`+strings.TrimSuffix(p, "/")+`">`) && p != "/" {
			t.Errorf("%s: canonical missing/wrong", p)
		}
		for _, l := range hrefRe.FindAllStringSubmatch(body, -1) {
			links[strings.TrimRight(l[1], "/")] = true
		}
		// every JSON-LD block must be valid JSON
		for _, j := range regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).FindAllStringSubmatch(body, -1) {
			var x any
			if err := json.Unmarshal([]byte(j[1]), &x); err != nil {
				t.Errorf("%s: invalid JSON-LD: %v", p, err)
			}
		}
	}
	// internal links must not be broken
	for l := range links {
		if l == "" || l == "/site.css" || l == "/site.js" || strings.HasPrefix(l, "/api") {
			continue
		}
		if l == "/admin" {
			continue
		}
		if code, _, _ := get(h, l); code != 200 {
			t.Errorf("broken internal link %s -> %d", l, code)
		}
	}
}

func TestRobotsAnd404(t *testing.T) {
	_, h := pagesServer(t)
	_, robots, _ := get(h, "/robots.txt")
	if !strings.Contains(robots, "Sitemap: https://macbill.example/sitemap.xml") || !strings.Contains(robots, "Disallow: /admin") {
		t.Fatalf("robots.txt wrong: %s", robots)
	}
	code, body, _ := get(h, "/no-such-page")
	if code != 404 || !strings.Contains(body, "noindex") {
		t.Fatalf("404 page should return 404 + noindex, got %d", code)
	}
	if code, _, _ := get(h, "/pricing/"); code != 200 {
		t.Fatalf("trailing slash should work: %d", code)
	}
	// the admin panel must not be indexable or linked from the public pages
	_, home, _ := get(h, "/")
	if strings.Contains(home, `href="/admin"`) {
		t.Fatal("public pages must not link to /admin")
	}
}

func TestSEOExtras(t *testing.T) {
	s, h := pagesServer(t)
	// ETag: second request with If-None-Match gets 304
	req := httptest.NewRequest("GET", "/pricing", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	et := w.Header().Get("ETag")
	if et == "" {
		t.Fatal("no ETag")
	}
	req = httptest.NewRequest("GET", "/pricing", nil)
	req.Header.Set("If-None-Match", et)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 304 {
		t.Fatalf("expected 304, got %d", w.Code)
	}
	// social card is a real 1200x630 PNG
	_, body, hd := get(h, "/og.png")
	if hd.Get("Content-Type") != "image/png" || !strings.HasPrefix(body, "\x89PNG") {
		t.Fatal("og.png is not a PNG")
	}
	cfg, err := pngConfig(body)
	if err != nil || cfg[0] != 1200 || cfg[1] != 630 {
		t.Fatalf("og.png size wrong: %v %v", cfg, err)
	}
	// feed and guide pages
	if _, feed, _ := get(h, "/feed.xml"); !strings.Contains(feed, "<rss") || strings.Count(feed, "<item>") < 14 {
		t.Fatal("feed missing items")
	}
	for _, p := range []string{"/restaurant-billing-software", "/restaurant-billing-software-bihar"} {
		code, b, _ := get(h, p)
		if code != 200 || !strings.Contains(b, `"@type":"FAQPage"`) {
			t.Fatalf("%s: code %d or FAQ schema missing", p, code)
		}
	}
	// home page carries social-card tags, hreflang and lang
	_, home, _ := get(h, "/")
	for _, want := range []string{`og:image" content="https://macbill.example/og.png"`, `hreflang="en-IN"`, `<html lang="en-IN">`, `twitter:card" content="summary_large_image"`} {
		if !strings.Contains(home, want) {
			t.Errorf("home page missing %s", want)
		}
	}
	// analytics + verification are off by default and CSP stays strict
	if strings.Contains(home, "googletagmanager") || strings.Contains(home, "google-site-verification") {
		t.Fatal("analytics/verification must be off by default")
	}
	// overrides: custom title shows, noindex works, reset works
	s.db.Exec(`INSERT INTO seo_overrides(path,title,descr,noindex) VALUES('/faq','Custom FAQ title for test','',1)`)
	s.ps.loadOverrides()
	_, faq, _ := get(h, "/faq")
	if !strings.Contains(faq, "<title>Custom FAQ title for test</title>") || !strings.Contains(faq, `content="noindex,follow"`) {
		t.Fatal("SEO override not applied")
	}
	// GA turns on only when an ID is saved, and then the CSP allows exactly Google's hosts
	s.db.Exec(`UPDATE settings SET v='G-ABCD1234' WHERE k='ga_id'`)
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "googletagmanager.com/gtag/js?id=G-ABCD1234") || !strings.Contains(w.Header().Get("Content-Security-Policy"), "googletagmanager.com") {
		t.Fatal("GA not wired when ID is set")
	}
}

func pngConfig(s string) ([2]int, error) {
	c, err := png.DecodeConfig(strings.NewReader(s))
	return [2]int{c.Width, c.Height}, err
}

func TestIconsAndManifest(t *testing.T) {
	_, h := pagesServer(t)
	for path, size := range map[string]int{"/icon-48.png": 48, "/icon-96.png": 96, "/icon-192.png": 192, "/icon-512.png": 512, "/icon-maskable-512.png": 512, "/apple-touch-icon.png": 180, "/logo.png": 512} {
		_, body, hd := get(h, path)
		cfg, err := pngConfig(body)
		if err != nil || cfg[0] != size || cfg[1] != size || hd.Get("Content-Type") != "image/png" {
			t.Errorf("%s: %v %v", path, cfg, err)
		}
	}
	_, ico, hd := get(h, "/favicon.ico")
	if hd.Get("Content-Type") != "image/x-icon" || len(ico) < 100 || ico[2] != 1 || ico[4] != 3 {
		t.Fatal("favicon.ico is not a 3-image ICO file")
	}
	if _, svg, _ := get(h, "/favicon.svg"); !strings.HasPrefix(svg, "<svg") {
		t.Fatal("favicon.svg missing")
	}
	_, mf, _ := get(h, "/site.webmanifest")
	var m map[string]any
	if json.Unmarshal([]byte(mf), &m) != nil || m["name"] != "MacBill System" || len(m["icons"].([]any)) != 3 {
		t.Fatalf("manifest wrong: %s", mf)
	}
	_, home, _ := get(h, "/")
	for _, want := range []string{`href="/favicon.ico"`, `href="/favicon.svg"`, `href="/icon-192.png"`, `rel="manifest"`, `rel="apple-touch-icon"`, `<span class="wm"><b>MacBill</b><i>System</i></span>`, `"logo":{"@type":"ImageObject"`} {
		if !strings.Contains(home, want) {
			t.Errorf("home page missing %s", want)
		}
	}
}
