package ui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func autostartFile() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		h, _ := os.UserHomeDir()
		dir = filepath.Join(h, ".config")
	}
	return filepath.Join(dir, "autostart", "drive.desktop")
}

func autostartEnabled() bool {
	_, err := os.Stat(autostartFile())
	return err == nil
}

// desktopQuote quotes an Exec argument (Desktop Entry spec).
func desktopQuote(s string) string {
	if !strings.ContainsAny(s, " \t\"'\\$`") {
		return s
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	return `"` + r.Replace(s) + `"`
}

func autostartInstall(args []string) error {
	q := make([]string, len(args))
	for i, a := range args {
		q[i] = desktopQuote(a)
	}
	body := "[Desktop Entry]\nType=Application\nName=Drive\nComment=Sync folders with your Urbit planet\n" +
		"Exec=" + strings.Join(q, " ") + "\nIcon=drive\nTerminal=false\nX-GNOME-Autostart-enabled=true\n"
	f := autostartFile()
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return os.WriteFile(f, []byte(body), 0o644)
}

func autostartRemove() error {
	if err := os.Remove(autostartFile()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
