/* ship.js — tiny zero-dependency client for talking to the ship from a page
 * served by the desk's fileserver (same origin, logged in with the urbauth cookie).
 *
 *   const ship = await Ship.connect('drive');       // learns ~ship from /x/state
 *   await ship.scry('items');                       // GET /~/scry/drive/items.json
 *   await ship.poke({ add: 'hello' });              // PUT /~/channel/<id>, mark %json
 *   ship.subscribe('/updates', (fact) => ...);      // SSE facts from the agent
 */
window.Ship = (() => {
  const state = { app: '', ship: '', id: `web-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`, seq: 0, es: null, pending: new Map(), subs: new Map() };
  const url = () => `/~/channel/${state.id}`;
  async function put(msgs) {
    const r = await fetch(url(), { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(msgs) });
    if (!r.ok) throw new Error(`channel ${r.status}`);
    if (!state.es) open();
  }
  function open() {
    const es = new EventSource(url()); state.es = es;
    es.onmessage = (ev) => {
      let d; try { d = JSON.parse(ev.data); } catch { return; }
      if (ev.lastEventId) put([{ id: ++state.seq, action: 'ack', 'event-id': Number(ev.lastEventId) }]).catch(() => {});
      if (d.response === 'poke') { const p = state.pending.get(d.id); if (!p) return; state.pending.delete(d.id); d.ok ? p.res() : p.rej(new Error(d.err || 'poke failed')); }
      if (d.response === 'subscribe' && !d.ok) { const s = state.subs.get(d.id); if (s) s.onErr && s.onErr(new Error(d.err)); }
      if (d.response === 'diff') { const s = state.subs.get(d.id); if (s) s.onFact(d.json); }   // facts arrive as "diff" with the mark's json
      if (d.response === 'quit') { const s = state.subs.get(d.id); if (s) { state.subs.delete(d.id); subscribe(s.path, s.onFact, s.onErr); } }
    };
    es.onerror = () => { es.close(); state.es = null; };
  }
  async function scry(path) {
    const r = await fetch(`/~/scry/${state.app}/${path}.json`, { cache: 'no-store' });
    if (!r.ok) throw new Error(`scry ${path}: ${r.status}`);
    return r.json();
  }
  async function poke(json, mark = 'json') {
    const id = ++state.seq;
    const p = new Promise((res, rej) => state.pending.set(id, { res, rej }));
    await put([{ id, action: 'poke', ship: state.ship, app: state.app, mark, json }]);
    setTimeout(() => { const pd = state.pending.get(id); if (pd) { state.pending.delete(id); pd.res(); } }, 3000);  // no SSE ack? assume ok
    return p;
  }
  async function subscribe(path, onFact, onErr) {
    const id = ++state.seq; state.subs.set(id, { path, onFact, onErr });
    await put([{ id, action: 'subscribe', ship: state.ship, app: state.app, path }]);
    return () => put([{ id: ++state.seq, action: 'unsubscribe', subscription: id }]);
  }
  async function connect(app) {
    state.app = app;
    const st = await scry('state');
    state.ship = String(st.ship || '~zod').replace(/^~/, '');
    return { scry, poke, subscribe, state, initial: st };
  }
  return { connect };
})();
