package ui

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/mopfel-winrux/urbit-drive/client/internal/pair"
	dsync "github.com/mopfel-winrux/urbit-drive/client/internal/sync"
)

// App is the desktop client: at most one sync session, and the local
// page that pairs the device and shows its settings.
type App struct {
	Home string
	Opts SessionOptions

	ctx   context.Context
	token string
	addr  string // 127.0.0.1:port, once Serve has run

	mu      sync.Mutex
	sess    *Session
	pairing bool
	err     string // the last pairing or sync error, for the page
}

// State is what the tray and the page show.
type State struct {
	Paired  bool
	Running bool
	Pairing bool
	Config  *Config
	Folders []dsync.FolderStatus
	// Excluded is each folder's paths not synced here, by folder name.
	Excluded map[string][]string
	// Files is the latest file events, moving ones first.
	Files []dsync.FileEvent
	Err   string
	Login bool // we start at login
}

func NewApp(ctx context.Context, home string, o SessionOptions) *App {
	var t [16]byte
	rand.Read(t[:])
	return &App{Home: home, Opts: o, ctx: ctx, token: hex.EncodeToString(t[:])}
}

// StartSync starts syncing the paired device, if it isn't already.
func (a *App) StartSync() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sess != nil {
		return nil
	}
	s, err := Start(a.ctx, a.Home, a.Opts)
	if err != nil {
		a.err = err.Error()
		return err
	}
	a.sess, a.err = s, ""
	go func() {
		if err := s.Wait(); err != nil {
			log.Printf("sync stopped: %v", err)
			a.mu.Lock()
			a.err = "sync stopped: " + err.Error()
			a.mu.Unlock()
		}
		a.mu.Lock()
		if a.sess == s {
			a.sess = nil
		}
		a.mu.Unlock()
	}()
	return nil
}

// Status is a snapshot for display.
func (a *App) Status() State {
	a.mu.Lock()
	st := State{Pairing: a.pairing, Err: a.err, Login: Autostart()}
	if st.Err == ErrNotPaired.Error() {
		st.Err = "" // not an error: the sign-in form says it
	}
	sess := a.sess
	a.mu.Unlock()
	if c, err := LoadConfig(a.Home); err == nil {
		st.Paired, st.Config = true, c
	}
	if sess != nil {
		st.Running = true
		if fs, err := sess.Follower.Snapshot(); err == nil {
			st.Folders = fs
		}
		if x, err := sess.Follower.Excluded(); err == nil {
			st.Excluded = x
		}
		if fs, err := sess.Follower.Recent(maxFileLines); err == nil {
			st.Files = fs
		}
	}
	return st
}

// Close stops syncing.
func (a *App) Close() {
	a.mu.Lock()
	s := a.sess
	a.sess = nil
	a.mu.Unlock()
	if s != nil {
		s.Stop()
	}
}

// Reset stops syncing and forgets this device's pairing, keys and sync
// state (the synced files stay where they are).
func (a *App) Reset() error {
	a.Close()
	a.mu.Lock()
	a.err = ""
	a.mu.Unlock()
	return os.RemoveAll(a.Home)
}

// URL is the local page, with the token that lets the browser in.
func (a *App) URL() string { return "http://" + a.addr + "/?t=" + a.token }

// Serve starts the local page on a random loopback port.
func (a *App) Serve() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	a.addr = ln.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.page)
	mux.HandleFunc("POST /pair", a.doPair)
	mux.HandleFunc("GET /status.json", func(w http.ResponseWriter, r *http.Request) {
		type line struct {
			Name  string `json:"name"`
			State string `json:"state"`
			Glyph string `json:"glyph,omitempty"`
			Where string `json:"where,omitempty"`
			When  int64  `json:"when,omitempty"` // unix ms, for a synced file's time
		}
		st := a.Status()
		out := struct {
			Summary string `json:"summary"`
			Files   []line `json:"files"`
			Folders []line `json:"folders"`
		}{Summary: Summary(st), Files: []line{}, Folders: []line{}}
		for _, e := range st.Files {
			l := line{path.Base(e.Path), FileState(e), glyphName(e.State), e.Folder + "/" + e.Path, 0}
			if e.State == dsync.Synced {
				l.When = e.When.UnixMilli()
			}
			out.Files = append(out.Files, l)
		}
		for _, f := range st.Folders {
			out.Folders = append(out.Folders, line{Name: f.Name, State: FolderLine(f)})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("POST /open", func(w http.ResponseWriter, r *http.Request) {
		if c, err := LoadConfig(a.Home); err == nil {
			Open(c.Dir)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /autostart", func(w http.ResponseWriter, r *http.Request) {
		if err := SetAutostart(a.Home, r.FormValue("on") == "1"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	selective := func(include bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			a.mu.Lock()
			s := a.sess
			a.mu.Unlock()
			if s == nil {
				http.Error(w, "not syncing", http.StatusConflict)
				return
			}
			folder, path := r.FormValue("folder"), r.FormValue("path")
			var err error
			if include {
				err = s.Follower.Include(folder, path)
			} else {
				err = s.Follower.Exclude(folder, path)
			}
			if err != nil {
				a.mu.Lock()
				a.err = err.Error()
				a.mu.Unlock()
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}
	mux.HandleFunc("POST /exclude", selective(false))
	mux.HandleFunc("POST /include", selective(true))
	mux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToUpper(strings.TrimSpace(r.FormValue("confirm"))) != "SIGN OUT" {
			a.mu.Lock()
			a.err = "To sign out, type SIGN OUT in the box first."
			a.mu.Unlock()
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err := a.Reset(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	srv := &http.Server{Handler: a.guard(mux)}
	go func() {
		<-a.ctx.Done()
		srv.Close()
	}()
	go srv.Serve(ln)
	log.Printf("settings at %s", a.URL())
	return nil
}

// guard lets in only our own browser: the right Host (no DNS
// rebinding), and our token, from the tray's link or its cookie
// (SameSite, so other sites can't post to us).
func (a *App) guard(h http.Handler) http.Handler {
	ok := func(s string) bool { return subtle.ConstantTimeCompare([]byte(s), []byte(a.token)) == 1 }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != a.addr {
			http.Error(w, "wrong host", http.StatusForbidden)
			return
		}
		if t := r.URL.Query().Get("t"); r.Method == "GET" && ok(t) {
			http.SetCookie(w, &http.Cookie{Name: "drive-ui", Value: a.token, Path: "/",
				HttpOnly: true, SameSite: http.SameSiteStrictMode})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if c, err := r.Cookie("drive-ui"); err != nil || !ok(c.Value) {
			http.Error(w, "open this page from the drive tray menu", http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (a *App) doPair(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	if a.pairing {
		a.mu.Unlock()
		http.Error(w, "already pairing", http.StatusConflict)
		return
	}
	a.pairing, a.err = true, ""
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.pairing = false
		a.mu.Unlock()
	}()
	dir := Untilde(strings.TrimSpace(r.FormValue("dir")))
	if dir == "" {
		dir = DefaultDir()
	}
	o := pair.Options{
		URL:       strings.TrimSpace(r.FormValue("url")),
		Code:      strings.TrimSpace(r.FormValue("code")),
		Name:      strings.TrimSpace(r.FormValue("name")),
		Publisher: strings.TrimSpace(r.FormValue("publisher")),
		Fake:      r.FormValue("fake") != "",
		Log:       func(f string, a ...any) { log.Printf(f, a...) },
	}
	var err error
	if o.URL == "" || o.Code == "" || o.Name == "" {
		err = errors.New("the planet's address, its +code and a device name are all needed")
	} else if _, err = Pair(a.Home, dir, o); err == nil {
		err = a.StartSync()
	}
	if err != nil {
		a.mu.Lock()
		a.err = err.Error()
		a.mu.Unlock()
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) page(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	st := a.Status()
	pageT.Execute(w, map[string]any{
		"S": st, "Host": host, "Dir": Tilde(DefaultDir()), "Summary": Summary(st),
	})
}

// LocalPath is where a file event's file is on this computer ("" when
// not syncing).
func (a *App) LocalPath(e dsync.FileEvent) string {
	a.mu.Lock()
	s := a.sess
	a.mu.Unlock()
	if s == nil {
		return ""
	}
	return s.Follower.LocalPath(e)
}

// glyphName is a file state's glyph, on the page (the tray draws the
// same ones in internal/icon).
func glyphName(st dsync.FileState) string {
	return map[dsync.FileState]string{dsync.Uploading: "up", dsync.Downloading: "down", dsync.Synced: "check",
		dsync.Conflict: "warn", dsync.Deleted: "x", dsync.Failed: "fail"}[st]
}

// Summary is the one line that answers "is everything synced?".
func Summary(st State) string {
	up, down := 0, 0
	for _, e := range st.Files {
		switch e.State {
		case dsync.Uploading:
			up++
		case dsync.Downloading:
			down++
		}
	}
	switch {
	case !st.Running:
		return "Not syncing"
	case up+down > 0:
		return moving(up, down)
	case len(st.Folders) == 0:
		return "Connecting to " + st.Config.Planet + "…"
	}
	for _, f := range st.Folders {
		if !f.UpToDate() {
			return "Syncing…"
		}
	}
	return "Up to date"
}

// FileLine is how a file reads in the tray: its name and its state.
func FileLine(e dsync.FileEvent) string { return path.Base(e.Path) + " · " + FileState(e) }

// FileState is a file's state in words: "Uploading 40%", "Synced 13:37".
func FileState(e dsync.FileEvent) string {
	pct := func() string {
		if e.Total <= 0 {
			return ""
		}
		return fmt.Sprintf(" %d%%", e.Done*100/e.Total)
	}
	switch e.State {
	case dsync.Uploading:
		return "Uploading" + pct()
	case dsync.Downloading:
		return "Downloading" + pct()
	case dsync.Synced:
		return "Synced " + e.When.Format("3:04 PM")
	case dsync.Conflict:
		return "Conflicted copy"
	case dsync.Deleted:
		return "Deleted"
	default:
		return "Couldn't sync"
	}
}

// FolderLine is how a folder's state reads, in the tray and the page.
func FolderLine(f dsync.FolderStatus) string {
	switch {
	case f.UpToDate():
		return "up to date"
	case f.Committing:
		return "uploading…"
	default:
		return fmt.Sprintf("syncing %d/%d…", f.Synced, f.Head)
	}
}

var pageT = template.Must(template.New("page").Funcs(template.FuncMap{
	"line": FolderLine, "file": FileState, "tilde": Tilde, "base": path.Base,
	"glyph": glyphName,
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Drive</title>
<style>
:root{--bg:#fff;--soft:#f7f5f2;--press:#eeeae4;--line:#e7e3dd;--fg:#1e1919;--fg2:#6b645d;--blue:#0061fe;--blue-h:#0057e4;--blue-ink:#fff;--green:#1f7a4d;--amber:#a85d00;--red:#c4312b;--red-soft:#fdecea;--select:#e9f1ff;color-scheme:light;font:14px/1.45 system-ui,-apple-system,"Segoe UI",Roboto,Ubuntu,sans-serif}
@media (prefers-color-scheme:dark){:root{--bg:#1a1918;--soft:#222120;--press:#33302d;--line:#34312e;--fg:#f4f1ec;--fg2:#aaa39b;--blue:#4d8dff;--blue-h:#6a9fff;--blue-ink:#0f1420;--green:#5fc792;--amber:#f0a64a;--red:#ff7a72;--red-soft:#3d1d1b;--select:#1b2b45;color-scheme:dark}}
*{box-sizing:border-box}[hidden]{display:none!important}
html,body{background:var(--bg)}body{margin:0;color:var(--fg);-webkit-font-smoothing:antialiased}
::selection{background:var(--select)}:focus-visible{outline:2px solid var(--blue);outline-offset:2px;border-radius:4px}
a{color:var(--blue);text-underline-offset:2px}
main{max-width:640px;margin:0 auto;padding:32px 24px 64px}
.ic{width:20px;height:20px;flex:none;fill:none;stroke:currentColor;stroke-width:1.75;stroke-linecap:round;stroke-linejoin:round}
header{display:flex;align-items:center;gap:12px;margin-bottom:24px}
header .logo{width:40px;height:40px;border-radius:10px;background:linear-gradient(135deg,#3d8bfd,#1f4fc4);display:grid;place-items:center;color:#fff}
header h1{font-size:20px;font-weight:650;margin:0;letter-spacing:-.01em}
header p{margin:0;color:var(--fg2)}
header .acts{margin-left:auto;display:flex;gap:8px;flex-wrap:wrap}
h2{font-size:16px;font-weight:650;margin:32px 0 8px}
p.lead{color:var(--fg2);margin:0 0 20px;max-width:60ch}
.btn{display:inline-flex;align-items:center;gap:8px;height:36px;padding:0 14px;border-radius:8px;border:1px solid var(--line);background:var(--bg);color:var(--fg);font:inherit;font-weight:550;cursor:pointer;text-decoration:none;white-space:nowrap}
.btn:hover{background:var(--soft)}.btn:active{background:var(--press)}
.btn.primary{background:var(--blue);border-color:var(--blue);color:var(--blue-ink)}.btn.primary:hover{background:var(--blue-h);border-color:var(--blue-h)}
.btn.danger{color:var(--red)}.btn.danger.solid{background:var(--red);border-color:var(--red);color:#fff}
.btn .ic{width:18px;height:18px}
label{display:flex;flex-direction:column;gap:6px;font-weight:550;margin:0 0 16px}
label small{font-weight:400;color:var(--fg2)}
.field{height:40px;padding:0 12px;border:1px solid var(--line);border-radius:8px;background:var(--bg);color:var(--fg);font:inherit;font-weight:400;width:100%}
.field:focus{outline:0;border-color:var(--blue)}
select.field{width:auto}
.check{flex-direction:row;align-items:center;gap:8px;font-weight:400}
details{margin:4px 0 20px}details summary{cursor:pointer;color:var(--fg2)}details[open] summary{margin-bottom:16px}
.err{background:var(--red-soft);color:var(--red);padding:12px 16px;border-radius:8px;margin:0 0 20px}
.rows{list-style:none;margin:0;padding:0;border-top:1px solid var(--line)}
.rows li{display:flex;align-items:center;gap:12px;min-height:48px;padding:6px 0;border-bottom:1px solid var(--line)}
.rows li .name{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.rows li .state{color:var(--fg2);font-variant-numeric:tabular-nums;white-space:nowrap}
.g-up,.g-down{color:var(--blue)}.g-check{color:var(--green)}.g-warn{color:var(--amber)}.g-x{color:var(--fg2)}.g-fail{color:var(--red)}
.empty{color:var(--fg2);padding:16px 0;margin:0}
.setting{display:flex;align-items:center;gap:16px;padding:16px 0;border-bottom:1px solid var(--line)}
.setting div{flex:1}.setting b{display:block}.setting p{margin:2px 0 0;color:var(--fg2)}
.inline{display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-top:12px}.inline .field{flex:1;min-width:180px}
code{font-size:13px;background:var(--soft);padding:1px 6px;border-radius:4px}
.id{color:var(--fg2);font-size:13px;overflow-wrap:anywhere}
@media (max-width:520px){header{flex-wrap:wrap}header .acts{margin-left:0;width:100%}}
</style></head><body>
<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>
<symbol id="up" viewBox="0 0 24 24"><path d="M12 19V5M6.5 10.5 12 5l5.5 5.5"/></symbol>
<symbol id="down" viewBox="0 0 24 24"><path d="M12 5v14M6.5 13.5 12 19l5.5-5.5"/></symbol>
<symbol id="check" viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></symbol>
<symbol id="warn" viewBox="0 0 24 24"><path d="M12 4 21 19.5H3zM12 9.5V14M12 16.8v.2"/></symbol>
<symbol id="x" viewBox="0 0 24 24"><path d="m6.5 6.5 11 11M17.5 6.5l-11 11"/></symbol>
<symbol id="fail" viewBox="0 0 24 24"><path d="m6.5 6.5 11 11M17.5 6.5l-11 11"/></symbol>
<symbol id="folder" viewBox="0 0 24 24"><path d="M3 6.5A1.5 1.5 0 0 1 4.5 5h4.38l2 2H19.5A1.5 1.5 0 0 1 21 8.5v9a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 17.5z"/></symbol>
<symbol id="web" viewBox="0 0 24 24"><circle cx="12" cy="12" r="8.5"/><path d="M3.5 12h17M12 3.5c2.5 2.6 2.5 14.4 0 17M12 3.5c-2.5 2.6-2.5 14.4 0 17"/></symbol>
<symbol id="sync" viewBox="0 0 24 24"><path d="M5 11a7 7 0 0 1 12-4.5M19 13a7 7 0 0 1-12 4.5M17 3v3.5h-3.5M7 21v-3.5h3.5"/></symbol>
</defs></svg>
<main>
{{if .S.Pairing}}
<header><span class="logo"><svg class="ic"><use href="#sync"/></svg></span><div><h1>Signing in…</h1><p>Setting up this computer on your ship. This page updates when it's done.</p></div></header>
<meta http-equiv="refresh" content="2">
{{else if not .S.Paired}}
<header><span class="logo"><svg class="ic"><use href="#sync"/></svg></span><div><h1>Sign in to Drive</h1><p>Keep a folder on this computer in sync with your ship.</p></div></header>
{{with .S.Err}}<p class="err">{{.}}</p>{{end}}
<form method="post" action="/pair">
<label>Your ship's web address<input class="field" type="url" name="url" placeholder="https://sampel-palnet.tlon.network" required autofocus>
<small>The address you open Landscape at.</small></label>
<label>Access code<input class="field" type="password" name="code" placeholder="xxxxxx-xxxxxx-xxxxxx-xxxxxx" required autocomplete="off">
<small>Your ship's login code. Landscape shows it in its settings, or type <code>+code</code> in your ship's terminal.</small></label>
<label>Name this computer<input class="field" type="text" name="name" value="{{.Host}}" required></label>
<label>Folder to sync<input class="field" type="text" name="dir" value="{{.Dir}}"></label>
<details><summary>Advanced</summary>
<label>Install Drive from<input class="field" type="text" name="publisher" placeholder="~publisher"><small>Only needed if Drive isn't on your ship yet.</small></label>
<label class="check"><input type="checkbox" name="fake"> Development ship (urbit -F)</label>
</details>
<button class="btn primary">Sign in</button>
<p class="lead" style="margin-top:16px">This computer gets its own identity on your ship, with keys made here. Your files go only between your computers and your ship.</p>
</form>
{{else}}
<header><span class="logo"><svg class="ic"><use href="#sync"/></svg></span>
<div><h1>Drive</h1><p id="summary">{{if not .S.Running}}Not syncing{{else}}{{.Summary}}{{end}}</p></div>
<div class="acts"><form method="post" action="/open"><button class="btn"><svg class="ic"><use href="#folder"/></svg>Open folder</button></form>
{{with .S.Config}}{{if .URL}}<a class="btn" href="{{.URL}}/drive/" target="_blank" rel="noopener"><svg class="ic"><use href="#web"/></svg>Open on the web</a>{{end}}{{end}}</div></header>
{{with .S.Err}}<p class="err">{{.}}</p>{{end}}
<h2>Recent files</h2>
<ul class="rows" id="files">{{range .S.Files}}<li><svg class="ic g-{{glyph .State}}"><use href="#{{glyph .State}}"/></svg><span class="name" title="{{.Folder}}/{{.Path}}">{{base .Path}}</span><span class="state">{{file .}}</span></li>{{else}}<li class="empty">Files you change here or on your other computers show up as they sync.</li>{{end}}</ul>

<h2>Folders</h2>
<ul class="rows" id="folders">{{range .S.Folders}}<li><svg class="ic"><use href="#folder"/></svg><span class="name">{{.Name}}</span><span class="state">{{line .}}</span></li>{{end}}</ul>
{{with .S.Config}}<p class="id">Synced to <code>{{tilde .Dir}}</code></p>{{end}}

{{if .S.Running}}
<h2>Selective sync</h2>
<p class="lead">Exclude a folder or file and it stays on your ship and your other computers, but not on this one.</p>
<ul class="rows">{{range $name, $xs := .S.Excluded}}{{range $xs}}<li><span class="name">{{$name}} / {{.}}</span>
<form method="post" action="/include"><input type="hidden" name="folder" value="{{$name}}"><input type="hidden" name="path" value="{{.}}"><button class="btn">Sync again</button></form></li>{{end}}{{end}}</ul>
<form class="inline" method="post" action="/exclude">
<select class="field" name="folder" aria-label="Folder">{{range .S.Folders}}<option>{{.Name}}</option>{{end}}</select>
<input class="field" type="text" name="path" placeholder="Path inside it, e.g. Videos/raw" required aria-label="Path to exclude">
<button class="btn">Exclude</button></form>
{{end}}

<h2>This computer</h2>
<div class="setting"><div><b>Start Drive when you log in</b><p>{{if .S.Login}}On: Drive opens in the background when you log in.{{else}}Off: start Drive yourself to sync.{{end}}</p></div>
<form method="post" action="/autostart"><input type="hidden" name="on" value="{{if .S.Login}}0{{else}}1{{end}}"><button class="btn">{{if .S.Login}}Turn off{{else}}Turn on{{end}}</button></form></div>
{{with .S.Config}}<div class="setting"><div><b>{{.Name}}</b><p class="id">Signed in to {{.Planet}} as {{.Moon}}</p></div></div>{{end}}
<div class="setting"><div><b>Sign out</b><p>Stops syncing and forgets this computer's keys. Your files stay where they are. Remove the computer on your ship too, under More › Computers.</p>
<form class="inline" method="post" action="/reset">
<input class="field" type="text" name="confirm" placeholder="Type SIGN OUT to confirm" aria-label="Type SIGN OUT to confirm" autocomplete="off"><button class="btn danger">Sign out</button></form></div></div>
{{end}}
</main>
<script>
// keep files and folders fresh without reloading (which would eat input)
const files = document.getElementById('files'), folders = document.getElementById('folders'), sum = document.getElementById('summary');
const li = (glyph, name, title, state) => {
  const l = document.createElement('li');
  if (glyph) { l.insertAdjacentHTML('beforeend', '<svg class="ic g-' + glyph + '"><use href="#' + glyph + '"/></svg>'); }
  const n = document.createElement('span'); n.className = 'name'; n.textContent = name; if (title) n.title = title;
  const s = document.createElement('span'); s.className = 'state'; s.textContent = state;
  l.append(n, s); return l;
};
const tick = async () => {
  try {
    const st = await (await fetch('/status.json')).json();
    if (sum) sum.textContent = st.summary;
    // times in the viewer's own format, as on the web page
    const state = (f) => f.when ? 'Synced ' + new Date(f.when).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' }) : f.state;
    if (st.files.length) files.replaceChildren(...st.files.map(f => li(f.glyph, f.name, f.where, state(f))));
    folders.replaceChildren(...st.folders.map(f => li('', f.name, '', f.state)));
    folders.querySelectorAll('li').forEach(l => l.insertAdjacentHTML('afterbegin', '<svg class="ic"><use href="#folder"/></svg>'));
  } catch (e) {}
};
if (files) { tick(); setInterval(tick, 2000); }
</script>
</body></html>`))
