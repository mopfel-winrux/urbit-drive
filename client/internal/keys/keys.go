// Package keys makes a moon's keys on the device, so the private half
// never leaves it: the planet is told only the public key (pass).
package keys

import (
	"crypto/rand"
	"encoding/hex"

	mesa "github.com/mopfel-winrux/libmesa/bindings/go"
)

// Moon is a moon's identity: its keyfile and public key.
type Moon struct {
	Ship    mesa.Ship
	Keyfile string   // [%2 ~] feed, as |moon prints
	Pass    [65]byte // 'b' ++ sgn ++ cry, little-endian as an atom
}

// New makes keys for [who] at life 1, rift 0: a suite %b ring of 'B'
// and 64 random bytes.
func New(who mesa.Ship) (*Moon, error) {
	var ring [65]byte
	ring[0] = 'B'
	if _, err := rand.Read(ring[1:]); err != nil {
		return nil, err
	}
	id := mesa.NewIdentity(who, 1, 0, ring)
	return &Moon{Ship: who, Keyfile: id.Keyfile(), Pass: id.Pass()}, nil
}

// PassHex is the public key as hoon's @ux digits (most significant
// first), without dots or 0x.
func (m *Moon) PassHex() string {
	var be [65]byte
	for i, b := range m.Pass {
		be[64-i] = b
	}
	return hex.EncodeToString(be[:])
}

// RandomMoon is a random moon of [parent]: its low 32 bits, and 32
// random nonzero bits above them.
func RandomMoon(parent mesa.Ship) (mesa.Ship, error) {
	var b [4]byte
	for {
		if _, err := rand.Read(b[:]); err != nil {
			return mesa.Ship{}, err
		}
		hi := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24
		if hi != 0 {
			return mesa.Ship{Lo: parent.Lo&0xffffffff | hi<<32}, nil
		}
	}
}
