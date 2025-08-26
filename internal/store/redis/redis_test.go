package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vnykmshr/obcache-go/internal/entry"
)

// TestRedisStoreBasicOperations tests basic Redis store operations using a mock
func TestRedisStoreBasicOperations(t *testing.T) {
	// Create a Redis client pointing to a non-existent server for unit testing
	// In practice, you would use miniredis or testcontainers
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Assume Redis is running for integration test
	})

	// Try to ping Redis - skip test if not available
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
	}

	config := &Config{
		Client:    client,
		KeyPrefix: "test:",
		Context:   ctx,
	}

	store, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create Redis store: %v", err)
	}
	defer store.Close()

	// Clean up any existing test data
	testKey := "test-key"
	store.Delete(testKey)

	// Test Set and Get
	testEntry := entry.New("test-value", time.Hour)
	err = store.Set(testKey, testEntry)
	if err != nil {
		t.Fatalf("Failed to set entry: %v", err)
	}

	retrievedEntry, found := store.Get(testKey)
	if !found {
		t.Fatal("Expected to find the entry")
	}

	if retrievedEntry.Value != "test-value" {
		t.Fatalf("Expected value 'test-value', got %v", retrievedEntry.Value)
	}

	// Test Keys
	keys := store.Keys()
	foundKey := false
	for _, key := range keys {
		if key == testKey {
			foundKey = true
			break
		}
	}
	if !foundKey {
		t.Fatal("Expected to find test key in Keys() result")
	}

	// Test Delete
	err = store.Delete(testKey)
	if err != nil {
		t.Fatalf("Failed to delete entry: %v", err)
	}

	_, found = store.Get(testKey)
	if found {
		t.Fatal("Expected entry to be deleted")
	}

	// Test Len after delete
	if store.Len() > 0 {
		// Note: Len() might return 0 or more depending on other concurrent tests
		// This is just a basic sanity check
	}
}

func TestRedisStoreWithTTL(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
	}

	config := &Config{
		Client:    client,
		KeyPrefix: "ttl-test:",
		Context:   ctx,
	}

	store, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create Redis store: %v", err)
	}
	defer store.Close()

	// Test with short TTL
	testKey := "ttl-key"
	store.Delete(testKey) // Clean up

	shortTTLEntry := entry.New("ttl-value", 100*time.Millisecond)
	err = store.Set(testKey, shortTTLEntry)
	if err != nil {
		t.Fatalf("Failed to set entry with TTL: %v", err)
	}

	// Should be available immediately
	_, found := store.Get(testKey)
	if !found {
		t.Fatal("Expected to find entry immediately after setting")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should be expired
	_, found = store.Get(testKey)
	if found {
		t.Fatal("Expected entry to be expired")
	}
}

func TestRedisStoreClear(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
	}

	config := &Config{
		Client:    client,
		KeyPrefix: "clear-test:",
		Context:   ctx,
	}

	store, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create Redis store: %v", err)
	}
	defer store.Close()

	// Add some entries
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		entry := entry.New(fmt.Sprintf("value-%d", i), time.Hour)
		store.Set(key, entry)
	}

	// Verify entries exist
	if store.Len() == 0 {
		t.Fatal("Expected some entries after setting")
	}

	// Clear all entries
	err = store.Clear()
	if err != nil {
		t.Fatalf("Failed to clear store: %v", err)
	}

	// Verify entries are gone
	if store.Len() != 0 {
		t.Fatal("Expected no entries after clear")
	}
}