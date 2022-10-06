package rand

import (
	mathrand "math/rand"
	"testing"
)

func BenchmarkRand_AlphaNum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AlphaNum64(16)
	}
	// On 2021 MacBook Pro M1 15": 103 ns/op
}

func BenchmarkRand_Intn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Intn(1000000)
	}
	// On 2021 MacBook Pro M1 15": 54 ns/op
}

func BenchmarkRand_MathIntn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mathrand.Intn(1000000)
	}
	// On 2021 MacBook Pro M1 15": 13.5 ns/op
}
