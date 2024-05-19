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
	for i := 0; i < 32; i++ {
		pool.Put(New())
	}
	pool.New = func() interface{} {
		return New()
	}
}

// New creates a new math random generator seeded by a crypto random number
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
	r := pool.Get().(*mathrand.Rand)
	reseed(r)
	n, err = r.Read(p)
	pool.Put(r)
	return n, err
}

// AlphaNum64 generates a random string of the specified length.
// The string will include only alphanumeric characters a-z, A-Z, 0-9
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
// The string will include only uppercase letters and numbers A-V, 0-9
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
	r := pool.Get().(*mathrand.Rand)
	reseed(r)
	m := r.Intn(n)
	pool.Put(r)
	return m
}

// reseed the math random generator that was pulled from the pool once every 256 operations
func reseed(r *mathrand.Rand) {
	// Perform once every 4096 operations
	o := atomic.AddInt32(&ops, 1)
	if o&0xFFF != 0 {
		return
	}

	// Generate crypto random 64-bit seed
	b := make([]byte, 8)
	cryptorand.Read(b)
	n := binary.LittleEndian.Uint64(b)

	// Reseed the math random generator
	r.Seed(int64(n))
}
