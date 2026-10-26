/* preview.js: Drive's file viewer, and its file-type icons. Used by the
 * web app (/drive/preview.js) and by share pages (/drive-api/preview.js,
 * public), so it brings its own styles and icons and reads only the
 * colour tokens both pages define.
 *
 *   DrivePreview.icon(name, cls)   an icon for the file's type
 *   DrivePreview.kind(name)        image video audio pdf text code sheet
 *                                  doc slides archive file
 *   DrivePreview.open({ items: [{name, size, mtime, url, download}], index,
 *     sub, closable, brand (html, in place of Close), actions(item) -> [{icon, label, href, download, run}],
 *     onNav(i), onClose() })      show items[index]; again to change it
 *   DrivePreview.close()
 */
window.DrivePreview = (() => {
  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const TEXT_MAX = 256 << 10;   // a text preview shows the first 256 KB

  // ----- kinds, by extension

  const KINDS = {
    image: 'png jpg jpeg gif webp avif bmp ico svg heic heif tif tiff',
    video: 'mp4 m4v webm mov ogv mkv avi',
    audio: 'mp3 m4a aac wav ogg oga opus flac aiff',
    pdf: 'pdf',
    sheet: 'csv tsv xls xlsx ods numbers',
    doc: 'txt md markdown log rtf doc docx odt pages tex',
    slides: 'ppt pptx odp key',
    archive: 'zip tar gz tgz bz2 xz 7z rar dmg iso zst',
    code: 'js mjs cjs ts tsx jsx json html htm css scss sass less py go rs c h cc cpp hpp java kt swift rb php sh bash zsh fish hoon nix sql yaml yml toml ini conf cfg xml lua pl r cs zig vue svelte dart ex exs erl hs ml clj el vim mk cmake gradle diff patch env gitignore dockerfile makefile',
  };
  const byExt = new Map();
  for (const [k, list] of Object.entries(KINDS)) for (const e of list.split(' ')) byExt.set(e, k);
  const ext = (name) => { const n = name.toLowerCase(), i = n.lastIndexOf('.'); return i > 0 ? n.slice(i + 1) : n; };
  const kind = (name) => byExt.get(ext(name)) || 'file';
  // what the viewer can show as text: these, and anything small that sniffs as text
  const TEXTY = new Set(['txt', 'md', 'markdown', 'log', 'csv', 'tsv', 'tex', 'svg', ...KINDS.code.split(' ')]);

  // ----- icons: a page outline with a mark for the kind, tinted per kind

  const page = '<path d="M6.5 3h7l4 4v12.5A1.5 1.5 0 0 1 16 21H6.5A1.5 1.5 0 0 1 5 19.5v-15A1.5 1.5 0 0 1 6.5 3z"/><path d="M13.5 3v4h4" fill="none"/>';
  const MARKS = {
    file: '',
    image: '<circle cx="9.25" cy="11.25" r="1.25"/><path d="m7.5 18 2.75-3 1.75 1.75 1.5-1.5 1.5 2.75"/>',
    video: '<path d="M9.75 11.75v5.5l4.5-2.75z"/>',
    audio: '<path d="M10.5 17.25v-5.5l4-1v5"/><circle cx="9.25" cy="17.25" r="1.25"/><circle cx="13.25" cy="15.75" r="1.25"/>',
    pdf: '<path d="M8 12h6.5M8 15h6.5M8 18h3.5"/>',
    doc: '<path d="M8 12h6.5M8 15h6.5M8 18h3.5"/>',
    code: '<path d="m9.5 12.5-2 2.5 2 2.5M13 12.5l2 2.5-2 2.5"/>',
    sheet: '<path d="M7.75 11.5h7v6.75h-7zM7.75 14.9h7M11.25 11.5v6.75"/>',
    slides: '<path d="M7.75 12h7v4.75h-7zM11.25 16.75v1.75"/>',
    archive: '<path d="M10.5 3.5v1.5M12 6.25v1.5M10.5 9v1.5M12 11.75v1.5M10.25 14.5h2v3h-2z"/>',
  };
  const UI = {
    close: '<path d="m6.5 6.5 11 11M17.5 6.5l-11 11"/>',
    prev: '<path d="m14.5 6-6 6 6 6"/>',
    next: '<path d="m9.5 6 6 6-6 6"/>',
    download: '<path d="M12 4.5V16M7.5 11.5 12 16l4.5-4.5M5 15.5v3A1.5 1.5 0 0 0 6.5 20h11a1.5 1.5 0 0 0 1.5-1.5v-3"/>',
    link: '<path d="M10 14a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1 1"/><path d="M14 10a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1-1"/>',
    history: '<path d="M4.5 12a7.5 7.5 0 1 0 2.2-5.3L4.5 9"/><path d="M4.5 4.5V9H9"/><path d="M12 8v4.5l3 1.8"/>',
  };
  const sym = (id, body) => `<symbol id="pv-${id}" viewBox="0 0 24 24">${body}</symbol>`;
  const SPRITE = `<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
    ${Object.entries(MARKS).map(([k, m]) => sym('t-' + k, page + m.replace(/<(path|circle)/g, '<$1 fill="none"'))).join('')}
    ${Object.entries(UI).map(([k, b]) => sym(k, b)).join('')}</defs></svg>`;
  const ui = (id) => `<svg class="pv-ic" aria-hidden="true"><use href="#pv-${id}"/></svg>`;
  const icon = (name, cls = '') => { const k = kind(name); return `<svg class="ic ft ft-${k} ${cls}" aria-hidden="true"><use href="#pv-t-${k}"/></svg>`; };

  const CSS = `
.ft { --ft: var(--fg2); color: var(--ft); fill: color-mix(in srgb, var(--ft) 12%, transparent); }
.ft-image { --ft: #0e7490; } .ft-video { --ft: #7c3aed; } .ft-audio { --ft: #be185d; } .ft-pdf { --ft: #c4312b; }
.ft-doc { --ft: #0061fe; } .ft-code { --ft: #57534e; } .ft-sheet { --ft: #15803d; } .ft-slides { --ft: #c2410c; } .ft-archive { --ft: #a85d00; }
@media (prefers-color-scheme: dark) {
  .ft-image { --ft: #5cc8db; } .ft-video { --ft: #a78bfa; } .ft-audio { --ft: #f472b6; } .ft-pdf { --ft: #ff7a72; }
  .ft-doc { --ft: #4d8dff; } .ft-code { --ft: #b5aea6; } .ft-sheet { --ft: #5fc792; } .ft-slides { --ft: #fb923c; } .ft-archive { --ft: #f0a64a; }
}
.pv { position: fixed; inset: 0; z-index: 50; display: flex; flex-direction: column; background: var(--bg); color: var(--fg); }
.pv-ic { width: 20px; height: 20px; flex: none; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.pv-bar { display: flex; align-items: center; gap: 12px; height: 64px; padding: 0 16px; border-bottom: 1px solid var(--line); flex: none; }
.pv-brand { display: flex; align-items: center; padding-right: 16px; margin-right: 4px; border-right: 1px solid var(--line); height: 32px; }
.pv-title { display: flex; align-items: center; gap: 12px; min-width: 0; margin-right: auto; }
.pv-title .ic { width: 28px; height: 28px; stroke-width: 1.5; flex: none; }
.pv-title div { display: flex; flex-direction: column; min-width: 0; }
.pv-title b { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pv-title small { color: var(--fg2); font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-variant-numeric: tabular-nums; }
.pv-acts { display: flex; gap: 8px; flex: none; }
.pv-btn { display: inline-flex; align-items: center; gap: 8px; height: 36px; padding: 0 14px; border-radius: 8px; border: 1px solid var(--line); background: var(--bg); color: var(--fg); font: inherit; font-weight: 550; cursor: pointer; white-space: nowrap; text-decoration: none; }
.pv-btn:hover { background: var(--rail); }
.pv-btn.primary { background: var(--blue); border-color: var(--blue); color: var(--blue-ink); }
.pv-btn.primary:hover { filter: brightness(1.06); }
.pv-btn.icon { width: 36px; padding: 0; justify-content: center; border-color: transparent; background: none; color: var(--fg2); }
.pv-btn.icon:hover { background: var(--rail); color: var(--fg); }
.pv-btn .pv-ic { width: 18px; height: 18px; }
.pv-stage { position: relative; flex: 1; min-height: 0; display: grid; place-items: center; overflow: auto; padding: 24px 72px;
  background: color-mix(in srgb, var(--fg) 4%, var(--bg)); }
.pv-stage > img, .pv-stage > video { max-width: 100%; max-height: calc(100dvh - 64px - 48px); object-fit: contain; border-radius: 4px; box-shadow: 0 2px 6px rgb(0 0 0 / .08), 0 12px 32px rgb(0 0 0 / .12); background: var(--bg); }
.pv-stage > img.checker { background: repeating-conic-gradient(color-mix(in srgb, var(--fg) 6%, var(--bg)) 0 25%, var(--bg) 0 50%) 0 0 / 16px 16px; }
.pv-stage > video { background: #000; }
.pv-stage > iframe { width: 100%; height: 100%; border: 0; border-radius: 4px; background: var(--bg); box-shadow: 0 2px 6px rgb(0 0 0 / .08); }
.pv-stage.pdf { padding: 0 64px; }
.pv-paper { align-self: start; width: min(880px, 100%); background: var(--bg); border-radius: 4px; box-shadow: 0 2px 6px rgb(0 0 0 / .06), 0 8px 24px rgb(0 0 0 / .08); }
.pv-paper pre { margin: 0; padding: 24px 28px; font: 13px/1.6 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; white-space: pre-wrap; overflow-wrap: anywhere; tab-size: 4; }
.pv-paper:has(table) { width: auto; max-width: 100%; }
.pv-paper .md { padding: 32px 40px; font-size: 15px; line-height: 1.6; overflow-wrap: anywhere; }
.pv-paper .md > :first-child { margin-top: 0; } .pv-paper .md > :last-child { margin-bottom: 0; }
.pv-paper .md h1 { font-size: 26px; line-height: 1.25; margin: 0 0 16px; letter-spacing: -0.015em; }
.pv-paper .md h2 { font-size: 20px; margin: 28px 0 10px; } .pv-paper .md h3, .pv-paper .md h4 { font-size: 16px; margin: 24px 0 8px; }
.pv-paper .md p, .pv-paper .md ul, .pv-paper .md ol, .pv-paper .md blockquote, .pv-paper .md pre { margin: 0 0 14px; }
.pv-paper .md ul, .pv-paper .md ol { padding-left: 24px; }
.pv-paper .md blockquote { border-left: 3px solid var(--line); padding: 2px 0 2px 16px; color: var(--fg2); }
.pv-paper .md code { font: 13px ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; background: var(--rail); padding: 1px 5px; border-radius: 4px; }
.pv-paper .md pre { background: var(--rail); border-radius: 6px; padding: 14px 16px; white-space: pre-wrap; }
.pv-paper .md pre code { background: none; padding: 0; }
.pv-paper .md hr { border: 0; border-top: 1px solid var(--line); margin: 24px 0; }
.pv-paper .more { margin: 0; padding: 12px 28px 20px; color: var(--fg2); font-size: 13px; border-top: 1px solid var(--line); }
.pv-paper table { border-collapse: collapse; font-size: 13px; font-variant-numeric: tabular-nums; }
.pv-paper .scroll { overflow: auto; max-height: 100%; }
.pv-paper td, .pv-paper th { border: 1px solid var(--line); padding: 6px 10px; text-align: left; white-space: nowrap; max-width: 320px; overflow: hidden; text-overflow: ellipsis; }
.pv-paper th { background: var(--rail); font-weight: 600; position: sticky; top: 0; }
.pv-card { text-align: center; max-width: 420px; }
.pv-card .ic { width: 96px; height: 96px; stroke-width: 1.1; }
.pv-card h2 { font-size: 18px; font-weight: 650; margin: 12px 0 4px; overflow-wrap: anywhere; }
.pv-card p { color: var(--fg2); margin: 0 0 20px; }
.pv-card audio { width: min(420px, 80vw); margin-top: 8px; }
.pv-nav { position: absolute; top: 50%; transform: translateY(-50%); width: 44px; height: 44px; border-radius: 50%; border: 1px solid var(--line); background: var(--bg); color: var(--fg); display: grid; place-items: center; cursor: pointer; box-shadow: 0 2px 6px rgb(0 0 0 / .08); z-index: 1; }
.pv-nav:hover { background: var(--rail); }
.pv-nav.prev { left: 16px; } .pv-nav.next { right: 16px; }
.pv-spin { width: 28px; height: 28px; border-radius: 50%; border: 3px solid var(--line); border-top-color: var(--blue); animation: pv-spin .8s linear infinite; }
@keyframes pv-spin { to { transform: rotate(360deg); } }
@media (max-width: 720px) {
  .pv-bar { padding: 0 8px; gap: 8px; }
  .pv-btn .lbl, .pv-brand span { display: none; } .pv-btn:not(.icon) { width: 36px; padding: 0; justify-content: center; }
  .pv-stage { padding: 12px; } .pv-stage.pdf { padding: 0; }
  .pv-nav { top: auto; bottom: 16px; transform: none; }
  .pv-paper pre { padding: 16px; } .pv-paper .md { padding: 20px; }
}
@media (prefers-reduced-motion: reduce) { .pv-spin { animation: none; } }`;

  let injected = false;
  function inject() {
    if (injected) return; injected = true;
    const st = document.createElement('style'); st.textContent = CSS; document.head.append(st);
    document.body.insertAdjacentHTML('afterbegin', SPRITE);
  }
  if (document.body) inject(); else addEventListener('DOMContentLoaded', inject);

  // ----- the viewer

  function size(n) {
    if (n == null) return '';
    if (n < 1024) return `${n} B`;
    const u = ['KB', 'MB', 'GB', 'TB']; let i = -1;
    do { n /= 1024; i++; } while (n >= 1024 && i < u.length - 1);
    return `${n < 10 ? n.toFixed(1) : Math.round(n)} ${u[i]}`;
  }

  let el = null, opts = null, idx = 0, seq = 0, back = null;

  function open(o) {
    inject();
    opts = o; idx = Math.max(0, Math.min(o.index || 0, o.items.length - 1));
    if (!el) {
      back = document.activeElement;
      el = document.createElement('div');
      el.className = 'pv'; el.setAttribute('role', 'dialog'); el.setAttribute('aria-modal', 'true');
      document.body.append(el);
      document.documentElement.style.overflow = 'hidden';
    }
    show();
  }

  function close(quiet) {
    if (!el) return;
    seq++; el.remove(); el = null;
    document.documentElement.style.overflow = '';
    back?.focus?.();
    const cb = opts?.onClose; opts = null;
    if (!quiet && cb) cb();
  }

  function go(d) {
    const n = idx + d;
    if (!opts || n < 0 || n >= opts.items.length) return;
    if (opts.onNav) opts.onNav(n); else { idx = n; show(); }
  }

  function show() {
    const it = opts.items[idx], many = opts.items.length > 1;
    const acts = (opts.actions ? opts.actions(it) : [{ icon: 'download', label: 'Download', href: it.download, download: it.name, primary: true }]);
    const meta = [many ? `${idx + 1} of ${opts.items.length}` : '', size(it.size), opts.sub || ''].filter(Boolean).join(' · ');
    el.setAttribute('aria-label', it.name);
    el.innerHTML = `<header class="pv-bar">
        ${opts.closable === false ? (opts.brand ? `<span class="pv-brand">${opts.brand}</span>` : '') : `<button class="pv-btn icon" data-pv="close" aria-label="Close">${ui('close')}</button>`}
        <div class="pv-title">${icon(it.name)}<div><b title="${esc(it.name)}">${esc(it.name)}</b><small>${esc(meta)}</small></div></div>
        <div class="pv-acts">${acts.map((a, i) => a.href
          ? `<a class="pv-btn ${a.primary ? 'primary' : ''}" href="${esc(a.href)}" ${a.download ? `download="${esc(a.download)}"` : ''} aria-label="${esc(a.label)}">${ui(a.icon)}<span class="lbl">${esc(a.label)}</span></a>`
          : `<button class="pv-btn ${a.primary ? 'primary' : ''}" data-act="${i}" aria-label="${esc(a.label)}">${ui(a.icon)}<span class="lbl">${esc(a.label)}</span></button>`).join('')}</div>
      </header>
      <div class="pv-stage"><div class="pv-spin" role="status" aria-label="Loading"></div></div>
      ${many && idx > 0 ? `<button class="pv-nav prev" data-pv="prev" aria-label="Previous file">${ui('prev')}</button>` : ''}
      ${many && idx < opts.items.length - 1 ? `<button class="pv-nav next" data-pv="next" aria-label="Next file">${ui('next')}</button>` : ''}`;
    el.querySelectorAll('[data-act]').forEach((b) => b.addEventListener('click', () => acts[+b.dataset.act].run(it)));
    (el.querySelector('[data-pv="close"]') || el.querySelector('.pv-btn'))?.focus();
    render(it, ++seq);
    // the next image, ready before it's asked for
    const nx = opts.items[idx + 1];
    if (nx && kind(nx.name) === 'image') new Image().src = nx.url;
  }

  const stage = () => el.querySelector('.pv-stage');
  function none(it, why) {
    const s = stage(); s.className = 'pv-stage';
    s.innerHTML = `<div class="pv-card">${icon(it.name)}<h2>${esc(it.name)}</h2><p>${esc(why || 'There’s no preview for this kind of file.')} ${size(it.size)}</p>
      <a class="pv-btn primary" href="${esc(it.download)}" download="${esc(it.name)}">${ui('download')}Download</a></div>`;
  }

  async function render(it, my) {
    const s = stage(), k = kind(it.name), e = ext(it.name);
    const fail = (why) => () => { if (my === seq) none(it, why); };
    if (k === 'image') {
      const img = new Image();
      img.alt = it.name; img.decoding = 'async';
      if (/^(png|gif|webp|svg|ico|avif)$/.test(e)) img.className = 'checker';
      img.onload = () => { if (my === seq) { s.className = 'pv-stage'; s.replaceChildren(img); } };
      img.onerror = fail('Your browser can’t show this image.');
      img.src = it.url;
      return;
    }
    if (k === 'video') {
      const v = document.createElement('video');
      v.controls = true; v.preload = 'metadata'; v.playsInline = true;
      v.onerror = fail('Your browser can’t play this video.');
      v.src = it.url; s.className = 'pv-stage'; s.replaceChildren(v);
      return;
    }
    if (k === 'audio') {
      s.className = 'pv-stage';
      s.innerHTML = `<div class="pv-card">${icon(it.name)}<h2>${esc(it.name)}</h2><audio controls preload="metadata"></audio></div>`;
      const a = s.querySelector('audio'); a.onerror = fail('Your browser can’t play this file.'); a.src = it.url;
      return;
    }
    if (k === 'pdf') {
      s.className = 'pv-stage pdf';
      s.innerHTML = `<iframe title="${esc(it.name)}" src="${esc(it.url)}"></iframe>`;
      return;
    }
    // text: known kinds, or anything small enough to sniff
    if (!TEXTY.has(e) && !(k === 'file' && it.size <= 2 << 20)) return none(it);
    let buf;
    try {
      const r = await fetch(it.url, { headers: { Range: `bytes=0-${TEXT_MAX - 1}` }, credentials: 'same-origin' });
      if (!r.ok) throw new Error(r.status);
      buf = new Uint8Array(await r.arrayBuffer());
    } catch { return fail('Couldn’t load this file. Try again.')(); }
    if (my !== seq) return;
    let text;
    try {
      if (buf.subarray(0, 8192).includes(0)) throw new Error('binary');
      text = new TextDecoder('utf-8', { fatal: true }).decode(buf.subarray(0, TEXT_MAX), { stream: true });
    } catch { return none(it); }
    const cut = it.size > TEXT_MAX;
    const more = cut ? `<p class="more">Showing the first ${size(TEXT_MAX)} of ${size(it.size)}. Download it to see the rest.</p>` : '';
    s.className = 'pv-stage';
    if ((e === 'csv' || e === 'tsv') && (text = text.replace(/^﻿/, ''))) {
      const rows = parseCsv(cut ? text.slice(0, text.lastIndexOf('\n')) : text, e === 'tsv' ? '\t' : ',', 1000);
      if (rows.length > 1) {
        const [h, ...body] = rows;
        s.innerHTML = `<div class="pv-paper"><div class="scroll"><table><thead><tr>${h.map((c) => `<th>${esc(c)}</th>`).join('')}</tr></thead>
          <tbody>${body.map((r) => `<tr>${r.map((c) => `<td title="${esc(c)}">${esc(c)}</td>`).join('')}</tr>`).join('')}</tbody></table></div>
          ${rows.length > 1000 || cut ? `<p class="more">Showing the first ${body.length} rows. Download it to see the rest.</p>` : ''}</div>`;
        return;
      }
    }
    if ((e === 'md' || e === 'markdown') && !cut) { s.innerHTML = `<div class="pv-paper"><div class="md">${markdown(text)}</div></div>`; return; }
    s.innerHTML = `<div class="pv-paper"><pre></pre>${more}</div>`;
    s.querySelector('pre').textContent = text;
  }

  // Markdown, the common parts: headings, lists, quotes, code, rules,
  // links and emphasis. Everything is escaped first; links must be web ones.
  function markdown(src) {
    const inline = (t) => esc(t)
      .replace(/`([^`]+)`/g, '<code>$1</code>')
      .replace(/\*\*([^*]+)\*\*|__([^_]+)__/g, (_, a, b) => `<strong>${a || b}</strong>`)
      .replace(/(^|[^*\w])\*([^*\s][^*]*)\*|(^|[^_\w])_([^_\s][^_]*)_/g, (_, p1, a, p2, b) => `${p1 ?? p2}<em>${a ?? b}</em>`)
      .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+|mailto:[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
    const out = [], lines = src.replace(/\r\n?/g, '\n').split('\n');
    let para = [], list = null, quote = [];
    const flush = () => {
      if (para.length) { out.push(`<p>${inline(para.join(' '))}</p>`); para = []; }
      if (list) { out.push(`<${list.tag}>${list.items.map((i) => `<li>${inline(i)}</li>`).join('')}</${list.tag}>`); list = null; }
      if (quote.length) { out.push(`<blockquote>${markdown(quote.join('\n'))}</blockquote>`); quote = []; }
    };
    for (let i = 0; i < lines.length; i++) {
      const l = lines[i]; let m;
      if (/^```/.test(l)) {
        flush(); const code = [];
        while (++i < lines.length && !/^```/.test(lines[i])) code.push(lines[i]);
        out.push(`<pre><code>${esc(code.join('\n'))}</code></pre>`);
      } else if ((m = l.match(/^(#{1,6})\s+(.*)$/))) { flush(); const n = Math.min(m[1].length, 4); out.push(`<h${n}>${inline(m[2])}</h${n}>`); }
      else if (/^\s*([-*_])(\s*\1){2,}\s*$/.test(l)) { flush(); out.push('<hr>'); }
      else if ((m = l.match(/^>\s?(.*)$/))) { if (para.length || list) flush(); quote.push(m[1]); }
      else if ((m = l.match(/^\s*([-*+]|\d+[.)])\s+(.*)$/))) {
        const tag = /\d/.test(m[1]) ? 'ol' : 'ul';
        if (para.length || quote.length || (list && list.tag !== tag)) flush();
        (list ||= { tag, items: [] }).items.push(m[2]);
      } else if (!l.trim()) flush();
      else if (list && /^\s+/.test(l)) list.items[list.items.length - 1] += ' ' + l.trim();
      else { if (list || quote.length) flush(); para.push(l.trim()); }
    }
    flush();
    return out.join('');
  }

  // CSV, with quoted fields; stops after [max] rows (plus one, to know)
  function parseCsv(t, sep, max) {
    const rows = []; let row = [], f = '', q = false;
    for (let i = 0; i < t.length; i++) {
      const c = t[i];
      if (q) { if (c === '"') { if (t[i + 1] === '"') { f += '"'; i++; } else q = false; } else f += c; continue; }
      if (c === '"' && f === '') q = true;
      else if (c === sep) { row.push(f); f = ''; }
      else if (c === '\n' || c === '\r') {
        if (c === '\r' && t[i + 1] === '\n') i++;
        row.push(f); f = ''; rows.push(row); row = [];
        if (rows.length > max) return rows;
      } else f += c;
    }
    if (f !== '' || row.length) { row.push(f); rows.push(row); }
    return rows;
  }

  // ----- keys, clicks and swipes, while open

  addEventListener('keydown', (ev) => {
    if (!el) return;
    const t = ev.target, inMedia = t.matches?.('video, audio, input, textarea, iframe');
    if (ev.key === 'Escape' && opts.closable !== false) { ev.preventDefault(); ev.stopImmediatePropagation(); close(); }
    else if (ev.key === 'ArrowLeft' && !inMedia) { ev.preventDefault(); ev.stopImmediatePropagation(); go(-1); }
    else if (ev.key === 'ArrowRight' && !inMedia) { ev.preventDefault(); ev.stopImmediatePropagation(); go(1); }
    else if (ev.key === 'Tab') {
      // keep focus inside the viewer
      const f = [...el.querySelectorAll('button, a[href], video, audio, iframe')];
      if (!f.length) return;
      const i = f.indexOf(document.activeElement);
      if (ev.shiftKey && i <= 0) { ev.preventDefault(); f[f.length - 1].focus(); }
      else if (!ev.shiftKey && i === f.length - 1) { ev.preventDefault(); f[0].focus(); }
    } else ev.stopImmediatePropagation();
  }, true);
  addEventListener('click', (ev) => {
    if (!el || !el.contains(ev.target)) return;
    const b = ev.target.closest('[data-pv]');
    if (!b) return;
    ev.preventDefault(); ev.stopPropagation();
    if (b.dataset.pv === 'close') close();
    else go(b.dataset.pv === 'prev' ? -1 : 1);
  }, true);
  let tx = null;
  addEventListener('touchstart', (ev) => { if (el && ev.touches.length === 1) tx = [ev.touches[0].clientX, ev.touches[0].clientY]; }, { passive: true });
  addEventListener('touchend', (ev) => {
    if (!el || !tx || ev.target.closest('video, audio, .pv-paper, iframe')) return (tx = null);
    const dx = ev.changedTouches[0].clientX - tx[0], dy = ev.changedTouches[0].clientY - tx[1]; tx = null;
    if (Math.abs(dx) > 60 && Math.abs(dx) > 2 * Math.abs(dy)) go(dx < 0 ? 1 : -1);
  });

  return { open, close, icon, kind, isOpen: () => !!el };
})();
