(() => {
  const $ = s => document.querySelector(s);
  const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const inr = n => '₹' + Number(n).toLocaleString('en-IN');

  // menu (mobile) + dropdown (keyboard friendly)
  const burger = $('#burger'), nav = $('#nav');
  burger.onclick = () => { const o = nav.classList.toggle('open'); burger.setAttribute('aria-expanded', o); };
  document.querySelectorAll('.dd').forEach(d => {
    d.querySelector('.ddb').onclick = e => { e.stopPropagation(); const o = d.classList.contains('open'); document.querySelectorAll('.dd').forEach(x => x.classList.remove('open')); d.classList.toggle('open', !o); };
  });
  document.addEventListener('click', e => { if (!e.target.closest('.dd')) document.querySelectorAll('.dd').forEach(x => x.classList.remove('open')); });
  document.addEventListener('keydown', e => { if (e.key === 'Escape') document.querySelectorAll('.dd').forEach(x => x.classList.remove('open')); });

  // pricing cards (home + pricing page) and the plan picker in the form
  const plansEl = $('#plans'), pick = $('#pl');
  if (plansEl || pick) {
    fetch('/api/public/plans').then(r => r.json()).then(plans => {
      if (plansEl) plansEl.innerHTML = plans.map(p => {
        const per = Math.round(p.price / p.months);
        const dur = p.months % 12 === 0 ? (p.months / 12 === 1 ? 'year' : p.months / 12 + ' years') : p.months + ' months';
        return `<div class="glass plan ${p.popular ? 'pop' : ''}">${p.popular ? '<span class="badge">★ MOST POPULAR</span>' : ''}
          <h3>${esc(p.name)}</h3><div class="tag">${esc(p.tagline)}</div>
          <div class="price">${inr(p.price)}<small> / ${esc(dur)}</small></div><div class="per">≈ ${inr(per)} per month</div>
          <ul>${String(p.features).split('|').filter(Boolean).map(f => `<li>${esc(f)}</li>`).join('')}</ul>
          <a class="btn ${p.popular ? 'pri' : 'ghost'}" href="/contact?plan=${encodeURIComponent(p.name)}">Choose this plan</a></div>`;
      }).join('') || '<div class="note">Plans coming soon.</div>';
      if (pick) {
        pick.innerHTML = '<option value="">Not sure yet</option>' + plans.map(p => `<option>${esc(p.name)}</option>`).join('');
        const want = new URLSearchParams(location.search).get('plan'); if (want) pick.value = want;
      }
    }).catch(() => { if (plansEl) plansEl.innerHTML = '<div class="note">Could not load plans. Please contact us.</div>'; });
  }

  // enquiry form (contact page)
  const form = $('#form');
  if (form) {
    const opened = Date.now();
    form.addEventListener('submit', async e => {
      e.preventDefault();
      const err = $('#err'), btn = $('#send');
      err.hidden = true;
      const d = Object.fromEntries(new FormData(form));
      if (!d.name.trim()) { err.textContent = 'Please enter your name.'; err.hidden = false; return $('#n').focus(); }
      if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(d.email.trim())) { err.textContent = 'Please enter a valid e-mail address - we will send our reply there.'; err.hidden = false; return $('#e').focus(); }
      btn.disabled = true; btn.textContent = 'Sending…';
      try {
        const r = await fetch('/api/public/enquiry', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...d, took: Date.now() - opened }) });
        const j = await r.json();
        if (!r.ok) throw new Error(j.error || 'Something went wrong');
        $('#tmail').textContent = d.email.trim();
        form.hidden = true; $('#thanks').hidden = false;
        $('#thanks').scrollIntoView({ behavior: 'smooth', block: 'center' });
      } catch (x) { err.textContent = x.message; err.hidden = false; btn.disabled = false; btn.textContent = 'Send enquiry →'; }
    });
  }
})();
