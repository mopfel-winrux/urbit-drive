package ui

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"fyne.io/systray"
)

const maxFolderLines = 8

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
	var lines [maxFolderLines]*systray.MenuItem
	for i := range lines {
		lines[i] = systray.AddMenuItem("", "")
		lines[i].Disable()
		lines[i].Hide()
	}
	systray.AddSeparator()
	openDir := systray.AddMenuItem("Open Drive folder", "Show the synced folder")
	openPlanet := systray.AddMenuItem("Open on planet", "%drive's page on your planet")
	settings := systray.AddMenuItem("Settings…", "Pairing, status and reset")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Stop syncing and quit")

	refresh := func() {
		st := a.Status()
		icon, title := iconOff, "Not syncing"
		switch {
		case st.Pairing:
			title = "Pairing…"
		case !st.Paired:
			title = "Not paired: open Settings to pair"
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
		systray.SetIcon(icon)
		systray.SetTooltip("Drive: " + title)
		status.SetTitle(title)
		for i, l := range lines {
			if i < len(st.Folders) {
				f := st.Folders[i]
				l.SetTitle(fmt.Sprintf("%s: %s", f.Name, FolderLine(f)))
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
			case <-quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func open(target string) {
	if err := Open(target); err != nil {
		log.Printf("open %s: %v", target, err)
	}
}
