package chunk

import (
	"bytes"
	"crypto/sha256"
	"math/rand/v2"
	"testing"
)

func data(n int, seed uint64) []byte {
	r := rand.New(rand.NewPCG(seed, seed))
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.Uint32())
	}
	return b
}

func split(t *testing.T, b []byte) (*File, [][]byte) {
	var parts [][]byte
	f, err := Split(bytes.NewReader(b), func(h Hash, c []byte) error {
		if Hash(sha256.Sum256(c)) != h {
			t.Fatal("chunk hash mismatch")
		}
		parts = append(parts, append([]byte(nil), c...))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return f, parts
}

func TestRoundTrip(t *testing.T) {
	for _, n := range []int{0, 1, Min, Min + 1, Avg, Max, Max + 1, 9*Avg + 12345} {
		b := data(n, uint64(n))
		f, parts := split(t, b)
		if got := bytes.Join(parts, nil); !bytes.Equal(got, b) {
			t.Fatalf("n=%d: chunks don't reassemble", n)
		}
		if f.Hash != sha256.Sum256(b) || f.Size != int64(n) {
			t.Fatalf("n=%d: whole hash or size wrong", n)
		}
		for i, p := range parts {
			if len(p) > Max || (len(p) < Min && i != len(parts)-1) {
				t.Fatalf("n=%d: chunk %d is %d bytes", n, i, len(p))
			}
		}
	}
}

// An insertion near the start changes only the chunks around it.
func TestLocality(t *testing.T) {
	b := data(16*Avg, 7)
	f1, _ := split(t, b)
	ed := append(append(append([]byte(nil), b[:100]...), []byte("inserted")...), b[100:]...)
	f2, _ := split(t, ed)
	old := map[Hash]bool{}
	for _, h := range f1.Chunks {
		old[h] = true
	}
	fresh := 0
	for _, h := range f2.Chunks {
		if !old[h] {
			fresh++
		}
	}
	if fresh > 2 {
		t.Fatalf("%d of %d chunks changed after a small insertion", fresh, len(f2.Chunks))
	}
}
