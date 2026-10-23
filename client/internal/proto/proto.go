// Package proto is %drive's wire format: the nouns of sur/drive.hoon,
// and the scry paths the planet serves and the device serves.
package proto

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	mesa "github.com/mopfel-winrux/libmesa/bindings/go"

	"github.com/mopfel-winrux/urbit-drive/client/internal/chunk"
)

const (
	Agent       = "drive"
	ActionMark  = "drive-action"
	UpdateMark  = "drive-update"
	FoldersPath = "/v1/folders"
	// UpPrefix is where a device serves the chunks of its commits (%shut).
	UpPrefix = "/g/x/0/drive//1/up"
)

// WavePath and BlobPath are a folder's bindings on the planet, read
// with the folder's coop key.
func WavePath(fid string, epoch, aeon uint64) string {
	return fmt.Sprintf("/g/x/1/drive//1/f/%s/%d/wave/%d", fid, epoch, aeon)
}

func BlobPath(fid string, epoch uint64, h chunk.Hash) string {
	return fmt.Sprintf("/g/x/1/drive//1/f/%s/%d/blob/%s", fid, epoch, h)
}

// HashNoun is a hash as the planet's @ux: the digest, big-endian.
func HashNoun(h chunk.Hash) *big.Int { return new(big.Int).SetBytes(h[:]) }

func hashOf(n mesa.Noun) (chunk.Hash, error) {
	var h chunk.Hash
	b := n.Big()
	if !n.IsAtom() || b.BitLen() > 256 {
		return h, errors.New("bad hash")
	}
	b.FillBytes(h[:])
	return h, nil
}

// Entry is a file at one revision.
type Entry struct {
	Hash   chunk.Hash
	Size   uint64
	Mtime  time.Time
	Exec   bool
	Chunks []chunk.Hash
}

func (e *Entry) Noun() mesa.Value {
	cs := make([]mesa.Value, len(e.Chunks))
	for i, c := range e.Chunks {
		cs[i] = HashNoun(c)
	}
	return mesa.Tuple(HashNoun(e.Hash), e.Size, mesa.Da(e.Mtime), e.Exec, mesa.List(cs...))
}

func entryOf(n mesa.Noun) (*Entry, error) {
	var e Entry
	var err error
	if !n.IsCell() {
		return nil, errors.New("bad entry")
	}
	if e.Hash, err = hashOf(n.Head()); err != nil {
		return nil, err
	}
	n = n.Tail()
	size, ok := n.Head().Uint64()
	if !ok {
		return nil, errors.New("bad size")
	}
	e.Size = size
	n = n.Tail()
	e.Mtime = mesa.Time(n.Head())
	n = n.Tail()
	x, _ := n.Head().Uint64()
	e.Exec = x == 0
	for _, c := range n.Tail().List() {
		h, err := hashOf(c)
		if err != nil {
			return nil, err
		}
		e.Chunks = append(e.Chunks, h)
	}
	return &e, nil
}

// Op is a file op in a wave: New nil deletes.
type Op struct {
	Path string
	New  *Entry
}

// Wave is one committed revision of a folder.
type Wave struct {
	Aeon   uint64
	Author mesa.Ship
	When   time.Time
	Ops    []Op
}

func atomShip(n mesa.Noun) mesa.Ship {
	b := n.Bytes() // little-endian
	var s mesa.Ship
	for i, c := range b {
		if i < 8 {
			s.Lo |= uint64(c) << (8 * i)
		} else if i < 16 {
			s.Hi |= uint64(c) << (8 * (i - 8))
		}
	}
	return s
}

// ParseWave reads a wave scried from the planet.
func ParseWave(n mesa.Noun) (*Wave, error) {
	if !n.IsCell() {
		return nil, errors.New("bad wave")
	}
	w := &Wave{}
	w.Aeon, _ = n.Head().Uint64()
	n = n.Tail()
	w.Author = atomShip(n.Head())
	n = n.Tail()
	w.When = mesa.Time(n.Head())
	for _, o := range n.Tail().List() {
		op := Op{Path: o.Head().Text()}
		if u := o.Tail(); u.IsCell() {
			e, err := entryOf(u.Tail())
			if err != nil {
				return nil, err
			}
			op.New = e
		}
		w.Ops = append(w.Ops, op)
	}
	return w, nil
}

// Cop is an op in a commit, with the hash we last saw at the path.
type Cop struct {
	Path string
	Base *chunk.Hash
	New  *Entry
}

// Commit is a %commit poke's payload.
type Commit struct {
	Cid  uint64
	Fid  string
	Base uint64
	Ops  []Cop
}

// Action is the %drive-action poke for the commit.
func (c *Commit) Action() mesa.Value {
	ops := make([]mesa.Value, len(c.Ops))
	for i, o := range c.Ops {
		var base, nw mesa.Value
		if o.Base != nil {
			base = mesa.Some(HashNoun(*o.Base))
		}
		if o.New != nil {
			nw = mesa.Some(o.New.Noun())
		}
		ops[i] = mesa.Tuple(o.Path, base, nw)
	}
	return mesa.Tuple("commit", c.Cid, c.Fid, c.Base, mesa.List(ops...))
}

// Update is a fact on /v1/folders; exactly one of its parts is set.
type Update struct {
	Folder *Folder
	Head   *struct {
		Fid  string
		Aeon uint64
	}
	Result *Result
}

type Folder struct {
	Fid   string
	Name  string
	Host  mesa.Ship
	Epoch uint64
	Head  uint64
}

type Result struct {
	Who  mesa.Ship
	Cid  uint64
	Aeon uint64 // when Err is ""
	Err  string
}

func u64(n mesa.Noun) uint64 { v, _ := n.Uint64(); return v }

// ParseUpdate reads a %drive-update fact.
func ParseUpdate(n mesa.Noun) (*Update, error) {
	if !n.IsCell() {
		return nil, errors.New("bad update")
	}
	u := &Update{}
	tag, t := n.Head().Text(), n.Tail()
	switch tag {
	case "folder":
		f := &Folder{Fid: t.Head().Text()}
		t = t.Tail()
		f.Name = t.Head().Text()
		t = t.Tail()
		f.Host = atomShip(t.Head())
		t = t.Tail()
		f.Epoch, f.Head = u64(t.Head()), u64(t.Tail())
		u.Folder = f
	case "head":
		u.Head = &struct {
			Fid  string
			Aeon uint64
		}{t.Head().Text(), u64(t.Tail())}
	case "result":
		r := &Result{Who: atomShip(t.Head())}
		t = t.Tail()
		r.Cid = u64(t.Head())
		res := t.Tail()
		if u64(res.Head()) == 0 {
			r.Aeon = u64(res.Tail())
		} else {
			r.Err = res.Tail().Text()
			if r.Err == "" {
				r.Err = "rejected"
			}
		}
		u.Result = r
	default:
		return nil, fmt.Errorf("unknown update %%%s", tag)
	}
	return u, nil
}

// Dec is a number as it appears in the planet's paths (no dots).
func Dec(n uint64) string { return strconv.FormatUint(n, 10) }
