package main

import (
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

var phoneRe = regexp.MustCompile(`^[0-9+()\-\s]{7,20}$`)

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.qmaps(`SELECT id,name,months,price,tagline,features,popular FROM plans WHERE active=1 ORDER BY sort,id`))
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"company": s.setting("company"), "email": s.setting("support_email"), "phone": s.setting("phone"), "address": s.setting("address"), "reply_hours": s.setting("reply_hours")})
}

// POST /api/public/enquiry - saves the enquiry, e-mails the visitor a thank-you and alerts the team.
func (s *Server) handleEnquiry(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !s.lim.allow("enq:"+ip, 5, time.Hour) {
		fail(w, 429, "Too many requests. Please try again in a little while or call us.")
		return
	}
	var in struct {
		Name, Email, Phone, Business, City, Plan, Message string
		Website                                           string // honeypot - real people never fill it
		Took                                              int    // ms the visitor took to fill the form
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Website != "" || (in.Took > 0 && in.Took < 2500) { // bots: pretend success
		writeJSON(w, 200, map[string]any{"ok": true})
		return
	}
	e := map[string]string{"name": clip(in.Name, 80), "email": strings.ToLower(clip(in.Email, 120)), "phone": clip(in.Phone, 20), "business": clip(in.Business, 100), "city": clip(in.City, 60), "plan": clip(in.Plan, 80), "message": clip(in.Message, 1000)}
	if len([]rune(e["name"])) < 2 {
		fail(w, 400, "Please enter your name.")
		return
	}
	if a, err := mail.ParseAddress(e["email"]); err != nil || a.Address != e["email"] || !strings.Contains(e["email"], ".") {
		fail(w, 400, "Please enter a valid e-mail address - we will send our reply there.")
		return
	}
	if e["phone"] != "" && !phoneRe.MatchString(e["phone"]) {
		fail(w, 400, "Please enter a valid phone number.")
		return
	}
	if !s.lim.allow("enq-mail:"+e["email"], 3, 24*time.Hour) { // do not let the form be used to spam one inbox
		writeJSON(w, 200, map[string]any{"ok": true})
		return
	}
	res, err := s.db.Exec(`INSERT INTO enquiries(name,email,phone,business,city,plan,message,ip,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		e["name"], e["email"], e["phone"], e["business"], e["city"], e["plan"], e["message"], ip, now())
	if err != nil {
		fail(w, 500, "Something went wrong. Please try again.")
		return
	}
	id, _ := res.LastInsertId()
	s.mailEnquiryThanks(e)
	s.mailEnquiryAlert(e, id)
	writeJSON(w, 200, map[string]any{"ok": true})
}
