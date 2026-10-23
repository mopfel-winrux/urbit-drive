package sync

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	mesa "github.com/mopfel-winrux/libmesa/bindings/go"

	"github.com/mopfel-winrux/urbit-drive/client/internal/chunk"
	"github.com/mopfel-winrux/urbit-drive/client/internal/proto"
)

func sha256Sum(b []byte) [32]byte { return sha256.Sum256(b) }

// row is what we last synced at a path: the planet's hash for it, and
// the file's size and mtime on disk then (to notice local edits without
// rehashing).
type row struct {
	hash  chunk.Hash
	stat  stat
	found bool
}

func (f *Follower) getFile(fid, p string) row {
	var r row
	var h []byte
	err := f.db.QueryRowContext(bg, `SELECT hash, size, mtime FROM drive_file WHERE fid = ? AND path = ?`, fid, p).
		Scan(&h, &r.stat.Size, &r.stat.Mtime)
	if err == nil {
		copy(r.hash[:], h)
		r.found = true
	}
	return r
}

func (f *Follower) setFile(fid, p string, h chunk.Hash, s stat) error {
	return f.exec(`INSERT OR REPLACE INTO drive_file(fid, path, hash, size, mtime) VALUES(?,?,?,?,?)`,
		fid, p, h[:], s.Size, s.Mtime)
}

func (f *Follower) delFile(fid, p string) error {
	return f.exec(`DELETE FROM drive_file WHERE fid = ? AND path = ?`, fid, p)
}

func statOf(fi fs.FileInfo) stat { return stat{fi.Size(), fi.ModTime().UnixNano()} }

// validRel accepts a relative slash path that stays inside the folder.
func validRel(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") || strings.ContainsRune(p, 0) {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}
	return path.Clean(p) == p
}

// ignored names are never synced: our own files, and OS/editor litter.
func ignored(name string) bool {
	switch {
	case name == ".drive", strings.HasPrefix(name, ".drive-tmp-"),
		name == ".DS_Store", name == "Thumbs.db", name == "desktop.ini",
		strings.HasPrefix(name, "~$"), strings.HasSuffix(name, ".swp"):
		return true
	}
	return false
}

// matchesDisk: the file at [p] is what we last synced, and that is [h].
func (f *Follower) matchesDisk(fo *folder, p string, h chunk.Hash) bool {
	r := f.getFile(fo.fid, p)
	if !r.found || r.hash != h {
		return false
	}
	fi, err := os.Lstat(filepath.Join(f.dir(fo), filepath.FromSlash(p)))
	return err == nil && fi.Mode().IsRegular() && statOf(fi) == r.stat
}

// applyWave writes a wave's ops to disk, then moves on to the next one.
func (f *Follower) applyWave(fo *folder, w *proto.Wave) {
	ours := w.Author == f.n.Ship()
	for _, op := range w.Ops {
		if err := f.applyOp(fo, op, ours); err != nil {
			f.cfg.Log("%s/%s: %v", fo.name, op.Path, err)
		}
	}
	if ours {
		fo.pending = nil
	}
	fo.synced, fo.pulling = w.Aeon, false
	f.endReplay(fo)
	f.saveFolder(fo)
	os.RemoveAll(f.cacheDir())
	f.cfg.Log("%q at aeon %d (%d ops)", fo.name, w.Aeon, len(w.Ops))
	f.pump(fo)
	f.scanSoon(fo)
}

// applyOp settles one op against the disk.
//
// Our row says what the planet and the disk agreed on last. If the disk
// still matches it, the op just happens. If it doesn't, the user edited
// the file meanwhile: an edit beats a delete; our own wave never
// overwrites a newer edit (it's committed next, on top); and someone
// else's change to a file we edited and haven't committed moves our edit
// aside as a conflicted copy, which the next scan uploads. If our edit
// is already in a commit in flight, the planet makes that copy instead.
func (f *Follower) applyOp(fo *folder, op proto.Op, ours bool) error {
	if !validRel(op.Path) {
		return errors.New("unsafe path, skipped")
	}
	if fo.excluded(op.Path) {
		return f.delFile(fo.fid, op.Path)
	}
	if fo.replaying(op.Path) {
		return nil
	}
	abs := filepath.Join(f.dir(fo), filepath.FromSlash(op.Path))
	r := f.getFile(fo.fid, op.Path)
	fi, err := os.Lstat(abs)
	exists := err == nil
	if exists && !fi.Mode().IsRegular() {
		return errors.New("not a regular file on disk, skipped")
	}
	changed := exists != r.found || exists && statOf(fi) != r.stat
	var sent *stat
	if fo.pending != nil {
		if s, ok := fo.pending.Paths[op.Path]; ok {
			sent = &s
		}
	}

	if op.New == nil {
		if exists && changed {
			return f.delFile(fo.fid, op.Path)
		}
		if exists {
			if err := os.Remove(abs); err != nil {
				return err
			}
			pruneDirs(filepath.Dir(abs), f.dir(fo))
		}
		return f.delFile(fo.fid, op.Path)
	}

	e := op.New
	if exists && !changed && r.hash == e.Hash {
		return f.setFile(fo.fid, op.Path, e.Hash, statOf(fi))
	}
	if ours {
		if exists && sent != nil && statOf(fi) == *sent {
			return f.setFile(fo.fid, op.Path, e.Hash, *sent)
		}
		if exists && changed {
			// edited again since we committed: the next scan rehashes
			return f.setFile(fo.fid, op.Path, e.Hash, stat{})
		}
	} else if exists && changed && sent == nil {
		if h, err := hashFile(abs); err == nil && h == e.Hash {
			return f.setFile(fo.fid, op.Path, e.Hash, statOf(fi))
		}
		alt := conflictName(abs, f.cfg.Device, time.Now())
		if err := os.Rename(abs, alt); err != nil {
			return err
		}
		f.cfg.Log("kept your edit of %s as %s", op.Path, filepath.Base(alt))
	}
	s, err := f.writeFile(abs, e)
	if err != nil {
		return err
	}
	return f.setFile(fo.fid, op.Path, e.Hash, s)
}

func hashFile(p string) (chunk.Hash, error) {
	cf, err := chunk.SplitFile(p, nil)
	if err != nil {
		return chunk.Hash{}, err
	}
	return cf.Hash, nil
}

// writeFile assembles [e] from its chunks beside [abs] and renames it in.
func (f *Follower) writeFile(abs string, e *proto.Entry) (stat, error) {
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return stat{}, err
	}
	tmp := filepath.Join(filepath.Dir(abs), fmt.Sprintf(".drive-tmp-%x", randU64()))
	mode := os.FileMode(0o644)
	if e.Exec {
		mode = 0o755
	}
	fd, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return stat{}, err
	}
	whole := sha256.New()
	var size uint64
	for _, h := range e.Chunks {
		b, err := f.readChunk(h)
		if err == nil {
			_, err = fd.Write(b)
		}
		if err != nil {
			fd.Close()
			os.Remove(tmp)
			return stat{}, fmt.Errorf("chunk %s: %w", h, err)
		}
		whole.Write(b)
		size += uint64(len(b))
	}
	if err := fd.Close(); err != nil {
		os.Remove(tmp)
		return stat{}, err
	}
	if size != e.Size || chunk.Hash(whole.Sum(nil)) != e.Hash {
		os.Remove(tmp)
		return stat{}, errors.New("assembled file doesn't match its hash")
	}
	os.Chtimes(tmp, e.Mtime, e.Mtime)
	if err := os.Rename(tmp, abs); err != nil {
		os.Remove(tmp)
		return stat{}, err
	}
	fi, err := os.Lstat(abs)
	if err != nil {
		return stat{}, err
	}
	return statOf(fi), nil
}

// pruneDirs removes empty directories from [dir] up to [root].
func pruneDirs(dir, root string) {
	for dir != root && strings.HasPrefix(dir, root) {
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

// conflictName is "name (conflicted copy, device, 2026-10-23).ext",
// numbered if that's taken; the planet makes the same names.
func conflictName(abs, device string, when time.Time) string {
	dir, base := filepath.Split(abs)
	ext := ""
	if i := strings.LastIndex(base, "."); i > 0 {
		base, ext = base[:i], base[i:]
	}
	name := fmt.Sprintf("%s (conflicted copy, %s, %s)", base, device, when.Format("2006-01-02"))
	alt := filepath.Join(dir, name+ext)
	for i := 2; ; i++ {
		if _, err := os.Lstat(alt); errors.Is(err, fs.ErrNotExist) {
			return alt
		}
		alt = filepath.Join(dir, fmt.Sprintf("%s %d%s", name, i, ext))
	}
}

// ----- scanning: local edits to commits

const scanBudget = 256 << 20 // bytes of new chunks per commit

// scanSoon scans the folder shortly, once it can: caught up, with no
// commit in flight.
func (f *Follower) scanSoon(fo *folder) {
	debugf("scanSoon %q scanning=%v pulling=%v pending=%v synced=%d head=%d", fo.name, fo.scanning, fo.pulling, fo.pending != nil, fo.synced, fo.head)
	if fo.readOnly {
		return
	}
	if fo.scanning || fo.pulling || fo.pending != nil || fo.synced < fo.head || !fo.live {
		fo.rescan = true // whatever is blocking it scans again when done
		return
	}
	fo.rescan, fo.scanning = false, true
	f.n.After(300*time.Millisecond, func() error { return f.scan(fo) })
}

type scanned struct {
	ops     []proto.Cop
	stats   map[string]stat
	touches map[string]stat // same contents, new mtime
	chunks  map[chunk.Hash][]byte
	more    bool
	err     error
}

// scan walks the folder off the loop and comes back with a commit.
func (f *Follower) scan(fo *folder) error {
	rows := map[string]row{}
	q, err := f.db.QueryContext(bg, `SELECT path, hash, size, mtime FROM drive_file WHERE fid = ?`, fo.fid)
	if err != nil {
		fo.scanning = false
		return err
	}
	for q.Next() {
		var p string
		var h []byte
		var r row
		q.Scan(&p, &h, &r.stat.Size, &r.stat.Mtime)
		copy(r.hash[:], h)
		r.found = true
		rows[p] = r
	}
	q.Close()
	dir, synced, exclude := f.dir(fo), fo.synced, append([]string(nil), fo.exclude...)
	debugf("scan %q in %s (%d rows)", fo.name, dir, len(rows))
	go func() {
		res := walk(dir, rows, exclude)
		err := f.n.Do(func() error { return f.finishScan(fo, synced, res) })
		if err != nil {
			f.cfg.Log("scan %q: %v", fo.name, err)
			f.n.Do(func() error { fo.scanning = false; f.scanSoon(fo); return nil })
		}
	}()
	return nil
}

func walk(dir string, rows map[string]row, exclude []string) *scanned {
	skip := &folder{exclude: exclude}
	res := &scanned{stats: map[string]stat{}, touches: map[string]stat{}, chunks: map[chunk.Hash][]byte{}}
	seen := map[string]bool{}
	budget := scanBudget
	res.err = filepath.WalkDir(dir, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // vanished meanwhile
		}
		if abs == dir {
			return nil
		}
		if ignored(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(dir, abs)
		rel = filepath.ToSlash(rel)
		if skip.excluded(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // directories are implied; links aren't synced yet
		}
		if !validRel(rel) {
			return nil
		}
		seen[rel] = true
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		r := rows[rel]
		if r.found && statOf(fi) == r.stat {
			return nil
		}
		if budget <= 0 {
			res.more = true
			return nil
		}
		got := map[chunk.Hash][]byte{}
		cf, err := chunk.SplitFile(abs, func(h chunk.Hash, b []byte) error {
			if _, ok := got[h]; !ok {
				got[h] = append([]byte(nil), b...)
			}
			return nil
		})
		if err != nil {
			return nil
		}
		fi2, err := os.Lstat(abs)
		if err != nil || statOf(fi2) != statOf(fi) {
			res.more = true // changing as we read: next time
			return nil
		}
		if r.found && r.hash == cf.Hash {
			res.touches[rel] = statOf(fi)
			return nil
		}
		for h, b := range got {
			res.chunks[h] = b
			budget -= len(b)
		}
		e := &proto.Entry{Hash: cf.Hash, Size: uint64(cf.Size), Mtime: fi.ModTime(),
			Exec: fi.Mode()&0o111 != 0, Chunks: cf.Chunks}
		var base *chunk.Hash
		if r.found {
			h := r.hash
			base = &h
		}
		res.ops = append(res.ops, proto.Cop{Path: rel, Base: base, New: e})
		res.stats[rel] = statOf(fi)
		return nil
	})
	for p, r := range rows {
		if !seen[p] {
			h := r.hash
			res.ops = append(res.ops, proto.Cop{Path: p, Base: &h})
		}
	}
	return res
}

func (f *Follower) finishScan(fo *folder, synced uint64, res *scanned) error {
	debugf("finishScan %q ops=%d touches=%d more=%v err=%v", fo.name, len(res.ops), len(res.touches), res.more, res.err)
	fo.scanning = false
	if res.err != nil {
		f.cfg.Log("scan %q: %v", fo.name, res.err)
	}
	if fo.synced != synced || fo.pending != nil || fo.pulling {
		fo.rescan = false
		f.scanSoon(fo)
		return nil
	}
	for p, s := range res.touches {
		r := f.getFile(fo.fid, p)
		if err := f.setFile(fo.fid, p, r.hash, s); err != nil {
			return err
		}
	}
	if len(res.ops) == 0 {
		if res.more || fo.rescan {
			fo.rescan = false
			f.scanSoon(fo)
		}
		return nil
	}
	c := &proto.Commit{Cid: randU64(), Fid: fo.fid, Base: fo.synced, Ops: res.ops}
	for h, b := range res.chunks {
		if err := f.exec(`INSERT OR IGNORE INTO drive_staged(hash, data, cid) VALUES(?,?,?)`, h[:], b, c.Cid); err != nil {
			return err
		}
	}
	fo.pending = &pending{Cid: c.Cid, Paths: res.stats}
	if err := f.saveFolder(fo); err != nil {
		return err
	}
	if _, err := f.pokes.Poke(f.n, f.cfg.Planet, proto.Agent, proto.ActionMark, c.Action()); err != nil {
		return err
	}
	f.cfg.Log("committing %d change(s) to %q", len(res.ops), fo.name)
	return nil
}

// ScanAll asks every folder to scan (from the watcher and the sweep).
func (f *Follower) ScanAll() error {
	return f.n.Do(func() error {
		for _, fo := range f.folders {
			f.scanSoon(fo)
		}
		return nil
	})
}

var _ mesa.App = (*Follower)(nil)

var debug = os.Getenv("DRIVE_DEBUG") != ""

func debugf(format string, a ...any) {
	if debug {
		log.Printf("debug: "+format, a...)
	}
}
