package ui

import (
	"os/exec"
	"runtime"

	"github.com/mopfel-winrux/urbit-drive/client/internal/icon"
)

// Open shows a URL or a folder with the desktop's default handler.
func Open(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

// The tray's icons (internal/icon draws desk/web/tile.svg): blue idle,
// orange syncing, grey off; ICO on Windows, PNG elsewhere.
var (
	iconIdle = trayIcon(icon.Idle)
	iconBusy = trayIcon(icon.Busy)
	iconOff  = trayIcon(icon.Off)
)

// glyphIcon is a file row's state glyph, for a menu item.
func glyphIcon(kind string) []byte {
	if runtime.GOOS == "windows" {
		return icon.WrapICO([]int{16}, [][]byte{icon.GlyphPNG(kind, 16)})
	}
	return icon.GlyphPNG(kind, 32)
}

func trayIcon(c icon.Style) []byte {
	if runtime.GOOS == "windows" {
		return icon.ICO([]int{16, 24, 32}, c)
	}
	return icon.PNG(c, 32)
}
