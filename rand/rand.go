/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

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
	"math/rand/v2"
)

// Void is an empty type.
type Void any

// Read generates random bytes and writes them into the buffer.
// It always returns the length of the buffer and a nil error.
func Read(bytes []byte) (n int, err error) {
	var x uint64
	length := len(bytes)
	for i := 0; i < length; i++ {
		if i%8 == 0 {
			x = rand.Uint64()
		} else {
			x = x >> 8
		}
		bytes[i] = byte(x & 0xFF)
	}
	return length, err
}

// AlphaNum64 generates a random string of the specified length.
// The string will include only alphanumeric characters a-z, A-Z, 0-9.
func AlphaNum64(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01"
	bytes := make([]byte, length)
	var x uint64
	for i := 0; i < length; i++ {
		if i%8 == 0 {
			x = rand.Uint64()
		} else {
			x = x >> 8
		}
		bytes[i] = letters[x&0x3F]
	}
	return string(bytes)
}

// AlphaNum32 generates a random string of the specified length.
// The string will include only uppercase letters and numbers A-V, 0-9.
func AlphaNum32(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUV"
	bytes := make([]byte, length)
	var x uint64
	for i := 0; i < length; i++ {
		if i%8 == 0 {
			x = rand.Uint64()
		} else {
			x = x >> 8
		}
		bytes[i] = letters[x&0x1F]
	}
	return string(bytes)
}

// IntN returns, as an int, a pseudo-random number in the half-open interval [0,n)
// from the default Source.
// It panics if n <= 0.
func IntN(n int) int {
	return rand.IntN(n)
}
