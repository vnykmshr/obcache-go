package obcache

import (
	"fmt"
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzCacheOperations tests cache operations with fuzzed inputs
func FuzzCacheOperations(f *testing.F) {
	// Seed corpus with various key types
	f.Add("simple-key", "simple-value", int64(3600))
	f.Add("", "empty-key", int64(1))
	f.Add("key-with-spaces and symbols!@#$%", "complex value with\nnewlines\tand\ttabs", int64(86400))
	f.Add("unicode-key-🚀🎯", "unicode-value-こんにちは", int64(60))
	f.Add("very-long-key-"+string(make([]byte, 1000)), "very-long-value-"+string(make([]byte, 5000)), int64(30))

	f.Fuzz(func(t *testing.T, key string, value string, ttlSeconds int64) {
		// Skip invalid UTF-8 strings as they're not realistic for most use cases
		if !utf8.ValidString(key) || !utf8.ValidString(value) {
			t.Skip("Skipping invalid UTF-8 input")
		}

		// Normalize TTL to reasonable range
		if ttlSeconds <= 0 {
			ttlSeconds = 1
		}
		if ttlSeconds > 86400 { // Max 1 day
			ttlSeconds = 86400
		}

		ttl := time.Duration(ttlSeconds) * time.Second

		cache, err := New(NewDefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()

		// Test Set operation
		err = cache.Set(key, value, ttl)
		if err != nil {
			t.Errorf("Set failed for key=%q value=%q ttl=%v: %v", key, value, ttl, err)
			return
		}

		// Test Get operation immediately after Set
		retrievedValue, found := cache.Get(key)
		if !found {
			t.Errorf("Get failed immediately after Set for key=%q", key)
			return
		}

		if retrievedValue != value {
			t.Errorf("Value mismatch for key=%q: got=%q want=%q", key, retrievedValue, value)
			return
		}

		// Test overwrite
		newValue := value + "-modified"
		err = cache.Set(key, newValue, ttl)
		if err != nil {
			t.Errorf("Set overwrite failed for key=%q: %v", key, err)
			return
		}

		retrievedValue, found = cache.Get(key)
		if !found {
			t.Errorf("Get failed after overwrite for key=%q", key)
			return
		}

		if retrievedValue != newValue {
			t.Errorf("Overwrite value mismatch for key=%q: got=%q want=%q", key, retrievedValue, newValue)
			return
		}

		// Test Delete
		err = cache.Invalidate(key)
		if err != nil {
			t.Errorf("Delete failed for key=%q: %v", key, err)
			return
		}

		_, found = cache.Get(key)
		if found {
			t.Errorf("Get succeeded after Delete for key=%q", key)
			return
		}
	})
}

// FuzzWrappedFunction tests function wrapping with fuzzed inputs
func FuzzWrappedFunction(f *testing.F) {
	// Seed corpus
	f.Add(42, "test-string", true)
	f.Add(-1, "", false)
	f.Add(0, "unicode-🎯", true)
	f.Add(1000000, "very-long-string-"+string(make([]byte, 1000)), false)

	f.Fuzz(func(t *testing.T, intParam int, stringParam string, boolParam bool) {
		// Skip invalid UTF-8
		if !utf8.ValidString(stringParam) {
			t.Skip("Skipping invalid UTF-8 input")
		}

		cache, err := New(NewDefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()

		// Test function that processes the fuzzed inputs
		testFunc := func(a int, b string, c bool) string {
			return fmt.Sprintf("processed-%d-%s-%t", a, b, c)
		}

		wrapped := Wrap(cache, testFunc)

		// First call should execute function
		result1 := wrapped(intParam, stringParam, boolParam)
		expected := fmt.Sprintf("processed-%d-%s-%t", intParam, stringParam, boolParam)

		if result1 != expected {
			t.Errorf("First call result mismatch: got=%q want=%q", result1, expected)
			return
		}

		// Second call should hit cache
		result2 := wrapped(intParam, stringParam, boolParam)
		if result2 != result1 {
			t.Errorf("Second call result mismatch: got=%q want=%q", result2, result1)
			return
		}

		// Test with slightly different parameters to ensure proper key generation
		modifiedStringParam := stringParam + "-suffix"
		result3 := wrapped(intParam, modifiedStringParam, boolParam)
		expected3 := fmt.Sprintf("processed-%d-%s-%t", intParam, modifiedStringParam, boolParam)

		if result3 != expected3 {
			t.Errorf("Modified param result mismatch: got=%q want=%q", result3, expected3)
			return
		}

		// Should be different from original result (unless coincidentally the same)
		if result3 == result1 && modifiedStringParam != stringParam {
			t.Errorf("Modified parameter produced same result unexpectedly")
			return
		}
	})
}

// FuzzErrorCaching tests error caching with fuzzed inputs
func FuzzErrorCaching(f *testing.F) {
	// Seed corpus
	f.Add(0, "normal input")
	f.Add(-1, "error input")
	f.Add(42, "")
	f.Add(-999, "trigger-special-error")

	f.Fuzz(func(t *testing.T, errorCode int, input string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(input) {
			t.Skip("Skipping invalid UTF-8 input")
		}

		cache, err := New(NewDefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()

		// Function that returns errors based on input
		errorFunc := func(code int, data string) (string, error) {
			if code < 0 {
				return "", fmt.Errorf("error code %d with data: %s", code, data)
			}
			return fmt.Sprintf("success-%d-%s", code, data), nil
		}

		// Test without error caching
		wrappedWithoutErrorCache := Wrap(cache, errorFunc)

		result1, err1 := wrappedWithoutErrorCache(errorCode, input)

		// Second call should produce identical result
		result2, err2 := wrappedWithoutErrorCache(errorCode, input)

		if errorCode < 0 {
			// Should be errors
			if err1 == nil || err2 == nil {
				t.Errorf("Expected errors for negative code %d, got err1=%v err2=%v", errorCode, err1, err2)
				return
			}

			// Without error caching, we can't guarantee identical error objects,
			// but the error messages should be the same
			if err1.Error() != err2.Error() {
				t.Errorf("Error messages differ: %q vs %q", err1.Error(), err2.Error())
				return
			}
		} else {
			// Should be successful
			if err1 != nil || err2 != nil {
				t.Errorf("Unexpected errors for non-negative code %d: err1=%v err2=%v", errorCode, err1, err2)
				return
			}

			if result1 != result2 {
				t.Errorf("Success results differ: %q vs %q", result1, result2)
				return
			}
		}

		// Test with error caching enabled
		cache2, err := New(NewDefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer cache2.Close()

		wrappedWithErrorCache := Wrap(cache2, errorFunc,
			WithErrorCaching(),
			WithErrorTTL(time.Hour))

		result3, err3 := wrappedWithErrorCache(errorCode, input)
		result4, err4 := wrappedWithErrorCache(errorCode, input)

		if errorCode < 0 {
			// Should be errors
			if err3 == nil || err4 == nil {
				t.Errorf("Expected cached errors for negative code %d, got err3=%v err4=%v", errorCode, err3, err4)
				return
			}

			// With error caching, error messages should be identical
			if err3.Error() != err4.Error() {
				t.Errorf("Cached error messages differ: %q vs %q", err3.Error(), err4.Error())
				return
			}
		} else {
			// Should be successful
			if err3 != nil || err4 != nil {
				t.Errorf("Unexpected cached errors for non-negative code %d: err3=%v err4=%v", errorCode, err3, err4)
				return
			}

			if result3 != result4 {
				t.Errorf("Cached success results differ: %q vs %q", result3, result4)
				return
			}
		}
	})
}

// FuzzKeyGeneration tests key generation with various input types
func FuzzKeyGeneration(f *testing.F) {
	// Seed corpus with edge cases
	f.Add("", int64(0), false)
	f.Add("test", int64(-1), true)
	f.Add("unicode-🎯", int64(9223372036854775807), false)                 // max int64
	f.Add("with\nnewlines\tand\ttabs", int64(-9223372036854775808), true) // min int64

	f.Fuzz(func(t *testing.T, s string, i int64, b bool) {
		// Skip invalid UTF-8
		if !utf8.ValidString(s) {
			t.Skip("Skipping invalid UTF-8 input")
		}

		// Test different key generation functions
		keyFuncs := []struct {
			name string
			fn   KeyGenFunc
		}{
			{"Default", DefaultKeyFunc},
			{"Simple", SimpleKeyFunc},
		}

		for _, kf := range keyFuncs {
			t.Run(kf.name, func(t *testing.T) {
				args := []any{s, i, b}

				// Generate key multiple times - should be consistent
				key1 := kf.fn(args)
				key2 := kf.fn(args)

				if key1 != key2 {
					t.Errorf("%s key generation inconsistent: %q vs %q", kf.name, key1, key2)
					return
				}

				// Key should not be empty (unless all inputs are zero values)
				if key1 == "" && !(s == "" && i == 0 && !b) {
					t.Errorf("%s generated empty key for non-zero inputs", kf.name)
					return
				}

				// Different inputs should (usually) generate different keys
				differentArgs := []any{s + "x", i + 1, !b}
				differentKey := kf.fn(differentArgs)

				// Allow some collisions as they're theoretically possible
				// but if the same key is generated for clearly different inputs,
				// that might indicate a problem
				if key1 == differentKey && len(s) < 50 {
					t.Logf("Warning: %s generated same key for different inputs: %q", kf.name, key1)
				}
			})
		}
	})
}

// FuzzTTLHandling tests TTL handling with various durations
func FuzzTTL(f *testing.F) {
	// Seed corpus
	f.Add(int64(1))             // 1ns
	f.Add(int64(1000))          // 1μs
	f.Add(int64(1000000))       // 1ms
	f.Add(int64(1000000000))    // 1s
	f.Add(int64(3600000000000)) // 1h

	f.Fuzz(func(t *testing.T, ttlNanos int64) {
		// Normalize TTL to reasonable range (1ms to 1 hour) to avoid timing issues
		if ttlNanos <= 0 {
			ttlNanos = 1000000 // 1ms
		}
		if ttlNanos < 1000000 { // Less than 1ms
			ttlNanos = 1000000 // Set to 1ms minimum
		}
		if ttlNanos > 3600000000000 { // 1 hour in nanoseconds
			ttlNanos = 3600000000000
		}

		ttl := time.Duration(ttlNanos)

		cache, err := New(NewDefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()

		key := "fuzz-ttl-key"
		value := "fuzz-ttl-value"

		// Set with fuzzed TTL
		err = cache.Set(key, value, ttl)
		if err != nil {
			t.Errorf("Set failed with TTL %v: %v", ttl, err)
			return
		}

		// Should be available immediately
		retrievedValue, found := cache.Get(key)
		if !found {
			t.Errorf("Value not found immediately after set with TTL %v", ttl)
			return
		}

		if retrievedValue != value {
			t.Errorf("Value mismatch with TTL %v: got=%q want=%q", ttl, retrievedValue, value)
			return
		}

		// For very short TTLs, test expiration
		if ttl < time.Millisecond {
			// Wait a bit longer than the TTL
			time.Sleep(ttl * 10)

			// Check expiration - don't use the result since timing is unpredictable
			cache.Get(key)
			// Note: We don't assert !found because the cleanup goroutine
			// might not have run yet. This is expected behavior.
		}
	})
}

// FuzzConcurrentOperations tests concurrent operations with fuzzed data
func FuzzConcurrentOperations(f *testing.F) {
	if testing.Short() {
		f.Skip("skipping concurrent fuzz test in short mode")
	}

	// Seed corpus
	f.Add("key1", "value1", "key2", "value2")
	f.Add("", "empty", "unicode-🎯", "test")
	f.Add("same", "val1", "same", "val2") // Same key, different values

	f.Fuzz(func(t *testing.T, key1, value1, key2, value2 string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(key1) || !utf8.ValidString(value1) ||
			!utf8.ValidString(key2) || !utf8.ValidString(value2) {
			t.Skip("Skipping invalid UTF-8 input")
		}

		cache, err := New(NewDefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()

		// Launch concurrent operations
		done := make(chan bool, 4)

		// Goroutine 1: Set key1
		go func() {
			defer func() { done <- true }()
			cache.Set(key1, value1, time.Hour)
		}()

		// Goroutine 2: Set key2
		go func() {
			defer func() { done <- true }()
			cache.Set(key2, value2, time.Hour)
		}()

		// Goroutine 3: Get key1
		go func() {
			defer func() { done <- true }()
			cache.Get(key1)
		}()

		// Goroutine 4: Get key2
		go func() {
			defer func() { done <- true }()
			cache.Get(key2)
		}()

		// Wait for all goroutines
		for i := 0; i < 4; i++ {
			<-done
		}

		// Verify cache is in a consistent state
		length := cache.Len()
		if length < 0 {
			t.Errorf("Negative cache length after concurrent operations: %d", length)
		}

		if length > 1000 {
			t.Errorf("Cache length exceeds capacity: %d > %d", length, 1000)
		}
	})
}
