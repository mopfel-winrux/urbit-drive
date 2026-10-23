package ui

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	dsync "github.com/mopfel-winrux/urbit-drive/client/internal/sync"
)

// Share links from this computer. The planet makes the link (it serves
// it); we ask over mesa and build the URL from its web address.

// Link is a share link to the file or folder at [local], on the planet.
func (a *App) Link(local string) (string, error) {
	a.mu.Lock()
	s := a.sess
	a.mu.Unlock()
	if s == nil {
		return "", errors.New("Drive isn't syncing; sign in first")
	}
	if s.Config.URL == "" {
		return "", errors.New("your ship's web address isn't known; sign out and sign in again")
	}
	token, err := s.Follower.AskLink(Untilde(local))
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s.Config.URL, "/") + "/drive-api/s/" + token, nil
}

// CopyLink makes a link and puts it on the clipboard. Without a working
// clipboard (a Linux desktop with no xclip, xsel or wl-copy) it opens the
// link in the browser instead, where it can be copied from the address bar.
func (a *App) CopyLink(local string) (string, error) {
	link, err := a.Link(local)
	if err != nil {
		return "", err
	}
	if clipboard.WriteAll(link) != nil {
		Open(link)
	}
	return link, nil
}

// localOf is the local path of a recent file (for the page's buttons).
func (a *App) localOf(folder, p string) string {
	return a.LocalPath(dsync.FileEvent{Folder: folder, Path: p})
}

// POST /link: folder+path (a recent file) or local (any path) -> {url}
func (a *App) doLink(w http.ResponseWriter, r *http.Request) {
	local := r.FormValue("local")
	if f, p := r.FormValue("folder"), r.FormValue("path"); f != "" {
		local = a.localOf(f, p)
	}
	w.Header().Set("Content-Type", "application/json")
	if local == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "say which file or folder"})
		return
	}
	link, err := a.Link(local)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"url": link})
}

// The control file: where a running Drive's page is, and its token, so
// `drive link` can ask it. Owner-only; removed when Drive stops.

type control struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
}

func controlPath(home string) string { return filepath.Join(home, "ui.json") }

func (a *App) writeControl() {
	b, _ := json.Marshal(control{a.addr, a.token})
	os.MkdirAll(a.Home, 0o700)
	if err := os.WriteFile(controlPath(a.Home), b, 0o600); err == nil {
		go func() { <-a.ctx.Done(); os.Remove(controlPath(a.Home)) }()
	}
}

// AskRunning asks the Drive running with [home] for a link to [local].
func AskRunning(home, local string) (string, error) {
	b, err := os.ReadFile(controlPath(home))
	if err != nil {
		return "", errors.New("Drive isn't running (start it with `drive` or `drive run`)")
	}
	var c control
	if err := json.Unmarshal(b, &c); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(Untilde(local))
	if err != nil {
		return "", err
	}
	req, _ := http.NewRequest("POST", "http://"+c.Addr+"/link", strings.NewReader(url.Values{"local": {abs}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Drive-Token", c.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.New("Drive isn't answering; is it still running?")
	}
	defer resp.Body.Close()
	var out struct{ URL, Error string }
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Error != "" {
		return "", errors.New(out.Error)
	}
	return out.URL, nil
}
