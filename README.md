# MacBill System - website + subscription admin

One small Go program (`macbill-site.exe`), its own SQLite database in `website/data/`.

| | |
|---|---|
| Website | http://localhost:4000 |
| Admin panel | http://localhost:4000/admin |

Run: `ADMIN_EMAIL=you@example.com ADMIN_PASSWORD=<strong password> go run .`
If you do not set `ADMIN_PASSWORD`, a random password is printed in the console on first start (change it in Admin > Settings).

## Make the e-mails really go out
Admin > **Settings > E-mail server (SMTP)**, e.g. for Gmail / Google Workspace:
host `smtp.gmail.com`, port `587`, login = your address, password = an **App Password**, from = `hello@macbill.in`.
Press **Send test**. Until this is set, every e-mail waits safely in **E-mail log** and can be sent later with *Send again*.
You can also use environment variables instead (password never stored in the database):
`MAIL_HOST`, `MAIL_PORT`, `MAIL_USER`, `MAIL_PASS`, `MAIL_FROM`.

## What happens when someone sends an enquiry
1. It is saved (Admin > Enquiries).
2. The visitor gets an official **"Thank you - we will contact you very soon"** e-mail.
3. Your team gets an alert e-mail (Settings > "Alert me at").

## Subscriptions
Admin > Customers: add a customer or turn an enquiry into one ("Make customer"), **Renew** (adds time after the current end date,
records the payment), see payment history, and automatic renewal reminders by e-mail N days before the end (default 15).
Plans and prices (Admin > Plans) update the website instantly.

## Production
Environment: `ADDR`, `SITE_URL=https://yourdomain`, `ADMIN_PASSWORD`, `TRUST_PROXY=1` (behind nginx/Caddy), `DB_PATH`.
`email-preview.html` shows how the thank-you e-mail looks.

## Pages (29, all SEO-ready)
Home, Features, Pricing, About, Contact, FAQ, Blog (6 guides), Privacy, Terms,
9 product pages (`/pos-billing`, `/kitchen-display`, `/qr-ordering`, `/inventory`, `/accounting`, `/hr-payroll`, `/delivery`, `/customers-loyalty`, `/reports-analytics`)
and 5 solution pages (`/solutions/restaurants`, `cafes`, `cloud-kitchens`, `restaurant-chains`, `bakeries-sweet-shops`).
Page text lives in `content.go` (add a page = add data), layout in `tpl/`.

SEO built in: unique title + description per page, canonical URL, Open Graph/Twitter tags, JSON-LD (Organization, SoftwareApplication with
your live prices, FAQPage, Article, BreadcrumbList), `/sitemap.xml`, `/robots.txt`, a real 404 page, gzip, clean URLs, one `<h1>` per page.
**Before launch set `SITE_URL=https://your-domain`** - the sitemap, canonicals and e-mail links use it. `go test` checks every page for SEO basics and broken links.
