package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

func oneLine(s string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ", "\x00", "").Replace(s))
}

func b64lines(s string) string {
	e := base64.StdEncoding.EncodeToString([]byte(s))
	var b strings.Builder
	for len(e) > 76 {
		b.WriteString(e[:76] + "\r\n")
		e = e[76:]
	}
	return b.String() + e
}

// queueMail stores the message in the outbox and sends it in the background.
// Nothing is ever lost: if SMTP is not set up or fails, it stays in the outbox and can be re-sent from the admin panel.
func (s *Server) queueMail(to, subject, kind, htmlBody, textBody string) int64 {
	to, subject = oneLine(to), oneLine(subject)
	res, err := s.db.Exec(`INSERT INTO outbox(to_addr,subject,kind,status,html,text,created_at) VALUES(?,?,?,?,?,?,?)`, to, subject, kind, "queued", htmlBody, textBody, now())
	if err != nil {
		return 0
	}
	id, _ := res.LastInsertId()
	go s.deliver(id, to, subject, htmlBody, textBody)
	return id
}

func (s *Server) markMail(id int64, status, errMsg string) {
	s.db.Exec(`UPDATE outbox SET status=?, error=? WHERE id=?`, status, clip(errMsg, 300), id)
}

func (s *Server) deliver(id int64, to, subject, htmlBody, textBody string) {
	if s.setting("smtp_host") == "" {
		s.markMail(id, "not_configured", "E-mail server (SMTP) is not set up yet - open Admin > Settings > E-mail")
		return
	}
	if err := s.smtpSend(to, subject, htmlBody, textBody); err != nil {
		s.markMail(id, "failed", err.Error())
		return
	}
	s.markMail(id, "sent", "")
}

func (s *Server) smtpSend(to, subject, htmlBody, textBody string) error {
	host, port, user, pass := s.setting("smtp_host"), s.setting("smtp_port"), s.setting("smtp_user"), s.setting("smtp_pass")
	if port == "" {
		port = "587"
	}
	fromAddr := s.setting("smtp_from")
	if fromAddr == "" {
		fromAddr = user
	}
	if _, err := mail.ParseAddress(fromAddr); err != nil {
		return fmt.Errorf("the 'from' e-mail address is not valid")
	}
	rcpt, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient address")
	}
	name := oneLine(s.setting("from_name"))
	if name == "" {
		name = "MacBill System"
	}
	boundary := "mb" + token()[:24]
	var m strings.Builder
	h := func(k, v string) { m.WriteString(k + ": " + v + "\r\n") }
	h("From", (&mail.Address{Name: name, Address: fromAddr}).String())
	h("To", rcpt.String())
	if rt := oneLine(s.setting("reply_to")); rt != "" {
		if a, e := mail.ParseAddress(rt); e == nil {
			h("Reply-To", a.String())
		}
	}
	h("Subject", "=?UTF-8?B?"+base64.StdEncoding.EncodeToString([]byte(subject))+"?=")
	h("Date", time.Now().Format(time.RFC1123Z))
	h("Message-ID", "<"+token()[:20]+"@macbill>")
	h("MIME-Version", "1.0")
	h("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
	m.WriteString("\r\n--" + boundary + "\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n" + b64lines(textBody))
	m.WriteString("\r\n--" + boundary + "\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n" + b64lines(htmlBody))
	m.WriteString("\r\n--" + boundary + "--\r\n")

	addr := net.JoinHostPort(host, port)
	d := net.Dialer{Timeout: 15 * time.Second}
	var c *smtp.Client
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if port == "465" {
		conn, err := tls.DialWithDialer(&d, "tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		if c, err = smtp.NewClient(conn, host); err != nil {
			return err
		}
	} else {
		conn, err := d.Dial("tcp", addr)
		if err != nil {
			return err
		}
		if c, err = smtp.NewClient(conn, host); err != nil {
			return err
		}
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsCfg); err != nil {
				return err
			}
		}
	}
	defer c.Close()
	if user != "" {
		if err := c.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
			return fmt.Errorf("login to the e-mail server failed: %v", err)
		}
	}
	if err := c.Mail(fromAddr); err != nil {
		return err
	}
	if err := c.Rcpt(rcpt.Address); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write([]byte(m.String())); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// ---------- official e-mail design ----------

func esc(s string) string { return html.EscapeString(s) }

// emailLayout wraps content in the MacBill System letterhead (table layout = works in Gmail, Outlook, phones).
func (s *Server) emailLayout(preheader, heading, body string) string {
	company, phone, support := esc(s.setting("company")), esc(s.setting("phone")), esc(s.setting("support_email"))
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + esc(heading) + `</title></head>
<body style="margin:0;padding:0;background:#f1f3f9;font-family:'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#1e293b">
<span style="display:none;max-height:0;overflow:hidden;opacity:0">` + esc(preheader) + `</span>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f1f3f9"><tr><td align="center" style="padding:28px 12px">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background:#ffffff;border-radius:18px;overflow:hidden;box-shadow:0 8px 30px rgba(30,41,90,.12)">
<tr><td style="background:#0f172a;background-image:linear-gradient(135deg,#0f172a,#1e1b4b);padding:26px 32px">
 <table role="presentation" width="100%"><tr><td style="font-size:24px;font-weight:800;letter-spacing:-.5px;color:#ffffff"><span style="display:inline-block;background:#f97316;color:#fff;border-radius:9px;padding:3px 9px;margin-right:9px">M</span>` + company + `</td>
 <td align="right" style="font-size:12px;color:#a5b4fc;letter-spacing:1px;text-transform:uppercase">Restaurant billing &amp; ERP</td></tr></table></td></tr>
<tr><td style="height:4px;background:#f97316;line-height:4px;font-size:0">&nbsp;</td></tr>
<tr><td style="padding:34px 32px 8px"><h1 style="margin:0 0 14px;font-size:24px;line-height:1.3;color:#0f172a">` + esc(heading) + `</h1>` + body + `</td></tr>
<tr><td style="padding:8px 32px 30px"><p style="margin:18px 0 0;font-size:15px;line-height:1.6;color:#334155">Warm regards,<br><b>Team ` + company + `</b></p></td></tr>
<tr><td style="background:#f8fafc;border-top:1px solid #e2e8f0;padding:20px 32px;font-size:12.5px;line-height:1.7;color:#64748b">
 <b style="color:#334155">Need help?</b> Write to <a href="mailto:` + support + `" style="color:#ea580c;text-decoration:none">` + support + `</a> or call <b>` + phone + `</b>.<br>
 This is an automated message from ` + company + `. Please do not share it with anyone you do not trust.</td></tr>
</table>
<p style="font-size:11.5px;color:#94a3b8;margin:14px 0 0">&copy; ` + fmt.Sprint(time.Now().Year()) + ` ` + company + `. All rights reserved.</p>
</td></tr></table></body></html>`
}

func p(s string) string {
	return `<p style="margin:0 0 14px;font-size:15.5px;line-height:1.65;color:#334155">` + s + `</p>`
}

func box(rows ...[2]string) string {
	var b strings.Builder
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin:6px 0 18px;background:#fff7ed;border:1px solid #fed7aa;border-radius:12px"><tr><td style="padding:14px 18px">`)
	for _, r := range rows {
		if r[1] == "" {
			continue
		}
		b.WriteString(`<div style="font-size:14px;line-height:1.9;color:#431407"><span style="color:#9a3412;display:inline-block;min-width:110px">` + esc(r[0]) + `</span><b>` + esc(r[1]) + `</b></div>`)
	}
	return b.String() + `</td></tr></table>`
}

func (s *Server) mailEnquiryThanks(e map[string]string) {
	hrs := s.setting("reply_hours")
	if hrs == "" {
		hrs = "24"
	}
	company := s.setting("company")
	body := p("Hi <b>"+esc(e["name"])+"</b>,") +
		p("Thank you for your interest in <b>"+esc(company)+"</b>. We have received your enquiry and our team will <b>contact you very soon</b> - usually within "+esc(hrs)+" hours.") +
		p("Here is what you sent us:") +
		box([2]string{"Business", e["business"]}, [2]string{"City", e["city"]}, [2]string{"Phone", e["phone"]}, [2]string{"Plan of interest", e["plan"]}, [2]string{"Your message", e["message"]}) +
		p("In the meantime, please keep ready the number of outlets and staff you have - it helps us suggest the right plan and set up your demo faster.")
	text := fmt.Sprintf("Hi %s,\n\nThank you for your interest in %s. We have received your enquiry and our team will contact you very soon (usually within %s hours).\n\nWarm regards,\nTeam %s\n%s | %s", e["name"], company, hrs, company, s.setting("support_email"), s.setting("phone"))
	s.queueMail(e["email"], "Thank you for contacting "+company, "enquiry_thanks", s.emailLayout("We got your enquiry - our team will contact you very soon.", "Thank you for contacting us", body), text)
}

func (s *Server) mailEnquiryAlert(e map[string]string, id int64) {
	to := s.setting("notify_email")
	if to == "" {
		to = s.setting("support_email")
	}
	body := p("A new enquiry just came in from the website.") + box([2]string{"Name", e["name"]}, [2]string{"E-mail", e["email"]}, [2]string{"Phone", e["phone"]}, [2]string{"Business", e["business"]}, [2]string{"City", e["city"]}, [2]string{"Plan", e["plan"]}, [2]string{"Message", e["message"]}) +
		p(`<a href="`+esc(s.site)+`/admin" style="display:inline-block;background:#f97316;color:#fff;text-decoration:none;font-weight:700;padding:11px 22px;border-radius:10px">Open in admin</a>`)
	s.queueMail(to, fmt.Sprintf("New enquiry #%d - %s", id, e["name"]), "enquiry_alert", s.emailLayout("New website enquiry", "New enquiry received", body),
		fmt.Sprintf("New enquiry #%d\n%s <%s> %s\n%s %s\n%s", id, e["name"], e["email"], e["phone"], e["business"], e["city"], e["message"]))
}

func (s *Server) mailSubscription(c map[string]any, renewed bool) {
	company := s.setting("company")
	title := "Your subscription is active"
	if renewed {
		title = "Your subscription has been renewed"
	}
	body := p("Hi <b>"+esc(fmt.Sprint(c["contact"]))+"</b>,") +
		p("Thank you for choosing <b>"+esc(company)+"</b>. Here are your subscription details:") +
		box([2]string{"Business", fmt.Sprint(c["business"])}, [2]string{"Plan", fmt.Sprint(c["plan_name"])}, [2]string{"Valid from", fmt.Sprint(c["starts"])}, [2]string{"Valid until", fmt.Sprint(c["ends"])}, [2]string{"Amount", "Rs. " + fmt.Sprint(c["amount"])}) +
		p("If you need anything - training for new staff, extra logins or a new outlet - just reply to this e-mail.")
	s.queueMail(fmt.Sprint(c["email"]), title+" - "+company, "subscription", s.emailLayout(title, title, body),
		fmt.Sprintf("%s\nPlan: %v\nValid until: %v\n\nTeam %s", title, c["plan_name"], c["ends"], company))
}

func (s *Server) mailReminder(c map[string]any, days int) {
	company := s.setting("company")
	when := fmt.Sprintf("in %d days", days)
	if days <= 0 {
		when = "today"
	}
	body := p("Hi <b>"+esc(fmt.Sprint(c["contact"]))+"</b>,") +
		p("This is a friendly reminder that your <b>"+esc(company)+"</b> subscription expires <b>"+when+"</b> (on <b>"+esc(fmt.Sprint(c["ends"]))+"</b>). Renew now so your billing, kitchen and reports keep running without a break.") +
		box([2]string{"Business", fmt.Sprint(c["business"])}, [2]string{"Plan", fmt.Sprint(c["plan_name"])}, [2]string{"Expires on", fmt.Sprint(c["ends"])}) +
		p("Just reply to this e-mail or call us and we will renew it for you.")
	s.queueMail(fmt.Sprint(c["email"]), "Your "+company+" subscription expires "+when, "reminder", s.emailLayout("Subscription expiring "+when, "Your subscription expires "+when, body),
		fmt.Sprintf("Your %s subscription expires %s (%v). Reply to this e-mail to renew.", company, when, c["ends"]))
}
