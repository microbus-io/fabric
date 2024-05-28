/*
Copyright (c) 2023-2024 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package rand

import (
	cryptorand "crypto/rand"
	mathrand "math/rand"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkRand_AlphaNum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AlphaNum64(16)
	}
	// On 2021 MacBook Pro M1 16": 69 ns/op
}

func BenchmarkRand_Intn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Intn(1000000)
	}
	// On 2021 MacBook Pro M1 16": 14.28 ns/op
}

func BenchmarkRand_MathIntn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mathrand.Intn(1000000)
	}
	// On 2021 MacBook Pro M1 16": 7.78 ns/op
}

func BenchmarkRand_CryptoRead(b *testing.B) {
	var buf [8]byte
	for i := 0; i < b.N; i++ {
		cryptorand.Read(buf[:])
	}
	// On 2021 MacBook Pro M1 16": 326 ns/op
}

func BenchmarkRand_New(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New()
	}
	// On 2021 MacBook Pro M1 16": 11368 ns/op
}

func TestRand_AlphaNum64(t *testing.T) {
	t.Parallel()

	re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	for i := 0; i < 1024; i++ {
		an64 := AlphaNum64(15)
		assert.Len(t, an64, 15)
		match := re.MatchString(an64)
		assert.True(t, match)
	}
}

func TestRand_AlphaNum32(t *testing.T) {
	t.Parallel()

	re := regexp.MustCompile(`^[A-Z0-9]+$`)
	for i := 0; i < 1024; i++ {
		an32 := AlphaNum32(15)
		assert.Len(t, an32, 15)
		match := re.MatchString(an32)
		assert.True(t, match)
	}
}

func TestRand_Intn(t *testing.T) {
	t.Parallel()

	for i := 0; i < 4096; i++ {
		n := Intn(100)
		assert.True(t, n >= 0)
		assert.True(t, n < 100)
	}
}
