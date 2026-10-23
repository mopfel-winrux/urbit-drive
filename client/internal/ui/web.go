package ui

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
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
	Err     string
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
	st := State{Pairing: a.pairing, Err: a.err}
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
		}
		out := []line{}
		for _, f := range a.Status().Folders {
			out = append(out, line{f.Name, FolderLine(f)})
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
	mux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("confirm") != "RESET" {
			http.Error(w, "type RESET to confirm", http.StatusBadRequest)
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
	dir := strings.TrimSpace(r.FormValue("dir"))
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
	pageT.Execute(w, map[string]any{
		"S": a.Status(), "Host": host, "Dir": DefaultDir(),
	})
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

var pageT = template.Must(template.New("page").Funcs(template.FuncMap{"line": FolderLine}).Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Drive</title>
<style>
:root{--bg:#fff;--fg:#1d1d1f;--mute:#6b6b70;--line:#e3e3e8;--acc:#2b6cd8;--bad:#c0392b}
@media (prefers-color-scheme:dark){:root{--bg:#18181b;--fg:#ececf0;--mute:#9a9aa3;--line:#2e2e34;--acc:#5b8ef0;--bad:#e5675a}}
body{background:var(--bg);color:var(--fg);font:15px/1.45 system-ui,sans-serif;max-width:560px;margin:32px auto;padding:0 16px}
h1{font-size:22px;margin:0 0 4px}h2{font-size:15px;margin:28px 0 8px;color:var(--mute);font-weight:600}
.mute{color:var(--mute)}.err{color:var(--bad);border:1px solid var(--bad);padding:8px 12px;border-radius:8px}
label{display:block;margin:12px 0 4px;font-weight:600}input[type=text],input[type=password],input[type=url]{width:100%;box-sizing:border-box;padding:8px;border:1px solid var(--line);border-radius:6px;background:var(--bg);color:var(--fg)}
button{margin-top:16px;padding:8px 16px;border:0;border-radius:6px;background:var(--acc);color:#fff;font-weight:600;cursor:pointer}
button.quiet{background:none;color:var(--acc);border:1px solid var(--line)}button.bad{background:var(--bad)}
table{width:100%;border-collapse:collapse}td{padding:6px 0;border-bottom:1px solid var(--line)}td:last-child{text-align:right}
code{font-size:13px}form.inline{display:inline}
</style></head><body>
<h1>Drive</h1>
{{with .S.Err}}<p class="err">{{.}}</p>{{end}}
{{if .S.Pairing}}<p>Pairing with your planet… this page refreshes when it's done.</p><meta http-equiv="refresh" content="2">
{{else if not .S.Paired}}
<p class="mute">Pair this computer with your planet. It becomes a moon of the planet, with keys made here, and syncs a folder with it.</p>
<form method="post" action="/pair">
<label>Planet address</label><input type="url" name="url" placeholder="https://sampel-palnet.tlon.network" required>
<label>+code</label><input type="password" name="code" placeholder="from your planet's dojo, or Landscape's settings" required>
<label>This device's name</label><input type="text" name="name" value="{{.Host}}" required>
<label>Folder to sync</label><input type="text" name="dir" value="{{.Dir}}">
<label>Install %drive from (if the planet doesn't have it)</label><input type="text" name="publisher" placeholder="~publisher">
<label><input type="checkbox" name="fake"> Fake ship (development)</label>
<button>Pair</button>
</form>
{{else}}{{with .S.Config}}
<p class="mute">{{.Name}} · <code>{{.Moon}}</code> on <code>{{.Planet}}</code></p>
<h2>Folders</h2>
{{end}}
{{if not .S.Running}}<p class="mute">Not syncing.</p>{{end}}
<table id="folders">{{range .S.Folders}}<tr><td>{{.Name}}</td><td class="mute">{{line .}}</td></tr>{{end}}</table>
{{with .S.Config}}
<p class="mute">Syncing <code>{{.Dir}}</code></p>
<form class="inline" method="post" action="/open"><button class="quiet">Open folder</button></form>
{{if .URL}}<a href="{{.URL}}/drive/" target="_blank"><button class="quiet" type="button">Open on planet</button></a>{{end}}
{{end}}
<h2>Reset</h2>
<p class="mute">Forget this pairing: stop syncing and delete the device's keys and sync state here. Your files stay. Unpair the moon on your planet too, from its Drive page.</p>
<form method="post" action="/reset" onsubmit="return confirm('Forget this pairing?')">
<input type="text" name="confirm" placeholder="type RESET"><button class="bad">Reset this device</button></form>
{{end}}
<script>
// keep the folder lines fresh without reloading (which would eat input)
const t = document.getElementById('folders');
if (t) setInterval(async () => {
  try {
    const fs = await (await fetch('/status.json')).json();
    t.replaceChildren(...fs.map(f => {
      const tr = document.createElement('tr');
      const a = document.createElement('td'), b = document.createElement('td');
      a.textContent = f.name; b.className = 'mute'; b.textContent = f.state;
      tr.append(a, b); return tr;
    }));
  } catch (e) {}
}, 2000);
</script>
</body></html>`))
