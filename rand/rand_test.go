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
	cryptorand "crypto/rand"
	mathrand "math/rand"
	"regexp"
	"testing"

	"github.com/microbus-io/testarossa"
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
		testarossa.StrLen(t, an64, 15)
		match := re.MatchString(an64)
		testarossa.True(t, match)
	}
}

func TestRand_AlphaNum32(t *testing.T) {
	t.Parallel()

	re := regexp.MustCompile(`^[A-Z0-9]+$`)
	for i := 0; i < 1024; i++ {
		an32 := AlphaNum32(15)
		testarossa.StrLen(t, an32, 15)
		match := re.MatchString(an32)
		testarossa.True(t, match)
	}
}

func TestRand_Intn(t *testing.T) {
	t.Parallel()

	for i := 0; i < 4096; i++ {
		n := Intn(100)
		testarossa.True(t, n >= 0)
		testarossa.True(t, n < 100)
	}
}
