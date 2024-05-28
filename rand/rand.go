/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package rand

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
	"sync"
	"sync/atomic"
)

// Void is an empty type.
type Void any

var (
	pool sync.Pool
	ops  int32
)

func init() {
	// Prepopulate the pool
	for i := 0; i < 16; i++ {
		go pool.Put(New())
	}
	pool.New = func() interface{} {
		return New()
	}
}

// New creates a new math random generator seeded by a crypto random number.
func New() *mathrand.Rand {
	// Generate crypto random 64-bit seed
	b := make([]byte, 8)
	cryptorand.Read(b)
	n := binary.LittleEndian.Uint64(b)
	// Create a math random generator seeded with the seed
	s := mathrand.NewSource(int64(n))
	return mathrand.New(s)
}

// Read generates len(p) random bytes and writes them into p. It
// always returns len(p) and a nil error.
func Read(p []byte) (n int, err error) {
	r := poolGet()
	n, err = r.Read(p)
	poolReturn(r)
	return n, err
}

// AlphaNum64 generates a random string of the specified length.
// The string will include only alphanumeric characters a-z, A-Z, 0-9.
func AlphaNum64(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01"
	bytes := make([]byte, length)
	Read(bytes)
	for i, b := range bytes {
		bytes[i] = letters[b&0x3F]
	}
	return string(bytes)
}

// AlphaNum32 generates a random string of the specified length.
// The string will include only uppercase letters and numbers A-V, 0-9.
func AlphaNum32(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUV"
	bytes := make([]byte, length)
	Read(bytes)
	for i, b := range bytes {
		bytes[i] = letters[b&0x1F]
	}
	return string(bytes)
}

// Intn generates a random number in the range [0,n).
func Intn(n int) int {
	if n == 1 {
		return 0
	}
	r := poolGet()
	m := r.Intn(n)
	poolReturn(r)
	return m
}

// poolGet returns a crypto-seeded math random generator.
func poolGet() *mathrand.Rand {
	return pool.Get().(*mathrand.Rand)
}

// poolReturn returns the math random generator to the pool, reseeding it every so often.
func poolReturn(r *mathrand.Rand) {
	// Reseed only once every 4096 operations
	o := atomic.AddInt32(&ops, 1)
	if o&0xFFF != 0 {
		pool.Put(r)
		return
	}
	// Do not block while performing crypto operation
	go func() {
		// Generate crypto random 64-bit seed
		b := make([]byte, 8)
		cryptorand.Read(b)
		n := binary.LittleEndian.Uint64(b)
		// Reseed the math random generator
		r.Seed(int64(n))
		pool.Put(r)
	}()
}
