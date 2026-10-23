// Package ui is the desktop side of the client: the configuration and
// sync session shared by the CLI and the tray, the tray menu, and a
// local web page for pairing and settings.
package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	mesa "github.com/mopfel-winrux/libmesa/bindings/go"
	_ "modernc.org/sqlite"

	"github.com/mopfel-winrux/urbit-drive/client/internal/pair"
	dsync "github.com/mopfel-winrux/urbit-drive/client/internal/sync"
)

// Config is a paired device: home/config.json.
type Config struct {
	Planet string `json:"planet"`
	Moon   string `json:"moon"`
	Name   string `json:"name"`
	Dir    string `json:"dir"`
	URL    string `json:"url,omitempty"` // the planet's web address
	Fake   bool   `json:"fake,omitempty"`
}

// DefaultHome is where state lives unless -home says otherwise.
func DefaultHome() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = "."
	}
	return filepath.Join(d, "drive")
}

// DefaultDir is the folder we sync unless told otherwise.
func DefaultDir() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Drive")
}

// ErrNotPaired: home has no config.json.
var ErrNotPaired = errors.New("not paired yet")

func LoadConfig(home string) (*Config, error) {
	b, err := os.ReadFile(filepath.Join(home, "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotPaired
	}
	if err != nil {
		return nil, err
	}
	var c Config
	return &c, json.Unmarshal(b, &c)
}

// Pair pairs this device and saves its config and keys in [home].
func Pair(home, dir string, o pair.Options) (*Config, error) {
	if _, err := os.Stat(filepath.Join(home, "config.json")); err == nil {
		return nil, fmt.Errorf("%s is already paired; use another -home or reset it", home)
	}
	res, err := pair.Pair(o)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, err
	}
	if res.Keyfile != "" {
		if err := os.WriteFile(filepath.Join(home, "moon.key"), []byte(res.Keyfile+"\n"), 0o600); err != nil {
			return nil, err
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	c := &Config{Planet: res.Planet.String(), Moon: res.Moon.String(), Name: o.Name, Dir: abs, URL: o.URL, Fake: o.Fake}
	b, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(filepath.Join(home, "config.json"), b, 0o600); err != nil {
		return nil, err
	}
	log.Printf("paired: %s is %s, syncing %s", o.Name, res.Moon, abs)
	return c, nil
}

// Session is a running sync: the node, its follower and the watcher.
type Session struct {
	Config   *Config
	Follower *dsync.Follower
	cancel   context.CancelFunc
	done     chan struct{} // closed when the node has closed
	err      error
}

// SessionOptions are `drive run`'s flags.
type SessionOptions struct {
	Port  int  // UDP port, 0 for any
	Trace bool // log every datagram
}

// Start opens the node for the device paired in [home] and syncs until
// ctx is done or Stop is called.
func Start(ctx context.Context, home string, o SessionOptions) (*Session, error) {
	c, err := LoadConfig(home)
	if err != nil {
		return nil, err
	}
	moon, err := mesa.ParseShip(c.Moon)
	if err != nil {
		return nil, err
	}
	planet, err := mesa.ParseShip(c.Planet)
	if err != nil {
		return nil, err
	}
	var id *mesa.Identity
	if c.Fake {
		id = mesa.FakeIdentity(moon)
	} else if id, err = mesa.LoadKeyfile(filepath.Join(home, "moon.key")); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(home, "drive.db")
	_, statErr := os.Stat(dbPath)
	fresh := errors.Is(statErr, os.ErrNotExist)
	st, err := mesa.OpenSQLite("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	f := dsync.New(dsync.Config{
		Planet: planet, Root: c.Dir, Device: c.Name,
		Log: func(f string, a ...any) { log.Printf(f, a...) },
	}, st.Conn())
	n, err := mesa.Open(mesa.Config{
		Identity: id, Store: st, App: f, Helpers: f.Helpers(),
		Port: o.Port, Breached: fresh && !c.Fake, Trace: o.Trace,
	})
	if err != nil {
		st.Close()
		return nil, err
	}
	if err := f.Start(n); err != nil {
		n.Close()
		st.Close()
		return nil, err
	}
	log.Printf("%s (%s) syncing %s with %s", c.Name, moon, c.Dir, planet)

	ctx, cancel := context.WithCancel(ctx)
	s := &Session{Config: c, Follower: f, cancel: cancel, done: make(chan struct{})}
	go func() {
		if err := f.WatchFS(ctx); err != nil {
			log.Printf("watcher: %v", err)
		}
	}()
	go func() {
		err := n.Run(ctx)
		n.Close()
		st.Close()
		if errors.Is(err, context.Canceled) {
			err = nil
		}
		s.err = err
		close(s.done)
	}()
	return s, nil
}

// Wait returns when the session ends, with its error.
func (s *Session) Wait() error {
	<-s.done
	return s.err
}

// Stop ends the session and waits for the node to close.
func (s *Session) Stop() error {
	s.cancel()
	return s.Wait()
}
