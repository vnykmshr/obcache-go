package obcache

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// Helper functions for benchmarking

// expensiveComputation simulates an expensive operation
func expensiveComputation(n int) int {
	// Simulate CPU-bound work
	result := 0
	for i := 0; i < 1000; i++ {
		result += i * n
	}
	return result
}

// expensiveIOOperation simulates an I/O bound operation
func expensiveIOOperation(n int) int {
	// Simulate I/O delay (database, API call, etc.)
	time.Sleep(time.Millisecond)
	return n * 2
}

// expensiveComputationWithError simulates an expensive operation that can fail
func expensiveComputationWithError(n int) (int, error) {
	if n < 0 {
		return 0, fmt.Errorf("negative number: %d", n)
	}
	return expensiveComputation(n), nil
}

// expensiveStringOperation simulates expensive string processing
func expensiveStringOperation(s string) string {
	// Simulate string processing work
	result := ""
	for i := 0; i < 100; i++ {
		result += fmt.Sprintf("%s-%d", s, i)
	}
	return result
}

// multiParamFunction simulates a function with multiple parameters
func multiParamFunction(a int, b string, c bool) string {
	time.Sleep(time.Microsecond) // Simulate some work
	return fmt.Sprintf("result-%d-%s-%t", a, b, c)
}

// Benchmark: Basic Cache Operations

func BenchmarkCacheSet(b *testing.B) {
	cache, _ := New(NewDefaultConfig())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, i, time.Hour)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache, _ := New(NewDefaultConfig())

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, i, time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkCacheGetMiss(b *testing.B) {
	cache, _ := New(NewDefaultConfig())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("missing-key-%d", i)
		cache.Get(key)
	}
}

// Benchmark: Function Wrapping vs Direct Calls

func BenchmarkDirectFunction(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		expensiveComputation(i % 100)
	}
}

func BenchmarkWrappedFunctionFirstCall(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Each call uses a different key, so no cache hits
		wrappedFunc(i)
	}
}

func BenchmarkWrappedFunctionCached(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation)

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		wrappedFunc(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// All calls will hit cache
		wrappedFunc(i % 100)
	}
}

func BenchmarkWrappedFunctionMixed(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 80% cache hits, 20% misses
		if i%5 == 0 {
			wrappedFunc(i) // Cache miss
		} else {
			wrappedFunc(i % 100) // Likely cache hit
		}
	}
}

// Benchmark: Function with Error Returns

func BenchmarkWrappedFunctionWithError(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputationWithError)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i % 100)
	}
}

func BenchmarkWrappedFunctionWithErrorCached(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputationWithError)

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		wrappedFunc(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i % 100)
	}
}

// Benchmark: String Operations

func BenchmarkDirectStringFunction(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		expensiveStringOperation(fmt.Sprintf("input-%d", i%100))
	}
}

func BenchmarkWrappedStringFunction(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveStringOperation)

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		wrappedFunc(fmt.Sprintf("input-%d", i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(fmt.Sprintf("input-%d", i%100))
	}
}

// Benchmark: Multi-parameter Functions

func BenchmarkDirectMultiParamFunction(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		multiParamFunction(i%100, fmt.Sprintf("str-%d", i%10), i%2 == 0)
	}
}

func BenchmarkWrappedMultiParamFunction(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, multiParamFunction)

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			wrappedFunc(i, fmt.Sprintf("str-%d", j), i%2 == 0)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i%100, fmt.Sprintf("str-%d", i%10), i%2 == 0)
	}
}

// Benchmark: Concurrent Access

func BenchmarkConcurrentCacheGet(b *testing.B) {
	cache, _ := New(NewDefaultConfig())

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), i, time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			cache.Get(key)
			i++
		}
	})
}

func BenchmarkConcurrentCacheSet(b *testing.B) {
	cache, _ := New(NewDefaultConfig())

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			cache.Set(key, i, time.Hour)
			i++
		}
	})
}

func BenchmarkConcurrentWrappedFunction(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation)

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		wrappedFunc(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			wrappedFunc(i % 100)
			i++
		}
	})
}

// Benchmark: Singleflight Effectiveness

func BenchmarkSingleflightBenefit(b *testing.B) {
	cache, _ := New(NewDefaultConfig())

	// Function that takes some time and counts calls
	callCount := int64(0)
	slowFunc := func(n int) int {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(time.Microsecond * 100) // Simulate slow operation
		return n * 2
	}

	wrappedFunc := Wrap(cache, slowFunc)

	b.ResetTimer()

	// Launch multiple goroutines calling the same function concurrently
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			wrappedFunc(42) // All goroutines call with same argument
		}
	})

	// The number of actual function calls should be much less than b.N
	// due to singleflight deduplication
	b.Logf("Total benchmark iterations: %d", b.N)
	b.Logf("Actual function calls: %d", atomic.LoadInt64(&callCount))
	b.Logf("Singleflight effectiveness: %.2f%%",
		(1.0-float64(atomic.LoadInt64(&callCount))/float64(b.N))*100)
}

// Benchmark: Different TTL Strategies

func BenchmarkShortTTL(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation, WithTTL(time.Millisecond))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i % 10)
		// Some entries will expire between calls
	}
}

func BenchmarkLongTTL(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation, WithTTL(time.Hour))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i % 10)
	}
}

// Benchmark: Cache Size Impact

func BenchmarkSmallCache(b *testing.B) {
	config := NewDefaultConfig().WithMaxEntries(10)
	cache, _ := New(config)
	wrappedFunc := Wrap(cache, expensiveComputation)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i % 100) // More keys than cache size
	}
}

func BenchmarkLargeCache(b *testing.B) {
	config := NewDefaultConfig().WithMaxEntries(10000)
	cache, _ := New(config)
	wrappedFunc := Wrap(cache, expensiveComputation)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i % 100) // Much fewer keys than cache size
	}
}

// Benchmark: Key Generation Impact

func BenchmarkWithDefaultKeyFunc(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation,
		WithKeyFunc(DefaultKeyFunc))

	// Pre-populate
	for i := 0; i < 100; i++ {
		wrappedFunc(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i % 100)
	}
}

func BenchmarkWithSimpleKeyFunc(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation,
		WithKeyFunc(SimpleKeyFunc))

	// Pre-populate
	for i := 0; i < 100; i++ {
		wrappedFunc(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wrappedFunc(i % 100)
	}
}

// Benchmark: Memory Allocation Patterns

func BenchmarkCacheMemoryUsage(b *testing.B) {
	cache, _ := New(NewDefaultConfig())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Set and get operations
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Set(key, i, time.Hour)
		cache.Get(key)
	}
}

// Benchmark: Real-world Simulation

func BenchmarkWebServerSimulation(b *testing.B) {
	// Simulate a web server with cached database queries
	cache, _ := New(NewDefaultConfig().WithMaxEntries(500))

	// Simulate database query
	dbQuery := func(userID int) map[string]interface{} {
		time.Sleep(time.Microsecond * 50) // Simulate DB latency
		return map[string]interface{}{
			"id":    userID,
			"name":  fmt.Sprintf("User %d", userID),
			"email": fmt.Sprintf("user%d@example.com", userID),
		}
	}

	cachedDBQuery := Wrap(cache, dbQuery, WithTTL(5*time.Minute))

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate realistic request pattern (Zipfian distribution)
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 80% of requests go to 20% of users (popular users)
			var userID int
			if i%10 < 8 {
				userID = i % 20 // Popular users (high cache hit rate)
			} else {
				userID = 20 + (i % 480) // Less popular users
			}

			cachedDBQuery(userID)
			i++
		}
	})
}

// Comprehensive Benchmark Suite

func BenchmarkCacheEffectivenessComparison(b *testing.B) {
	// This benchmark compares the same workload with and without caching

	// Setup
	cache, _ := New(NewDefaultConfig())
	wrappedFunc := Wrap(cache, expensiveComputation)

	// Pre-populate cache for realistic hit rates
	for i := 0; i < 50; i++ {
		wrappedFunc(i)
	}

	b.Run("CPUBound-Uncached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			expensiveComputation(i % 50)
		}
	})

	b.Run("CPUBound-Cached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			wrappedFunc(i % 50)
		}
	})

	// I/O bound operations show cache benefits better
	ioCache, _ := New(NewDefaultConfig())
	wrappedIOFunc := Wrap(ioCache, expensiveIOOperation)

	// Pre-populate cache
	for i := 0; i < 10; i++ {
		wrappedIOFunc(i)
	}

	b.Run("IOBound-Uncached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			expensiveIOOperation(i % 10)
		}
	})

	b.Run("IOBound-Cached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			wrappedIOFunc(i % 10)
		}
	})
}
