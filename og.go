package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ---------- RSS feed for the blog ----------

func (ps *pageSite) feed(w http.ResponseWriter, r *http.Request) {
	site := strings.TrimRight(ps.srv.site, "/")
	company := ps.srv.setting("company")
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom"><channel>`)
	fmt.Fprintf(&b, "<title>%s blog</title><link>%s/blog</link><description>Restaurant management guides from %s</description><language>en-IN</language>", esc(company), site, esc(company))
	fmt.Fprintf(&b, `<atom:link href="%s/feed.xml" rel="self" type="application/rss+xml"/>`, site)
	for _, p := range posts {
		fmt.Fprintf(&b, "<item><title>%s</title><link>%s%s</link><guid>%s%s</guid><pubDate>%s</pubDate><description>%s</description></item>", esc(p.Title), site, p.Path(), site, p.Path(), rssDate(p.Date), esc(p.Desc))
	}
	b.WriteString("</channel></rss>")
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(b.String()))
}

func rssDate(d string) string {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return ""
	}
	return t.Format(time.RFC1123Z)
}
