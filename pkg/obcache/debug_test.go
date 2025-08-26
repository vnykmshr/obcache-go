package obcache

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDebugHandler(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add some test data
	_ = cache.Set("key1", "value1", time.Hour)
	_ = cache.Set("key2", 42, time.Minute)
	_ = cache.Warmup("key3", "warmed-value")

	// Trigger some stats
	_, _ = cache.Get("key1")    // hit
	_, _ = cache.Get("missing") // miss

	handler := cache.DebugHandler()

	t.Run("StatsOnly", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/stats", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("Expected JSON content type, got %s", w.Header().Get("Content-Type"))
		}

		var response DebugResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check stats
		if response.Stats.Hits != 1 {
			t.Fatalf("Expected 1 hit, got %d", response.Stats.Hits)
		}
		if response.Stats.Misses != 1 {
			t.Fatalf("Expected 1 miss, got %d", response.Stats.Misses)
		}
		if response.Stats.Total != 2 {
			t.Fatalf("Expected 2 total requests, got %d", response.Stats.Total)
		}
		if response.Stats.HitRate != 50.0 {
			t.Fatalf("Expected 50%% hit rate, got %f", response.Stats.HitRate)
		}

		// Should not include keys
		if len(response.Keys) != 0 {
			t.Fatalf("Expected no keys in /stats endpoint, got %d", len(response.Keys))
		}

		// Check config
		if response.Stats.Config.MaxEntries != 1000 {
			t.Fatalf("Expected MaxEntries 1000, got %d", response.Stats.Config.MaxEntries)
		}
		if response.Stats.Config.DefaultTTL != 5*time.Minute {
			t.Fatalf("Expected DefaultTTL 5m, got %v", response.Stats.Config.DefaultTTL)
		}
	})

	t.Run("KeysEndpoint", func(t *testing.T) {
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

		// Should include keys
		if len(response.Keys) != 3 {
			t.Fatalf("Expected 3 keys, got %d", len(response.Keys))
		}

		// Check key metadata
		keyFound := false
		for _, key := range response.Keys {
			if key.Key == "key1" {
				keyFound = true
				if key.Value != "value1" {
					t.Fatalf("Expected value1, got %v", key.Value)
				}
				if key.ExpiresAt == nil {
					t.Fatal("Expected expiration time for key1")
				}
				if key.TTL == "" {
					t.Fatal("Expected TTL for key1")
				}
				if key.Age == "" {
					t.Fatal("Expected age for key1")
				}
			}
		}

		if !keyFound {
			t.Fatal("key1 not found in response")
		}
	})

	t.Run("RootEndpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response DebugResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Root endpoint should behave like /keys
		if len(response.Keys) != 3 {
			t.Fatalf("Expected 3 keys, got %d", len(response.Keys))
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/stats", strings.NewReader("test"))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("Expected status 405, got %d", w.Code)
		}
	})
}

func TestDebugHandlerWithExpiredKeys(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add keys - one that will expire soon, one valid
	_ = cache.Set("soon-expired", "value", 50*time.Millisecond)
	_ = cache.Set("valid", "value", time.Hour)

	// Make request before expiration to check TTL formatting
	handler := cache.DebugHandler()
	req := httptest.NewRequest("GET", "/keys", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response DebugResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have both keys
	if len(response.Keys) != 2 {
		t.Fatalf("Expected 2 keys before expiration, got %d", len(response.Keys))
	}

	// Check TTL formatting for valid keys
	validFound := false
	for _, key := range response.Keys {
		if key.Key == "valid" {
			validFound = true
			if key.TTL == "" {
				t.Fatal("Expected non-empty TTL for valid key")
			}
			if key.ExpiresAt == nil {
				t.Fatal("Expected expiration time for valid key")
			}
		}
	}
	if !validFound {
		t.Fatal("Valid key not found")
	}

	// Wait for the short TTL key to expire
	time.Sleep(100 * time.Millisecond)

	// Make another request - expired key should be filtered out by store
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req)

	var response2 DebugResponse
	if err := json.NewDecoder(w2.Body).Decode(&response2); err != nil {
		t.Fatalf("Failed to decode response after expiration: %v", err)
	}

	// Should only have the valid key now
	if len(response2.Keys) != 1 {
		t.Fatalf("Expected 1 key after expiration, got %d", len(response2.Keys))
	}

	if response2.Keys[0].Key != "valid" {
		t.Fatalf("Expected valid key to remain, got %s", response2.Keys[0].Key)
	}
}

func TestDebugHandlerWithoutTTL(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Use Warmup which uses DefaultTTL, then test key without explicit TTL by direct store access
	_ = cache.Warmup("key1", "value1")

	handler := cache.DebugHandler()
	req := httptest.NewRequest("GET", "/keys", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response DebugResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Keys) != 1 {
		t.Fatalf("Expected 1 key, got %d", len(response.Keys))
	}

	key := response.Keys[0]
	if key.TTL == "" {
		t.Fatal("Expected TTL to be present for warmed key")
	}
}

func TestNewDebugServer(t *testing.T) {
	cache, err := New(NewDefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	server := cache.NewDebugServer(":0") // Use port 0 for testing
	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.Addr != ":0" {
		t.Fatalf("Expected addr :0, got %s", server.Addr)
	}

	// Test that the server has the correct routes by making test requests
	_ = cache.Set("test", "value", time.Hour)

	testCases := []struct {
		path    string
		hasKeys bool
	}{
		{"/", true},
		{"/keys", true},
		{"/stats", false},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()

		server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Path %s: expected status 200, got %d", tc.path, w.Code)
		}

		var response DebugResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Path %s: failed to decode response: %v", tc.path, err)
		}

		if tc.hasKeys && len(response.Keys) == 0 {
			t.Fatalf("Path %s: expected keys but got none", tc.path)
		}
		if !tc.hasKeys && len(response.Keys) > 0 {
			t.Fatalf("Path %s: expected no keys but got %d", tc.path, len(response.Keys))
		}
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Nanosecond, "500ns"},
		{1500 * time.Microsecond, "1ms"}, // Truncated to millisecond
		{1500 * time.Millisecond, "1s"},  // Truncated to second
		{90 * time.Second, "1m0s"},       // Truncated to minute
		{3661 * time.Second, "1h0m0s"},   // Truncated to hour
	}

	for _, tc := range testCases {
		result := formatDuration(tc.duration)
		if result != tc.expected {
			t.Fatalf("formatDuration(%v): expected %s, got %s", tc.duration, tc.expected, result)
		}
	}
}

func TestDebugResponseSerialization(t *testing.T) {
	// Test that our debug response structures serialize correctly to JSON
	response := DebugResponse{
		Stats: &DebugStats{
			Hits:          10,
			Misses:        5,
			Evictions:     2,
			Invalidations: 1,
			KeyCount:      3,
			InFlight:      0,
			HitRate:       66.67,
			Total:         15,
			Config: &DebugConfig{
				MaxEntries:      1000,
				DefaultTTL:      5 * time.Minute,
				CleanupInterval: time.Minute,
			},
		},
		Keys: []DebugKey{
			{
				Key:       "test",
				Value:     "value",
				ExpiresAt: func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
				CreatedAt: time.Now(),
				Age:       "1m",
				TTL:       "59m",
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to serialize debug response: %v", err)
	}

	var decoded DebugResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to deserialize debug response: %v", err)
	}

	if decoded.Stats.Hits != 10 {
		t.Fatalf("Expected 10 hits after round-trip, got %d", decoded.Stats.Hits)
	}
	if len(decoded.Keys) != 1 {
		t.Fatalf("Expected 1 key after round-trip, got %d", len(decoded.Keys))
	}
}
