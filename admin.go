package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessCookie = "mb_admin"

type ctxAdmin struct{ ID int64 }

// qmaps runs a query and returns rows as maps (never nil, so the screens never receive null lists).
func (s *Server) qmaps(q string, args ...any) []map[string]any {
	out := []map[string]any{}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	for rows.Next() {
		vals := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range vals {
			ptr[i] = &vals[i]
		}
		if rows.Scan(ptr...) != nil {
			continue
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			if b, ok := vals[i].([]byte); ok {
				m[c] = string(b)
			} else {
				m[c] = vals[i]
			}
		}
		out = append(out, m)
	}
	return out
}

func (s *Server) one(q string, args ...any) map[string]any {
	r := s.qmaps(q, args...)
	if len(r) == 0 {
		return nil
	}
	return r[0]
}

func (s *Server) adminID(r *http.Request) int64 {
	c, err := r.Cookie(sessCookie)
	if err != nil || len(c.Value) != 64 {
		return 0
	}
	var id, exp int64
	if s.db.QueryRow(`SELECT admin_id,expires FROM sessions WHERE token_hash=?`, sha(c.Value)).Scan(&id, &exp) != nil || exp < now() {
		return 0
	}
	return id
}

// auth guards every admin call. State-changing calls must also carry the custom header (blocks cross-site forms).
func (s *Server) auth(h func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.adminID(r) == 0 {
			fail(w, 401, "Please log in")
			return
		}
		if r.Method != http.MethodGet && r.Header.Get("X-Requested-With") != "macbill" {
			fail(w, 403, "blocked")
			return
		}
		h(w, r)
	}
}

func pid(r *http.Request) int64 { n, _ := strconv.ParseInt(r.PathValue("id"), 10, 64); return n }

func (s *Server) adminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/admin/login", s.login)
	mux.HandleFunc("POST /api/admin/logout", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessCookie); err == nil {
			s.db.Exec(`DELETE FROM sessions WHERE token_hash=?`, sha(c.Value))
		}
		http.SetCookie(w, &http.Cookie{Name: sessCookie, Path: "/", MaxAge: -1})
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	A := s.auth
	mux.HandleFunc("GET /api/admin/me", func(w http.ResponseWriter, r *http.Request) {
		id := s.adminID(r)
		if id == 0 {
			fail(w, 401, "Please log in")
			return
		}
		writeJSON(w, 200, s.one(`SELECT id,email,name FROM admins WHERE id=?`, id))
	})
	mux.HandleFunc("GET /api/admin/dashboard", A(s.dashboard))
	mux.HandleFunc("GET /api/admin/enquiries", A(s.listEnquiries))
	mux.HandleFunc("PUT /api/admin/enquiries/{id}", A(s.updateEnquiry))
	mux.HandleFunc("DELETE /api/admin/enquiries/{id}", A(func(w http.ResponseWriter, r *http.Request) {
		s.db.Exec(`DELETE FROM enquiries WHERE id=?`, pid(r))
		writeJSON(w, 200, map[string]any{"ok": true})
	}))
	mux.HandleFunc("POST /api/admin/enquiries/{id}/convert", A(s.convertEnquiry))
	mux.HandleFunc("GET /api/admin/customers", A(s.listCustomers))
	mux.HandleFunc("POST /api/admin/customers", A(s.saveCustomer))
	mux.HandleFunc("PUT /api/admin/customers/{id}", A(s.saveCustomer))
	mux.HandleFunc("DELETE /api/admin/customers/{id}", A(func(w http.ResponseWriter, r *http.Request) {
		s.db.Exec(`DELETE FROM payments WHERE customer_id=?`, pid(r))
		s.db.Exec(`DELETE FROM customers WHERE id=?`, pid(r))
		writeJSON(w, 200, map[string]any{"ok": true})
	}))
	mux.HandleFunc("POST /api/admin/customers/{id}/renew", A(s.renew))
	mux.HandleFunc("POST /api/admin/customers/{id}/remind", A(s.remindNow))
	mux.HandleFunc("GET /api/admin/customers/{id}/payments", A(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, s.qmaps(`SELECT * FROM payments WHERE customer_id=? ORDER BY id DESC`, pid(r)))
	}))
	mux.HandleFunc("GET /api/admin/plans", A(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, s.qmaps(`SELECT * FROM plans ORDER BY sort,id`))
	}))
	mux.HandleFunc("POST /api/admin/plans", A(s.savePlan))
	mux.HandleFunc("PUT /api/admin/plans/{id}", A(s.savePlan))
	mux.HandleFunc("DELETE /api/admin/plans/{id}", A(func(w http.ResponseWriter, r *http.Request) {
		s.db.Exec(`UPDATE plans SET active=0 WHERE id=?`, pid(r)) // hidden, not deleted - old customers still reference it
		writeJSON(w, 200, map[string]any{"ok": true})
	}))
	mux.HandleFunc("GET /api/admin/settings", A(s.getSettings))
	mux.HandleFunc("PUT /api/admin/settings", A(s.putSettings))
	mux.HandleFunc("POST /api/admin/settings/test-email", A(s.testEmail))
	mux.HandleFunc("GET /api/admin/outbox", A(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, s.qmaps(`SELECT id,to_addr,subject,kind,status,error,created_at FROM outbox ORDER BY id DESC LIMIT 200`))
	}))
	mux.HandleFunc("POST /api/admin/outbox/{id}/resend", A(func(w http.ResponseWriter, r *http.Request) {
		m := s.one(`SELECT to_addr,subject,html,text FROM outbox WHERE id=?`, pid(r))
		if m == nil {
			fail(w, 404, "not found")
			return
		}
		s.db.Exec(`UPDATE outbox SET status='queued', error='' WHERE id=?`, pid(r))
		go s.deliver(pid(r), fmt.Sprint(m["to_addr"]), fmt.Sprint(m["subject"]), fmt.Sprint(m["html"]), fmt.Sprint(m["text"]))
		writeJSON(w, 200, map[string]any{"ok": true})
	}))
	mux.HandleFunc("POST /api/admin/password", A(s.changePassword))
	mux.HandleFunc("GET /api/admin/seo", A(s.seoList))
	mux.HandleFunc("PUT /api/admin/seo/page", A(s.seoSave))
}

// ---------- login ----------

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	if !s.lim.allow("login:"+clientIP(r), 20, 10*time.Minute) {
		fail(w, 429, "Too many attempts. Please wait a few minutes.")
		return
	}
	var id, fails, locked int64
	var hash string
	err := s.db.QueryRow(`SELECT id,hash,fails,locked_until FROM admins WHERE email=?`, strings.ToLower(strings.TrimSpace(in.Email))).Scan(&id, &hash, &fails, &locked)
	if err == nil && locked > now() {
		fail(w, 429, "Account locked for a few minutes after too many wrong passwords.")
		return
	}
	if err != nil { // spend the same time as a real check so the e-mail cannot be guessed from timing
		bcrypt.CompareHashAndPassword([]byte("$2a$11$0123456789012345678901uJ1b6o9mQz1x2o4n0m1aQp3m3n5k7yS"), []byte(in.Password))
		fail(w, 401, "E-mail or password is wrong")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fails++
		lock := int64(0)
		if fails >= 5 {
			lock, fails = now()+900, 0
		}
		s.db.Exec(`UPDATE admins SET fails=?, locked_until=? WHERE id=?`, fails, lock, id)
		fail(w, 401, "E-mail or password is wrong")
		return
	}
	s.db.Exec(`UPDATE admins SET fails=0, locked_until=0 WHERE id=?`, id)
	s.db.Exec(`DELETE FROM sessions WHERE expires<?`, now())
	tk := token()
	s.db.Exec(`INSERT INTO sessions(token_hash,admin_id,expires) VALUES(?,?,?)`, sha(tk), id, now()+12*3600)
	http.SetCookie(w, &http.Cookie{Name: sessCookie, Value: tk, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: strings.HasPrefix(s.site, "https://"), MaxAge: 12 * 3600})
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var in struct{ Old, New string }
	if !readJSON(w, r, &in) {
		return
	}
	id := s.adminID(r)
	var hash string
	s.db.QueryRow(`SELECT hash FROM admins WHERE id=?`, id).Scan(&hash)
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Old)) != nil {
		fail(w, 400, "Current password is wrong")
		return
	}
	if len(in.New) < 10 {
		fail(w, 400, "New password must be at least 10 characters")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(in.New), 11)
	s.db.Exec(`UPDATE admins SET hash=? WHERE id=?`, string(h), id)
	c, _ := r.Cookie(sessCookie)
	s.db.Exec(`DELETE FROM sessions WHERE admin_id=? AND token_hash<>?`, id, sha(c.Value)) // log out other devices
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ---------- dashboard ----------

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	cnt := func(q string, a ...any) int64 { var n int64; s.db.QueryRow(q, a...).Scan(&n); return n }
	today := time.Now().Format("2006-01-02")
	in30 := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	year := time.Now().Format("2006") + "-01-01"
	month := time.Now().Format("2006-01") + "-01"
	writeJSON(w, 200, map[string]any{
		"new_enquiries": cnt(`SELECT COUNT(*) FROM enquiries WHERE status='new'`),
		"enquiries":     cnt(`SELECT COUNT(*) FROM enquiries`),
		"active":        cnt(`SELECT COUNT(*) FROM customers WHERE status='active' AND ends>=?`, today),
		"expiring":      cnt(`SELECT COUNT(*) FROM customers WHERE status='active' AND ends>=? AND ends<=?`, today, in30),
		"expired":       cnt(`SELECT COUNT(*) FROM customers WHERE status='active' AND ends<?`, today),
		"revenue_year":  cnt(`SELECT COALESCE(SUM(amount),0) FROM payments WHERE paid_on>=?`, year),
		"revenue_month": cnt(`SELECT COALESCE(SUM(amount),0) FROM payments WHERE paid_on>=?`, month),
		"recent":        s.qmaps(`SELECT id,name,business,city,plan,status,created_at FROM enquiries ORDER BY id DESC LIMIT 6`),
		"renewals":      s.qmaps(`SELECT id,business,contact,phone,email,plan_name,ends FROM customers WHERE status='active' AND ends<=? ORDER BY ends LIMIT 8`, in30),
		"by_month":      s.qmaps(`SELECT strftime('%Y-%m', created_at,'unixepoch') AS m, COUNT(*) AS n FROM enquiries GROUP BY m ORDER BY m DESC LIMIT 6`),
	})
}

// ---------- enquiries ----------

func (s *Server) listEnquiries(w http.ResponseWriter, r *http.Request) {
	q, st := "%"+strings.TrimSpace(r.URL.Query().Get("q"))+"%", r.URL.Query().Get("status")
	writeJSON(w, 200, s.qmaps(`SELECT e.*, COALESCE((SELECT status FROM outbox o WHERE o.to_addr=e.email AND o.kind='enquiry_thanks' ORDER BY o.id DESC LIMIT 1),'') AS mail
		FROM enquiries e WHERE (?='' OR e.status=?) AND (e.name LIKE ? OR e.email LIKE ? OR e.phone LIKE ? OR e.business LIKE ? OR e.city LIKE ?) ORDER BY e.id DESC LIMIT 500`, st, st, q, q, q, q, q))
}

func (s *Server) updateEnquiry(w http.ResponseWriter, r *http.Request) {
	var in struct{ Status, Note string }
	if !readJSON(w, r, &in) {
		return
	}
	switch in.Status {
	case "new", "contacted", "demo", "won", "lost":
	default:
		fail(w, 400, "bad status")
		return
	}
	s.db.Exec(`UPDATE enquiries SET status=?, note=? WHERE id=?`, in.Status, clip(in.Note, 1000), pid(r))
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) convertEnquiry(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PlanID int64 `json:"plan_id"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	e := s.one(`SELECT * FROM enquiries WHERE id=?`, pid(r))
	pl := s.one(`SELECT * FROM plans WHERE id=?`, in.PlanID)
	if e == nil || pl == nil {
		fail(w, 404, "enquiry or plan not found")
		return
	}
	months := toInt(pl["months"])
	start := time.Now()
	res, err := s.db.Exec(`INSERT INTO customers(business,contact,email,phone,city,plan_id,plan_name,amount,starts,ends,notes,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		firstNonEmpty(fmt.Sprint(e["business"]), fmt.Sprint(e["name"])), e["name"], e["email"], e["phone"], e["city"], in.PlanID, pl["name"], toInt(pl["price"]),
		start.Format("2006-01-02"), start.AddDate(0, int(months), 0).Format("2006-01-02"), e["message"], now())
	if err != nil {
		fail(w, 500, "could not create customer")
		return
	}
	cid, _ := res.LastInsertId()
	s.db.Exec(`UPDATE enquiries SET status='won' WHERE id=?`, pid(r))
	s.db.Exec(`INSERT INTO payments(customer_id,amount,plan_name,months,method,paid_on,created_at) VALUES(?,?,?,?,?,?,?)`, cid, toInt(pl["price"]), pl["name"], months, "", start.Format("2006-01-02"), now())
	c := s.one(`SELECT * FROM customers WHERE id=?`, cid)
	s.mailSubscription(c, false)
	writeJSON(w, 200, map[string]any{"ok": true, "customer": cid})
}

func firstNonEmpty(a ...string) string {
	for _, x := range a {
		if strings.TrimSpace(x) != "" && x != "<nil>" {
			return x
		}
	}
	return ""
}

func toInt(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

// ---------- customers & subscriptions ----------

func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request) {
	q, st := "%"+strings.TrimSpace(r.URL.Query().Get("q"))+"%", r.URL.Query().Get("state")
	today := time.Now().Format("2006-01-02")
	in30 := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	where := ""
	switch st {
	case "active":
		where = ` AND status='active' AND ends>='` + today + `'`
	case "expiring":
		where = ` AND status='active' AND ends>='` + today + `' AND ends<='` + in30 + `'`
	case "expired":
		where = ` AND status='active' AND ends<'` + today + `'`
	case "cancelled":
		where = ` AND status='cancelled'`
	}
	rows := s.qmaps(`SELECT *, CASE WHEN status='cancelled' THEN 'cancelled' WHEN ends<? THEN 'expired' WHEN ends<=? THEN 'expiring' ELSE 'active' END AS state
		FROM customers WHERE (business LIKE ? OR contact LIKE ? OR email LIKE ? OR phone LIKE ? OR city LIKE ?)`+where+` ORDER BY ends LIMIT 1000`, today, in30, q, q, q, q, q)
	writeJSON(w, 200, rows)
}

func validDate(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }

func (s *Server) saveCustomer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Business, Contact, Email, Phone, City, Starts, Ends, Notes, Status string
		PlanID                                                             int64 `json:"plan_id"`
		Amount                                                             int64
	}
	if !readJSON(w, r, &in) {
		return
	}
	if clip(in.Business, 100) == "" {
		fail(w, 400, "Business name is required")
		return
	}
	if in.Email != "" {
		if _, err := mail.ParseAddress(in.Email); err != nil {
			fail(w, 400, "E-mail address is not valid")
			return
		}
	}
	if !validDate(in.Starts) || !validDate(in.Ends) || in.Ends < in.Starts {
		fail(w, 400, "Please choose valid start and end dates")
		return
	}
	if in.Status != "cancelled" {
		in.Status = "active"
	}
	planName := ""
	if pl := s.one(`SELECT name FROM plans WHERE id=?`, in.PlanID); pl != nil {
		planName = fmt.Sprint(pl["name"])
	}
	id := pid(r)
	if id == 0 {
		res, err := s.db.Exec(`INSERT INTO customers(business,contact,email,phone,city,plan_id,plan_name,amount,starts,ends,status,notes,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			clip(in.Business, 100), clip(in.Contact, 80), clip(in.Email, 120), clip(in.Phone, 20), clip(in.City, 60), in.PlanID, planName, in.Amount, in.Starts, in.Ends, in.Status, clip(in.Notes, 2000), now())
		if err != nil {
			fail(w, 500, "could not save")
			return
		}
		id, _ = res.LastInsertId()
		if in.Amount > 0 {
			s.db.Exec(`INSERT INTO payments(customer_id,amount,plan_name,months,paid_on,created_at) VALUES(?,?,?,?,?,?)`, id, in.Amount, planName, 0, in.Starts, now())
		}
		if in.Email != "" {
			s.mailSubscription(s.one(`SELECT * FROM customers WHERE id=?`, id), false)
		}
	} else {
		s.db.Exec(`UPDATE customers SET business=?,contact=?,email=?,phone=?,city=?,plan_id=?,plan_name=?,amount=?,starts=?,ends=?,status=?,notes=? WHERE id=?`,
			clip(in.Business, 100), clip(in.Contact, 80), clip(in.Email, 120), clip(in.Phone, 20), clip(in.City, 60), in.PlanID, planName, in.Amount, in.Starts, in.Ends, in.Status, clip(in.Notes, 2000), id)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "id": id})
}

// renew extends from the later of today / current end date, so paying early never loses days.
func (s *Server) renew(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PlanID         int64 `json:"plan_id"`
		Amount, Months int64
		Method, Ref    string
		Notify         bool
	}
	if !readJSON(w, r, &in) {
		return
	}
	c := s.one(`SELECT * FROM customers WHERE id=?`, pid(r))
	pl := s.one(`SELECT * FROM plans WHERE id=?`, in.PlanID)
	if c == nil || pl == nil {
		fail(w, 404, "customer or plan not found")
		return
	}
	months := in.Months
	if months <= 0 {
		months = toInt(pl["months"])
	}
	if months > 60 {
		fail(w, 400, "Too many months")
		return
	}
	amount := in.Amount
	if amount <= 0 {
		amount = toInt(pl["price"])
	}
	base := time.Now()
	if cur, err := time.Parse("2006-01-02", fmt.Sprint(c["ends"])); err == nil && cur.After(base) {
		base = cur
	}
	ends := base.AddDate(0, int(months), 0).Format("2006-01-02")
	s.db.Exec(`UPDATE customers SET plan_id=?,plan_name=?,amount=?,ends=?,status='active',reminded_for='' WHERE id=?`, in.PlanID, pl["name"], amount, ends, pid(r))
	s.db.Exec(`INSERT INTO payments(customer_id,amount,plan_name,months,method,ref,paid_on,created_at) VALUES(?,?,?,?,?,?,?,?)`, pid(r), amount, pl["name"], months, clip(in.Method, 30), clip(in.Ref, 60), time.Now().Format("2006-01-02"), now())
	if in.Notify && fmt.Sprint(c["email"]) != "" {
		s.mailSubscription(s.one(`SELECT * FROM customers WHERE id=?`, pid(r)), true)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "ends": ends})
}

func (s *Server) remindNow(w http.ResponseWriter, r *http.Request) {
	c := s.one(`SELECT * FROM customers WHERE id=?`, pid(r))
	if c == nil || fmt.Sprint(c["email"]) == "" {
		fail(w, 400, "This customer has no e-mail address")
		return
	}
	s.mailReminder(c, daysLeft(fmt.Sprint(c["ends"])))
	writeJSON(w, 200, map[string]any{"ok": true})
}

func daysLeft(ends string) int {
	t, err := time.ParseInLocation("2006-01-02", ends, time.Local)
	if err != nil {
		return 0
	}
	return int(time.Until(t.AddDate(0, 0, 1)).Hours() / 24)
}

// reminderLoop e-mails each customer once, a set number of days before their plan ends.
func (s *Server) reminderLoop() {
	for {
		days, _ := strconv.Atoi(s.setting("reminder_days"))
		if days > 0 {
			limit := time.Now().AddDate(0, 0, days).Format("2006-01-02")
			for _, c := range s.qmaps(`SELECT * FROM customers WHERE status='active' AND email<>'' AND ends>=? AND ends<=? AND reminded_for<>ends`, time.Now().Format("2006-01-02"), limit) {
				s.mailReminder(c, daysLeft(fmt.Sprint(c["ends"])))
				s.db.Exec(`UPDATE customers SET reminded_for=ends WHERE id=?`, c["id"])
			}
		}
		time.Sleep(time.Hour)
	}
}

// ---------- plans ----------

func (s *Server) savePlan(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name, Tagline, Features string
		Months, Price           int64
		Popular, Active         bool
		Sort                    int64
	}
	if !readJSON(w, r, &in) {
		return
	}
	if clip(in.Name, 80) == "" || in.Months < 1 || in.Months > 60 || in.Price < 0 {
		fail(w, 400, "Please fill the plan name, months (1-60) and price")
		return
	}
	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	if id := pid(r); id != 0 {
		s.db.Exec(`UPDATE plans SET name=?,months=?,price=?,tagline=?,features=?,popular=?,active=?,sort=? WHERE id=?`, clip(in.Name, 80), in.Months, in.Price, clip(in.Tagline, 120), clip(in.Features, 2000), b2i(in.Popular), b2i(in.Active), in.Sort, id)
	} else {
		s.db.Exec(`INSERT INTO plans(name,months,price,tagline,features,popular,active,sort) VALUES(?,?,?,?,?,?,?,?)`, clip(in.Name, 80), in.Months, in.Price, clip(in.Tagline, 120), clip(in.Features, 2000), b2i(in.Popular), 1, in.Sort)
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ---------- settings ----------

var settingKeys = []string{"company", "legal_name", "support_email", "phone", "address", "notify_email", "from_name", "reply_to", "smtp_host", "smtp_port", "smtp_user", "smtp_pass", "smtp_from", "reminder_days", "reply_hours", "google_verify", "bing_verify", "ga_id"}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{}
	for _, k := range settingKeys {
		out[k] = s.setting(k)
	}
	out["smtp_pass"] = "" // never sent back to the browser
	out["smtp_pass_set"] = s.setting("smtp_pass") != ""
	writeJSON(w, 200, out)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	var in map[string]string
	if !readJSON(w, r, &in) {
		return
	}
	if v := strings.TrimSpace(in["ga_id"]); v != "" && !gaRe.MatchString(v) {
		fail(w, 400, "Google Analytics ID looks like G-XXXXXXXXXX")
		return
	}
	for _, k := range []string{"google_verify", "bing_verify"} {
		if v := strings.TrimSpace(in[k]); v != "" && !verifyRe.MatchString(v) {
			fail(w, 400, "The site-verification code should be just the code, not the whole tag")
			return
		}
	}
	for _, k := range settingKeys {
		v, ok := in[k]
		if !ok || (k == "smtp_pass" && v == "") { // empty password = keep the old one
			continue
		}
		s.db.Exec(`INSERT INTO settings(k,v) VALUES(?,?) ON CONFLICT(k) DO UPDATE SET v=excluded.v`, k, clip(oneLine(v), 200))
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) testEmail(w http.ResponseWriter, r *http.Request) {
	var in struct{ To string }
	if !readJSON(w, r, &in) {
		return
	}
	if _, err := mail.ParseAddress(in.To); err != nil {
		fail(w, 400, "Please enter a valid e-mail address")
		return
	}
	if s.setting("smtp_host") == "" {
		fail(w, 400, "Fill in and save the e-mail server details first")
		return
	}
	body := p("This is a test message from your <b>" + esc(s.setting("company")) + "</b> admin panel. If you can read this, your e-mail is set up correctly and customers will receive their thank-you e-mails.")
	if err := s.smtpSend(in.To, "Test e-mail from "+s.setting("company"), s.emailLayout("E-mail test", "E-mail is working", body), "Your e-mail setup works."); err != nil {
		fail(w, 400, "Could not send: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

var _ = sql.ErrNoRows

var (
	gaRe     = regexp.MustCompile(`^G-[A-Z0-9]{4,14}$`)
	verifyRe = regexp.MustCompile(`^[A-Za-z0-9_\-]{10,120}$`)
)

// seoList returns every public page with its default and overridden title/description.
func (s *Server) seoList(w http.ResponseWriter, r *http.Request) {
	out := []map[string]any{}
	for _, path := range s.ps.order {
		def := s.ps.pages[path]
		o, _ := s.ps.override(path)
		out = append(out, map[string]any{"path": path, "default_title": def.meta.Title, "default_desc": def.meta.Desc, "title": o.Title, "desc": o.Desc, "noindex": o.NoIndex})
	}
	writeJSON(w, 200, out)
}

func (s *Server) seoSave(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path, Title, Desc string
		NoIndex           bool
	}
	if !readJSON(w, r, &in) {
		return
	}
	if _, ok := s.ps.pages[in.Path]; !ok {
		fail(w, 404, "page not found")
		return
	}
	in.Title, in.Desc = clip(oneLine(in.Title), 110), clip(oneLine(in.Desc), 220)
	if in.Title == "" && in.Desc == "" && !in.NoIndex {
		s.db.Exec(`DELETE FROM seo_overrides WHERE path=?`, in.Path)
	} else {
		ni := 0
		if in.NoIndex {
			ni = 1
		}
		s.db.Exec(`INSERT INTO seo_overrides(path,title,descr,noindex) VALUES(?,?,?,?) ON CONFLICT(path) DO UPDATE SET title=excluded.title, descr=excluded.descr, noindex=excluded.noindex`, in.Path, in.Title, in.Desc, ni)
	}
	s.ps.loadOverrides()
	writeJSON(w, 200, map[string]any{"ok": true})
}
