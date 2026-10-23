// drive keeps a folder in sync with your planet's %drive.
//
//	drive pair -url https://sampel-palnet.tlon.network -code ... -name laptop
//	drive run
//	drive status
//
// State lives in -home (default: the OS config dir, /drive): config.json,
// the moon's keyfile, and drive.db (the mesa node's store and ours).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	mesa "github.com/mopfel-winrux/libmesa/bindings/go"
	_ "modernc.org/sqlite"

	"github.com/mopfel-winrux/urbit-drive/client/internal/pair"
	dsync "github.com/mopfel-winrux/urbit-drive/client/internal/sync"
)

type config struct {
	Planet string `json:"planet"`
	Moon   string `json:"moon"`
	Name   string `json:"name"`
	Dir    string `json:"dir"`
	Fake   bool   `json:"fake,omitempty"`
}

func defaultHome() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = "."
	}
	return filepath.Join(d, "drive")
}

func main() {
	log.SetFlags(log.Ltime)
	mesa.Logf = log.Printf
	if len(os.Args) < 2 {
		usage()
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "pair":
		err = cmdPair(args)
	case "run":
		err = cmdRun(args)
	case "status":
		err = cmdStatus(args)
	default:
		usage()
	}
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: drive pair -url URL -code CODE [-name NAME] [-dir DIR] [-publisher ~ship] [-fake]
       drive run [-home DIR] [-port N]
       drive status [-home DIR]`)
	os.Exit(2)
}

func loadConfig(home string) (*config, error) {
	b, err := os.ReadFile(filepath.Join(home, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("not paired yet (run drive pair): %w", err)
	}
	var c config
	return &c, json.Unmarshal(b, &c)
}

func cmdPair(args []string) error {
	fs := flag.NewFlagSet("pair", flag.ExitOnError)
	home := fs.String("home", defaultHome(), "state directory")
	url := fs.String("url", "", "the planet's web address")
	code := fs.String("code", "", "the planet's +code")
	host, _ := os.Hostname()
	name := fs.String("name", host, "this device's name")
	userHome, _ := os.UserHomeDir()
	dir := fs.String("dir", filepath.Join(userHome, "Drive"), "folder to sync")
	publisher := fs.String("publisher", "", "ship to install %drive from, if the planet lacks it")
	fake := fs.Bool("fake", false, "the planet is a fake ship (urbit -F)")
	fs.Parse(args)
	if *url == "" || *code == "" {
		usage()
	}
	if _, err := os.Stat(filepath.Join(*home, "config.json")); err == nil {
		return fmt.Errorf("%s is already paired; use another -home or remove it", *home)
	}
	res, err := pair.Pair(pair.Options{
		URL: *url, Code: *code, Name: *name, Publisher: *publisher, Fake: *fake,
		Log: func(f string, a ...any) { log.Printf(f, a...) },
	})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*home, 0o700); err != nil {
		return err
	}
	if res.Keyfile != "" {
		if err := os.WriteFile(filepath.Join(*home, "moon.key"), []byte(res.Keyfile+"\n"), 0o600); err != nil {
			return err
		}
	}
	abs, _ := filepath.Abs(*dir)
	c := config{Planet: res.Planet.String(), Moon: res.Moon.String(), Name: *name, Dir: abs, Fake: *fake}
	b, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(filepath.Join(*home, "config.json"), b, 0o600); err != nil {
		return err
	}
	log.Printf("paired: %s is %s, syncing %s", *name, res.Moon, abs)
	return nil
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	home := fs.String("home", defaultHome(), "state directory")
	port := fs.Int("port", 0, "UDP port (0: any)")
	trace := fs.Bool("trace", false, "log every datagram")
	fs.Parse(args)
	c, err := loadConfig(*home)
	if err != nil {
		return err
	}
	moon, err := mesa.ParseShip(c.Moon)
	if err != nil {
		return err
	}
	planet, err := mesa.ParseShip(c.Planet)
	if err != nil {
		return err
	}
	var id *mesa.Identity
	if c.Fake {
		id = mesa.FakeIdentity(moon)
	} else if id, err = mesa.LoadKeyfile(filepath.Join(*home, "moon.key")); err != nil {
		return err
	}
	dbPath := filepath.Join(*home, "drive.db")
	_, statErr := os.Stat(dbPath)
	fresh := errors.Is(statErr, os.ErrNotExist)
	st, err := mesa.OpenSQLite("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	f := dsync.New(dsync.Config{
		Planet: planet, Root: c.Dir, Device: c.Name,
		Log: func(f string, a ...any) { log.Printf(f, a...) },
	}, st.Conn())
	n, err := mesa.Open(mesa.Config{
		Identity: id, Store: st, App: f, Helpers: f.Helpers(),
		Port: *port, Breached: fresh && !c.Fake, Trace: *trace,
	})
	if err != nil {
		return err
	}
	if err := f.Start(n); err != nil {
		return err
	}
	log.Printf("%s (%s) syncing %s with %s", c.Name, moon, c.Dir, planet)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := f.WatchFS(ctx); err != nil {
			log.Printf("watcher: %v", err)
		}
	}()
	return n.Run(ctx)
}

func cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	home := fs.String("home", defaultHome(), "state directory")
	fs.Parse(args)
	c, err := loadConfig(*home)
	if err != nil {
		return err
	}
	fmt.Printf("device  %s (%s)\nplanet  %s\nfolder  %s\n", c.Name, c.Moon, c.Planet, c.Dir)
	db, err := sql.Open("sqlite", "file:"+filepath.Join(*home, "drive.db")+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT f.name, f.fid, f.synced, f.head, f.pending IS NOT NULL,
		(SELECT count(*) FROM drive_file WHERE fid = f.fid) FROM drive_folder f`)
	if err != nil {
		return nil // never ran
	}
	defer rows.Close()
	for rows.Next() {
		var name, fid string
		var synced, head, files int64
		var pend bool
		rows.Scan(&name, &fid, &synced, &head, &pend, &files)
		s := fmt.Sprintf("  %-16s %s  aeon %d/%d  %d files", name, fid, synced, head, files)
		if pend {
			s += "  (committing)"
		}
		fmt.Println(s)
	}
	return nil
}
