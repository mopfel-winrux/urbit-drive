package ui

import (
	"os"
	"strings"
	"testing"
)

func TestAutostartLinux(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if Autostart() {
		t.Fatal("on before install")
	}
	if err := SetAutostart("/tmp/a home", true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(autostartFile())
	if err != nil || !Autostart() {
		t.Fatalf("not installed: %v", err)
	}
	if !strings.Contains(string(b), ` tray -home "/tmp/a home"`) {
		t.Fatalf("Exec line: %s", b)
	}
	if err := SetAutostart("", false); err != nil || Autostart() {
		t.Fatalf("still on: %v", err)
	}
	if err := SetAutostart("", false); err != nil {
		t.Fatalf("removing twice: %v", err)
	}
}
