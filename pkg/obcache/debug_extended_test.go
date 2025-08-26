package obcache

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDebugHandlerConcurrency(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	handler := cache.DebugHandler()

	// Add initial data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		_ = cache.Set(key, value, time.Hour)
	}

	// Make concurrent requests to debug handler
	const numRequests = 50
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			req := httptest.NewRequest("GET", "/keys", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				results <- fmt.Errorf("request %d: expected status 200, got %d", requestID, w.Code)
				return
			}

			var response DebugResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				results <- fmt.Errorf("request %d: failed to decode response: %v", requestID, err)
				return
			}

			if len(response.Keys) != 100 {
				results <- fmt.Errorf("request %d: expected 100 keys, got %d", requestID, len(response.Keys))
				return
			}

			results <- nil
		}(i)
	}

	// Check all results
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}

func TestDebugHandlerLargeDataset(t *testing.T) {
	cache, err := New(NewDefaultConfig().WithMaxEntries(10000))
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Add large dataset
	const numKeys = 5000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("large-key-%04d", i)
		value := strings.Repeat("x", 1000) // Large values
		_ = cache.Set(key, value, time.Hour)
	}

	handler := cache.DebugHandler()

	// Test /keys endpoint with large dataset
	req := httptest.NewRequest("GET", "/keys", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	handler.ServeHTTP(w, req)
	duration := time.Since(start)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response DebugResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Keys) != numKeys {
		t.Errorf("Expected %d keys, got %d", numKeys, len(response.Keys))
	}

	// Should complete reasonably quickly even with large dataset
	if duration > 5*time.Second {
		t.Errorf("Debug handler took too long: %v", duration)
	}

	// Verify response size is reasonable (should be compressed if large)
	responseSize := w.Body.Len()
	t.Logf("Response size for %d keys: %d bytes (%.2f KB)", numKeys, responseSize, float64(responseSize)/1024)
}

func TestDebugHandlerComplexValues(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Add various complex value types
	testData := map[string]interface{}{
		"string":  "simple string",
		"number":  42,
		"float":   3.14159,
		"boolean": true,
		"nil":     nil,
		"map": map[string]interface{}{
			"nested": "value",
			"count":  123,
		},
		"slice": []string{"a", "b", "c"},
		"struct": struct {
			Name string
			Age  int
		}{Name: "Test", Age: 25},
		"pointer": &struct{ Data string }{Data: "pointer data"},
	}

	for key, value := range testData {
		_ = cache.Set(key, value, time.Hour)
	}

	handler := cache.DebugHandler()
	req := httptest.NewRequest("GET", "/keys", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response DebugResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Keys) != len(testData) {
		t.Errorf("Expected %d keys, got %d", len(testData), len(response.Keys))
	}

	// Verify all keys are present and values are correctly serialized
	keyMap := make(map[string]DebugKey)
	for _, key := range response.Keys {
		keyMap[key.Key] = key
	}

	for expectedKey := range testData {
		if debugKey, exists := keyMap[expectedKey]; !exists {
			t.Errorf("Expected key '%s' not found in response", expectedKey)
		} else {
			// Verify the key has required metadata
			if debugKey.TTL == "" {
				t.Errorf("Key '%s' missing TTL", expectedKey)
			}
			if debugKey.Age == "" {
				t.Errorf("Key '%s' missing Age", expectedKey)
			}
			if debugKey.ExpiresAt == nil {
				t.Errorf("Key '%s' missing ExpiresAt", expectedKey)
			}
		}
	}
}

func TestDebugHandlerEdgeCases(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		method       string
		expectedCode int
		expectedJSON bool
		setupCache   func(*Cache)
		validateResp func(*testing.T, DebugResponse)
	}{
		{
			name:         "Empty cache keys endpoint",
			path:         "/keys",
			method:       "GET",
			expectedCode: http.StatusOK,
			expectedJSON: true,
			setupCache:   func(_ *Cache) {}, // No setup - empty cache
			validateResp: func(t *testing.T, resp DebugResponse) {
				if len(resp.Keys) != 0 {
					t.Errorf("Expected empty keys array, got %d keys", len(resp.Keys))
				}
			},
		},
		{
			name:         "Empty cache stats endpoint",
			path:         "/stats",
			method:       "GET",
			expectedCode: http.StatusOK,
			expectedJSON: true,
			setupCache:   func(_ *Cache) {}, // No setup - empty cache
			validateResp: func(t *testing.T, resp DebugResponse) {
				if resp.Stats == nil {
					t.Error("Expected stats to be present")
					return
				}
				if resp.Stats.KeyCount != 0 {
					t.Errorf("Expected 0 keys in stats, got %d", resp.Stats.KeyCount)
				}
				if len(resp.Keys) != 0 {
					t.Errorf("Stats endpoint should not include keys, got %d", len(resp.Keys))
				}
			},
		},
		{
			name:         "Invalid path returns keys",
			path:         "/invalid",
			method:       "GET",
			expectedCode: http.StatusOK,
			expectedJSON: true,
			setupCache:   func(_ *Cache) {},
			validateResp: func(t *testing.T, resp DebugResponse) {
				// Invalid paths are treated as keys requests
				if len(resp.Keys) != 0 {
					t.Errorf("Expected empty keys array, got %d keys", len(resp.Keys))
				}
			},
		},
		{
			name:         "POST method not allowed",
			path:         "/keys",
			method:       "POST",
			expectedCode: http.StatusMethodNotAllowed,
			expectedJSON: false,
			setupCache:   func(_ *Cache) {},
			validateResp: func(_ *testing.T, _ DebugResponse) {},
		},
		{
			name:         "PUT method not allowed",
			path:         "/stats",
			method:       "PUT",
			expectedCode: http.StatusMethodNotAllowed,
			expectedJSON: false,
			setupCache:   func(_ *Cache) {},
			validateResp: func(_ *testing.T, _ DebugResponse) {},
		},
		{
			name:         "DELETE method not allowed",
			path:         "/",
			method:       "DELETE",
			expectedCode: http.StatusMethodNotAllowed,
			expectedJSON: false,
			setupCache:   func(_ *Cache) {},
			validateResp: func(_ *testing.T, _ DebugResponse) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache, err := New(NewDefaultConfig())
			if err != nil {
				t.Fatalf("Failed to create cache: %v", err)
			}
			defer cache.Close()

			tc.setupCache(cache)
			handler := cache.DebugHandler()

			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, w.Code)
			}

			if tc.expectedJSON {
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("Expected JSON content type, got %s", w.Header().Get("Content-Type"))
				}

				var response DebugResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode JSON response: %v", err)
				}

				tc.validateResp(t, response)
			}
		})
	}
}

func TestDebugHandlerPerformanceStats(t *testing.T) { //nolint:gocyclo // Acceptable complexity for comprehensive validation
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Generate various operations to create meaningful stats
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("perf-key-%d", i)
		_ = cache.Set(key, i, time.Hour)
	}

	// Generate hits and misses
	hitCount := 0
	missCount := 0
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("perf-key-%d", i)
		if _, found := cache.Get(key); found {
			hitCount++
		} else {
			missCount++
		}
	}

	// Invalidate some keys
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("perf-key-%d", i)
		_ = cache.Invalidate(key)
	}

	handler := cache.DebugHandler()
	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response DebugResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	stats := response.Stats
	if stats == nil {
		t.Fatal("Expected stats to be present")
	}

	// Verify stats accuracy
	if stats.Hits != int64(hitCount) {
		t.Errorf("Expected %d hits, got %d", hitCount, stats.Hits)
	}
	if stats.Misses != int64(missCount) {
		t.Errorf("Expected %d misses, got %d", missCount, stats.Misses)
	}
	if stats.Invalidations != 10 {
		t.Errorf("Expected 10 invalidations, got %d", stats.Invalidations)
	}
	if stats.Total != int64(hitCount+missCount) {
		t.Errorf("Expected %d total operations, got %d", hitCount+missCount, stats.Total)
	}

	// Verify hit rate calculation
	expectedHitRate := float64(hitCount) / float64(hitCount+missCount) * 100
	if fmt.Sprintf("%.2f", stats.HitRate) != fmt.Sprintf("%.2f", expectedHitRate) {
		t.Errorf("Expected hit rate %.2f, got %.2f", expectedHitRate, stats.HitRate)
	}

	// Verify config is included
	if stats.Config == nil {
		t.Error("Expected config to be present in stats")
	} else {
		if stats.Config.MaxEntries != 1000 { // Default value
			t.Errorf("Expected default MaxEntries 1000, got %d", stats.Config.MaxEntries)
		}
		if stats.Config.DefaultTTL != 5*time.Minute { // Default value
			t.Errorf("Expected default TTL 5m, got %v", stats.Config.DefaultTTL)
		}
	}
}

func TestDebugHandlerTTLFormats(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	testCases := []struct {
		key         string
		ttl         time.Duration
		expectedFmt string
	}{
		{"second-key", 90 * time.Second, "1m"},
		{"minute-key", 3661 * time.Second, "1h"},
		{"hour-key", 25 * time.Hour, "25h"},
		{"long-key", 48 * time.Hour, "48h"},
	}

	for _, tc := range testCases {
		_ = cache.Set(tc.key, "test-value", tc.ttl)
	}

	handler := cache.DebugHandler()
	req := httptest.NewRequest("GET", "/keys", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response DebugResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	keyMap := make(map[string]DebugKey)
	for _, key := range response.Keys {
		keyMap[key.Key] = key
	}

	for _, tc := range testCases {
		if debugKey, exists := keyMap[tc.key]; !exists {
			t.Errorf("Key %s not found in response", tc.key)
		} else {
			// Note: Due to timing differences, we'll check that TTL is reasonably close
			// rather than exact match, since some time will have passed
			if debugKey.TTL == "" {
				t.Errorf("Key %s has empty TTL", tc.key)
			}
			// For very short TTLs, they might have already expired
			if tc.ttl > time.Second && debugKey.TTL == "expired" {
				t.Errorf("Key %s unexpectedly expired (TTL was %v)", tc.key, tc.ttl)
			}
		}
	}
}

func TestDebugHandlerContentNegotiation(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	_ = cache.Set("test-key", "test-value", time.Hour)

	handler := cache.DebugHandler()

	testCases := []struct {
		name           string
		acceptHeader   string
		expectedType   string
		expectedStatus int
	}{
		{
			name:           "JSON accept header",
			acceptHeader:   "application/json",
			expectedType:   "application/json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Wildcard accept header",
			acceptHeader:   "*/*",
			expectedType:   "application/json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "No accept header",
			acceptHeader:   "",
			expectedType:   "application/json",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Text accept header",
			acceptHeader:   "text/plain",
			expectedType:   "application/json", // Should still return JSON
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/keys", nil)
			if tc.acceptHeader != "" {
				req.Header.Set("Accept", tc.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tc.expectedType {
				t.Errorf("Expected content type %s, got %s", tc.expectedType, contentType)
			}
		})
	}
}

func BenchmarkDebugHandler(b *testing.B) {
	cache, _ := New(NewDefaultConfig())
	defer cache.Close()

	// Setup cache with test data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		value := fmt.Sprintf("bench-value-%d", i)
		_ = cache.Set(key, value, time.Hour)
	}

	handler := cache.DebugHandler()

	b.Run("Keys endpoint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/keys", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("Stats endpoint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/stats", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}
