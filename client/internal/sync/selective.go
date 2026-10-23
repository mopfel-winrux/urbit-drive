package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Selective sync: a folder's excluded paths (a file, or a directory and
// everything in it) are neither downloaded nor uploaded here. Excluding
// one removes our unedited copies of it; including it again re-reads the
// folder's history for just those paths (a replay) until we're back
// where we were.

func under(p, prefix string) bool {
	return p == prefix || strings.HasPrefix(p, prefix+"/")
}

func (fo *folder) excluded(p string) bool {
	for _, x := range fo.exclude {
		if under(p, x) {
			return true
		}
	}
	return false
}

// replaying: an op on [p] is from history we already applied, unless p
// was just included again.
func (fo *folder) replaying(p string) bool {
	if fo.replay == nil || fo.synced >= fo.replay.Until {
		return false
	}
	for _, x := range fo.replay.Only {
		if under(p, x) {
			return false
		}
	}
	return true
}

// wanted: we apply ops on [p] now.
func (f *Follower) wanted(fo *folder, p string) bool {
	return !fo.excluded(p) && !fo.replaying(p)
}

func (f *Follower) loadSelective() error {
	q, err := f.db.QueryContext(bg, `SELECT fid, prefix FROM drive_exclude`)
	if err != nil {
		return err
	}
	for q.Next() {
		var fid, p string
		q.Scan(&fid, &p)
		if fo, ok := f.folders[fid]; ok {
			fo.exclude = append(fo.exclude, p)
		}
	}
	q.Close()
	q, err = f.db.QueryContext(bg, `SELECT fid, body FROM drive_replay`)
	if err != nil {
		return err
	}
	defer q.Close()
	for q.Next() {
		var fid string
		var b []byte
		q.Scan(&fid, &b)
		if fo, ok := f.folders[fid]; ok {
			fo.replay = &replay{}
			json.Unmarshal(b, fo.replay)
		}
	}
	return nil
}

func (f *Follower) saveReplay(fo *folder) error {
	if fo.replay == nil {
		return f.exec(`DELETE FROM drive_replay WHERE fid = ?`, fo.fid)
	}
	b, _ := json.Marshal(fo.replay)
	return f.exec(`INSERT OR REPLACE INTO drive_replay(fid, body) VALUES(?,?)`, fo.fid, b)
}

// endReplay: called after each wave; the replay ends where we were.
func (f *Follower) endReplay(fo *folder) {
	if fo.replay != nil && fo.synced >= fo.replay.Until {
		fo.replay = nil
		f.saveReplay(fo)
	}
}

func (f *Follower) folderNamed(name string) (*folder, error) {
	for _, fo := range f.folders {
		if fo.name == name || fo.fid == name {
			return fo, nil
		}
	}
	return nil, fmt.Errorf("no folder %q", name)
}

func cleanRel(p string) (string, error) {
	p = strings.Trim(filepath.ToSlash(p), "/")
	if !validRel(p) {
		return "", fmt.Errorf("bad path %q", p)
	}
	return p, nil
}

// Exclude stops syncing [p] in [folder] (by name) on this device.
func (f *Follower) Exclude(folder, p string) error {
	p, err := cleanRel(p)
	if err != nil {
		return err
	}
	return f.n.Do(func() error {
		fo, err := f.folderNamed(folder)
		if err != nil {
			return err
		}
		if fo.excluded(p) {
			return nil
		}
		if err := f.exec(`INSERT OR IGNORE INTO drive_exclude(fid, prefix) VALUES(?,?)`, fo.fid, p); err != nil {
			return err
		}
		fo.exclude = append(fo.exclude, p)
		// forget our copies; delete the ones nobody edited here
		q, err := f.db.QueryContext(bg, `SELECT path, size, mtime FROM drive_file WHERE fid = ?`, fo.fid)
		if err != nil {
			return err
		}
		var gone []string
		for q.Next() {
			var rp string
			var st stat
			q.Scan(&rp, &st.Size, &st.Mtime)
			if !under(rp, p) {
				continue
			}
			gone = append(gone, rp)
			abs := filepath.Join(f.dir(fo), filepath.FromSlash(rp))
			if fi, err := os.Lstat(abs); err == nil && statOf(fi) == st {
				os.Remove(abs)
				f.pruneDirs(fo, filepath.Dir(abs))
			}
		}
		q.Close()
		for _, rp := range gone {
			if err := f.delFile(fo.fid, rp); err != nil {
				return err
			}
		}
		f.cfg.Log("%q: no longer syncing %s here", fo.name, p)
		return nil
	})
}

// Include syncs [p] in [folder] here again, replaying the folder's
// history for it.
func (f *Follower) Include(folder, p string) error {
	p, err := cleanRel(p)
	if err != nil {
		return err
	}
	return f.n.Do(func() error {
		fo, err := f.folderNamed(folder)
		if err != nil {
			return err
		}
		var keep []string
		found := false
		for _, x := range fo.exclude {
			if x == p {
				found = true
			} else {
				keep = append(keep, x)
			}
		}
		if !found {
			return fmt.Errorf("%s isn't excluded from %q", p, fo.name)
		}
		if err := f.exec(`DELETE FROM drive_exclude WHERE fid = ? AND prefix = ?`, fo.fid, p); err != nil {
			return err
		}
		fo.exclude = keep
		until := fo.synced
		if fo.replay != nil {
			until = max(until, fo.replay.Until)
			fo.replay.Only = append(fo.replay.Only, p)
		} else {
			fo.replay = &replay{Only: []string{p}}
		}
		fo.replay.Until = until
		f.cancel(fo)
		fo.synced = 0
		if err := f.saveReplay(fo); err != nil {
			return err
		}
		if err := f.saveFolder(fo); err != nil {
			return err
		}
		f.cfg.Log("%q: syncing %s here again", fo.name, p)
		f.pump(fo)
		return nil
	})
}

// Excluded lists each folder's excluded paths, by folder name.
func (f *Follower) Excluded() (map[string][]string, error) {
	out := map[string][]string{}
	err := f.n.Do(func() error {
		for _, fo := range f.folders {
			xs := append([]string(nil), fo.exclude...)
			sort.Strings(xs)
			out[fo.name] = xs
		}
		return nil
	})
	return out, err
}
