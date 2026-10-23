/* service worker: installable + offline shell. Bump VERSION to invalidate caches.
 * Only runs over HTTPS or localhost. Registered from app.js with scope /drive/. */
const VERSION = 'drive-v4';
const SHELL = ['/drive/', '/drive/index.html', '/drive/app.js', '/drive/ship.js', '/drive/style.css', '/drive/manifest.json', '/drive/tile.svg'];
self.addEventListener('install', (e) => e.waitUntil(caches.open(VERSION).then((c) => c.addAll(SHELL)).then(() => self.skipWaiting())));
self.addEventListener('activate', (e) => e.waitUntil(caches.keys().then((ks) => Promise.all(ks.filter((k) => k !== VERSION).map((k) => caches.delete(k)))).then(() => self.clients.claim())));
self.addEventListener('fetch', (e) => {
  const req = e.request; if (req.method !== 'GET') return;
  const url = new URL(req.url); if (url.origin !== self.location.origin) return;
  if (url.pathname.startsWith('/~/scry/')) {          // ship data: network first, cache fallback
    e.respondWith(fetch(req).then((r) => { if (r.ok) caches.open(VERSION).then((c) => c.put(req, r.clone())); return r; })
      .catch(() => caches.match(req).then((r) => r || new Response('{"offline":true}', { status: 503, headers: { 'Content-Type': 'application/json' } }))));
  } else if (url.pathname === '/drive' || url.pathname.startsWith('/drive/')) {   // shell: stale-while-revalidate
    const key = url.pathname === '/drive' ? '/drive/' : req;
    e.respondWith(caches.match(key).then((cached) => {
      const net = fetch(req).then((r) => { if (r.ok) caches.open(VERSION).then((c) => c.put(key, r.clone())); return r; }).catch(() => cached);
      return cached || net;
    }));
  }
});
