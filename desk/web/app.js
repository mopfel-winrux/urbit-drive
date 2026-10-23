/* app.js: Drive on the planet. Folders, Activity and More, after Dropbox's
 * web app. Reads %drive's scries, pokes it with JSON actions, and uploads
 * chunk bytes to /drive-api/blob. */
(async () => {
  const $ = (s, el = document) => el.querySelector(s);
  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const icon = (id, cls = '') => `<svg class="ic ${cls}" aria-hidden="true"><use href="#i-${id}"/></svg>`;
  const CHUNK = 4 << 20;
  const RELEASES = 'https://github.com/mopfel-winrux/urbit-drive/releases/latest';

  function size(n) {
    if (n == null) return '—';
    if (n < 1024) return `${n} B`;
    const u = ['KB', 'MB', 'GB', 'TB']; let i = -1;
    do { n /= 1024; i++; } while (n >= 1024 && i < u.length - 1);
    return `${n < 10 ? n.toFixed(1) : Math.round(n)} ${u[i]}`;
  }
  const dayStart = (d) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  function when(ms) {
    if (!ms) return '—';
    const d = new Date(ms), now = new Date();
    const t = d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
    const days = Math.round((dayStart(now) - dayStart(d)) / 864e5);
    if (days === 0) return t;
    if (days === 1) return `Yesterday ${t}`;
    return d.toLocaleDateString([], { month: 'short', day: 'numeric', year: d.getFullYear() === now.getFullYear() ? undefined : 'numeric' });
  }
  const base = (p) => p.replace(/\/$/, '').split('/').pop();
  const keepText = () => st.keep >= 1 ? `${st.keep} ${st.keep === 1 ? 'day' : 'days'}` : 'less than a day';
  const join = (dir, name) => (dir ? `${dir}/${name}` : name);
  const parent = (p) => p.replace(/\/$/, '').split('/').slice(0, -1).join('/');

  let ship, st = { folders: [], devices: [], invites: [], stored: { bytes: 0, chunks: 0 }, keep: 30 };
  const trees = new Map();      // fid -> [{path, hash, size, mtime}]
  const olds = new Map();       // fid -> [{path, deleted, versions}]
  let activity = null;
  let selection = new Set();    // names in the current listing
  let sort = { key: 'name', dir: 1 };
  let renaming = null;          // name being renamed inline
  let query = '';
  let links = [];               // share links, from /x/links

  // ----- routing: #/  #/f/<fid>/<path>?sel=<name>  #/activity  #/more/<tab>

  function route() {
    const [path, qs] = location.hash.slice(1).split('?');
    const parts = (path || '/').split('/').filter(Boolean).map(decodeURIComponent);
    const sel = new URLSearchParams(qs || '').get('sel');
    if (parts[0] === 'f' && parts[1]) return { view: 'folder', fid: parts[1], dir: parts.slice(2).join('/'), sel };
    if (parts[0] === 'activity') return { view: 'activity' };
    if (parts[0] === 'more') return { view: 'more', tab: parts[1] || 'deleted' };
    return { view: 'root', sel };
  }
  const folderHref = (fid, dir = '', sel) =>
    `#/f/${encodeURIComponent(fid)}${dir ? '/' + dir.split('/').map(encodeURIComponent).join('/') : ''}${sel ? '?sel=' + encodeURIComponent(sel) : ''}`;
  const folderOf = (fid) => st.folders.find((f) => f.fid === fid);
  const mine = (f) => f.host === st.ship;

  // ----- data

  async function loadTree(fid, force) {
    if (!force && trees.has(fid)) return trees.get(fid);
    const [t, o] = await Promise.all([ship.scry(`tree/${fid}`), ship.scry(`old/${fid}`)]);
    trees.set(fid, t); olds.set(fid, o);
    return t;
  }

  // the entries directly inside [dir] of a folder's tree
  function listDir(tree, dir) {
    const files = [], dirs = new Map();
    const pre = dir ? dir + '/' : '';
    for (const e of tree) {
      if (!e.path.startsWith(pre)) continue;
      const rel = e.path.slice(pre.length);
      if (!rel) continue;
      const cut = rel.indexOf('/');
      if (cut === -1) { files.push({ kind: 'file', name: rel, path: e.path, size: e.size, mtime: e.mtime, hash: e.hash }); continue; }
      const name = rel.slice(0, cut);
      const d = dirs.get(name) || { kind: 'dir', name, path: pre + name, size: 0, mtime: 0, count: 0 };
      if (rel !== name + '/') { d.size += e.size; d.count++; d.mtime = Math.max(d.mtime, e.mtime); }
      dirs.set(name, d);
    }
    return [...dirs.values(), ...files];
  }
  function allDirs(tree) {
    const out = new Set(['']);
    for (const e of tree) {
      const parts = e.path.split('/'); parts.pop();
      for (let i = 1; i <= parts.length; i++) out.add(parts.slice(0, i).join('/'));
    }
    return [...out].sort((a, b) => a.localeCompare(b));
  }
  function sorted(items) {
    const k = sort.key, d = sort.dir;
    return [...items].sort((a, b) => {
      if (a.kind !== b.kind) return a.kind === 'dir' ? -1 : 1;
      const x = k === 'name' ? a.name.localeCompare(b.name, undefined, { numeric: true }) : (a[k] || 0) - (b[k] || 0);
      return x * d || a.name.localeCompare(b.name);
    });
  }
  const conflicted = (name) => / \(conflicted copy, /.test(name);

  // ----- small UI pieces

  function toast(msg, bad) {
    const t = $('#toast'); t.textContent = msg; t.className = 'toast' + (bad ? ' bad' : ''); t.hidden = false;
    clearTimeout(toast.t); toast.t = setTimeout(() => (t.hidden = true), 4500);
  }
  async function act(json, done) {
    try { await ship.poke(json); if (done) toast(done); return true; }
    catch (e) { toast(e.message || 'That didn’t work. Try again.', true); return false; }
  }

  function closeMenu() { const m = $('.menu'); if (m) { m.remove(); document.querySelectorAll('[aria-expanded="true"]').forEach((b) => b.setAttribute('aria-expanded', 'false')); } }
  function openMenu(btn, items) {
    closeMenu();
    const m = document.createElement('div');
    m.className = 'menu'; m.setAttribute('role', 'menu');
    m.innerHTML = items.map((it, i) => it === '-' ? '<hr>' :
      it.href ? `<a role="menuitem" href="${esc(it.href)}" ${it.download ? `download="${esc(it.download)}"` : ''} data-i="${i}">${icon(it.icon)}${esc(it.label)}</a>`
        : `<button role="menuitem" class="${it.danger ? 'danger' : ''}" data-i="${i}">${icon(it.icon)}${esc(it.label)}</button>`).join('');
    document.body.appendChild(m);
    const r = btn.getBoundingClientRect();
    const top = Math.min(r.bottom + 4, innerHeight - m.offsetHeight - 8);
    m.style.top = `${Math.max(8, top)}px`;
    m.style.left = `${Math.max(8, Math.min(r.right - m.offsetWidth, innerWidth - m.offsetWidth - 8))}px`;
    btn.setAttribute('aria-expanded', 'true');
    m.addEventListener('click', (ev) => {
      const el = ev.target.closest('[data-i]'); if (!el) return;
      const it = items[+el.dataset.i]; if (it.run) { ev.preventDefault(); it.run(); }
      closeMenu();
    });
    m.querySelector('[data-i]')?.focus();
  }

  function dialog(html, onSubmit) {
    const d = $('#dialog');
    d.innerHTML = `<form method="dialog">${html}</form>`;
    const f = d.querySelector('form');
    f.addEventListener('submit', async (ev) => {
      if (ev.submitter?.value === 'cancel') return;
      ev.preventDefault();
      const ok = await onSubmit(f, ev.submitter);
      if (ok !== false) d.close();
    });
    d.showModal();
    f.querySelector('input:not([type=hidden]), select, button.primary')?.focus();
    return d;
  }

  // ----- views

  function renderRail(r) {
    document.querySelectorAll('.rail nav a').forEach((a) => {
      const on = a.dataset.nav === (r.view === 'root' || r.view === 'folder' ? 'folders' : r.view);
      on ? a.setAttribute('aria-current', 'page') : a.removeAttribute('aria-current');
    });
    $('#who').textContent = st.ship || '';
    const total = st.stored.bytes;
    $("#usage").textContent = `${size(total)} stored on your ship`;
  }

  function header(r) {
    if (r.view === 'root') return { crumbs: '', title: 'Folders' };
    const f = folderOf(r.fid);
    const segs = r.dir ? r.dir.split('/') : [];
    let crumbs = `<a href="#/">Folders</a>`;
    if (segs.length) crumbs += `${icon('chevron')}<a href="${folderHref(r.fid)}">${esc(f?.name || 'Folder')}</a>`;
    segs.slice(0, -1).forEach((s, i) => { crumbs += `${icon('chevron')}<a href="${folderHref(r.fid, segs.slice(0, i + 1).join('/'))}">${esc(s)}</a>`; });
    return { crumbs, title: segs.length ? segs[segs.length - 1] : (f?.name || 'Folder') };
  }

  function access(f) {
    if (!mine(f)) return `Shared by ${esc(f.host)}`;
    return f.members.length ? `Shared with ${f.members.length} ${f.members.length === 1 ? 'ship' : 'ships'}` : 'Only you';
  }

  function toolbar(r, f) {
    if (selection.size) {
      const items = currentItems(r);
      const sel = items.filter((i) => selection.has(i.name));
      const files = sel.filter((i) => i.kind === 'file');
      const rw = !f || f.role === 'rw';
      let b = `<div class="selbar"><span class="count">${sel.length} selected</span>`;
      if (r.view === 'root') {
        if (sel.length === 1 && mine(sel[0].f)) b += `<button class="btn" data-do="share">${icon('share')}<span class="lbl">Share</span></button>`;
        if (sel.length === 1 && !mine(sel[0].f)) b += `<button class="btn danger" data-do="leave">Leave folder</button>`;
      } else {
        if (files.length === sel.length) b += `<button class="btn" data-do="download">${icon('download')}<span class="lbl">Download</span></button>`;
        if (rw && sel.length === 1) b += `<button class="btn" data-do="rename">${icon('rename')}<span class="lbl">Rename</span></button>`;
        if (rw) b += `<button class="btn" data-do="move">${icon('move')}<span class="lbl">Move</span></button>`;
        if (rw) b += `<button class="btn danger" data-do="delete">${icon('trash')}<span class="lbl">Delete</span></button>`;
      }
      return b + `<button class="btn ghost" data-do="clear">Clear</button></div>`;
    }
    if (r.view === 'root') return `<button class="btn primary" data-do="new-folder">${icon('plus-folder')}New folder</button>`;
    const rw = f.role === 'rw';
    let b = '';
    if (rw) b += `<button class="btn primary" data-do="upload">${icon('upload')}Upload</button><button class="btn" data-do="mkdir">${icon('plus-folder')}<span class="lbl">New folder</span></button>`;
    else b += `<span class="tag ro">View only</span>`;
    if (mine(f) && !r.dir) b += `<button class="btn" data-do="share-this">${icon('share')}<span class="lbl">Share</span></button>`;
    return b;
  }

  function currentItems(r) {
    if (r.view === 'root') return st.folders.map((f) => ({ kind: 'top', name: f.name, f, size: f.bytes, mtime: 0 }));
    const t = trees.get(r.fid); return t ? listDir(t, r.dir) : [];
  }

  function rows(r, f, items) {
    const head = `<thead><tr>
      <th class="c-check"><input type="checkbox" data-all aria-label="Select all" ${items.length && selection.size === items.length ? 'checked' : ''}></th>
      <th class="c-name"><button data-sort="name">Name${sort.key === 'name' ? icon('chevron', sort.dir > 0 ? 'sort up' : 'sort down') : ''}</button></th>
      <th class="c-access">Who can access</th>
      <th class="c-mod"><button data-sort="mtime">Modified${sort.key === 'mtime' ? icon('chevron', sort.dir > 0 ? 'sort up' : 'sort down') : ''}</button></th>
      <th class="c-size"><button data-sort="size">Size${sort.key === 'size' ? icon('chevron', sort.dir > 0 ? 'sort up' : 'sort down') : ''}</button></th>
      <th class="c-act" aria-label="Actions"></th></tr></thead>`;
    const body = sorted(items).map((it) => {
      const sel = selection.has(it.name);
      const href = it.kind === 'top' ? folderHref(it.f.fid) : it.kind === 'dir' ? folderHref(r.fid, it.path) : folderHref(r.fid, r.dir, it.name);
      const ic = it.kind === 'file' ? icon('file', conflicted(it.name) ? 'conflict' : '') : icon(it.kind === 'top' && (it.f.members.length || !mine(it.f)) ? 'folder-shared' : 'folder', 'folder');
      const label = renaming === it.name
        ? `<input value="${esc(it.name)}" data-rename="${esc(it.name)}" aria-label="New name">`
        : `<a href="${href}" ${it.kind === 'file' ? 'data-open' : ''} ${conflicted(it.name) ? 'title="Conflicted copy: two computers changed this file at once"' : ''}>${esc(it.name)}</a>${it.kind === 'top' && it.f.role === 'r' ? '<span class="tag ro">View only</span>' : ''}`;
      const acc = it.kind === 'top' ? access(it.f) : access(f);
      return `<tr data-name="${esc(it.name)}" class="${sel ? 'sel' : ''}">
        <td class="c-check"><input type="checkbox" data-pick="${esc(it.name)}" aria-label="Select ${esc(it.name)}" ${sel ? 'checked' : ''}></td>
        <td class="c-name"><div class="name">${ic}<div class="nm">${label}<small class="meta">${it.kind === 'dir' ? `${it.count} ${it.count === 1 ? 'file' : 'files'}` : it.kind === 'top' ? esc(access(it.f)) : `${when(it.mtime)} · ${size(it.size)}`}</small></div></div></td>
        <td class="c-access">${acc}</td>
        <td class="c-mod">${it.kind === 'file' ? when(it.mtime) : it.mtime ? when(it.mtime) : '—'}</td>
        <td class="c-size">${it.kind === 'dir' ? `${it.count} ${it.count === 1 ? 'file' : 'files'}` : size(it.size)}</td>
        <td class="c-act"><button class="icon-btn row-menu" data-menu="${esc(it.name)}" aria-label="More actions for ${esc(it.name)}" aria-haspopup="menu" aria-expanded="false">${icon('more')}</button></td></tr>`;
    }).join('');
    return `<table class="files">${head}<tbody>${body}</tbody></table>`;
  }

  function invitesBanner() {
    return st.invites.map((i) => `<div class="banner">${icon('folder-shared')}
      <p><b>${esc(i.host)}</b> shared <b>${esc(i.name)}</b> with you${i.role === 'r' ? ' (view only)' : ''}.</p>
      <button class="btn primary" data-accept="${esc(i.host)}|${esc(i.fid)}">Add to my folders</button>
      <button class="btn ghost" data-decline="${esc(i.host)}|${esc(i.fid)}">Decline</button></div>`).join('');
  }

  async function renderFolders(r) {
    const c = $('#content');
    let f = null;
    if (r.view === 'folder') {
      f = folderOf(r.fid);
      if (!f) { c.innerHTML = emptyState('folder', 'This folder isn’t here', 'It may have been unshared or left.', `<a class="btn" href="#/">Back to Folders</a>`); return; }
      if (!trees.has(r.fid)) {
        const h = header(r);
        c.innerHTML = `<div class="crumbs">${h.crumbs}</div><div class="title-row"><h1>${esc(h.title)}</h1></div>
          <table class="files"><tbody>${'<tr class="skel"><td></td><td><div></div></td><td><div></div></td><td><div></div></td><td><div></div></td><td></td></tr>'.repeat(5)}</tbody></table>`;
        try { await loadTree(r.fid); } catch { c.innerHTML = emptyState('folder', 'Couldn’t load this folder', 'Your ship didn’t answer. Check that it’s running, then reload.', ''); return; }
        if (location.hash && route().fid !== r.fid) return;
      }
    }
    const h = header(r);
    const items = currentItems(r);
    let out = `<div class="crumbs">${h.crumbs}</div><div class="title-row"><h1>${esc(h.title)}</h1>${toolbar(r, f)}</div>`;
    if (r.view === 'root') out += invitesBanner();
    if (!items.length) {
      out += r.view === 'root'
        ? emptyState('folder', 'No folders yet', 'Create a folder, or pair a computer and the folder it syncs appears here.', `<button class="btn primary" data-do="new-folder">${icon('plus-folder')}New folder</button>`)
        : f.role === 'rw'
          ? emptyState('upload', 'Drop files here', 'Files you upload sync to every computer paired with your ship.', `<button class="btn primary" data-do="upload">${icon('upload')}Upload files</button>`)
          : emptyState('folder', 'Nothing here yet', `Files ${esc(f.host)} adds will appear here.`, '');
    } else out += rows(r, f, items);
    c.innerHTML = out;
    $('[data-rename]')?.focus();
    $('[data-rename]')?.select();
    renderDetails(r);
  }

  function emptyState(ic, title, text, action) {
    return `<div class="empty">${icon(ic)}<h2>${esc(title)}</h2><p>${text}</p>${action}</div>`;
  }

  const fileUrl = (fid, hash) => `/~/scry/drive/file/${fid}/${hash}.mime`;

  function renderDetails(r) {
    const d = $('#details');
    if (r.view !== 'folder' || !r.sel) { d.hidden = true; d.innerHTML = ''; return; }
    const f = folderOf(r.fid), t = trees.get(r.fid) || [];
    const path = join(r.dir, r.sel);
    const e = t.find((x) => x.path === path);
    const hist = (olds.get(r.fid) || []).find((o) => o.path === path);
    if (!e) { d.hidden = true; return; }
    const versions = hist ? hist.versions : [];
    d.hidden = false;
    d.innerHTML = `<div class="head">${icon('file', conflicted(e.path) ? 'big conflict' : 'big')}<h2>${esc(r.sel)}</h2>
        <a class="icon-btn" href="${folderHref(r.fid, r.dir)}" aria-label="Close details">${icon('close')}</a></div>
      ${conflicted(r.sel) ? '<p class="muted">Two computers changed this file at once. This is the copy Drive kept so neither edit was lost; compare it with the original and delete the one you don’t need.</p>' : ''}
      <dl><dt>Size</dt><dd>${size(e.size)}</dd><dt>Modified</dt><dd>${new Date(e.mtime).toLocaleString()}</dd>
        <dt>Folder</dt><dd>${esc(f.name)}${r.dir ? ' / ' + esc(r.dir) : ''}</dd></dl>
      <div class="actions"><a class="btn primary" href="${fileUrl(r.fid, e.hash)}" download="${esc(r.sel)}">${icon('download')}Download</a></div>
      <h3>Version history</h3>
      ${versions.length ? '' : `<p class="muted">Earlier versions appear here when the file changes, for ${keepText()}.</p>`}
      <ul class="versions">
        <li class="current"><b>${when(e.mtime)}</b><small>${size(e.size)}</small></li>
        ${versions.map((v) => `<li><b>${when(v.mtime)}</b><small>${size(v.size)} · replaced ${when(v.when)}</small>
          <span class="vbtns"><a class="icon-btn" href="${fileUrl(r.fid, v.hash)}" download="${esc(r.sel)}" aria-label="Download this version">${icon('download')}</a>
          ${f.role === 'rw' ? `<button class="btn" data-restore="${esc(v.hash)}" data-path="${esc(path)}">Restore</button>` : ''}</span></li>`).join('')}
      </ul>`;
  }

  async function renderActivity() {
    const c = $('#content');
    c.innerHTML = `<div class="crumbs"></div><div class="title-row"><h1>Activity</h1></div><p class="sub">Changes to your folders from every computer and ship, newest first.</p><div id="feed"><table class="files"><tbody>${'<tr class="skel"><td></td><td><div></div></td><td><div></div></td></tr>'.repeat(4)}</tbody></table></div>`;
    try { activity = await ship.scry('activity'); } catch { activity = []; }
    if (route().view !== 'activity') return;
    if (!activity.length) { $('#feed').innerHTML = emptyState('activity', 'No changes yet', 'When files are added, edited or deleted on any of your computers, it shows up here.', ''); return; }
    let out = '', day = '';
    for (const a of activity) {
      const d = new Date(a.when), label = dayStart(d) === dayStart(new Date()) ? 'Today' : when(dayStart(d)).replace(/ .*/, '') === 'Yesterday' ? 'Yesterday' : d.toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' });
      if (label !== day) { if (day) out += '</ul>'; out += `<div class="day">${label}</div><ul class="events">`; day = label; }
      const ops = a.ops.filter((o) => o.kind !== 'folder' || true);
      const kinds = new Set(ops.map((o) => o.kind));
      const verb = ops.length === 1
        ? ({ added: 'added', edited: 'edited', deleted: 'deleted', folder: 'created the folder' })[ops[0].kind]
        : kinds.size === 1 ? `${[...kinds][0] === 'folder' ? 'created' : [...kinds][0]} ${ops.length} items` : `changed ${ops.length} items`;
      const what = ops.length === 1 ? ` <b>${esc(base(ops[0].path))}</b>` : '';
      const ic = kinds.size === 1 && kinds.has('deleted') ? 'trash' : ops.length === 1 && ops[0].kind === 'folder' ? 'folder' : 'file';
      const by = a.who === st.ship ? 'You' : a.by;
      out += `<li>${icon(ic)}<div class="what"><b>${esc(by)}</b> ${verb}${what} in <a href="${folderHref(a.fid)}">${esc(a.folder)}</a></div><span class="when">${new Date(a.when).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}</span>
        ${ops.length > 1 ? `<ul>${ops.slice(0, 5).map((o) => `<li><span class="k ${o.kind}">${o.kind === 'folder' ? 'folder' : o.kind}</span>${esc(o.path)}</li>`).join('')}${ops.length > 5 ? `<li>and ${ops.length - 5} more</li>` : ''}</ul>` : ''}</li>`;
    }
    $('#feed').innerHTML = out + '</ul>';
  }

  async function renderMore(r) {
    const c = $('#content');
    const tab = (id, ic, label) => `<a href="#/more/${id}" ${r.tab === id ? 'aria-current="page"' : ''}>${icon(ic)}${label}</a>`;
    c.innerHTML = `<div class="crumbs"></div><div class="title-row"><h1>More</h1></div>
      <nav class="tabs" aria-label="More">${tab('deleted', 'trash', 'Deleted files')}${tab('links', 'link', 'Links')}${tab('computers', 'computer', 'Computers')}${tab('settings', 'settings', 'Settings')}</nav><div id="pane"></div>`;
    const p = $('#pane');
    if (r.tab === 'computers') {
      const ds = [...st.devices].sort((a, b) => (a.revoked !== null) - (b.revoked !== null) || b.since - a.since);
      p.innerHTML = `<p class="sub" style="margin-top:0">Each computer you pair gets its own identity on your ship (a <i>moon</i>), so you can remove one without affecting the others. To add a computer, <a href="${RELEASES}" target="_blank" rel="noopener">get the desktop app</a> and sign in with this ship’s address and <code>+code</code>.</p>` +
        (ds.length ? `<table class="files"><thead><tr><th class="c-name">Computer</th><th class="c-access">Identity</th><th class="c-mod">Paired</th><th class="c-act"></th></tr></thead><tbody>${ds.map((d) => `
          <tr><td class="c-name"><div class="name">${icon('computer')}<span class="label ${d.revoked ? 'revoked' : ''}">${esc(d.name)}</span>${d.revoked ? '<span class="tag ro">Removed</span>' : ''}</div></td>
          <td class="c-access" title="${esc(d.moon)}">${esc(d.moon)}</td><td class="c-mod">${when(d.revoked || d.since)}</td>
          <td class="c-act">${d.revoked ? '' : `<button class="icon-btn" data-unpair="${esc(d.moon)}" data-name="${esc(d.name)}" aria-label="Remove ${esc(d.name)}">${icon('trash')}</button>`}</td></tr>`).join('')}</tbody></table>`
          : emptyState('computer', 'No computers yet', 'Install the desktop app on a computer and pair it with your ship; its folder starts syncing right away.', `<a class="btn primary" href="${RELEASES}" target="_blank" rel="noopener">${icon('desktop-app')}Get the desktop app</a>`));
    } else if (r.tab === 'links') {
      const ls = [...links].sort((a, b) => b.made - a.made);
      p.innerHTML = `<p class="sub" style="margin-top:0">Anyone with a link can open what it points to, without a ship. Links always show the current version. Turn one off and it stops working at once.</p>` +
        (ls.length ? `<table class="files"><thead><tr><th class="c-name">Shared item</th><th class="c-access">Who can open it</th><th class="c-mod">Created</th><th class="c-size">Views</th><th class="c-size">Downloads</th><th class="c-act" aria-label="Actions"></th></tr></thead><tbody>${ls.map((l) => {
          const expired = l.expires && l.expires < Date.now();
          const where = l.path ? `${esc(l.folder)}${parent(l.path) ? ' / ' + esc(parent(l.path)) : ''}` : 'The whole folder';
          const access = expired ? 'Expired' : `${l.password ? 'Needs the password' : 'Anyone with the link'}${l.expires ? ` · until ${when(l.expires)}` : ''}`;
          return `<tr class="${expired || !l.live ? 'revoked' : ''}"><td class="c-name"><div class="name">${l.kind === 'file' ? icon('file') : icon('folder', 'folder')}<div class="nm"><span class="label">${esc(l.path ? base(l.path) : l.folder)}</span><small class="meta" style="display:block">${where}${l.live ? '' : ' · no longer there'}</small></div></div></td>
          <td class="c-access">${access}</td><td class="c-mod">${when(l.made)}</td><td class="c-size">${l.views}</td><td class="c-size">${l.downloads}</td>
          <td class="c-act"><button class="icon-btn" data-linkmenu="${esc(l.token)}" aria-label="Actions for this link" aria-haspopup="menu" aria-expanded="false">${icon('more')}</button></td></tr>`;
        }).join('')}</tbody></table>`
          : emptyState('link', 'No links yet', 'Choose Copy link on a file or folder you own to share it with anyone, ship or not.', ''));
    } else if (r.tab === 'settings') {
      p.innerHTML = `
        <div class="setting"><b>Keep deleted files and earlier versions</b><p>After this, they’re removed from your ship for good.</p>
          <form class="ctl" id="keep-form"><input class="field num" id="keep" type="number" min="1" max="3650" value="${st.keep}" aria-label="Days"><span class="muted">days</span><button class="btn">Save</button></form></div>
        <div class="setting"><b>Storage</b><p>${size(st.stored.bytes)} on your ship, counting each piece of a file once even when several files or versions share it.</p>
          <div class="ctl"><button class="btn" data-do="prune">Clean up now</button></div></div>`;
    } else {
      p.innerHTML = `<table class="files"><tbody>${'<tr class="skel"><td><div></div></td><td><div></div></td><td><div></div></td></tr>'.repeat(3)}</tbody></table>`;
      await Promise.all(st.folders.map((f) => loadTree(f.fid).catch(() => null)));
      if (route().tab !== r.tab && !(r.tab === 'deleted' && route().view === 'more')) return;
      const gone = [];
      for (const f of st.folders) for (const o of olds.get(f.fid) || []) if (o.deleted && o.versions.length) gone.push({ f, o, v: o.versions[0] });
      gone.sort((a, b) => b.v.when - a.v.when);
      p.innerHTML = `<p class="sub" style="margin-top:0">Files deleted in the last ${keepText()}. Restoring puts a file back where it was, on every computer.</p>` +
        (gone.length ? `<table class="files"><thead><tr><th class="c-name">Name</th><th class="c-access">Folder</th><th class="c-mod">Deleted</th><th class="c-size">Size</th><th class="c-act"></th></tr></thead><tbody>${gone.map(({ f, o, v }) => `
          <tr><td class="c-name"><div class="name">${icon('file')}<span class="label">${esc(base(o.path))}</span></div></td>
          <td class="c-access">${esc(f.name)}${parent(o.path) ? ' / ' + esc(parent(o.path)) : ''}</td><td class="c-mod">${when(v.when)}</td><td class="c-size">${size(v.size)}</td>
          <td class="c-act">${f.role === 'rw' ? `<button class="icon-btn" data-restore="${esc(v.hash)}" data-path="${esc(o.path)}" data-fid="${esc(f.fid)}" aria-label="Restore ${esc(base(o.path))}">${icon('restore')}</button>` : ''}</td></tr>`).join('')}</tbody></table>`
          : emptyState('trash', 'No deleted files', `Deleted files stay here for ${keepText()}, so you can bring them back.`, ''));
    }
  }

  async function renderSearch() {
    const c = $('#content');
    $('#details').hidden = true;
    c.innerHTML = `<div class="crumbs"></div><div class="title-row"><h1>Results for “${esc(query)}”</h1></div><div id="results"></div>`;
    await Promise.all(st.folders.map((f) => loadTree(f.fid).catch(() => null)));
    if (!query) return;
    const q = query.toLowerCase(), hits = [];
    for (const f of st.folders) for (const e of trees.get(f.fid) || []) {
      if (base(e.path).toLowerCase().includes(q)) hits.push({ f, e });
    }
    $('#results').innerHTML = hits.length ? `<table class="files"><thead><tr><th class="c-name">Name</th><th class="c-access">Location</th><th class="c-mod">Modified</th><th class="c-size">Size</th></tr></thead><tbody>${hits.slice(0, 200).map(({ f, e }) => {
      const dir = e.path.endsWith('/'), p = e.path.replace(/\/$/, '');
      return `<tr><td class="c-name"><div class="name">${icon(dir ? 'folder' : 'file', dir ? 'folder' : '')}<a href="${dir ? folderHref(f.fid, p) : folderHref(f.fid, parent(p), base(p))}">${esc(base(p))}</a></div></td>
        <td class="c-access">${esc(f.name)}${parent(p) ? ' / ' + esc(parent(p)) : ''}</td><td class="c-mod">${dir ? '—' : when(e.mtime)}</td><td class="c-size">${dir ? '—' : size(e.size)}</td></tr>`;
    }).join('')}</tbody></table>` : emptyState('search', 'No matches', 'Search looks at file and folder names in every folder.', '');
  }

  async function render() {
    closeMenu();
    const r = route();
    renderRail(r);
    $('#rail').classList.remove('open');
    if (query) return renderSearch();
    if (r.view === 'activity') { $('#details').hidden = true; return renderActivity(); }
    if (r.view === 'more') { $('#details').hidden = true; return renderMore(r); }
    return renderFolders(r);
  }

  async function refresh() {
    st = await ship.scry('state');
    links = await ship.scry('links').catch(() => links);
    const r = route();
    for (const fid of [...trees.keys()]) { if (fid !== r.fid) { trees.delete(fid); olds.delete(fid); } }
    if (r.view === 'folder') await loadTree(r.fid, true).catch(() => null);
    render();
  }

  // ----- actions

  function selectedItems() { const r = route(); return currentItems(r).filter((i) => selection.has(i.name)); }

  function confirmDelete(items, r) {
    const n = items.length, one = items[0];
    dialog(`<h2>Delete ${n === 1 ? `“${esc(one.name)}”` : `${n} items`}?</h2>
      <p>${n === 1 && one.kind === 'dir' ? 'The folder and everything in it will be' : 'They’ll be'} removed from every computer. You can restore deleted files from More › Deleted files for ${keepText()}.</p>
      <div class="foot"><button class="btn" value="cancel">Cancel</button><button class="btn danger solid" value="ok">Delete</button></div>`,
    async () => {
      for (const it of items) await act({ del: { fid: r.fid, path: it.path } });
      toast(n === 1 ? `Deleted ${one.name}` : `Deleted ${n} items`);
      selection.clear(); await refresh();
    });
  }

  function moveDialog(items, r) {
    const dirs = allDirs(trees.get(r.fid) || []).filter((d) => !items.some((i) => i.kind === 'dir' && (d === i.path || d.startsWith(i.path + '/'))));
    const f = folderOf(r.fid);
    let target = r.dir;
    const d = dialog(`<h2>Move ${items.length === 1 ? `“${esc(items[0].name)}”` : `${items.length} items`}</h2><p>Choose a folder in ${esc(f.name)}.</p>
      <div class="picker" role="listbox">${dirs.map((p) => `<button type="button" data-to="${esc(p)}" aria-pressed="${p === target}" style="padding-left:${12 + (p ? p.split('/').length : 0) * 16}px">${icon('folder')}${esc(p ? base(p) : f.name)}</button>`).join('')}</div>
      <p class="err" hidden></p><div class="foot"><button class="btn" value="cancel">Cancel</button><button class="btn primary" value="ok">Move</button></div>`,
    async (form) => {
      if (target === r.dir) { form.querySelector('.err').textContent = 'Pick a different folder.'; form.querySelector('.err').hidden = false; return false; }
      for (const it of items) await act({ mv: { fid: r.fid, from: it.path, to: join(target, it.name) } });
      toast(`Moved to ${target ? base(target) : f.name}`); selection.clear(); await refresh();
    });
    d.querySelector('.picker').addEventListener('click', (ev) => {
      const b = ev.target.closest('[data-to]'); if (!b) return;
      target = b.dataset.to;
      d.querySelectorAll('[data-to]').forEach((x) => x.setAttribute('aria-pressed', String(x === b)));
    });
  }

  function shareDialog(f) {
    const roleSel = (v, extra = '') => `<select class="field" ${extra}><option value="rw"${v === 'rw' ? ' selected' : ''}>Can edit</option><option value="r"${v === 'r' ? ' selected' : ''}>Can view</option></select>`;
    const d = dialog(`<h2>Share “${esc(f.name)}”</h2>
      <p>People on other ships get a copy of this folder on their ship and their computers. Removing someone stops future changes reaching them; it can’t take back what they already have.</p>
      <label>Add a ship<div class="row"><input class="field" name="ship" placeholder="~sampel-palnet" autocomplete="off" required pattern="~?[a-z]{3}([a-z\\-]*)">${roleSel('rw', 'name="role"')}</div></label>
      <p class="err" hidden></p>
      ${f.members.length ? `<ul class="members">${f.members.map((m) => `<li><span>${esc(m.ship)}</span>${roleSel(m.role, `data-role="${esc(m.ship)}" aria-label="Access for ${esc(m.ship)}"`)}<button type="button" class="icon-btn" data-remove="${esc(m.ship)}" aria-label="Remove ${esc(m.ship)}">${icon('close')}</button></li>`).join('')}</ul>` : ''}
      <div class="foot"><button class="btn" value="cancel">Done</button><button class="btn primary" value="ok">Share</button></div>`,
    async (form) => {
      let s = form.ship.value.trim().toLowerCase(); if (!s.startsWith('~')) s = '~' + s;
      if (s === st.ship) { form.querySelector('.err').textContent = 'That’s your own ship.'; form.querySelector('.err').hidden = false; return false; }
      const ok = await act({ share: { fid: f.fid, ship: s, role: form.role.value } }, `Invited ${s}`);
      if (ok) await refresh();
    });
    d.addEventListener('change', async (ev) => {
      const s = ev.target.dataset.role; if (!s) return;
      if (await act({ share: { fid: f.fid, ship: s, role: ev.target.value } }, `${s} can now ${ev.target.value === 'rw' ? 'edit' : 'view'}`)) refresh();
    });
    d.addEventListener('click', async (ev) => {
      const s = ev.target.closest('[data-remove]')?.dataset.remove; if (!s) return;
      if (await act({ unshare: { fid: f.fid, ship: s } }, `Stopped sharing with ${s}`)) { d.close(); refresh(); }
    });
  }

  function newFolderDialog(r) {
    dialog(`<h2>New folder</h2><label>Name<input class="field" name="name" required autocomplete="off" maxlength="200"></label><p class="err" hidden></p>
      <div class="foot"><button class="btn" value="cancel">Cancel</button><button class="btn primary" value="ok">Create</button></div>`,
    async (form) => {
      const name = form.name.value.trim().replace(/\//g, '-');
      const err = (m) => { form.querySelector('.err').textContent = m; form.querySelector('.err').hidden = false; return false; };
      if (!name || name === '.' || name === '..') return err('Give the folder a name.');
      if (r.view === 'root') {
        if (st.folders.some((f) => f.name === name)) return err('You already have a folder with that name.');
        if (await act({ 'new-folder': name }, `Created ${name}`)) await refresh();
      } else {
        if (listDir(trees.get(r.fid) || [], r.dir).some((i) => i.name === name)) return err('Something here already has that name.');
        if (await act({ mkdir: { fid: r.fid, path: join(r.dir, name) } }, `Created ${name}`)) await refresh();
      }
    });
  }

  async function commitRename(input) {
    const r = route(), from = input.dataset.rename, to = input.value.trim().replace(/\//g, '-');
    renaming = null;
    if (!to || to === from) return render();
    if (listDir(trees.get(r.fid) || [], r.dir).some((i) => i.name === to)) { toast('Something here already has that name.', true); return render(); }
    await act({ mv: { fid: r.fid, from: join(r.dir, from), to: join(r.dir, to) } }, `Renamed to ${to}`);
    selection.clear(); await refresh();
  }

  // ----- share links

  const linkUrl = (tok) => `${location.origin}/drive-api/s/${tok}`;
  // the item's working link, if it has one (expired ones can't be reused)
  const linkFor = (fid, path) => links.find((l) => l.fid === fid && l.path === path && !(l.expires && l.expires < Date.now()));
  const newToken = () => [...crypto.getRandomValues(new Uint8Array(16))].map((b) => b.toString(16).padStart(2, '0')).join('');

  async function copyText(text) {
    try { await navigator.clipboard.writeText(text); return true; } catch {}
    const i = document.createElement('input'); i.value = text; document.body.appendChild(i); i.select();
    const ok = document.execCommand('copy'); i.remove(); return ok;
  }

  async function copyLink(fid, path) {
    let l = linkFor(fid, path);
    if (!l) {
      const token = newToken();
      if (!await act({ link: { token, fid, path, expires: null, password: null } })) return;
      links = await ship.scry('links').catch(() => links);
      l = linkFor(fid, path) || { token };
    }
    (await copyText(linkUrl(l.token))) ? toast('Link copied. Anyone with it can open this.') : toast(linkUrl(l.token));
  }

  function linkDialog(fid, path, name) {
    const l = linkFor(fid, path);
    const token = l ? l.token : newToken();
    const exp = l && l.expires ? `<option value="keep" selected>Until ${esc(when(l.expires))}</option>` : '';
    const d = dialog(`<h2>Link to “${esc(name)}”</h2>
      <p>Anyone with this link can open it, ship or not, and always sees the current version.</p>
      <div class="row"><input class="field" value="${esc(linkUrl(token))}" readonly aria-label="Link">
        <button type="button" class="btn" data-copy>${icon('link')}Copy</button></div>
      <label>Link works<select class="field" name="expires">${exp}<option value="0"${l && l.expires ? '' : ' selected'}>Until you turn it off</option><option value="1">For 1 day</option><option value="7">For 7 days</option><option value="30">For 30 days</option></select></label>
      <label>Password<input class="field" type="password" name="pw" autocomplete="new-password" placeholder="${l && l.password ? 'A password is set. Type a new one to change it' : 'Optional'}"></label>
      ${l && l.password ? '<label class="check" style="flex-direction:row;font-weight:400"><input type="checkbox" name="nopw"> Remove the password</label>' : ''}
      <div class="foot">${l ? '<button type="button" class="btn danger" data-unlink style="margin-right:auto">Turn off link</button>' : ''}<button class="btn" value="cancel">Cancel</button><button class="btn primary" value="ok">${l ? 'Save' : 'Create link'}</button></div>`,
    async (form) => {
      const e = form.expires.value;
      const expires = e === 'keep' ? l.expires : e === '0' ? null : Date.now() + Number(e) * 864e5;
      const password = form.nopw?.checked ? '' : form.pw.value ? form.pw.value : (l ? null : '');
      if (await act({ link: { token, fid, path, expires, password } }, l ? 'Link updated' : 'Link created')) {
        links = await ship.scry('links').catch(() => links);
        if (!l) await copyText(linkUrl(token));
        render();
      }
    });
    d.querySelector('[data-copy]').addEventListener('click', async () => {
      if (!l) { await act({ link: { token, fid, path, expires: null, password: null } }); links = await ship.scry('links').catch(() => links); }
      toast((await copyText(linkUrl(token))) ? 'Link copied' : linkUrl(token));
    });
    d.querySelector('[data-unlink]')?.addEventListener('click', async () => {
      if (await act({ unlink: token }, 'Link turned off')) { links = await ship.scry('links').catch(() => links); d.close(); render(); }
    });
  }

  // ----- upload: each 4 MiB piece to /drive-api/blob, then %put

  const hasSubtle = !!(window.crypto && crypto.subtle);
  const hex = (buf) => [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, '0')).join('');

  const uploads = [];
  function renderProgress() {
    const p = $('#progress');
    if (!uploads.length) { p.hidden = true; return; }
    const busy = uploads.filter((u) => !u.done && !u.failed).length;
    p.hidden = false;
    p.innerHTML = `<header><span>${busy ? `Uploading ${busy} ${busy === 1 ? 'item' : 'items'}` : 'Uploads complete'}</span>${busy ? '' : `<button class="icon-btn" data-do="close-progress" aria-label="Close">${icon('close')}</button>`}</header>
      <ul>${uploads.map((u) => `<li class="${u.done ? 'done' : ''} ${u.failed ? 'failed' : ''}"><span>${esc(u.name)}</span><small>${u.failed ? 'Failed' : u.done ? 'Done' : `${Math.round(u.pct)}%`}</small><div class="bar"><i style="width:${u.done ? 100 : u.pct}%"></i></div></li>`).join('')}</ul>`;
  }
  async function upload(files, fid, dir) {
    const f = folderOf(fid);
    if (!f || f.role !== 'rw') { toast('This folder is view only.', true); return; }
    const jobs = [...files].map((file) => ({ file, name: file.name, pct: 0, done: false, failed: false }));
    uploads.push(...jobs); renderProgress();
    for (const u of jobs) {
      try {
        const buf = await u.file.arrayBuffer();
        const chunks = [];
        for (let off = 0; off < buf.byteLength || (off === 0 && !chunks.length && buf.byteLength === 0); off += CHUNK) {
          if (!buf.byteLength) break;
          const piece = buf.slice(off, off + CHUNK);
          const res = await fetch('/drive-api/blob', { method: 'POST', body: piece, credentials: 'include', headers: { 'Content-Type': 'application/octet-stream' } });
          if (!res.ok) throw new Error(`upload ${res.status}`);
          chunks.push((await res.json()).hash);
          u.pct = Math.min(99, ((off + piece.byteLength) / buf.byteLength) * 100); renderProgress();
        }
        const hash = hasSubtle ? hex(await crypto.subtle.digest('SHA-256', buf)) : sha256(new Uint8Array(buf));
        await ship.poke({ put: { fid, path: join(dir, u.name), hash, size: buf.byteLength, mtime: u.file.lastModified || Date.now(), chunks } });
        u.done = true;
      } catch (e) { u.failed = true; console.error(e); }
      renderProgress();
    }
    await refresh();
  }

  // ----- events

  document.addEventListener('click', async (ev) => {
    const r = route();
    const t = ev.target;
    if (!t.closest('.menu') && !t.closest('[data-menu]')) closeMenu();
    const doEl = t.closest('[data-do]');
    if (doEl) {
      const what = doEl.dataset.do, items = selectedItems(), f = r.fid && folderOf(r.fid);
      if (what === 'upload') $('#file-input').click();
      else if (what === 'new-folder' || what === 'mkdir') newFolderDialog(r);
      else if (what === 'share-this') shareDialog(f);
      else if (what === 'share') shareDialog(items[0].f);
      else if (what === 'leave') leave(items[0].f);
      else if (what === 'clear') { selection.clear(); render(); }
      else if (what === 'delete') confirmDelete(items, r);
      else if (what === 'move') moveDialog(items, r);
      else if (what === 'rename') { renaming = items[0].name; render(); }
      else if (what === 'download') for (const it of items) { const a = document.createElement('a'); a.href = fileUrl(r.fid, it.hash); a.download = it.name; a.click(); }
      else if (what === 'prune') { await act({ prune: null }, 'Cleaned up history older than your setting'); refresh(); }
      else if (what === 'close-progress') { uploads.length = 0; renderProgress(); }
      return;
    }
    const menuBtn = t.closest('[data-menu]');
    if (menuBtn) {
      const it = currentItems(r).find((i) => i.name === menuBtn.dataset.menu); if (!it) return;
      const f = it.kind === 'top' ? it.f : folderOf(r.fid), rw = f.role === 'rw';
      // share links, for folders we host: the whole folder, a subfolder, or a file
      const fid = it.kind === 'top' ? it.f.fid : r.fid, lpath = it.kind === 'top' ? '' : it.path;
      const linkItems = mine(f) ? [{ icon: 'link', label: 'Copy link', run: () => copyLink(fid, lpath) },
        { icon: 'settings', label: 'Link settings…', run: () => linkDialog(fid, lpath, it.name) }] : [];
      const items = it.kind === 'top'
        ? [{ icon: 'folder', label: 'Open', run: () => (location.hash = folderHref(it.f.fid)) }, ...linkItems,
          mine(it.f) ? { icon: 'share', label: 'Share with ships', run: () => shareDialog(it.f) } : { icon: 'close', label: 'Leave folder', danger: true, run: () => leave(it.f) }]
        : it.kind === 'dir'
          ? [{ icon: 'folder', label: 'Open', run: () => (location.hash = folderHref(r.fid, it.path)) }, ...linkItems,
            ...(rw ? ['-', { icon: 'rename', label: 'Rename', run: () => { renaming = it.name; render(); } }, { icon: 'move', label: 'Move', run: () => moveDialog([it], r) }, { icon: 'trash', label: 'Delete', danger: true, run: () => confirmDelete([it], r) }] : [])]
          : [{ icon: 'download', label: 'Download', href: fileUrl(r.fid, it.hash), download: it.name },
            ...linkItems,
            { icon: 'history', label: 'Version history', run: () => (location.hash = folderHref(r.fid, r.dir, it.name)) },
            ...(rw ? ['-', { icon: 'rename', label: 'Rename', run: () => { renaming = it.name; render(); } }, { icon: 'move', label: 'Move', run: () => moveDialog([it], r) }, { icon: 'trash', label: 'Delete', danger: true, run: () => confirmDelete([it], r) }] : [])];
      openMenu(menuBtn, items); return;
    }
    const lm = t.closest('[data-linkmenu]');
    if (lm) {
      const l = links.find((x) => x.token === lm.dataset.linkmenu); if (!l) return;
      openMenu(lm, [{ icon: 'link', label: 'Copy link', run: async () => toast((await copyText(linkUrl(l.token))) ? 'Link copied' : linkUrl(l.token)) },
        { icon: 'settings', label: 'Link settings…', run: () => linkDialog(l.fid, l.path, l.path ? base(l.path) : l.folder) },
        '-', { icon: 'close', label: 'Turn off link', danger: true, run: async () => { if (await act({ unlink: l.token }, 'Link turned off')) { links = await ship.scry('links').catch(() => links); render(); } } }]);
      return;
    }
    const sortBtn = t.closest('[data-sort]');
    if (sortBtn) { const k = sortBtn.dataset.sort; sort = { key: k, dir: sort.key === k ? -sort.dir : 1 }; render(); return; }
    const acc = t.closest('[data-accept]');
    if (acc) { const [host, fid] = acc.dataset.accept.split('|'); if (await act({ accept: { host, fid } }, 'Added to your folders')) refresh(); return; }
    const dec = t.closest('[data-decline]');
    if (dec) { const [host, fid] = dec.dataset.decline.split('|'); if (await act({ decline: { host, fid } })) refresh(); return; }
    const rest = t.closest('[data-restore]');
    if (rest) {
      const fid = rest.dataset.fid || r.fid;
      if (await act({ restore: { fid, path: rest.dataset.path, hash: rest.dataset.restore } }, `Restored ${base(rest.dataset.path)}`)) { trees.delete(fid); refresh(); }
      return;
    }
    const unp = t.closest('[data-unpair]');
    if (unp) {
      dialog(`<h2>Remove “${esc(unp.dataset.name)}”?</h2><p>It stops syncing, and its access to your folders is revoked. Files already on it stay there.</p>
        <div class="foot"><button class="btn" value="cancel">Cancel</button><button class="btn danger solid" value="ok">Remove</button></div>`,
      async () => { if (await act({ unpair: unp.dataset.unpair }, `Removed ${unp.dataset.name}`)) refresh(); });
      return;
    }
    if (t.closest('#open-rail')) { $('#rail').classList.add('open'); return; }
    // clicking a row selects it; a file's name opens its details
    const row = t.closest('tbody tr[data-name]');
    if (row && !t.closest('a, button, input')) {
      selection = new Set([row.dataset.name]); render();
    }
  });

  document.addEventListener('change', (ev) => {
    const t = ev.target;
    if (t.matches('[data-pick]')) { t.checked ? selection.add(t.dataset.pick) : selection.delete(t.dataset.pick); render(); }
    else if (t.matches('[data-all]')) { selection = t.checked ? new Set(currentItems(route()).map((i) => i.name)) : new Set(); render(); }
    else if (t.id === 'file-input') { const r = route(); if (r.view === 'folder') upload(t.files, r.fid, r.dir); t.value = ''; }
  });

  document.addEventListener('submit', async (ev) => {
    if (ev.target.id !== 'keep-form') return;
    ev.preventDefault();
    const days = parseInt($('#keep').value, 10);
    if (days >= 1) { if (await act({ keep: days }, `Keeping deleted files and versions for ${days} days`)) refresh(); }
  });

  document.addEventListener('keydown', (ev) => {
    if (ev.target.matches?.('[data-rename]')) {
      if (ev.key === 'Enter') { ev.preventDefault(); commitRename(ev.target); }
      if (ev.key === 'Escape') { renaming = null; render(); }
      return;
    }
    if (ev.key === 'Escape') {
      if ($('.menu')) return closeMenu();
      if ($('#rail').classList.contains('open')) return $('#rail').classList.remove('open');
      const r = route();
      if (r.sel) { location.hash = folderHref(r.fid, r.dir); return; }
      if (selection.size) { selection.clear(); render(); }
    }
  });
  document.addEventListener('focusout', (ev) => { if (ev.target.matches?.('[data-rename]') && renaming) commitRename(ev.target); });

  let qT;
  $('#search').addEventListener('input', (ev) => {
    clearTimeout(qT);
    qT = setTimeout(() => { query = ev.target.value.trim(); render(); }, 150);
  });
  $('#search').addEventListener('keydown', (ev) => { if (ev.key === 'Escape') { ev.target.value = ''; query = ''; render(); } });

  // drag files onto a folder to upload
  let dragDepth = 0;
  addEventListener('dragenter', (ev) => {
    const r = route();
    if (r.view !== 'folder' || !ev.dataTransfer?.types.includes('Files') || folderOf(r.fid)?.role !== 'rw') return;
    dragDepth++; $('#drop-target').textContent = r.dir ? base(r.dir) : folderOf(r.fid).name; $('#drop-veil').hidden = false;
  });
  addEventListener('dragleave', () => { if (--dragDepth <= 0) { dragDepth = 0; $('#drop-veil').hidden = true; } });
  addEventListener('dragover', (ev) => { if (!$('#drop-veil').hidden) ev.preventDefault(); });
  addEventListener('drop', (ev) => {
    if ($('#drop-veil').hidden) return;
    ev.preventDefault(); dragDepth = 0; $('#drop-veil').hidden = true;
    const r = route(); upload(ev.dataTransfer.files, r.fid, r.dir);
  });

  async function leave(f) {
    dialog(`<h2>Leave “${esc(f.name)}”?</h2><p>Your computers keep the files they have, but stop syncing this folder. ${esc(f.host)} can share it with you again.</p>
      <div class="foot"><button class="btn" value="cancel">Cancel</button><button class="btn danger solid" value="ok">Leave</button></div>`,
    async () => { if (await act({ leave: f.fid }, `Left ${f.name}`)) { selection.clear(); refresh(); } });
  }

  addEventListener('hashchange', () => { selection.clear(); renaming = null; query = ''; $('#search').value = ''; render(); });

  // ----- sha-256, for ships served over plain http (no crypto.subtle)
  function sha256(bytes) {
    const K = new Uint32Array([0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2]);
    const H = new Uint32Array([0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19]);
    const len = bytes.length, total = ((len + 9 + 63) >> 6) << 6, m = new Uint8Array(total);
    m.set(bytes); m[len] = 0x80;
    const dv = new DataView(m.buffer); dv.setUint32(total - 4, len << 3); dv.setUint32(total - 8, Math.floor(len / 0x20000000));
    const W = new Uint32Array(64), ror = (x, n) => (x >>> n) | (x << (32 - n));
    for (let o = 0; o < total; o += 64) {
      for (let i = 0; i < 16; i++) W[i] = dv.getUint32(o + i * 4);
      for (let i = 16; i < 64; i++) {
        const s0 = ror(W[i - 15], 7) ^ ror(W[i - 15], 18) ^ (W[i - 15] >>> 3), s1 = ror(W[i - 2], 17) ^ ror(W[i - 2], 19) ^ (W[i - 2] >>> 10);
        W[i] = (W[i - 16] + s0 + W[i - 7] + s1) | 0;
      }
      let [a, b, c, d, e, f, g, h] = H;
      for (let i = 0; i < 64; i++) {
        const t1 = (h + (ror(e, 6) ^ ror(e, 11) ^ ror(e, 25)) + ((e & f) ^ (~e & g)) + K[i] + W[i]) | 0;
        const t2 = ((ror(a, 2) ^ ror(a, 13) ^ ror(a, 22)) + ((a & b) ^ (a & c) ^ (b & c))) | 0;
        h = g; g = f; f = e; e = (d + t1) | 0; d = c; c = b; b = a; a = (t1 + t2) | 0;
      }
      H[0] += a; H[1] += b; H[2] += c; H[3] += d; H[4] += e; H[5] += f; H[6] += g; H[7] += h;
    }
    return [...H].map((x) => x.toString(16).padStart(8, '0')).join('');
  }

  // ----- start

  try {
    ship = await Ship.connect('drive');
    st = ship.initial;
    links = await ship.scry('links').catch(() => []);
    await render();
    let t;
    await ship.subscribe('/v1/folders', () => { clearTimeout(t); t = setTimeout(refresh, 300); },
      () => toast('Live updates stopped. Reload to see new changes.', true));
  } catch (e) {
    $('#content').innerHTML = emptyState('folder', 'Can’t reach Drive on your ship', 'Make sure the Drive app is installed and running, then reload this page.', '');
  }
  if ('serviceWorker' in navigator) navigator.serviceWorker.register('/drive/sw.js', { scope: '/drive/' }).catch(() => {});
})();
