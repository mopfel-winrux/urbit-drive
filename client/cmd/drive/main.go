// drive keeps a folder in sync with your planet's %drive.
//
//	drive                      the tray app (pairs from a local page if needed)
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
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/atotto/clipboard"
	mesa "github.com/mopfel-winrux/libmesa/bindings/go"
	_ "modernc.org/sqlite"

	"github.com/mopfel-winrux/urbit-drive/client/internal/pair"
	"github.com/mopfel-winrux/urbit-drive/client/internal/ui"
)

// version is set at release builds (-X main.version=...).
var version = "dev"

func main() {
	log.SetFlags(log.Ltime)
	mesa.Logf = log.Printf
	cmd, args := "tray", os.Args[1:]
	// older macOS passes -psn_<n> to apps started from the Finder
	if len(args) > 0 && strings.HasPrefix(args[0], "-psn_") {
		args = args[1:]
	}
	if len(args) > 0 && args[0] != "" && args[0][0] != '-' {
		cmd, args = args[0], args[1:]
	}
	var err error
	switch cmd {
	case "tray":
		err = cmdTray(args)
	case "pair":
		err = cmdPair(args)
	case "run":
		err = cmdRun(args)
	case "status":
		err = cmdStatus(args)
	case "link":
		err = cmdLink(args)
	case "exclude", "include":
		err = cmdSelective(cmd, args)
	case "version":
		fmt.Println("drive", version)
	default:
		usage()
	}
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: drive [tray] [-home DIR] [-port N]
       drive pair -url URL -code CODE [-name NAME] [-dir DIR] [-publisher ~ship] [-fake]
       drive run [-home DIR] [-port N]
       drive status [-home DIR]
       drive link [-home DIR] PATH
       drive exclude|include [-home DIR] FOLDER PATH
       drive version`)
	os.Exit(2)
}

func cmdTray(args []string) error {
	fs := flag.NewFlagSet("tray", flag.ExitOnError)
	home := fs.String("home", ui.DefaultHome(), "state directory")
	port := fs.Int("port", 0, "UDP port (0: any)")
	fs.Parse(args)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	a := ui.NewApp(ctx, *home, ui.SessionOptions{Port: *port})
	if err := a.Serve(); err != nil {
		return err
	}
	switch err := a.StartSync(); {
	case errors.Is(err, ui.ErrNotPaired):
		ui.Open(a.URL()) // the pairing form
	case err != nil:
		log.Printf("sync: %v", err) // shown in the tray and on the page
	}
	ui.RunTray(a)
	return nil
}

func cmdPair(args []string) error {
	fs := flag.NewFlagSet("pair", flag.ExitOnError)
	home := fs.String("home", ui.DefaultHome(), "state directory")
	url := fs.String("url", "", "the planet's web address")
	code := fs.String("code", "", "the planet's +code")
	host, _ := os.Hostname()
	name := fs.String("name", host, "this device's name")
	dir := fs.String("dir", ui.DefaultDir(), "folder to sync")
	publisher := fs.String("publisher", "", "ship to install %drive from, if the planet lacks it")
	fake := fs.Bool("fake", false, "the planet is a fake ship (urbit -F)")
	fs.Parse(args)
	if *url == "" || *code == "" {
		usage()
	}
	_, err := ui.Pair(*home, *dir, pair.Options{
		URL: *url, Code: *code, Name: *name, Publisher: *publisher, Fake: *fake,
		Log: func(f string, a ...any) { log.Printf(f, a...) },
	})
	return err
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	home := fs.String("home", ui.DefaultHome(), "state directory")
	port := fs.Int("port", 0, "UDP port (0: any)")
	trace := fs.Bool("trace", false, "log every datagram")
	fs.Parse(args)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// headless, but with the local page (not opened), so `drive link` works
	a := ui.NewApp(ctx, *home, ui.SessionOptions{Port: *port, Trace: *trace})
	if err := a.Serve(); err != nil {
		return err
	}
	if err := a.StartSync(); err != nil {
		return err
	}
	<-ctx.Done()
	a.Close()
	return nil
}

// cmdLink asks the running Drive for a share link to a file or folder,
// prints it, and copies it.
func cmdLink(args []string) error {
	fs := flag.NewFlagSet("link", flag.ExitOnError)
	home := fs.String("home", ui.DefaultHome(), "state directory")
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: drive link [-home DIR] PATH") }
	fs.Parse(args)
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	link, err := ui.AskRunning(*home, fs.Arg(0))
	if err != nil {
		return err
	}
	fmt.Println(link)
	if clipboard.WriteAll(link) == nil {
		fmt.Fprintln(os.Stderr, "(copied)")
	}
	return nil
}

func cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	home := fs.String("home", ui.DefaultHome(), "state directory")
	fs.Parse(args)
	c, err := ui.LoadConfig(*home)
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

// cmdSelective excludes a path in a folder from this device, or includes
// it again. It runs a session briefly, so the tray or daemon must not
// be running (use the settings page then).
func cmdSelective(cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	home := fs.String("home", ui.DefaultHome(), "state directory")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: drive %s [-home DIR] FOLDER PATH\n", cmd)
	}
	fs.Parse(args)
	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(2)
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	s, err := ui.Start(ctx, *home, ui.SessionOptions{})
	if err != nil {
		return fmt.Errorf("%w (is drive already running? use its settings page)", err)
	}
	if cmd == "exclude" {
		err = s.Follower.Exclude(fs.Arg(0), fs.Arg(1))
	} else {
		err = s.Follower.Include(fs.Arg(0), fs.Arg(1))
	}
	s.Stop()
	return err
}
