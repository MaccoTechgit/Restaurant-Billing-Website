package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed tpl
var tplFS embed.FS

type Crumb struct{ Name, Path string }

type PageMeta struct {
	Title, Desc, Path, OGType, Robots string
	Crumbs                            []Crumb
	Modified                          string
}

// View is everything a template can use.
type View struct {
	PageMeta
	Company, Legal, Email, Phone, Address, Site, Year, Reply, WA, MapsURL, GA, GoogleVerify, BingVerify string
	Products, Solutions                                                                                 []*Detail
	Posts                                                                                               []*Post
	D                                                                                                   *Detail
	P                                                                                                   *Post
	Guides                                                                                              []*Post
	RelDetails                                                                                          []*Detail
	RelPosts                                                                                            []*Post
	GeneralFAQ                                                                                          []QA
	LD                                                                                                  []template.JS
	Canonical                                                                                           string
	Tmpl                                                                                                string
}

var generalFAQ = []QA{
	{"What is MacBill System?", "MacBill System is restaurant billing and management software. It covers POS and billing, a kitchen display, QR table ordering, inventory, accounting, staff attendance and payroll, delivery and reports in one place."},
	{"Do I need special hardware?", "No. It runs in a web browser on the PC, laptop or tablet you already have. For receipts, any 58mm or 80mm thermal printer works."},
	{"How long does it take to start?", "Most restaurants can start billing the same day. We help you set up your menu, tables, taxes, printer and staff logins, and we give a free training call."},
	{"How does pricing work?", "You pay one fixed price for the plan period - 6 months or 1 year. There are no per-bill or per-order charges. See the pricing page for current plans."},
	{"What happens when my plan ends?", "We remind you by e-mail before it ends. Your data stays safe, and you can renew whenever you are ready."},
	{"Can different staff see different things?", "Yes. There are 12 ready roles, plus custom roles. A waiter, a cashier and an owner each see only what they should."},
	{"Can I manage more than one outlet?", "Yes. The Chain plan supports up to five branches on one dashboard, with separate stock and staff per branch."},
	{"Is my data safe?", "Passwords are stored securely, staff access is permission-controlled, important actions are logged, and the system is built to resist common web attacks. Ask us about backups and hosting for your setup."},
	{"Can I move my menu from my old software?", "Yes. We can import your items, prices and categories from Excel or CSV during setup."},
	{"How do I get a demo?", "Send an enquiry from the contact page. Our team will contact you very soon to show a live demo on your own menu."},
}

type pageDef struct {
	meta  PageMeta
	tmpl  string
	build func(v *View)
}

type pageSite struct {
	srv   *Server
	sets  map[string]*template.Template
	pages map[string]*pageDef
	order []string
	mu    sync.Mutex
	over  map[string]seoOverride
}

func (s *Server) newPageSite() *pageSite {
	ps := &pageSite{srv: s, sets: map[string]*template.Template{}, pages: map[string]*pageDef{}, over: map[string]seoOverride{}}
	funcs := template.FuncMap{
		"inr": func(n int) string { return "₹" + commas(n) },
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"add":     func(a, b int) int { return a + b },
		"ico":     ico,
		"barcode": barcodeSVG,
		"qr":      qrSVG,
		"wm": func(name string) template.HTML {
			first, rest, _ := strings.Cut(name, " ")
			if rest == "" {
				return template.HTML("<b>" + template.HTMLEscapeString(first) + "</b>")
			}
			return template.HTML("<b>" + template.HTMLEscapeString(first) + "</b><i>" + template.HTMLEscapeString(rest) + "</i>")
		},
	}
	for _, n := range []string{"home", "detail", "features", "pricing", "about", "contact", "faq", "blog", "post", "legal", "notfound"} {
		t, err := template.New(n).Funcs(funcs).ParseFS(tplFS, "tpl/layout.html", "tpl/"+n+".html")
		if err != nil {
			panic(fmt.Sprintf("template %s: %v", n, err))
		}
		ps.sets[n] = t
	}
	add := func(path, tmpl string, m PageMeta, b func(*View)) {
		m.Path = path
		if m.OGType == "" {
			m.OGType = "website"
		}
		ps.pages[path] = &pageDef{meta: m, tmpl: tmpl, build: b}
		ps.order = append(ps.order, path)
	}
	home := Crumb{"Home", "/"}
	add("/", "home", PageMeta{Title: "MacBill System: Restaurant Billing & POS Software", Desc: "Restaurant billing software with POS, kitchen display, QR table ordering, inventory, accounting, HR and delivery in one simple system. Free demo."}, nil)
	add("/features", "features", PageMeta{Title: "Restaurant Software Features | MacBill System", Desc: "Explore every MacBill System feature: POS, kitchen display, QR ordering, inventory, accounting, HR & payroll, delivery, loyalty and reports.", Crumbs: []Crumb{home, {"Features", "/features"}}}, nil)
	add("/pricing", "pricing", PageMeta{Title: "Restaurant Software Pricing & Plans | MacBill System", Desc: "Simple, fixed-price plans for restaurants: 6 months, 1 year and multi-branch. No per-bill charges. Free setup and training.", Crumbs: []Crumb{home, {"Pricing", "/pricing"}}}, nil)
	add("/about", "about", PageMeta{Title: "About MacBill System - Made for Restaurants", Desc: "Why we build MacBill System: simple, fast, professional restaurant software that any team can use from day one.", Crumbs: []Crumb{home, {"About", "/about"}}}, nil)
	add("/contact", "contact", PageMeta{Title: "Contact & Free Demo | MacBill System", Desc: "Talk to the MacBill System team. Send an enquiry and we will contact you very soon with a free demo for your restaurant.", Crumbs: []Crumb{home, {"Contact", "/contact"}}}, nil)
	add("/faq", "faq", PageMeta{Title: "FAQ - Restaurant Billing Software | MacBill System", Desc: "Answers to common questions about MacBill System: hardware, setup time, pricing, staff roles, multiple outlets, data safety and demos.", Crumbs: []Crumb{home, {"FAQ", "/faq"}}}, nil)
	add("/blog", "blog", PageMeta{Title: "Restaurant Management Blog | MacBill System", Desc: "Practical guides for restaurant owners: choosing billing software, GST bills, reducing wastage, QR ordering, kitchen flow and staff permissions.", Crumbs: []Crumb{home, {"Blog", "/blog"}}}, nil)
	add("/privacy", "legal", PageMeta{Title: "Privacy Policy | MacBill System", Desc: "How MacBill System collects, uses and protects the information you send us.", Crumbs: []Crumb{home, {"Privacy Policy", "/privacy"}}}, func(v *View) { v.Tmpl = "privacy" })
	add("/terms", "legal", PageMeta{Title: "Terms of Service | MacBill System", Desc: "The terms that apply to using the MacBill System website and subscription plans.", Crumbs: []Crumb{home, {"Terms of Service", "/terms"}}}, func(v *View) { v.Tmpl = "terms" })
	for _, d := range append(append([]*Detail{}, products...), solutions...) {
		d := d
		crumbs := []Crumb{home, {"Features", "/features"}, {d.Name, d.Path()}}
		if d.Kind == "solution" {
			crumbs = []Crumb{home, {"Solutions", "/features#solutions"}, {d.Name, d.Path()}}
		}
		add(d.Path(), "detail", PageMeta{Title: d.Title, Desc: d.Desc, Crumbs: crumbs}, func(v *View) {
			v.D = d
			for _, r := range d.Related {
				if x := findProduct(r); x != nil {
					v.RelDetails = append(v.RelDetails, x)
				}
			}
			for _, r := range d.BlogLinks {
				if x := findPost(r); x != nil {
					v.RelPosts = append(v.RelPosts, x)
				}
			}
		})
	}
	for _, p := range guides {
		p := p
		add(p.Path(), "post", PageMeta{Title: guideTitle(p.Title), Desc: p.Desc, OGType: "article", Modified: p.Date, Crumbs: []Crumb{home, {"Guides", "/blog"}, {p.Name(), p.Path()}}}, func(v *View) {
			v.P = p
			for _, r := range p.Related {
				if x := findProduct(r); x != nil {
					v.RelDetails = append(v.RelDetails, x)
				}
			}
			v.RelPosts = posts[:3]
		})
	}
	for _, p := range posts {
		p := p
		add(p.Path(), "post", PageMeta{Title: postTitle(p.Title), Desc: p.Desc, OGType: "article", Modified: p.Date, Crumbs: []Crumb{home, {"Blog", "/blog"}, {p.Title, "/blog/" + p.Slug}}}, func(v *View) {
			v.P = p
			for _, r := range p.Related {
				if x := findProduct(r); x != nil {
					v.RelDetails = append(v.RelDetails, x)
				}
			}
			for _, o := range posts {
				if o.Slug != p.Slug && len(v.RelPosts) < 3 {
					v.RelPosts = append(v.RelPosts, o)
				}
			}
			v.Guides = guides
		})
	}
	return ps
}

func commas(n int) string { // Indian digit grouping: 1,23,456
	s := fmt.Sprint(n)
	if len(s) <= 3 {
		return s
	}
	head, tail := s[:len(s)-3], s[len(s)-3:]
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}

func (ps *pageSite) view(def *pageDef) *View {
	s := ps.srv
	v := &View{PageMeta: def.meta, Company: s.setting("company"), Legal: s.setting("legal_name"), Email: s.setting("support_email"), Phone: s.setting("phone"), Address: s.setting("address"),
		Site: strings.TrimRight(s.site, "/"), Year: fmt.Sprint(time.Now().Year()), Reply: s.setting("reply_hours"),
		Products: products, Solutions: solutions, Posts: posts, GeneralFAQ: generalFAQ}
	if v.Reply == "" {
		v.Reply = "24"
	}
	if v.Legal == "" {
		v.Legal = v.Company
	}
	v.WA = waNumber(v.Phone)
	v.MapsURL = "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(v.Address)
	v.Canonical = v.Site + def.meta.Path
	if def.build != nil {
		def.build(v)
	}
	if v.Path == "/" {
		v.RelPosts = posts[:3]
	}
	v.GA, v.GoogleVerify, v.BingVerify = s.setting("ga_id"), s.setting("google_verify"), s.setting("bing_verify")
	v.Guides = guides
	if o, ok := ps.override(def.meta.Path); ok {
		if o.Title != "" {
			v.Title = o.Title
		}
		if o.Desc != "" {
			v.Desc = o.Desc
		}
		if o.NoIndex {
			v.Robots = "noindex,follow"
		}
	}
	v.LD = ps.jsonLD(v)
	return v
}

func ld(v any) template.JS { b, _ := json.Marshal(v); return template.JS(b) }

func (ps *pageSite) jsonLD(v *View) []template.JS {
	out := []template.JS{}
	out = append(out, ld(map[string]any{"@context": "https://schema.org", "@type": "Organization", "@id": v.Site + "/#org", "name": v.Company, "url": v.Site, "logo": map[string]any{"@type": "ImageObject", "url": v.Site + "/logo.png", "width": 512, "height": 512}, "image": v.Site + "/og.png", "email": v.Email, "telephone": "+" + v.WA, "legalName": v.Legal, "address": map[string]any{"@type": "PostalAddress", "streetAddress": v.Address, "addressCountry": "IN"}}))
	if v.Path == "/" {
		out = append(out, ld(map[string]any{"@context": "https://schema.org", "@type": "WebSite", "name": v.Company, "alternateName": "MacBill", "url": v.Site, "publisher": map[string]any{"@id": v.Site + "/#org"}}))
	}
	if len(v.Crumbs) > 1 {
		items := []map[string]any{}
		for i, c := range v.Crumbs {
			items = append(items, map[string]any{"@type": "ListItem", "position": i + 1, "name": c.Name, "item": v.Site + c.Path})
		}
		out = append(out, ld(map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": items}))
	}
	if v.Path == "/" || v.Path == "/pricing" || (v.D != nil && v.D.Kind == "product") {
		offers := []map[string]any{}
		for _, p := range ps.srv.qmaps(`SELECT name,price FROM plans WHERE active=1 ORDER BY sort,id`) {
			offers = append(offers, map[string]any{"@type": "Offer", "name": p["name"], "price": fmt.Sprint(p["price"]), "priceCurrency": "INR"})
		}
		name := v.Company
		if v.D != nil {
			name = v.Company + " " + v.D.Name
		}
		out = append(out, ld(map[string]any{"@context": "https://schema.org", "@type": "SoftwareApplication", "name": name, "applicationCategory": "BusinessApplication", "operatingSystem": "Web", "description": v.Desc, "offers": offers}))
	}
	faq := []QA{}
	if v.D != nil {
		faq = v.D.FAQ
	} else if v.Path == "/faq" {
		faq = v.GeneralFAQ
	} else if v.P != nil {
		faq = v.P.FAQ
	}
	if len(faq) > 0 {
		qs := []map[string]any{}
		for _, q := range faq {
			qs = append(qs, map[string]any{"@type": "Question", "name": q.Q, "acceptedAnswer": map[string]any{"@type": "Answer", "text": q.A}})
		}
		out = append(out, ld(map[string]any{"@context": "https://schema.org", "@type": "FAQPage", "mainEntity": qs}))
	}
	if v.P != nil {
		out = append(out, ld(map[string]any{"@context": "https://schema.org", "@type": "Article", "headline": v.P.Title, "description": v.P.Desc, "datePublished": v.P.Date, "dateModified": v.P.Date, "mainEntityOfPage": v.Canonical,
			"author": map[string]any{"@type": "Organization", "name": v.Company}, "publisher": map[string]any{"@type": "Organization", "name": v.Company}}))
	}
	return out
}

func (ps *pageSite) serve(w http.ResponseWriter, r *http.Request, path string) bool {
	def, ok := ps.pages[path]
	if !ok {
		return false
	}
	v := ps.view(def)
	var buf bytes.Buffer
	if err := ps.sets[def.tmpl].ExecuteTemplate(&buf, "layout", v); err != nil {
		http.Error(w, "page error", 500)
		return true
	}
	sum := sha256.Sum256(buf.Bytes())
	etag := `W/"` + hex.EncodeToString(sum[:8]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	w.Write(buf.Bytes())
	return true
}

func (ps *pageSite) notFound(w http.ResponseWriter, r *http.Request) {
	v := ps.view(&pageDef{meta: PageMeta{Title: "Page not found | MacBill System", Desc: "This page could not be found.", Path: r.URL.Path, Robots: "noindex"}})
	var buf bytes.Buffer
	ps.sets["notfound"].ExecuteTemplate(&buf, "layout", v)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(404)
	w.Write(buf.Bytes())
}

func (ps *pageSite) sitemap(w http.ResponseWriter, r *http.Request) {
	site := strings.TrimRight(ps.srv.site, "/")
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	paths := append([]string{}, ps.order...)
	sort.SliceStable(paths, func(i, j int) bool { return len(paths[i]) < len(paths[j]) })
	day := time.Now().Format("2006-01-02")
	for _, p := range paths {
		pri := "0.6"
		switch {
		case p == "/":
			pri = "1.0"
		case p == "/pricing" || p == "/contact" || p == "/features":
			pri = "0.9"
		case p == "/restaurant-billing-software" || p == "/restaurant-billing-software-bihar":
			pri = "0.9"
		case strings.HasPrefix(p, "/blog/"):
			pri = "0.5"
		case p == "/privacy" || p == "/terms":
			pri = "0.2"
		}
		lm := day
		if pg := ps.pages[p]; pg.meta.Modified != "" {
			lm = pg.meta.Modified
		}
		fmt.Fprintf(&b, "<url><loc>%s%s</loc><lastmod>%s</lastmod><priority>%s</priority></url>\n", site, p, lm, pri)
	}
	b.WriteString("</urlset>\n")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	io.WriteString(w, b.String())
}

func (ps *pageSite) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /api/\n\nSitemap: %s/sitemap.xml\n", strings.TrimRight(ps.srv.site, "/"))
}

// ---------- gzip ----------

type gzWriter struct {
	http.ResponseWriter
	zw *gzip.Writer
}

func (g *gzWriter) Write(b []byte) (int, error) { return g.zw.Write(b) }
func (g *gzWriter) WriteHeader(c int) {
	g.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(c)
}

func gz(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(w)
		defer zw.Close()
		next.ServeHTTP(&gzWriter{ResponseWriter: w, zw: zw}, r)
	})
}

var _ fs.FS

func postTitle(t string) string {
	if len(t) <= 56 {
		return t + " | MacBill System"
	}
	return t
}

// waNumber turns "+91 85410 39703" into the digits WhatsApp links need (91...).
func waNumber(phone string) string {
	d := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	if len(d) == 10 {
		d = "91" + d
	}
	return d
}

func guideTitle(t string) string {
	if len(t) <= 56 {
		return t + " | MacBill System"
	}
	return t
}

func (p *Post) Name() string {
	switch p.Slug {
	case "restaurant-billing-software":
		return "Billing software guide"
	case "restaurant-billing-software-bihar":
		return "Software for Bihar"
	}
	return p.Title
}

// ---------- SEO overrides (edited in Admin > SEO) ----------

type seoOverride struct {
	Title, Desc string
	NoIndex     bool
}

func (ps *pageSite) loadOverrides() {
	m := map[string]seoOverride{}
	for _, r := range ps.srv.qmaps(`SELECT path,title,descr,noindex FROM seo_overrides`) {
		m[fmt.Sprint(r["path"])] = seoOverride{Title: fmt.Sprint(r["title"]), Desc: fmt.Sprint(r["descr"]), NoIndex: toInt(r["noindex"]) == 1}
	}
	ps.mu.Lock()
	ps.over = m
	ps.mu.Unlock()
}

func (ps *pageSite) override(path string) (seoOverride, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	o, ok := ps.over[path]
	return o, ok
}
