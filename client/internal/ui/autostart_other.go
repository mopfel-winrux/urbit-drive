//go:build !linux && !darwin && !windows

package ui

import "errors"

func autostartEnabled() bool { return false }

func autostartInstall([]string) error { return errors.New("start at login isn't supported here") }

func autostartRemove() error { return nil }
