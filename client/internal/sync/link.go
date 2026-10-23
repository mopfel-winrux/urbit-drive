package sync

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mopfel-winrux/urbit-drive/client/internal/proto"
)

// Share links from this computer: we ask our planet for a link to its
// copy of a file or folder, and it answers on our subscription with the
// token to use (a plain link it already has, or the one we offered).

type linkAnswer struct{ token, err string }

// ErrNoAnswer: the planet didn't answer in time.
var ErrNoAnswer = errors.New("your ship didn't answer; check that it's running, then try again")

// AskLink gets a share link token for the file or folder at [local] (a
// path on this computer, inside one of our folders). Call it from any
// goroutine but the node's; it waits up to 20 seconds.
func (f *Follower) AskLink(local string) (string, error) {
	abs, err := filepath.Abs(local)
	if err != nil {
		return "", err
	}
	var b [16]byte
	rand.Read(b[:])
	token := hex.EncodeToString(b[:])
	answer := make(chan linkAnswer, 1)
	err = f.n.Do(func() error {
		fo, rel, err := f.locate(abs)
		if err != nil {
			return err
		}
		if fo.readOnly {
			return fmt.Errorf("%q is shared with you; only its owner can make links", fo.name)
		}
		if f.asking == nil {
			f.asking = map[string]chan linkAnswer{}
		}
		f.asking[token] = answer
		seq, err := f.pokes.Poke(f.n, f.cfg.Planet, proto.Agent, proto.ActionMark, proto.LinkAsk(token, fo.fid, rel))
		if err == nil {
			if f.askSeq == nil {
				f.askSeq = map[uint64]string{}
			}
			f.askSeq[seq] = token
		}
		return err
	})
	if err != nil {
		return "", err
	}
	select {
	case a := <-answer:
		if a.err != "" {
			return "", errors.New(a.err)
		}
		return a.token, nil
	case <-time.After(20 * time.Second):
		f.n.Do(func() error { delete(f.asking, token); return nil })
		return "", ErrNoAnswer
	}
}

// locate: which of our folders [abs] is in, and its path there ("" for
// the folder itself). Loop only.
func (f *Follower) locate(abs string) (*folder, string, error) {
	for _, fo := range f.folders {
		dir := f.dir(fo)
		rel, err := filepath.Rel(dir, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if rel == "." {
			return fo, "", nil
		}
		rel = filepath.ToSlash(rel)
		if fo.excluded(rel) {
			return nil, "", fmt.Errorf("%s isn't synced on this computer", rel)
		}
		fi, err := os.Stat(abs)
		if err != nil {
			return nil, "", err
		}
		if !fi.IsDir() && !f.getFile(fo.fid, rel).found {
			return nil, "", errors.New("that file hasn't synced to your ship yet; wait a moment and try again")
		}
		return fo, rel, nil
	}
	return nil, "", fmt.Errorf("%s isn't in a Drive folder", abs)
}

// linkPokeDone: the planet acked or nacked a link-ask. A nack (an older
// %drive that doesn't know %link-ask, say) answers it at once.
func (f *Follower) linkPokeDone(seq uint64, nack bool) {
	token, ok := f.askSeq[seq]
	if !ok {
		return
	}
	delete(f.askSeq, seq)
	if c, ok := f.asking[token]; ok && nack {
		delete(f.asking, token)
		c <- linkAnswer{err: "your ship refused; update Drive on it and try again"}
	}
}
