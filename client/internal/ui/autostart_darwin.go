package ui

import (
	"errors"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func autostartFile() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Library", "LaunchAgents", AutostartLabel+".plist")
}

func autostartEnabled() bool {
	_, err := os.Stat(autostartFile())
	return err == nil
}

// autostartInstall writes a LaunchAgent; launchd loads it at the next
// login (we're already running now).
func autostartInstall(args []string) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>` + AutostartLabel + `</string>
  <key>ProgramArguments</key>
  <array>
`)
	for _, a := range args {
		b.WriteString("    <string>" + html.EscapeString(a) + "</string>\n")
	}
	b.WriteString(`  </array>
  <key>RunAtLoad</key><true/>
  <key>ProcessType</key><string>Interactive</string>
</dict>
</plist>
`)
	f := autostartFile()
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return os.WriteFile(f, []byte(b.String()), 0o644)
}

func autostartRemove() error {
	if err := os.Remove(autostartFile()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
