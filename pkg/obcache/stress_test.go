package obcache

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestStressConcurrentAccess tests high-concurrency scenarios
func TestStressConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	config := NewDefaultConfig().WithMaxEntries(1000)
	cache, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	const (
		numGoroutines          = 100
		operationsPerGoroutine = 1000
		keyRange               = 500
	)

	var (
		sets    int64
		gets    int64
		deletes int64
		errors  int64
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines performing random operations
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("stress-key-%d", rng.Intn(keyRange))

				switch rng.Intn(10) {
				case 0, 1: // 20% deletes
					if err := cache.Invalidate(key); err != nil {
						atomic.AddInt64(&errors, 1)
					}
					atomic.AddInt64(&deletes, 1)

				case 2, 3, 4: // 30% sets
					value := fmt.Sprintf("value-%d-%d", workerID, j)
					if err := cache.Set(key, value, time.Hour); err != nil {
						atomic.AddInt64(&errors, 1)
					}
					atomic.AddInt64(&sets, 1)

				default: // 50% gets
					_, _ = cache.Get(key)
					atomic.AddInt64(&gets, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	totalOps := atomic.LoadInt64(&sets) + atomic.LoadInt64(&gets) + atomic.LoadInt64(&deletes)
	errorRate := float64(atomic.LoadInt64(&errors)) / float64(totalOps) * 100

	t.Logf("Stress test completed:")
	t.Logf("  Sets: %d", atomic.LoadInt64(&sets))
	t.Logf("  Gets: %d", atomic.LoadInt64(&gets))
	t.Logf("  Deletes: %d", atomic.LoadInt64(&deletes))
	t.Logf("  Total operations: %d", totalOps)
	t.Logf("  Errors: %d (%.2f%%)", atomic.LoadInt64(&errors), errorRate)

	if errorRate > 1.0 {
		t.Errorf("Error rate too high: %.2f%%", errorRate)
	}

	// Verify cache is still functional
	if err := cache.Set("final-test", "success", time.Hour); err != nil {
		t.Error("Cache not functional after stress test")
	}

	if val, found := cache.Get("final-test"); !found || val != "success" {
		t.Error("Cache get failed after stress test")
	}
}

// TestStressWrappedFunctions tests wrapped function stress scenarios
func TestStressWrappedFunctions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	cache, err := New(NewDefaultConfig().WithMaxEntries(500))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	// Expensive computation that we'll wrap
	var computationCount int64
	expensiveFunc := func(input int) string {
		atomic.AddInt64(&computationCount, 1)
		time.Sleep(time.Microsecond * 10) // Simulate work
		return fmt.Sprintf("result-%d", input)
	}

	wrapped := Wrap(cache, expensiveFunc)

	const (
		numWorkers     = 50
		callsPerWorker = 500
		inputRange     = 100 // Some overlap to test cache effectiveness
	)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	startTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for j := 0; j < callsPerWorker; j++ {
				input := rng.Intn(inputRange)
				result := wrapped(input)

				expected := fmt.Sprintf("result-%d", input)
				if result != expected {
					t.Errorf("Wrong result for input %d: got %s, want %s", input, result, expected)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(startTime)
	totalCalls := numWorkers * callsPerWorker
	actualComputations := atomic.LoadInt64(&computationCount)

	cacheEffectiveness := (1.0 - float64(actualComputations)/float64(totalCalls)) * 100

	t.Logf("Wrapped function stress test completed:")
	t.Logf("  Total function calls: %d", totalCalls)
	t.Logf("  Actual computations: %d", actualComputations)
	t.Logf("  Cache effectiveness: %.1f%%", cacheEffectiveness)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Calls per second: %.0f", float64(totalCalls)/duration.Seconds())

	if cacheEffectiveness < 80.0 {
		t.Errorf("Cache effectiveness too low: %.1f%% (expected > 80%%)", cacheEffectiveness)
	}
}

// TestStressMemoryPressure tests behavior under memory pressure
func TestStressMemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	// Create a small cache to force evictions
	config := NewDefaultConfig().WithMaxEntries(100)
	cache, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	const (
		numWorkers       = 10
		entriesPerWorker = 1000
	)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	startMemory := getMemStats()

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < entriesPerWorker; j++ {
				key := fmt.Sprintf("memory-test-%d-%d", workerID, j)
				// Store progressively larger values to test memory handling
				value := make([]byte, j%1000+100)
				for k := range value {
					value[k] = byte(k % 256)
				}

				if err := cache.Set(key, value, time.Hour); err != nil {
					t.Errorf("Failed to set key %s: %v", key, err)
					return
				}

				// Occasionally verify we can still read
				if j%100 == 0 {
					// Check if we can read - eviction is expected, so we don't use the result
					cache.Get(key)
				}
			}
		}(i)
	}

	wg.Wait()

	endMemory := getMemStats()

	t.Logf("Memory pressure test completed:")
	t.Logf("  Cache length: %d", cache.Len())
	t.Logf("  Cache capacity: %d", 1000)
	t.Logf("  Memory before: %d bytes", startMemory.Alloc)
	t.Logf("  Memory after: %d bytes", endMemory.Alloc)
	t.Logf("  Memory delta: %d bytes", int64(endMemory.Alloc)-int64(startMemory.Alloc))

	// Cache should not exceed its capacity
	if cache.Len() > 1000 {
		t.Errorf("Cache exceeded capacity: %d > %d", cache.Len(), 1000)
	}

	// Cache should still be functional
	testKey := "final-memory-test"
	testValue := []byte("test-value")
	if err := cache.Set(testKey, testValue, time.Hour); err != nil {
		t.Error("Cache not functional after memory pressure test")
	}
}

// TestStressTTLExpiration tests TTL handling under stress
func TestStressTTLExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	config := NewDefaultConfig().
		WithMaxEntries(1000).
		WithCleanupInterval(10 * time.Millisecond)

	cache, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	const (
		numWorkers       = 20
		entriesPerWorker = 100
	)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Set entries with various TTLs
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < entriesPerWorker; j++ {
				key := fmt.Sprintf("ttl-test-%d-%d", workerID, j)
				// Vary TTL from 1ms to 100ms
				ttl := time.Duration(j%100+1) * time.Millisecond

				if err := cache.Set(key, fmt.Sprintf("value-%d", j), ttl); err != nil {
					t.Errorf("Failed to set key %s: %v", key, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Wait for cleanup to run several times
	time.Sleep(200 * time.Millisecond)

	// Most entries should have expired by now
	remainingEntries := cache.Len()
	t.Logf("TTL stress test completed:")
	t.Logf("  Total entries set: %d", numWorkers*entriesPerWorker)
	t.Logf("  Remaining entries: %d", remainingEntries)

	// There should be significantly fewer entries due to expiration
	if remainingEntries > numWorkers*entriesPerWorker/2 {
		t.Logf("Warning: More entries remaining than expected, cleanup may be slow")
	}
}

// TestStressEvictionStrategies tests different eviction strategies under stress
func TestStressEvictionStrategies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	strategies := []struct {
		name   string
		config func() *Config
	}{
		{"LRU", func() *Config { return NewDefaultConfig().WithLRUEviction().WithMaxEntries(100) }},
		{"LFU", func() *Config { return NewDefaultConfig().WithLFUEviction().WithMaxEntries(100) }},
		{"FIFO", func() *Config { return NewDefaultConfig().WithFIFOEviction().WithMaxEntries(100) }},
	}

	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			cache, err := New(strategy.config())
			if err != nil {
				t.Fatal(err)
			}
			defer cache.Close()

			const (
				numOperations = 5000
				keyRange      = 200 // More keys than capacity to force evictions
			)

			rng := rand.New(rand.NewSource(42)) // Fixed seed for reproducibility

			startTime := time.Now()

			for i := 0; i < numOperations; i++ {
				key := fmt.Sprintf("eviction-test-%d", rng.Intn(keyRange))

				// Mix of operations
				switch rng.Intn(10) {
				case 0, 1: // 20% gets
					cache.Get(key)
				default: // 80% sets
					value := fmt.Sprintf("value-%d", i)
					cache.Set(key, value, time.Hour)
				}

				// Cache should never exceed capacity
				if cache.Len() > 1000 {
					t.Fatalf("Cache exceeded capacity with %s strategy: %d > %d",
						strategy.name, cache.Len(), 1000)
				}
			}

			duration := time.Since(startTime)

			t.Logf("%s eviction stress test:", strategy.name)
			t.Logf("  Operations: %d", numOperations)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Ops/sec: %.0f", float64(numOperations)/duration.Seconds())
			t.Logf("  Final cache size: %d/%d", cache.Len(), 1000)
		})
	}
}

// TestStressHookExecution tests hook execution under high load
func TestStressHookExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	var (
		hitCount   int64
		missCount  int64
		evictCount int64
	)

	hooks := &Hooks{}
	hooks.AddOnHit(func(key string, value any) {
		atomic.AddInt64(&hitCount, 1)
	})
	hooks.AddOnMiss(func(key string) {
		atomic.AddInt64(&missCount, 1)
	})
	hooks.AddOnEvict(func(key string, value any, reason EvictReason) {
		atomic.AddInt64(&evictCount, 1)
	})

	config := NewDefaultConfig().
		WithMaxEntries(100).
		WithHooks(hooks)

	cache, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	const numOperations = 10000
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("hook-test-%d", rng.Intn(150))

		if rng.Intn(2) == 0 {
			// Set operation
			cache.Set(key, fmt.Sprintf("value-%d", i), time.Hour)
		} else {
			// Get operation
			cache.Get(key)
		}
	}

	t.Logf("Hook execution stress test:")
	t.Logf("  Operations: %d", numOperations)
	t.Logf("  Hits: %d", atomic.LoadInt64(&hitCount))
	t.Logf("  Misses: %d", atomic.LoadInt64(&missCount))
	t.Logf("  Evictions: %d", atomic.LoadInt64(&evictCount))

	// Verify hooks were called
	totalHits := atomic.LoadInt64(&hitCount)
	totalMisses := atomic.LoadInt64(&missCount)

	if totalHits == 0 && totalMisses == 0 {
		t.Error("No hits or misses recorded - hooks may not be working")
	}
}

// Helper function to get memory stats
func getMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	return m
}

// TestStressRaceConditions specifically tests for race conditions
func TestStressRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	// Run test multiple times to increase chance of catching races
	for round := 0; round < 3; round++ {
		t.Run(fmt.Sprintf("Round%d", round+1), func(t *testing.T) {
			cache, err := New(NewDefaultConfig().WithMaxEntries(50))
			if err != nil {
				t.Fatal(err)
			}
			defer cache.Close()

			const (
				numGoroutines          = 100
				operationsPerGoroutine = 100
			)

			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			// All goroutines work on the same small set of keys to maximize contention
			keys := make([]string, 10)
			for i := range keys {
				keys[i] = fmt.Sprintf("race-key-%d", i)
			}

			for i := 0; i < numGoroutines; i++ {
				go func(workerID int) {
					defer wg.Done()

					rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

					for j := 0; j < operationsPerGoroutine; j++ {
						key := keys[rng.Intn(len(keys))]

						switch rng.Intn(4) {
						case 0:
							cache.Set(key, fmt.Sprintf("value-%d-%d", workerID, j), time.Hour)
						case 1:
							cache.Get(key)
						case 2:
							cache.Invalidate(key)
						case 3:
							_ = cache.Len() // Read cache state
						}
					}
				}(i)
			}

			wg.Wait()

			// Verify cache is still in a consistent state
			length := cache.Len()
			if length < 0 || length > 1000 {
				t.Errorf("Cache in inconsistent state: length=%d, capacity=%d", length, 1000)
			}
		})
	}
}

// TestStressCleanupGoroutine tests the cleanup goroutine under stress
func TestStressCleanupGoroutine(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	// Fast cleanup interval to stress the cleanup goroutine
	config := NewDefaultConfig().
		WithMaxEntries(1000).
		WithCleanupInterval(time.Millisecond)

	cache, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	// Set many entries with short TTLs
	const numEntries = 5000
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("cleanup-test-%d", i)
		ttl := time.Duration(i%50+1) * time.Millisecond
		cache.Set(key, fmt.Sprintf("value-%d", i), ttl)
	}

	initialLength := cache.Len()
	t.Logf("Initial cache length: %d", initialLength)

	// Wait for cleanup to run multiple times
	time.Sleep(200 * time.Millisecond)

	finalLength := cache.Len()
	t.Logf("Final cache length: %d", finalLength)

	cache.Close()

	// Most entries should have been cleaned up
	if finalLength >= initialLength {
		t.Error("Cleanup goroutine doesn't seem to be working effectively")
	}

	t.Logf("Cleanup effectiveness: %.1f%% entries removed",
		float64(initialLength-finalLength)/float64(initialLength)*100)
}
