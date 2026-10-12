(() => {
  const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const inr = n => '₹' + Number(n || 0).toLocaleString('en-IN');
  const when = t => t ? new Date(t * 1000).toLocaleString('en-IN', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }) : '';
  const chip = s => `<span class="chip ${esc(s)}">${esc(String(s).replace('_', ' '))}</span>`;
  const today = () => new Date().toISOString().slice(0, 10);
  const addMonths = (d, m) => { const x = new Date(d + 'T00:00:00'); const day = x.getDate(); x.setMonth(x.getMonth() + (+m)); if (x.getDate() !== day) x.setDate(0); return new Date(x.getTime() - x.getTimezoneOffset() * 6e4).toISOString().slice(0, 10); };
  let ME = null, PLANS = [];

  async function api(path, method = 'GET', body) {
    const r = await fetch('/api/admin' + path, { method, headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'macbill' }, body: body === undefined ? undefined : JSON.stringify(body) });
    const j = await r.json().catch(() => ({}));
    if (r.status === 401 && path !== '/login') { showLogin(); throw new Error('Please log in'); }
    if (!r.ok) throw new Error(j.error || 'Something went wrong');
    return j;
  }
  function toast(m, bad) { const d = document.createElement('div'); d.textContent = m; if (bad) d.className = 'bad'; $('#toast').append(d); setTimeout(() => d.remove(), 3800); }
  const run = async (fn, ok) => { try { const r = await fn(); if (ok) toast(ok); return r === undefined ? true : r; } catch (e) { toast(e.message, true); return false; } };
  function modal(html) { $('#modal').innerHTML = `<div class="mbox">${html}</div>`; const m = $('#modal'); m.onclick = e => { if (e.target === m) closeModal(); }; return $('.mbox'); }
  function closeModal() { $('#modal').innerHTML = ''; }
  document.addEventListener('keydown', e => { if (e.key === 'Escape') closeModal(); });
  const confirmBox = msg => new Promise(res => { const b = modal(`<h2>Please confirm</h2><p>${esc(msg)}</p><div class="mact"><button class="btn" id="n">Cancel</button><button class="btn pri" id="y">Yes, continue</button></div>`); $('#n', b).onclick = () => { closeModal(); res(false); }; $('#y', b).onclick = () => { closeModal(); res(true); }; });

  /* ---------- login ---------- */
  function showLogin() { $('#app').hidden = true; $('#login').hidden = false; }
  $('#lf').onsubmit = async e => {
    e.preventDefault(); const er = $('#lerr'); er.hidden = true;
    try { await api('/login', 'POST', { email: $('#le').value, password: $('#lp').value }); $('#lp').value = ''; boot(); }
    catch (x) { er.textContent = x.message; er.hidden = false; }
  };
  $('#out').onclick = async () => { await api('/logout', 'POST').catch(() => { }); showLogin(); };
  $('#mb').onclick = () => $('#side').classList.toggle('open');

  /* ---------- router ---------- */
  const PAGES = {
    dashboard: ['📊', 'Dashboard', dashboard], enquiries: ['📨', 'Enquiries', enquiries], customers: ['🏪', 'Customers & subscriptions', customers],
    plans: ['💎', 'Plans & pricing', plans], seo: ['🔎', 'SEO', seo], emails: ['✉️', 'E-mail log', emails], settings: ['⚙️', 'Settings', settings],
  };
  let page = 'dashboard', newCount = 0;
  function renderMenu() {
    $('#menu').innerHTML = Object.entries(PAGES).map(([k, v]) => `<a data-p="${k}" class="${k === page ? 'on' : ''}">${v[0]} ${esc(v[1].split(' & ')[0])}${k === 'enquiries' && newCount ? `<span class="cnt">${newCount}</span>` : ''}</a>`).join('');
  }
  $('#menu').onclick = e => { const a = e.target.closest('[data-p]'); if (a) { location.hash = '#' + a.dataset.p; $('#side').classList.remove('open'); } };
  async function route() {
    page = PAGES[location.hash.slice(1)] ? location.hash.slice(1) : 'dashboard';
    $('#title').textContent = PAGES[page][1]; renderMenu();
    $('#view').innerHTML = '<div class="empty">Loading…</div>';
    try { await PAGES[page][2]($('#view')); } catch (e) { $('#view').innerHTML = `<div class="err">${esc(e.message)}</div>`; }
  }
  window.addEventListener('hashchange', () => ME && route());
  async function boot() {
    try { ME = await api('/me'); } catch (e) { return showLogin(); }
    $('#login').hidden = true; $('#app').hidden = false; $('#who').textContent = ME.email;
    PLANS = await api('/plans');
    const d = await api('/dashboard').catch(() => ({})); newCount = d.new_enquiries || 0;
    route();
  }

  /* ---------- dashboard ---------- */
  async function dashboard(el) {
    const d = await api('/dashboard'); newCount = d.new_enquiries; renderMenu();
    el.innerHTML = `<div class="cards">
      <div class="glass stat ${d.new_enquiries ? 'hot' : ''}"><span>New enquiries</span><b>${d.new_enquiries}</b></div>
      <div class="glass stat"><span>Active customers</span><b>${d.active}</b></div>
      <div class="glass stat ${d.expiring ? 'hot' : ''}"><span>Expiring in 30 days</span><b>${d.expiring}</b></div>
      <div class="glass stat"><span>Expired (not renewed)</span><b>${d.expired}</b></div>
      <div class="glass stat"><span>Collected this month</span><b>${inr(d.revenue_month)}</b></div>
      <div class="glass stat"><span>Collected this year</span><b>${inr(d.revenue_year)}</b></div></div>
      <div class="cols">
      <div class="glass card"><h3>Latest enquiries</h3>${d.recent.length ? d.recent.map(e => `<div class="row"><span><b>${esc(e.name)}</b> <span class="muted">${esc(e.business || e.city || '')}</span><br><small class="muted">${when(e.created_at)}</small></span>${chip(e.status)}</div>`).join('') : '<div class="empty">No enquiries yet</div>'}<div style="margin-top:12px"><a class="btn sm" href="#enquiries">Open all enquiries</a></div></div>
      <div class="glass card"><h3>Renewals due</h3>${d.renewals.length ? d.renewals.map(c => `<div class="row"><span><b>${esc(c.business)}</b><br><small class="muted">${esc(c.plan_name)} · ${esc(c.phone || c.email)}</small></span><span style="text-align:right">${esc(c.ends)}<br>${new Date(c.ends + 'T23:59:59') < new Date() ? chip('expired') : chip('expiring')}</span></div>`).join('') : '<div class="empty">Nothing due - good!</div>'}<div style="margin-top:12px"><a class="btn sm" href="#customers">Open customers</a></div></div></div>`;
  }

  /* ---------- enquiries ---------- */
  async function enquiries(el) {
    el.innerHTML = `<div class="bar"><input id="q" type="search" placeholder="🔍 Search name, e-mail, phone, city…"><select id="st"><option value="">All statuses</option><option>new</option><option>contacted</option><option>demo</option><option>won</option><option>lost</option></select><span class="sp"></span></div><div id="tb"></div>`;
    const load = async () => {
      const rows = await api('/enquiries?q=' + encodeURIComponent($('#q').value) + '&status=' + $('#st').value);
      $('#tb').innerHTML = rows.length ? `<div class="tw glass"><table><thead><tr><th>Received</th><th>Name</th><th>Contact</th><th>Business</th><th>Plan</th><th>Status</th><th>Thank-you mail</th><th></th></tr></thead><tbody>${rows.map(r => `<tr><td>${when(r.created_at)}</td><td><b>${esc(r.name)}</b></td><td>${esc(r.phone)}<br><small class="muted">${esc(r.email)}</small></td><td>${esc(r.business)}<br><small class="muted">${esc(r.city)}</small></td><td>${esc(r.plan)}</td><td>${chip(r.status)}</td><td>${r.mail ? chip(r.mail) : '-'}</td><td><button class="btn sm" data-open="${r.id}">Open</button></td></tr>`).join('')}</tbody></table></div>` : '<div class="glass empty">No enquiries found.</div>';
      $$('[data-open]', el).forEach(b => b.onclick = () => openEnquiry(rows.find(r => r.id == b.dataset.open), load));
    };
    let t; $('#q').oninput = () => { clearTimeout(t); t = setTimeout(load, 250); }; $('#st').onchange = load; load();
  }
  function openEnquiry(r, done) {
    const b = modal(`<h2>${esc(r.name)}</h2><div class="muted">${when(r.created_at)}</div>
      <div class="fg"><div><label>E-mail</label><div>${esc(r.email)}</div></div><div><label>Phone</label><div>${esc(r.phone) || '-'}</div></div><div><label>Business</label><div>${esc(r.business) || '-'}</div></div><div><label>City</label><div>${esc(r.city) || '-'}</div></div><div class="full"><label>Plan of interest</label><div>${esc(r.plan) || '-'}</div></div>
      <div class="full"><label>Message</label><div class="pre">${esc(r.message) || '-'}</div></div>
      <div><label>Status</label><select id="s">${['new', 'contacted', 'demo', 'won', 'lost'].map(x => `<option ${x === r.status ? 'selected' : ''}>${x}</option>`).join('')}</select></div>
      <div class="full"><label>Your notes (private)</label><textarea id="nt">${esc(r.note)}</textarea></div></div>
      <div class="mact"><button class="btn bad" id="d">Delete</button><a class="btn" href="mailto:${esc(r.email)}">✉ Reply</a><button class="btn ok" id="cv">➜ Make customer</button><button class="btn pri" id="sv">Save</button></div>`);
    $('#sv', b).onclick = async () => { if (await run(() => api('/enquiries/' + r.id, 'PUT', { status: $('#s', b).value, note: $('#nt', b).value }), 'Saved')) { closeModal(); done(); } };
    $('#d', b).onclick = async () => { if (await confirmBox('Delete this enquiry permanently?')) { await run(() => api('/enquiries/' + r.id, 'DELETE'), 'Deleted'); closeModal(); done(); } };
    $('#cv', b).onclick = () => {
      const b2 = modal(`<h2>Make ${esc(r.business || r.name)} a customer</h2><p class="muted">Starts today. A confirmation e-mail with the plan and end date is sent to ${esc(r.email)}.</p><label>Plan</label><select id="p">${PLANS.filter(p => p.active).map(p => `<option value="${p.id}">${esc(p.name)} - ${inr(p.price)}</option>`).join('')}</select><div class="mact"><button class="btn" id="c">Cancel</button><button class="btn pri" id="go">Create & send e-mail</button></div>`);
      $('#c', b2).onclick = closeModal;
      $('#go', b2).onclick = async () => { if (await run(() => api('/enquiries/' + r.id + '/convert', 'POST', { plan_id: +$('#p', b2).value }), 'Customer created')) { closeModal(); location.hash = '#customers'; } };
    };
  }

  /* ---------- customers ---------- */
  async function customers(el) {
    el.innerHTML = `<div class="bar"><input id="q" type="search" placeholder="🔍 Search business, name, phone…"><select id="st"><option value="">All</option><option value="active">Active</option><option value="expiring">Expiring in 30 days</option><option value="expired">Expired</option><option value="cancelled">Cancelled</option></select><span class="sp"></span><button class="btn pri" id="add">+ Add customer</button></div><div id="tb"></div>`;
    const load = async () => {
      const rows = await api('/customers?q=' + encodeURIComponent($('#q').value) + '&state=' + $('#st').value);
      $('#tb').innerHTML = rows.length ? `<div class="tw glass"><table><thead><tr><th>Business</th><th>Contact</th><th>Plan</th><th>Valid until</th><th>Status</th><th></th></tr></thead><tbody>${rows.map(c => `<tr><td><b>${esc(c.business)}</b><br><small class="muted">${esc(c.city)}</small></td><td>${esc(c.contact)}<br><small class="muted">${esc(c.phone)} ${esc(c.email)}</small></td><td>${esc(c.plan_name)}<br><small class="muted">${inr(c.amount)}</small></td><td><b>${esc(c.ends)}</b><br><small class="muted">from ${esc(c.starts)}</small></td><td>${chip(c.state)}</td>
        <td><div class="acts"><button class="btn sm ok" data-r="${c.id}">Renew</button><button class="btn sm" data-e="${c.id}">Edit</button>${c.email ? `<button class="btn sm" data-m="${c.id}" title="Send renewal reminder">🔔</button>` : ''}<button class="btn sm" data-h="${c.id}">Payments</button></div></td></tr>`).join('')}</tbody></table></div>` : '<div class="glass empty">No customers yet. Add one, or convert an enquiry.</div>';
      const by = id => rows.find(c => c.id == id);
      $$('[data-r]', el).forEach(b => b.onclick = () => renewModal(by(b.dataset.r), load));
      $$('[data-e]', el).forEach(b => b.onclick = () => customerModal(by(b.dataset.e), load));
      $$('[data-m]', el).forEach(b => b.onclick = async () => { if (await confirmBox('Send a renewal reminder e-mail to ' + by(b.dataset.m).email + '?')) run(() => api('/customers/' + b.dataset.m + '/remind', 'POST', {}), 'Reminder queued'); });
      $$('[data-h]', el).forEach(b => b.onclick = () => payments(by(b.dataset.h)));
    };
    let t; $('#q').oninput = () => { clearTimeout(t); t = setTimeout(load, 250); }; $('#st').onchange = load; $('#add').onclick = () => customerModal(null, load); load();
  }
  function customerModal(c, done) {
    c = c || { business: '', contact: '', email: '', phone: '', city: '', plan_id: PLANS.find(p => p.active)?.id || 0, amount: 0, starts: today(), ends: '', status: 'active', notes: '' };
    const plansOpt = PLANS.filter(p => p.active || p.id == c.plan_id).map(p => `<option value="${p.id}" ${p.id == c.plan_id ? 'selected' : ''}>${esc(p.name)}</option>`).join('');
    const b = modal(`<h2>${c.id ? 'Edit customer' : 'Add customer'}</h2><div class="fg">
      <div class="full"><label>Business / restaurant *</label><input id="bu" value="${esc(c.business)}"></div>
      <div><label>Contact person</label><input id="co" value="${esc(c.contact)}"></div><div><label>Phone</label><input id="ph" value="${esc(c.phone)}"></div>
      <div><label>E-mail</label><input id="em" type="email" value="${esc(c.email)}"></div><div><label>City</label><input id="ci" value="${esc(c.city)}"></div>
      <div><label>Plan</label><select id="pl">${plansOpt}</select></div><div><label>Amount paid (₹)</label><input id="am" type="number" min="0" value="${c.amount}"></div>
      <div><label>Starts</label><input id="s" type="date" value="${esc(c.starts)}"></div><div><label>Ends</label><input id="e" type="date" value="${esc(c.ends)}"></div>
      <div><label>Status</label><select id="sx"><option value="active" ${c.status !== 'cancelled' ? 'selected' : ''}>Active</option><option value="cancelled" ${c.status === 'cancelled' ? 'selected' : ''}>Cancelled</option></select></div>
      <div class="full"><label>Notes</label><textarea id="no">${esc(c.notes)}</textarea></div></div>
      <div class="err" id="er" hidden></div><div class="mact">${c.id ? '<button class="btn bad" id="d">Delete</button>' : ''}<button class="btn" id="c">Cancel</button><button class="btn pri" id="sv">Save</button></div>`);
    const fill = () => { const p = PLANS.find(x => x.id == $('#pl', b).value); if (!p) return; if (!c.id || confirmed) { $('#am', b).value = p.price; } if ($('#s', b).value) $('#e', b).value = addMonths($('#s', b).value, p.months); };
    let confirmed = false;
    $('#pl', b).onchange = () => { confirmed = true; fill(); }; $('#s', b).onchange = () => { const p = PLANS.find(x => x.id == $('#pl', b).value); if (p) $('#e', b).value = addMonths($('#s', b).value, p.months); };
    if (!c.id) fill();
    $('#c', b).onclick = closeModal;
    if (c.id) $('#d', b).onclick = async () => { if (await confirmBox('Delete this customer and their payment history?')) { await run(() => api('/customers/' + c.id, 'DELETE'), 'Deleted'); closeModal(); done(); } };
    $('#sv', b).onclick = async () => {
      const body = { business: $('#bu', b).value, contact: $('#co', b).value, phone: $('#ph', b).value, email: $('#em', b).value, city: $('#ci', b).value, plan_id: +$('#pl', b).value, amount: +$('#am', b).value || 0, starts: $('#s', b).value, ends: $('#e', b).value, status: $('#sx', b).value, notes: $('#no', b).value };
      try { await api('/customers' + (c.id ? '/' + c.id : ''), c.id ? 'PUT' : 'POST', body); closeModal(); toast('Saved'); done(); } catch (x) { $('#er', b).textContent = x.message; $('#er', b).hidden = false; }
    };
  }
  function renewModal(c, done) {
    const cur = PLANS.find(p => p.id == c.plan_id) || PLANS[0];
    const b = modal(`<h2>Renew ${esc(c.business)}</h2><p class="muted">Currently valid until <b>${esc(c.ends)}</b>. The new period is added after that date (or from today if it already ended).</p><div class="fg">
      <div class="full"><label>Plan</label><select id="pl">${PLANS.filter(p => p.active || p.id == c.plan_id).map(p => `<option value="${p.id}" ${p.id == cur?.id ? 'selected' : ''}>${esc(p.name)} - ${inr(p.price)}</option>`).join('')}</select></div>
      <div><label>Months</label><input id="mo" type="number" min="1" max="60" value="${cur?.months || 12}"></div><div><label>Amount received (₹)</label><input id="am" type="number" min="0" value="${cur?.price || 0}"></div>
      <div><label>Payment method</label><select id="me"><option>UPI</option><option>Bank transfer</option><option>Cash</option><option>Card</option><option>Cheque</option></select></div><div><label>Reference / UTR</label><input id="rf"></div>
      <div class="full"><label style="display:flex;gap:8px;align-items:center;font-weight:600"><input type="checkbox" id="nf" style="width:auto" ${c.email ? 'checked' : 'disabled'}> E-mail the renewal confirmation to the customer</label></div></div>
      <div class="mact"><button class="btn" id="c">Cancel</button><button class="btn pri" id="go">Renew</button></div>`);
    $('#pl', b).onchange = () => { const p = PLANS.find(x => x.id == $('#pl', b).value); $('#mo', b).value = p.months; $('#am', b).value = p.price; };
    $('#c', b).onclick = closeModal;
    $('#go', b).onclick = async () => { const r = await run(() => api('/customers/' + c.id + '/renew', 'POST', { plan_id: +$('#pl', b).value, months: +$('#mo', b).value, amount: +$('#am', b).value, method: $('#me', b).value, ref: $('#rf', b).value, notify: $('#nf', b).checked })); if (r) { toast('Renewed until ' + r.ends); closeModal(); done(); } };
  }
  async function payments(c) {
    const rows = await api('/customers/' + c.id + '/payments');
    const b = modal(`<h2>${esc(c.business)}</h2><p class="muted">Payment history</p>${rows.length ? `<div class="tw"><table><thead><tr><th>Date</th><th>Plan</th><th>Months</th><th>Amount</th><th>Method</th></tr></thead><tbody>${rows.map(p => `<tr><td>${esc(p.paid_on)}</td><td>${esc(p.plan_name)}</td><td>${p.months || '-'}</td><td><b>${inr(p.amount)}</b></td><td>${esc(p.method)} ${esc(p.ref)}</td></tr>`).join('')}</tbody></table></div>` : '<div class="empty">No payments recorded</div>'}<div class="mact"><button class="btn pri" id="c">Close</button></div>`);
    $('#c', b).onclick = closeModal;
  }

  /* ---------- plans ---------- */
  async function plans(el) {
    PLANS = await api('/plans');
    el.innerHTML = `<div class="hint">These plans appear on your website's pricing section instantly. Put each feature on its own line.</div><div class="bar"><span class="sp"></span><button class="btn pri" id="add">+ New plan</button></div>
      <div class="tw glass"><table><thead><tr><th>Plan</th><th>Period</th><th>Price</th><th>Shown on site</th><th></th></tr></thead><tbody>${PLANS.map(p => `<tr><td><b>${esc(p.name)}</b>${p.popular ? ' ⭐' : ''}<br><small class="muted">${esc(p.tagline)}</small></td><td>${p.months} months</td><td><b>${inr(p.price)}</b></td><td>${p.active ? chip('active') : chip('cancelled').replace('cancelled', 'hidden')}</td><td><button class="btn sm" data-e="${p.id}">Edit</button></td></tr>`).join('')}</tbody></table></div>`;
    $('#add', el).onclick = () => planModal(null); $$('[data-e]', el).forEach(b => b.onclick = () => planModal(PLANS.find(p => p.id == b.dataset.e)));
  }
  function planModal(p) {
    p = p || { name: '', months: 12, price: 5999, tagline: '', features: '', popular: 0, active: 1, sort: PLANS.length };
    const b = modal(`<h2>${p.id ? 'Edit plan' : 'New plan'}</h2><div class="fg"><div class="full"><label>Plan name *</label><input id="n" value="${esc(p.name)}"></div>
      <div><label>Months *</label><input id="m" type="number" min="1" max="60" value="${p.months}"></div><div><label>Price (₹) *</label><input id="pr" type="number" min="0" value="${p.price}"></div>
      <div class="full"><label>Short line under the name</label><input id="t" value="${esc(p.tagline)}"></div>
      <div class="full"><label>Features (one per line)</label><textarea id="f" style="min-height:140px">${esc(String(p.features).split('|').join('\n'))}</textarea></div>
      <div><label>Order on site</label><input id="so" type="number" value="${p.sort}"></div>
      <div><label>&nbsp;</label><label style="display:flex;gap:8px;align-items:center;font-weight:600"><input type="checkbox" id="po" style="width:auto" ${p.popular ? 'checked' : ''}> ⭐ Most popular</label>${p.id ? `<label style="display:flex;gap:8px;align-items:center;font-weight:600"><input type="checkbox" id="ac" style="width:auto" ${p.active ? 'checked' : ''}> Show on website</label>` : ''}</div></div>
      <div class="err" id="er" hidden></div><div class="mact"><button class="btn" id="c">Cancel</button><button class="btn pri" id="sv">Save</button></div>`);
    $('#c', b).onclick = closeModal;
    $('#sv', b).onclick = async () => {
      const body = { name: $('#n', b).value, months: +$('#m', b).value, price: +$('#pr', b).value, tagline: $('#t', b).value, features: $('#f', b).value.split('\n').map(x => x.trim()).filter(Boolean).join('|'), popular: $('#po', b).checked, active: p.id ? $('#ac', b).checked : true, sort: +$('#so', b).value || 0 };
      try { await api('/plans' + (p.id ? '/' + p.id : ''), p.id ? 'PUT' : 'POST', body); closeModal(); toast('Plan saved'); route(); } catch (x) { $('#er', b).textContent = x.message; $('#er', b).hidden = false; }
    };
  }


  /* ---------- SEO ---------- */
  async function seo(el) {
    const [rows, s] = await Promise.all([api('/seo'), api('/settings')]);
    const lenCls = (n, lo, hi) => n === 0 ? 'chip' : (n >= lo && n <= hi ? 'chip sent' : 'chip queued');
    el.innerHTML = `<div class="hint good">Every page already has its own title, description, canonical link, social card and structured data. Use this screen to fine-tune the words people see in Google, and to connect Google tools.</div>
      <div class="cols"><div class="glass card"><h3>Connect Google &amp; Bing</h3>
        <label>Google Search Console verification code</label><input id="gv" value="${esc(s.google_verify)}" placeholder="the code only, e.g. AbC123…">
        <label>Bing Webmaster verification code</label><input id="bv" value="${esc(s.bing_verify)}" placeholder="the code only">
        <label>Google Analytics 4 ID (optional)</label><input id="ga" value="${esc(s.ga_id)}" placeholder="G-XXXXXXXXXX">
        <div class="mact"><button class="btn pri" id="sv">Save</button></div></div>
      <div class="glass card"><h3>Your SEO files</h3>
        <div class="row"><span>Sitemap</span><a href="/sitemap.xml" target="_blank" rel="noopener">/sitemap.xml ↗</a></div>
        <div class="row"><span>Robots</span><a href="/robots.txt" target="_blank" rel="noopener">/robots.txt ↗</a></div>
        <div class="row"><span>Blog feed</span><a href="/feed.xml" target="_blank" rel="noopener">/feed.xml ↗</a></div>
        <div class="row"><span>Social card</span><a href="/og.png" target="_blank" rel="noopener">/og.png ↗</a></div>
        <p class="muted" style="font-size:13.5px;margin:12px 0 0">After going live: add the site in Google Search Console, then submit <b>/sitemap.xml</b>. Set <b>SITE_URL</b> to your real https address.</p></div></div>
      <h3 style="margin:22px 0 10px">Page titles &amp; descriptions</h3>
      <div class="bar"><input id="q" type="search" placeholder="🔍 Filter pages…"><span class="sp"></span><span class="muted">Good title: 30-60 characters · Good description: 110-160</span></div>
      <div id="tb"></div>`;
    const draw = () => {
      const q = $('#q', el).value.toLowerCase();
      $('#tb', el).innerHTML = `<div class="tw glass"><table><thead><tr><th>Page</th><th>Google preview</th><th></th></tr></thead><tbody>${rows.filter(r => r.path.includes(q) || (r.title || r.default_title).toLowerCase().includes(q)).map(r => {
        const t = r.title || r.default_title, d = r.desc || r.default_desc;
        return `<tr><td style="white-space:nowrap"><b>${esc(r.path)}</b><br>${r.title || r.desc ? chip('custom') : '<small class="muted">default</small>'}${r.noindex ? ' ' + chip('noindex') : ''}</td>
          <td><div style="color:#1a0dab;font-size:16px">${esc(t)}</div><div style="color:#006621;font-size:12.5px">${esc(location.origin + r.path)}</div><div style="color:#545454;font-size:13.5px;max-width:560px">${esc(d)}</div>
          <small><span class="${lenCls(t.length, 30, 65)}">title ${t.length}</span> <span class="${lenCls(d.length, 100, 165)}">description ${d.length}</span></small></td>
          <td><button class="btn sm" data-e="${esc(r.path)}">Edit</button></td></tr>`;
      }).join('')}</tbody></table></div>`;
      $$('[data-e]', el).forEach(b => b.onclick = () => edit(rows.find(r => r.path === b.dataset.e)));
    };
    const edit = r => {
      const b = modal(`<h2>${esc(r.path)}</h2><p class="muted">Leave a field empty to use the default.</p>
        <label>Title <span class="muted" id="tc"></span></label><input id="t" maxlength="110" value="${esc(r.title)}" placeholder="${esc(r.default_title)}">
        <label>Description <span class="muted" id="dc"></span></label><textarea id="d" maxlength="220" placeholder="${esc(r.default_desc)}">${esc(r.desc)}</textarea>
        <label style="display:flex;gap:8px;align-items:center;font-weight:600"><input type="checkbox" id="ni" style="width:auto" ${r.noindex ? 'checked' : ''}> Hide this page from Google (noindex)</label>
        <div class="mact"><button class="btn" id="c">Cancel</button><button class="btn pri" id="s">Save</button></div>`);
      const cnt = () => { $('#tc', b).textContent = ($('#t', b).value || r.default_title).length + ' chars'; $('#dc', b).textContent = ($('#d', b).value || r.default_desc).length + ' chars'; };
      $('#t', b).oninput = $('#d', b).oninput = cnt; cnt();
      $('#c', b).onclick = closeModal;
      $('#s', b).onclick = async () => { if (await run(() => api('/seo/page', 'PUT', { path: r.path, title: $('#t', b).value, desc: $('#d', b).value, noindex: $('#ni', b).checked }), 'Saved - live now')) { closeModal(); route(); } };
    };
    $('#q', el).oninput = draw; draw();
    $('#sv', el).onclick = async () => { if (await run(() => api('/settings', 'PUT', { google_verify: $('#gv', el).value, bing_verify: $('#bv', el).value, ga_id: $('#ga', el).value }), 'Saved')) route(); };
  }

  /* ---------- e-mail log ---------- */
  async function emails(el) {
    const rows = await api('/outbox');
    const bad = rows.filter(r => r.status === 'not_configured' || r.status === 'failed').length;
    el.innerHTML = `${bad ? `<div class="hint">⚠ ${bad} e-mail(s) did not go out. Set up the e-mail server in <a href="#settings"><b>Settings</b></a>, then press <b>Send again</b>.</div>` : ''}${rows.length ? `<div class="tw glass"><table><thead><tr><th>When</th><th>To</th><th>Subject</th><th>Status</th><th></th></tr></thead><tbody>${rows.map(r => `<tr><td>${when(r.created_at)}</td><td>${esc(r.to_addr)}</td><td>${esc(r.subject)}${r.error ? `<br><small style="color:#b91c1c">${esc(r.error)}</small>` : ''}</td><td>${chip(r.status)}</td><td>${r.status !== 'sent' && r.status !== 'queued' ? `<button class="btn sm" data-r="${r.id}">Send again</button>` : ''}</td></tr>`).join('')}</tbody></table></div>` : '<div class="glass empty">No e-mails yet.</div>'}`;
    $$('[data-r]', el).forEach(b => b.onclick = async () => { await run(() => api('/outbox/' + b.dataset.r + '/resend', 'POST', {}), 'Sending…'); setTimeout(route, 1800); });
  }

  /* ---------- settings ---------- */
  async function settings(el) {
    const s = await api('/settings');
    const f = (id, label, type = 'text', ph = '') => `<div><label for="${id}">${label}</label><input id="${id}" type="${type}" value="${esc(s[id])}" placeholder="${esc(ph)}"></div>`;
    el.innerHTML = `<div class="cols"><div>
      <div class="glass card"><h3>Company details (shown on website &amp; e-mails)</h3><div class="fg">${f('company', 'Company name')}${f('phone', 'Phone')}${f('support_email', 'Public e-mail')}${f('address', 'Address / city')}${f('reply_hours', 'Reply promise (hours)', 'number')}${f('reminder_days', 'Renewal reminder (days before end)', 'number')}</div></div>
      <div class="glass card" style="margin-top:16px"><h3>Change my password</h3><div class="fg"><div><label>Current password</label><input id="op" type="password" autocomplete="current-password"></div><div><label>New password (10+ characters)</label><input id="np" type="password" autocomplete="new-password"></div></div><div class="mact"><button class="btn" id="cp">Change password</button></div></div></div>
      <div class="glass card"><h3>E-mail server (SMTP)</h3><div class="hint">Customers get their thank-you e-mail from this account. Use your business e-mail's SMTP details (e.g. Google Workspace, Zoho, Hostinger). For Gmail use an <b>App Password</b>, not your normal password.</div>
      <div class="fg">${f('smtp_host', 'SMTP server', 'text', 'smtp.gmail.com')}${f('smtp_port', 'Port', 'number', '587')}${f('smtp_user', 'Login / username', 'text', 'hello@macbill.in')}<div><label for="smtp_pass">Password ${s.smtp_pass_set ? '<span class="chip sent">saved</span>' : ''}</label><input id="smtp_pass" type="password" autocomplete="new-password" placeholder="${s.smtp_pass_set ? 'leave empty to keep' : ''}"></div>
      ${f('smtp_from', 'Send from (e-mail)', 'email', 'hello@macbill.in')}${f('from_name', 'Sender name')}${f('reply_to', 'Reply-to (optional)', 'email')}${f('notify_email', 'Alert me at (new enquiries)', 'email', 'your-own@email.com')}</div>
      <div class="mact"><button class="btn pri" id="sv">Save settings</button></div>
      <hr style="border:0;border-top:1px solid var(--line);margin:18px 0"><label>Send a test e-mail to</label><div style="display:flex;gap:8px"><input id="te" type="email" value="${esc(ME.email)}"><button class="btn" id="tb">Send test</button></div></div></div>`;
    $('#sv', el).onclick = async () => { const keys = ['company', 'phone', 'support_email', 'address', 'reply_hours', 'reminder_days', 'smtp_host', 'smtp_port', 'smtp_user', 'smtp_pass', 'smtp_from', 'from_name', 'reply_to', 'notify_email']; const body = {}; keys.forEach(k => body[k] = $('#' + k, el).value); if (await run(() => api('/settings', 'PUT', body), 'Settings saved')) route(); };
    $('#tb', el).onclick = async () => { $('#tb', el).disabled = true; await run(() => api('/settings/test-email', 'POST', { to: $('#te', el).value }), 'Test e-mail sent ✔'); $('#tb', el).disabled = false; };
    $('#cp', el).onclick = async () => { if (await run(() => api('/password', 'POST', { old: $('#op', el).value, new: $('#np', el).value }), 'Password changed')) { $('#op', el).value = $('#np', el).value = ''; } };
  }

  boot();
})();
