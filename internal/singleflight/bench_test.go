package singleflight

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkSingleflightBasic tests basic singleflight performance
func BenchmarkSingleflightBasic(b *testing.B) {
	g := &Group[string, int]{}
	
	fn := func() (int, error) {
		return 42, nil
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		g.Do("key", fn)
	}
}

// BenchmarkSingleflightConcurrent tests concurrent singleflight performance
func BenchmarkSingleflightConcurrent(b *testing.B) {
	g := &Group[string, int]{}
	
	callCount := int64(0)
	fn := func() (int, error) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(time.Microsecond) // Simulate some work
		return 42, nil
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Do("same-key", fn)
		}
	})
	
	b.Logf("Total benchmark iterations: %d", b.N)
	b.Logf("Actual function calls: %d", atomic.LoadInt64(&callCount))
	b.Logf("Deduplication effectiveness: %.2f%%", 
		(1.0 - float64(atomic.LoadInt64(&callCount))/float64(b.N)) * 100)
}

// BenchmarkSingleflightDifferentKeys tests performance with different keys
func BenchmarkSingleflightDifferentKeys(b *testing.B) {
	g := &Group[string, int]{}
	
	fn := func() (int, error) {
		return 42, nil
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		g.Do(string(rune(i%100)), fn)
	}
}

// BenchmarkSingleflightWithContext tests context performance
func BenchmarkSingleflightWithContext(b *testing.B) {
	g := &Group[string, int]{}
	
	fn := func() (int, error) {
		return 42, nil
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		g.DoContext(context.Background(), "key", fn)
	}
}

// BenchmarkSingleflightDoChan tests channel-based interface
func BenchmarkSingleflightDoChan(b *testing.B) {
	g := &Group[string, int]{}
	
	fn := func() (int, error) {
		return 42, nil
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		ch := g.DoChan("key", fn)
		<-ch
	}
}

// BenchmarkSingleflightMemoryUsage tests memory allocation patterns
func BenchmarkSingleflightMemoryUsage(b *testing.B) {
	g := &Group[string, int]{}
	
	fn := func() (int, error) {
		return 42, nil
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Mix of same and different keys to test memory allocation
		key := string(rune(i % 10))
		g.Do(key, fn)
		if i%10 == 9 {
			// Forget some keys to test cleanup
			g.Forget(string(rune(i % 10)))
		}
	}
}