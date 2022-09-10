package rand

import (
	mathrand "math/rand"
	"testing"
)

func BenchmarkAlphaNum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AlphaNum64(16)
	}
	// Took 114 ns/op
}

func BenchmarkIntn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Intn(1000000)
	}
	// Took 64 ns/op
}

func BenchmarkMathIntn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mathrand.Intn(1000000)
	}
	// Took 13.5 ns/op
}
