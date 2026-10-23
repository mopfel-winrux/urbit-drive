package ui

import (
	"os"
	"path/filepath"
)

// Start at login: each OS's own mechanism, under the same names the
// installers use (packaging/), so the toggle sees what an installer set.
//
//	linux    $XDG_CONFIG_HOME/autostart/drive.desktop
//	darwin   ~/Library/LaunchAgents/<AutostartLabel>.plist
//	windows  HKCU\Software\Microsoft\Windows\CurrentVersion\Run, value "Drive"
const AutostartLabel = "io.github.mopfel-winrux.drive"

// autostartArgs is how login should start us: this executable, the tray,
// and our home if it isn't the default.
func autostartArgs(home string) ([]string, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	args := []string{exe, "tray"}
	if home != "" && home != DefaultHome() {
		args = append(args, "-home", home)
	}
	return args, nil
}

// Autostart reports whether we start at login.
func Autostart() bool { return autostartEnabled() }

// SetAutostart turns starting at login on or off.
func SetAutostart(home string, on bool) error {
	if !on {
		return autostartRemove()
	}
	args, err := autostartArgs(home)
	if err != nil {
		return err
	}
	return autostartInstall(args)
}
