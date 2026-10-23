package sync

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/mopfel-winrux/urbit-drive/client/internal/chunk"
	"github.com/mopfel-winrux/urbit-drive/client/internal/proto"
)

// FileState is where one file is in syncing, as the tray shows it.
type FileState string

const (
	Uploading   FileState = "uploading"
	Downloading FileState = "downloading"
	Synced      FileState = "synced"
	Deleted     FileState = "deleted"
	Conflict    FileState = "conflict"
	Failed      FileState = "failed"
)

// FileEvent is the latest thing that happened to a file.
type FileEvent struct {
	Folder string // folder name
	Path   string // in the folder
	State  FileState
	Done   int64 // bytes moved so far, while uploading or downloading
	Total  int64
	When   time.Time
}

// Active: still moving.
func (e FileEvent) Active() bool { return e.State == Uploading || e.State == Downloading }

const keepEvents = 64

// activity is the follower's recent file events, newest last; loop only.
type activity struct {
	byKey map[string]*FileEvent
	order []*FileEvent
	// chunk -> events waiting on it, to count progress
	waiting map[chunk.Hash][]*FileEvent
	sizes   map[chunk.Hash]int64
	// commit -> what it does, to settle when its result comes
	commits map[uint64][]proto.Cop
}

func (f *Follower) act() *activity {
	if f.recent == nil {
		f.recent = &activity{byKey: map[string]*FileEvent{}, waiting: map[chunk.Hash][]*FileEvent{},
			sizes: map[chunk.Hash]int64{}, commits: map[uint64][]proto.Cop{}}
	}
	return f.recent
}

// note records [state] for a file, replacing what was there.
func (f *Follower) note(fo *folder, p string, state FileState, total int64) *FileEvent {
	a := f.act()
	key := fo.fid + "\x00" + p
	e := a.byKey[key]
	if e == nil {
		e = &FileEvent{Folder: fo.name, Path: p}
		a.byKey[key] = e
	} else {
		for i, x := range a.order {
			if x == e {
				a.order = append(a.order[:i], a.order[i+1:]...)
				break
			}
		}
	}
	e.State, e.Done, e.Total, e.When = state, 0, total, time.Now()
	a.order = append(a.order, e)
	if len(a.order) > keepEvents {
		old := a.order[0]
		a.order = a.order[1:]
		for k, v := range a.byKey {
			if v == old {
				delete(a.byKey, k)
			}
		}
	}
	return e
}

// expect: [e] moves these chunks, of these sizes.
func (f *Follower) expect(e *FileEvent, chunks []chunk.Hash, sizes func(chunk.Hash) int64) {
	a := f.act()
	for _, h := range chunks {
		a.waiting[h] = append(a.waiting[h], e)
		if sizes != nil {
			a.sizes[h] = sizes(h)
		}
	}
}

// moved: a chunk went up (the planet read it) or came down.
func (f *Follower) moved(h chunk.Hash, n int64) {
	a := f.act()
	for _, e := range a.waiting[h] {
		if e.Active() {
			e.Done = min(e.Total, e.Done+n)
		}
	}
	delete(a.waiting, h)
}

// noteCommit: a commit is on its way up.
func (f *Follower) noteCommit(fo *folder, c *proto.Commit, chunks map[chunk.Hash][]byte) {
	f.act().commits[c.Cid] = c.Ops
	for _, op := range c.Ops {
		if op.New == nil || isMarker(op.Path) {
			continue
		}
		e := f.note(fo, op.Path, Uploading, int64(op.New.Size))
		var mine []chunk.Hash
		for _, h := range op.New.Chunks {
			if b, ok := chunks[h]; ok {
				mine = append(mine, h)
				f.act().sizes[h] = int64(len(b))
			}
		}
		if len(mine) == 0 { // the planet has every chunk already
			e.Done = e.Total
		}
		f.expect(e, mine, nil)
	}
}

// settleCommit: its result came.
func (f *Follower) settleCommit(fo *folder, cid uint64, ok bool) {
	a := f.act()
	for _, op := range a.commits[cid] {
		switch {
		case !ok:
			if op.New != nil {
				f.note(fo, op.Path, Failed, 0)
			}
		case op.New == nil:
			f.note(fo, op.Path, Deleted, 0)
		case !isMarker(op.Path):
			e := f.note(fo, op.Path, Synced, int64(op.New.Size))
			e.Done = e.Total
		}
	}
	delete(a.commits, cid)
}

// Recent is the latest file events, newest first, at most n; call it
// from any goroutine but the node's.
func (f *Follower) Recent(n int) ([]FileEvent, error) {
	var out []FileEvent
	err := f.n.Do(func() error {
		for _, e := range f.act().order {
			out = append(out, *e)
		}
		return nil
	})
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Active() != out[j].Active() {
			return out[i].Active()
		}
		return out[i].When.After(out[j].When)
	})
	if len(out) > n {
		out = out[:n]
	}
	return out, err
}

// LocalPath is where a file event's file lives on this computer.
func (f *Follower) LocalPath(e FileEvent) string {
	return filepath.Join(f.cfg.Root, safeName(e.Folder), filepath.FromSlash(e.Path))
}
