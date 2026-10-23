// Package chunk splits files into content-defined chunks (FastCDC) and
// hashes them, so an edit re-sends only the chunks around it and equal
// chunks anywhere are stored once.
package chunk

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
)

const (
	Min = 256 << 10
	Avg = 1 << 20
	Max = 4 << 20 // the planet refuses larger chunks
)

// Hash is a sha-256 digest.
type Hash [32]byte

func (h Hash) String() string { return hex.EncodeToString(h[:]) }

// ParseHash reads 64 hex digits.
func ParseHash(s string) (Hash, bool) {
	var h Hash
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return h, false
	}
	copy(h[:], b)
	return h, true
}

// File is a file's chunking: its whole hash, and its chunks in order.
type File struct {
	Hash   Hash
	Size   int64
	Chunks []Hash
}

var gear [256]uint64

func init() {
	// fixed for all time: changing it would re-chunk every file
	for i := range gear {
		d := sha256.Sum256([]byte{'d', 'r', 'v', byte(i)})
		gear[i] = binary.LittleEndian.Uint64(d[:8])
	}
}

// normalized chunking: harder to cut before Avg, easier after
const (
	maskS = uint64(1<<22-1) << 42 // 22 bits
	maskL = uint64(1<<18-1) << 46 // 18 bits
)

// cut is where the first chunk of [b] ends.
func cut(b []byte) int {
	n := len(b)
	if n <= Min {
		return n
	}
	if n > Max {
		n = Max
	}
	norm := min(Avg, n)
	var fp uint64
	i := Min
	for ; i < norm; i++ {
		fp = fp<<1 + gear[b[i]]
		if fp&maskS == 0 {
			return i
		}
	}
	for ; i < n; i++ {
		fp = fp<<1 + gear[b[i]]
		if fp&maskL == 0 {
			return i
		}
	}
	return n
}

// Split reads [r] to the end and calls [fn] with each chunk and its hash
// (the slice is reused after fn returns).
func Split(r io.Reader, fn func(h Hash, data []byte) error) (*File, error) {
	f := &File{}
	whole := sha256.New()
	buf := make([]byte, 0, 2*Max)
	br := bufio.NewReaderSize(r, 1<<20)
	eof := false
	for {
		for !eof && len(buf) < Max {
			n, err := br.Read(buf[len(buf):cap(buf)])
			buf = buf[:len(buf)+n]
			if err == io.EOF {
				eof = true
			} else if err != nil {
				return nil, err
			}
		}
		if len(buf) == 0 {
			break
		}
		c := buf
		if !eof || len(buf) > Max {
			c = buf[:cut(buf)]
		}
		h := Hash(sha256.Sum256(c))
		whole.Write(c)
		f.Size += int64(len(c))
		f.Chunks = append(f.Chunks, h)
		if fn != nil {
			if err := fn(h, c); err != nil {
				return nil, err
			}
		}
		buf = buf[:copy(buf, buf[len(c):])]
	}
	copy(f.Hash[:], whole.Sum(nil))
	return f, nil
}

// SplitFile is Split on the file at [path].
func SplitFile(path string, fn func(h Hash, data []byte) error) (*File, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return Split(fd, fn)
}
