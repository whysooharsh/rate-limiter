package limiter

import "testing"

// Benchmarks for TokenBucket.Allow.

func BenchmarkAllow(b *testing.B) {

	bucket := NewTokenBucket(1000000, 1000000)

	for i := 0; i < b.N; i++ {
		bucket.Allow()
	}

}

func BenchmarkAllowParallel(b *testing.B) {
	bucket := NewTokenBucket(1000000, 1000000)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bucket.Allow()
		}
	})
}
