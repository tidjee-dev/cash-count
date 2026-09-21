package store

import (
	"crypto/rand"
	"fmt"
)

const idAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// NewID generates an opaque text ID with a type prefix, e.g. cnt_K7f3Qx9mZ2aB4.
// 72 bits of crypto randomness make collisions practically impossible.
func NewID(prefix string) (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("new id: %w", err)
	}
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = idAlphabet[int(v)%len(idAlphabet)]
	}
	return prefix + "_" + string(out), nil
}

// MustID is NewID for contexts where rand failure is unrecoverable.
func MustID(prefix string) string {
	id, err := NewID(prefix)
	if err != nil {
		panic(err)
	}
	return id
}
