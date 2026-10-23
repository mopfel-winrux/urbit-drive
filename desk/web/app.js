/* app.js — the page logic. Replace with your own UI; keep the shape:
 * load state by scry, mutate by poke, refresh from /updates facts. */
'use strict';
const $ = (s) => document.querySelector(s);
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
let ship, items = [], title = '';

function render() {
  $('#title').textContent = title;
  $('#items').innerHTML = items.length ? items.map((i) => `
    <li class="${i.done ? 'done' : ''}" data-id="${i.id}">
      <input type="checkbox" ${i.done ? 'checked' : ''} data-toggle="${i.id}">
      <span>${esc(i.text)}</span> <small>${new Date(i.when).toLocaleString()}</small>
      <button data-del="${i.id}">×</button>
    </li>`).join('') : '<li class="muted">Nothing yet.</li>';
  $('#status').textContent = `~${ship.state.ship} · ${items.length} items`;
}
function applyFact(f) {
  if (f.item) { const i = items.findIndex((x) => x.id === f.item.id); if (i >= 0) items[i] = f.item; else items.push(f.item); }
  if (f.del != null) items = items.filter((x) => x.id !== f.del);
  if (f.title != null) title = f.title;
  render();
}
async function main() {
  if ('serviceWorker' in navigator) navigator.serviceWorker.register('/drive/sw.js', { scope: '/drive/' }).catch(() => {});
  ship = await Ship.connect('drive');
  items = ship.initial.items; title = ship.initial.title;
  render();
  ship.subscribe('/updates', applyFact, (e) => console.error('subscription failed', e));
  $('#add').onsubmit = async (e) => {
    e.preventDefault(); const text = $('#text').value.trim(); if (!text) return;
    try { await ship.poke({ add: text }); $('#text').value = ''; } catch (err) { alert(`Ship rejected: ${err.message}`); }
  };
  $('#items').onclick = async (e) => {
    const del = e.target.dataset.del, tog = e.target.dataset.toggle;
    if (del) await ship.poke({ del: Number(del) });
    if (tog) { const it = items.find((x) => x.id === Number(tog)); await ship.poke({ edit: { id: it.id, item: { text: it.text, when: it.when, done: !it.done } } }); }
  };
  $('#title').onblur = () => ship.poke({ 'set-title': $('#title').textContent.trim() || 'drive' });
}
window.addEventListener('DOMContentLoaded', () => main().catch((e) => { $('#status').textContent = `cannot reach the ship: ${e.message}`; }));
