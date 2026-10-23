// Package sync is the follower: it keeps a local directory in sync with
// the planet's %drive folders over mesa.
//
// Down: a subscription to /v1/folders says each folder's head; we read
// the waves we lack (and the chunks they name) by remote scry, under the
// folder's %shut key, and apply them to disk in order.
//
// Up: a scan finds files that differ from what we last synced, stages
// their chunks where the planet can read them (/up/<hex>, %shut to the
// planet only) and pokes a %commit naming them. The planet settles
// conflicts and publishes the result as a wave like any other.
//
// Everything that touches the node or the database runs on the node's
// loop, inside the transaction of the event that caused it; hashing and
// directory walks run on goroutines and come back through Node.Do.
package sync

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	mesa "github.com/mopfel-winrux/libmesa/bindings/go"

	"github.com/mopfel-winrux/urbit-drive/client/internal/chunk"
	"github.com/mopfel-winrux/urbit-drive/client/internal/proto"
)

// Config is what a follower needs.
type Config struct {
	Planet mesa.Ship
	Root   string // each folder syncs to Root/<folder name>
	Device string // our name, for conflicted copies
	Log    func(string, ...any)
}

type folder struct {
	fid, name     string
	epoch, head   uint64
	synced        uint64
	pending       *pending // our commit in flight
	pulling       bool
	live          bool // heard its head this run: don't commit on a stale base
	readOnly      bool
	scanning      bool
	rescan        bool // a scan was asked for while one couldn't run
	key           *shutKey
	keyAsked      bool
	waitingChunks int
}

// pending is our commit in flight, persisted with the folder.
type pending struct {
	Cid   uint64
	Paths map[string]stat // what we hashed, to skip rehashing on its wave
}

type stat struct {
	Size  int64
	Mtime int64 // ns
}

type shutKey struct {
	kid uint64
	key [32]byte
}

// Follower is the sync engine; it is a mesa.App.
type Follower struct {
	mesa.BaseApp
	cfg   Config
	n     *mesa.Node
	db    *sql.Conn
	subs  *mesa.Subs
	pokes *mesa.Pokes

	folders map[string]*folder
	scries  map[string]func(noun mesa.Noun, ok bool)
	keyBone uint64
	keyQ    []string // fids asked for on keyBone, in order
	upKid   uint64
}

// New makes a follower; pass its Helpers() to mesa.Open with it as the
// App, then call Start.
func New(cfg Config, db *sql.Conn) *Follower {
	if cfg.Log == nil {
		cfg.Log = func(string, ...any) {}
	}
	f := &Follower{cfg: cfg, db: db, folders: map[string]*folder{}, scries: map[string]func(mesa.Noun, bool){}}
	f.subs = &mesa.Subs{OnFact: f.onFact, OnWatch: func(n *mesa.Node, s mesa.Sub, err *mesa.Nack) {
		if err != nil {
			cfg.Log("subscription refused: %s %v", err.Tag, err.Lines)
		} else {
			cfg.Log("subscribed to %s", cfg.Planet)
		}
	}, OnKick: func(n *mesa.Node, s mesa.Sub) {
		debugf("kicked from %s %s; resubscribing", s.Ship, s.Path)
	}}
	f.pokes = &mesa.Pokes{Name: "drive", OnDone: func(n *mesa.Node, her mesa.Ship, agent string, seq uint64, err *mesa.Nack) {
		if err != nil {
			cfg.Log("commit poke nacked: %s %v", err.Tag, err.Lines)
		}
	}}
	return f
}

func (f *Follower) Helpers() []mesa.Helper { return []mesa.Helper{f.subs, f.pokes} }

const schema = `
CREATE TABLE IF NOT EXISTS drive_folder(fid TEXT PRIMARY KEY, name TEXT, epoch INT, head INT, synced INT, pending BLOB);
CREATE TABLE IF NOT EXISTS drive_file(fid TEXT, path TEXT, hash BLOB, size INT, mtime INT, PRIMARY KEY(fid, path));
CREATE TABLE IF NOT EXISTS drive_staged(hash BLOB PRIMARY KEY, data BLOB, cid INT);
CREATE TABLE IF NOT EXISTS drive_meta(k TEXT PRIMARY KEY, v INT);
`

var bg = context.Background()

func (f *Follower) exec(q string, args ...any) error {
	_, err := f.db.ExecContext(bg, q, args...)
	return err
}

// Start loads our state and subscribes; call it after mesa.Open (it
// runs its own transaction).
func (f *Follower) Start(n *mesa.Node) error {
	f.n = n
	return n.Do(func() error {
		for _, q := range strings.Split(schema, ";") {
			if strings.TrimSpace(q) != "" {
				if err := f.exec(q); err != nil {
					return err
				}
			}
		}
		if err := f.loadMeta(); err != nil {
			return err
		}
		rows, err := f.db.QueryContext(bg, `SELECT fid, name, epoch, head, synced, pending FROM drive_folder`)
		if err != nil {
			return err
		}
		for rows.Next() {
			fo := &folder{}
			var pend []byte
			if err := rows.Scan(&fo.fid, &fo.name, &fo.epoch, &fo.head, &fo.synced, &pend); err != nil {
				rows.Close()
				return err
			}
			if len(pend) > 0 {
				fo.pending = &pending{}
				json.Unmarshal(pend, fo.pending)
			}
			f.folders[fo.fid] = fo
		}
		rows.Close()
		for _, fo := range f.folders {
			f.pump(fo) // scries don't survive a restart: ask again
			f.scanSoon(fo)
		}
		// a fresh watch: its first facts are every folder's head, now.
		// The old flow's facts may be stuck in the planet's backoff for
		// us, if we were offline a while.
		for _, s := range f.subs.List() {
			if err := f.subs.Unsubscribe(n, s.Ship, s.Agent, s.Path); err != nil {
				return err
			}
		}
		return f.subs.Subscribe(n, f.cfg.Planet, proto.Agent, proto.FoldersPath)
	})
}

// loadMeta sets up what lives across runs: the key we serve our staged
// chunks under, and the flow we ask for the planet's keys on.
func (f *Follower) loadMeta() error {
	get := func(k string) (uint64, bool) {
		var v uint64
		err := f.db.QueryRowContext(bg, `SELECT v FROM drive_meta WHERE k = ?`, k).Scan(&v)
		return v, err == nil
	}
	put := func(k string, v uint64) error {
		return f.exec(`INSERT OR REPLACE INTO drive_meta(k, v) VALUES(?, ?)`, k, v)
	}
	var ok bool
	if f.upKid, ok = get("up-kid"); !ok {
		var key [32]byte
		rand.Read(key[:])
		kid, err := f.n.ShutAdd(proto.UpPrefix, key)
		if err != nil {
			return err
		}
		f.upKid = kid
		if err := put("up-kid", kid); err != nil {
			return err
		}
	}
	if f.keyBone, ok = get("key-bone"); !ok {
		bone, err := f.n.OpenFlow(f.cfg.Planet)
		if err != nil {
			return err
		}
		f.keyBone = bone
		return put("key-bone", bone)
	}
	return nil
}

func (f *Follower) saveFolder(fo *folder) error {
	var pend []byte
	if fo.pending != nil {
		pend, _ = json.Marshal(fo.pending)
	}
	return f.exec(`INSERT OR REPLACE INTO drive_folder(fid, name, epoch, head, synced, pending) VALUES(?,?,?,?,?,?)`,
		fo.fid, fo.name, fo.epoch, fo.head, fo.synced, pend)
}

func (f *Follower) dir(fo *folder) string { return filepath.Join(f.cfg.Root, safeName(fo.name)) }

func safeName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '_'
		}
		return r
	}, s)
	if s == "" || s == "." || s == ".." {
		return "_"
	}
	return s
}

// ----- down: facts, waves, chunks

func (f *Follower) onFact(n *mesa.Node, s mesa.Sub, mark string, noun mesa.Noun) {
	u, err := proto.ParseUpdate(noun)
	debugf("fact %s %v", mark, noun)
	if err != nil {
		f.cfg.Log("bad fact: %v", err)
		return
	}
	switch {
	case u.Hello != nil:
		for _, uf := range u.Hello.Folders {
			f.takeFolder(uf)
		}
		for _, r := range u.Hello.Results {
			f.takeResult(n, r)
		}
	case u.Folder != nil:
		f.takeFolder(u.Folder)
	case u.Head != nil:
		if fo, ok := f.folders[u.Head.Fid]; ok && u.Head.Aeon > fo.head {
			fo.head = u.Head.Aeon
			f.saveFolder(fo)
			f.pump(fo)
		}
	case u.Gone != "":
		// left or unshared: stop syncing it, keep the files
		if fo, ok := f.folders[u.Gone]; ok {
			f.cfg.Log("folder %q is no longer shared with us; keeping its files", fo.name)
			delete(f.folders, u.Gone)
			f.exec(`DELETE FROM drive_folder WHERE fid = ?`, u.Gone)
			f.exec(`DELETE FROM drive_file WHERE fid = ?`, u.Gone)
		}
	case u.Result != nil:
		f.takeResult(n, u.Result)
	}
}

func (f *Follower) takeFolder(uf *proto.Folder) {
	fo, ok := f.folders[uf.Fid]
	if !ok {
		fo = &folder{fid: uf.Fid}
		f.folders[uf.Fid] = fo
		f.cfg.Log("folder %q (%s)", uf.Name, uf.Fid)
	}
	if fo.epoch != uf.Epoch {
		fo.key, fo.keyAsked = nil, false
	}
	fo.name, fo.epoch, fo.head, fo.live = uf.Name, uf.Epoch, uf.Head, true
	if ro := uf.Role == "r"; ro != fo.readOnly {
		fo.readOnly = ro
		if ro {
			f.cfg.Log("%q is read-only for us: local changes to it won't upload", fo.name)
		}
	}
	os.MkdirAll(f.dir(fo), 0o755)
	f.saveFolder(fo)
	f.pump(fo)
	f.scanSoon(fo)
}

func (f *Follower) takeResult(n *mesa.Node, r *proto.Result) {
	if r.Who != n.Ship() {
		return
	}
	for _, fo := range f.folders {
		if fo.pending == nil || fo.pending.Cid != r.Cid {
			continue
		}
		if r.Err != "" {
			f.cfg.Log("commit to %q failed: %s", fo.name, r.Err)
			fo.pending = nil
		} else {
			f.cfg.Log("committed to %q at aeon %d", fo.name, r.Aeon)
			if r.Aeon > fo.head {
				fo.head = r.Aeon
			}
			// fo.pending stays until we apply our wave (applyWave
			// uses its stats), unless the commit changed nothing
			if r.Aeon <= fo.synced {
				fo.pending = nil
			}
		}
		f.exec(`DELETE FROM drive_staged WHERE cid = ?`, r.Cid)
		f.saveFolder(fo)
		f.pump(fo)
		f.scanSoon(fo)
	}
}

// withKey runs [fn] with the folder's coop key, asking the planet for it
// first if we don't have it.
func (f *Follower) withKey(fo *folder, fn func(k *shutKey)) {
	if fo.key != nil {
		fn(fo.key)
		return
	}
	if fo.keyAsked {
		return // pump runs again when it arrives
	}
	fo.keyAsked = true
	path := fmt.Sprintf("/g/x/1/drive//1/f/%s/%d", fo.fid, fo.epoch)
	if err := f.n.RequestKey(f.cfg.Planet, f.keyBone, path); err != nil {
		f.cfg.Log("key request: %v", err)
		fo.keyAsked = false
		return
	}
	f.keyQ = append(f.keyQ, fo.fid)
}

// Boon is the planet's answer to a key request.
func (f *Follower) Boon(n *mesa.Node, her mesa.Ship, bone uint64, payload mesa.Noun) {
	if bone != f.keyBone || len(f.keyQ) == 0 {
		return
	}
	fid := f.keyQ[0]
	f.keyQ = f.keyQ[1:]
	fo, ok := f.folders[fid]
	if !ok {
		return
	}
	fo.keyAsked = false
	kid, key, ok := mesa.ParseKey(payload)
	if !ok {
		f.cfg.Log("the planet refused the key to %q (unpaired?)", fo.name)
		return
	}
	fo.key = &shutKey{kid, key}
	f.pump(fo)
}

func (f *Follower) scry(k *shutKey, path string, fn func(mesa.Noun, bool)) {
	if _, busy := f.scries[path]; busy {
		f.scries[path] = fn
		return
	}
	f.scries[path] = fn
	if err := f.n.ScryShut(f.cfg.Planet, k.kid, k.key, path); err != nil {
		delete(f.scries, path)
		f.cfg.Log("scry %s: %v", path, err)
	}
}

// Scry is the answer to one of our reads.
func (f *Follower) Scry(n *mesa.Node, her mesa.Ship, path, mark string, noun mesa.Noun, ok bool) {
	fn, found := f.scries[path]
	if !found {
		return
	}
	delete(f.scries, path)
	fn(noun, ok)
}

// pump reads the next wave a folder lacks.
func (f *Follower) pump(fo *folder) {
	if fo.pulling || fo.synced >= fo.head {
		return
	}
	f.withKey(fo, func(k *shutKey) {
		fo.pulling = true
		aeon := fo.synced + 1
		f.scry(k, proto.WavePath(fo.fid, fo.epoch, aeon), func(noun mesa.Noun, ok bool) {
			w, err := proto.ParseWave(noun)
			if !ok || err != nil || w.Aeon != aeon {
				f.cfg.Log("wave %d of %q unreadable: %v", aeon, fo.name, err)
				fo.pulling = false
				return
			}
			f.fetchChunks(fo, k, w)
		})
	})
}

// cacheDir holds downloaded chunks until their wave is applied.
func (f *Follower) cacheDir() string { return filepath.Join(f.cfg.Root, ".drive", "cache") }

func (f *Follower) haveChunk(h chunk.Hash) bool {
	if _, err := os.Stat(filepath.Join(f.cacheDir(), h.String())); err == nil {
		return true
	}
	var one int
	return f.db.QueryRowContext(bg, `SELECT 1 FROM drive_staged WHERE hash = ?`, h[:]).Scan(&one) == nil
}

func (f *Follower) readChunk(h chunk.Hash) ([]byte, error) {
	if b, err := os.ReadFile(filepath.Join(f.cacheDir(), h.String())); err == nil {
		return b, nil
	}
	var b []byte
	err := f.db.QueryRowContext(bg, `SELECT data FROM drive_staged WHERE hash = ?`, h[:]).Scan(&b)
	return b, err
}

// fetchChunks reads every chunk the wave needs that we don't have, then
// applies it.
func (f *Follower) fetchChunks(fo *folder, k *shutKey, w *proto.Wave) {
	need := map[chunk.Hash]bool{}
	for _, op := range w.Ops {
		if op.New == nil || f.matchesDisk(fo, op.Path, op.New.Hash) {
			continue
		}
		for _, h := range op.New.Chunks {
			if !need[h] && !f.haveChunk(h) {
				need[h] = true
			}
		}
	}
	if len(need) == 0 {
		f.applyWave(fo, w)
		return
	}
	os.MkdirAll(f.cacheDir(), 0o755)
	fo.waitingChunks = len(need)
	failed := false
	for h := range need {
		h := h
		f.scry(k, proto.BlobPath(fo.fid, fo.epoch, h), func(noun mesa.Noun, ok bool) {
			data, err := chunkOf(noun, h)
			if !ok || err != nil {
				f.cfg.Log("chunk %s: %v", h, err)
				failed = true
			} else if err := os.WriteFile(filepath.Join(f.cacheDir(), h.String()), data, 0o600); err != nil {
				f.cfg.Log("cache: %v", err)
				failed = true
			}
			fo.waitingChunks--
			if fo.waitingChunks > 0 {
				return
			}
			if failed {
				fo.pulling = false
				f.n.After(5*time.Second, func() error { f.pump(fo); return nil })
				return
			}
			f.applyWave(fo, w)
		})
	}
}

// chunkOf checks a [len dat] chunk against its hash.
func chunkOf(noun mesa.Noun, h chunk.Hash) ([]byte, error) {
	if !noun.IsCell() {
		return nil, errors.New("not a chunk")
	}
	n, ok := noun.Head().Uint64()
	if !ok || n > chunk.Max {
		return nil, errors.New("bad length")
	}
	b := noun.Tail().Bytes()
	if uint64(len(b)) > n {
		return nil, errors.New("longer than its length")
	}
	data := make([]byte, n)
	copy(data, b)
	if chunk.Hash(sha256Sum(data)) != h {
		return nil, errors.New("hash mismatch")
	}
	return data, nil
}

// ----- up: serving our staged chunks

// Peek serves /up/<hex>, our staged chunks (only ever %shut-encrypted).
func (f *Follower) Peek(n *mesa.Node, her *mesa.Ship, path string) (string, mesa.Value, bool) {
	_, rest, ok := strings.Cut(path, "/drive//1/up/")
	if !ok {
		return "", nil, false
	}
	h, ok := chunk.ParseHash(rest)
	if !ok {
		return "", nil, false
	}
	var data []byte
	if f.db.QueryRowContext(bg, `SELECT data FROM drive_staged WHERE hash = ?`, h[:]).Scan(&data) != nil {
		return "", nil, false
	}
	return "noun", mesa.Tuple(uint64(len(data)), data), true
}

// ShutKey gives the key to our staged chunks to our planet only.
func (f *Follower) ShutKey(n *mesa.Node, her mesa.Ship, agent, spur string) uint64 {
	if her == f.cfg.Planet && agent == proto.Agent && strings.HasPrefix(spur, "/up/") {
		return f.upKid
	}
	return 0
}

// Status describes each folder, for `drive status`.
func (f *Follower) Status() []string {
	var out []string
	for _, fo := range f.folders {
		s := fmt.Sprintf("%-20s %s  aeon %d/%d", fo.name, fo.fid, fo.synced, fo.head)
		if fo.pending != nil {
			s += "  (committing)"
		}
		out = append(out, s)
	}
	return out
}

// randU64 is 63 random bits: SQLite integers are signed.
func randU64() uint64 {
	var b [8]byte
	rand.Read(b[:])
	return binary.LittleEndian.Uint64(b[:]) >> 1
}
