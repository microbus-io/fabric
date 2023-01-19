/*
Copyright 2023 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rand

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
	"sync"
	"sync/atomic"
)

var (
	pool sync.Pool
	ops  int32
)

func init() {
	// Prepopulate the pool
	for i := 0; i < 16; i++ {
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

// AlphaNum64 generates a random string of the specified length.
// The string will include only alphanumeric characters a-z, A-Z, 0-9
func AlphaNum64(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01"
	bytes := make([]byte, length)
	r := pool.Get().(*mathrand.Rand)
	reseed(r)
	_, _ = r.Read(bytes)
	for i, b := range bytes {
		bytes[i] = letters[b&0x3F]
	}
	pool.Put(r)
	return string(bytes)
}

// AlphaNum32 generates a random string of the specified length.
// The string will include only uppercase letters and numbers A-V, 0-9
func AlphaNum32(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUV"
	bytes := make([]byte, length)
	r := pool.Get().(*mathrand.Rand)
	reseed(r)
	_, _ = r.Read(bytes)
	for i, b := range bytes {
		bytes[i] = letters[b&0x1F]
	}
	pool.Put(r)
	return string(bytes)
}

// Intn generates a random number in the range [0,n).
func Intn(n int) int {
	r := pool.Get().(*mathrand.Rand)
	reseed(r)
	m := r.Intn(n)
	pool.Put(r)
	return m
}

// reseed the math random generator that was pulled from the pool once every 256 operations
func reseed(r *mathrand.Rand) {
	// Perform once every 256 operations
	o := atomic.AddInt32(&ops, 1)
	if o&0xFF != 0 {
		return
	}

	// Generate crypto random 64-bit seed
	b := make([]byte, 8)
	cryptorand.Read(b)
	n := binary.LittleEndian.Uint64(b)

	// Reseed the math random generator
	r.Seed(int64(n))
}
