package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/systray"

	"github.com/mopfel-winrux/urbit-drive/client/internal/icon"
	dsync "github.com/mopfel-winrux/urbit-drive/client/internal/sync"
)

const maxFileLines = 8

// RunTray shows the tray icon and menu until Quit, then stops the app.
// It must run on the main goroutine (macOS needs the main thread).
// Without a tray host (a headless session, a desktop with no
// StatusNotifier) systray only logs; syncing and the page go on, and
// the process runs until it is signalled.
func RunTray(a *App) {
	defer a.Close()
	if runtime.GOOS == "linux" && os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		// systray would panic on Quit with no bus to talk to
		log.Printf("no D-Bus session: running without a tray icon")
		<-a.ctx.Done()
		return
	}
	systray.Run(func() { trayReady(a) }, func() {})
}

func trayReady(a *App) {
	systray.SetIcon(iconOff)
	systray.SetTitle("Drive")
	systray.SetTooltip("Drive")

	status := systray.AddMenuItem("Starting…", "")
	status.Disable()
	systray.AddSeparator()
	// the latest files: moving ones first, then what changed most recently
	var lines [maxFileLines]*systray.MenuItem
	var shown [maxFileLines]dsync.FileEvent
	var copies, shows [maxFileLines]*systray.MenuItem
	for i := range lines {
		lines[i] = systray.AddMenuItem("", "")
		copies[i] = lines[i].AddSubMenuItem("Copy link", "A link anyone can open, served by your ship")
		shows[i] = lines[i].AddSubMenuItem("Show in folder", "")
		lines[i].Hide()
	}
	icons := map[dsync.FileState][]byte{
		dsync.Uploading: glyphIcon(icon.GlyphUp), dsync.Downloading: glyphIcon(icon.GlyphDown),
		dsync.Synced: glyphIcon(icon.GlyphCheck), dsync.Conflict: glyphIcon(icon.GlyphConflict),
		dsync.Failed: glyphIcon(icon.GlyphFailed), dsync.Deleted: glyphIcon(icon.GlyphDeleted),
	}
	systray.AddSeparator()
	openDir := systray.AddMenuItem("Open Drive folder", "Show the synced folder")
	openPlanet := systray.AddMenuItem("Open on the web", "Drive on your ship, in the browser")
	settings := systray.AddMenuItem("Settings…", "Sign in, selective sync, start at login")
	login := systray.AddMenuItemCheckbox("Start at login", "Start Drive when you log in", Autostart())
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Stop syncing and quit")

	refresh := func() {
		st := a.Status()
		icon, title := iconOff, "Not syncing"
		switch {
		case st.Pairing:
			title = "Signing in…"
		case !st.Paired:
			title = "Signed out: open Settings to sign in"
		case !st.Running:
			if st.Err != "" {
				title = "Stopped: " + st.Err
			}
		case len(st.Folders) == 0:
			icon, title = iconBusy, "Connecting to "+st.Config.Planet+"…"
		default:
			icon, title = iconIdle, "Up to date"
			for _, f := range st.Folders {
				if !f.UpToDate() {
					icon, title = iconBusy, "Syncing…"
				}
			}
		}
		up, down := 0, 0
		for _, e := range st.Files {
			switch e.State {
			case dsync.Uploading:
				up++
			case dsync.Downloading:
				down++
			}
		}
		if st.Running && up+down > 0 {
			icon, title = iconBusy, moving(up, down)
		}
		systray.SetIcon(icon)
		systray.SetTooltip("Drive: " + title)
		status.SetTitle(title)
		for i, l := range lines {
			if i < len(st.Files) {
				e := st.Files[i]
				shown[i] = e
				l.SetTitle(FileLine(e))
				l.SetTooltip(e.Folder + "/" + e.Path)
				if b, ok := icons[e.State]; ok {
					l.SetIcon(b)
				}
				l.Show()
			} else {
				l.Hide()
			}
		}
		if st.Paired {
			openDir.Enable()
		} else {
			openDir.Disable()
		}
		if st.Paired && st.Config.URL != "" {
			openPlanet.Enable()
		} else {
			openPlanet.Disable()
		}
	}
	refresh()

	showClicks, copyClicks := clicks(shows[:]), clicks(copies[:])
	tick := time.NewTicker(2 * time.Second)
	go func() {
		defer tick.Stop()
		for {
			select {
			case <-a.ctx.Done():
				systray.Quit()
				return
			case <-tick.C:
				refresh()
			case i := <-showClicks:
				if p := a.LocalPath(shown[i]); p != "" {
					open(filepath.Dir(p))
				}
			case i := <-copyClicks:
				e := shown[i]
				go func() {
					status.SetTitle("Getting a link to " + filepath.Base(e.Path) + "…")
					if _, err := a.CopyLink(a.LocalPath(e)); err != nil {
						log.Printf("link: %v", err)
						status.SetTitle("Couldn't make a link: " + err.Error())
						return
					}
					status.SetTitle("Link copied: " + filepath.Base(e.Path))
				}()
			case <-openDir.ClickedCh:
				if c, err := LoadConfig(a.Home); err == nil {
					open(c.Dir)
				}
			case <-openPlanet.ClickedCh:
				if c, err := LoadConfig(a.Home); err == nil && c.URL != "" {
					open(c.URL + "/drive/")
				}
			case <-settings.ClickedCh:
				open(a.URL())
			case <-login.ClickedCh:
				if err := SetAutostart(a.Home, !login.Checked()); err != nil {
					log.Printf("start at login: %v", err)
				}
				if Autostart() {
					login.Check()
				} else {
					login.Uncheck()
				}
			case <-quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// clicks merges the file rows' click channels into one that says which.
func clicks(items []*systray.MenuItem) <-chan int {
	out := make(chan int)
	for i, it := range items {
		go func(i int, c <-chan struct{}) {
			for range c {
				out <- i
			}
		}(i, it.ClickedCh)
	}
	return out
}

func moving(up, down int) string {
	files := func(n int) string {
		if n == 1 {
			return "1 file"
		}
		return fmt.Sprintf("%d files", n)
	}
	switch {
	case up > 0 && down > 0:
		return fmt.Sprintf("Uploading %s, downloading %s…", files(up), files(down))
	case up > 0:
		return "Uploading " + files(up) + "…"
	default:
		return "Downloading " + files(down) + "…"
	}
}

func open(target string) {
	if err := Open(target); err != nil {
		log.Printf("open %s: %v", target, err)
	}
}
