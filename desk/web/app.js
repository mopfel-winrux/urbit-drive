/* app.js: the %drive web page. Reads /x/state, re-reads it whenever a
 * fact arrives on /v1/folders, and pokes %drive with JSON actions. */
(async () => {
  const $ = (s) => document.querySelector(s);
  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const size = (n) => n < 1024 ? `${n} B` : n < 1 << 20 ? `${(n / 1024).toFixed(1)} KB` : n < 1 << 30 ? `${(n / (1 << 20)).toFixed(1)} MB` : `${(n / (1 << 30)).toFixed(2)} GB`;
  const date = (ms) => new Date(ms).toLocaleDateString();
  const open = new Set();          // folders whose details are shown
  const trees = new Map();         // fid -> file list, while open
  let ship, st;

  function toast(msg, bad) {
    const t = $('#toast'); t.textContent = msg; t.className = bad ? 'bad' : ''; t.hidden = false;
    clearTimeout(toast.t); toast.t = setTimeout(() => (t.hidden = true), 4000);
  }
  async function act(json, done) {
    try { await ship.poke(json); if (done) toast(done); await refresh(); }
    catch (e) { toast(e.message, true); }
  }

  function renderInvites() {
    $('#invites-section').hidden = !st.invites.length;
    $('#invites').innerHTML = st.invites.map((i) => `
      <li><span><b>${esc(i.name)}</b> from ${esc(i.host)} <small class="muted">${i.role === 'rw' ? 'can edit' : 'read only'}</small></span>
        <button data-accept="${esc(i.host)}|${esc(i.fid)}">Accept</button>
        <button class="quiet" data-decline="${esc(i.host)}|${esc(i.fid)}">Decline</button></li>`).join('');
  }

  function renderFolder(f) {
    const mine = f.host === st.ship;
    const behind = f.behind ? ` · ${f.behind} behind` : '';
    const who = mine ? (f.members.length ? `shared with ${f.members.length}` : 'only you') : `from ${esc(f.host)} · ${f.role === 'rw' ? 'can edit' : 'read only'}`;
    let body = '';
    if (open.has(f.fid)) {
      const files = trees.get(f.fid);
      body += '<div class="details">';
      body += files ? (files.length ? `<table class="files">${files.map((e) => `
        <tr><td>${esc(e.path)}</td><td class="num">${size(e.size)}</td><td class="num muted">${date(e.mtime)}</td></tr>`).join('')}</table>`
        : '<p class="muted">No files yet.</p>') : '<p class="muted">Loading…</p>';
      if (mine) {
        body += `<h3>Sharing</h3><ul class="rows">${f.members.map((m) => `
          <li><span>${esc(m.ship)}</span>
            <select data-role="${esc(f.fid)}|${esc(m.ship)}">
              <option value="rw"${m.role === 'rw' ? ' selected' : ''}>can edit</option>
              <option value="r"${m.role === 'r' ? ' selected' : ''}>read only</option></select>
            <button class="quiet" data-unshare="${esc(f.fid)}|${esc(m.ship)}">Remove</button></li>`).join('')}</ul>
          <form class="inline" data-share="${esc(f.fid)}">
            <input name="ship" placeholder="~sampel-palnet" required pattern="~?[a-z-]+">
            <select name="role"><option value="rw">can edit</option><option value="r">read only</option></select>
            <button>Share</button></form>`;
      } else {
        body += `<p><button class="quiet" data-leave="${esc(f.fid)}">Leave this folder</button>
          <span class="muted">Your devices keep their copies.</span></p>`;
      }
      body += '</div>';
    }
    return `<div class="folder${open.has(f.fid) ? ' open' : ''}">
      <div class="summary" data-toggle="${esc(f.fid)}">
        <b>${esc(f.name)}</b><span class="muted">${who}</span>
        <span class="num muted">${f.files} files · ${size(f.bytes)} · rev ${f.head}${behind}</span></div>${body}</div>`;
  }

  function renderDevices() {
    const ds = [...st.devices].sort((a, b) => (a.revoked !== null) - (b.revoked !== null) || b.since - a.since);
    $('#devices').innerHTML = ds.length ? ds.map((d) => `
      <li class="${d.revoked ? 'revoked' : ''}"><span><b>${esc(d.name)}</b> <small class="muted">${esc(d.moon)}</small></span>
        <small class="muted">${d.revoked ? `removed ${date(d.revoked)}` : `paired ${date(d.since)}`}</small>
        ${d.revoked ? '' : `<button class="quiet" data-unpair="${esc(d.moon)}" data-name="${esc(d.name)}">Remove</button>`}</li>`).join('')
      : '<li class="muted">No devices yet.</li>';
  }

  function render() {
    $('#ship').textContent = st.ship;
    renderInvites();
    const fs = [...st.folders].sort((a, b) => (a.host !== st.ship) - (b.host !== st.ship) || a.name.localeCompare(b.name));
    $('#folders').innerHTML = fs.map(renderFolder).join('');
    renderDevices();
  }

  async function refresh() {
    st = await ship.scry('state');
    await Promise.all([...open].map(async (fid) => {
      try { trees.set(fid, await ship.scry(`tree/${fid}`)); } catch { open.delete(fid); }
    }));
    render();
  }

  document.addEventListener('click', async (ev) => {
    const el = ev.target.closest('[data-toggle],[data-accept],[data-decline],[data-unshare],[data-leave],[data-unpair]');
    if (!el) return;
    const d = el.dataset;
    if (d.toggle) {
      open.has(d.toggle) ? open.delete(d.toggle) : open.add(d.toggle);
      render(); if (open.has(d.toggle)) refresh();
    } else if (d.accept) {
      const [host, fid] = d.accept.split('|'); act({ accept: { host, fid } }, 'Joined; your devices will sync it.');
    } else if (d.decline) {
      const [host, fid] = d.decline.split('|'); act({ decline: { host, fid } });
    } else if (d.unshare) {
      const [fid, s] = d.unshare.split('|');
      if (confirm(`Stop sharing with ${s}? They keep what they already have.`)) act({ unshare: { fid, ship: s } }, `No longer shared with ${s}`);
    } else if (d.leave) {
      if (confirm('Leave this folder? Your devices keep their copies but stop syncing it.')) act({ leave: d.leave }, 'Left the folder');
    } else if (d.unpair) {
      if (confirm(`Remove ${d.name}? It stops syncing; files already on it stay there.`)) act({ unpair: d.unpair }, `Removed ${d.name}`);
    }
  });
  document.addEventListener('change', (ev) => {
    const d = ev.target.dataset;
    if (d.role) { const [fid, s] = d.role.split('|'); act({ share: { fid, ship: s, role: ev.target.value } }, 'Role changed'); }
  });
  document.addEventListener('submit', (ev) => {
    ev.preventDefault();
    const f = ev.target;
    if (f.id === 'new-folder') {
      const name = $('#new-folder-name').value.trim();
      if (name) act({ 'new-folder': name }, `Created ${name}`).then(() => f.reset());
    } else if (f.dataset.share) {
      let s = f.ship.value.trim(); if (!s.startsWith('~')) s = '~' + s;
      act({ share: { fid: f.dataset.share, ship: s, role: f.role.value } }, `Invited ${s}`).then(() => f.reset());
    }
  });

  try {
    ship = await Ship.connect('drive');
    st = ship.initial; render();
    $('#status').textContent = '';
    let t;
    await ship.subscribe('/v1/folders', () => { clearTimeout(t); t = setTimeout(refresh, 250); },
      () => ($('#status').textContent = 'live updates unavailable'));
  } catch (e) {
    $('#status').textContent = `can't reach %drive: ${e.message}`;
  }
  if ('serviceWorker' in navigator) navigator.serviceWorker.register('/drive/sw.js', { scope: '/drive/' }).catch(() => {});
})();
