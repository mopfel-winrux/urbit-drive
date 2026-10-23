package ui

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func autostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue("Drive")
	return err == nil
}

func autostartInstall(args []string) error {
	q := make([]string, len(args))
	for i, a := range args {
		q[i] = `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue("Drive", strings.Join(q, " "))
}

func autostartRemove() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	if err := k.DeleteValue("Drive"); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}
