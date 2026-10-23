package sync

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchFS scans when anything under Root changes (debounced by scanSoon's
// delay and its one-scan-at-a-time rule), and sweeps every 30s besides,
// in case the OS dropped an event. It returns when ctx is done.
func (f *Follower) WatchFS(ctx context.Context) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	os.MkdirAll(f.cfg.Root, 0o755)
	addTree := func(root string) {
		filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if ignored(d.Name()) && p != root {
				return filepath.SkipDir
			}
			w.Add(p)
			return nil
		})
	}
	addTree(f.cfg.Root)
	sweep := time.NewTicker(30 * time.Second)
	defer sweep.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-w.Events:
			if ignored(filepath.Base(ev.Name)) {
				continue
			}
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := os.Lstat(ev.Name); err == nil && fi.IsDir() {
					addTree(ev.Name)
				}
			}
			f.ScanAll()
		case err := <-w.Errors:
			f.cfg.Log("watch: %v", err)
		case <-sweep.C:
			addTree(f.cfg.Root)
			f.ScanAll()
		}
	}
}
