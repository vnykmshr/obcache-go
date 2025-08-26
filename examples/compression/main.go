package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vnykmshr/obcache-go/pkg/compression"
	"github.com/vnykmshr/obcache-go/pkg/obcache"
)

// User represents a sample data structure
type User struct {
	ID       int               `json:"id"`
	Name     string            `json:"name"`
	Email    string            `json:"email"`
	Bio      string            `json:"bio"`
	Tags     []string          `json:"tags"`
	Settings map[string]string `json:"settings"`
}

// generateLargeUser creates a user with large data for compression demonstration
func generateLargeUser(id int) *User {
	// Generate a large bio for compression testing
	bio := strings.Repeat("This is a very long user biography that contains lots of repetitive text. ", 50)

	tags := make([]string, 20)
	for i := 0; i < 20; i++ {
		tags[i] = fmt.Sprintf("tag_%d", i)
	}

	settings := make(map[string]string)
	for i := 0; i < 30; i++ {
		settings[fmt.Sprintf("setting_%d", i)] = fmt.Sprintf("This is a configuration value for setting %d with some extra text to make it larger", i)
	}

	return &User{
		ID:       id,
		Name:     fmt.Sprintf("User %d", id),
		Email:    fmt.Sprintf("user%d@example.com", id),
		Bio:      bio,
		Tags:     tags,
		Settings: settings,
	}
}

// generateLargeJSON creates a large JSON string for compression testing
func generateLargeJSON() string {
	data := map[string]any{
		"users": make([]User, 100),
		"metadata": map[string]string{
			"version":   "1.0",
			"generated": time.Now().Format(time.RFC3339),
			"notes":     strings.Repeat("This is a repetitive note that should compress well. ", 100),
		},
	}

	users := data["users"].([]User)
	for i := 0; i < 100; i++ {
		users[i] = *generateLargeUser(i)
	}

	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func main() {
	fmt.Println("🗜️ obcache-go Compression Examples")
	fmt.Println("=====================================")

	// Example 1: Compression disabled (default)
	fmt.Println("\n1. Testing without compression (baseline)")
	noCompressionExample()

	// Example 2: Gzip compression
	fmt.Println("\n2. Testing with Gzip compression")
	gzipCompressionExample()

	// Example 3: Deflate compression
	fmt.Println("\n3. Testing with Deflate compression")
	deflateCompressionExample()

	// Example 4: Compression with different minimum sizes
	fmt.Println("\n4. Testing compression thresholds")
	compressionThresholdExample()

	// Example 5: Performance comparison
	fmt.Println("\n5. Performance comparison")
	performanceComparisonExample()

	fmt.Println("\n✨ All compression examples completed!")
}

func noCompressionExample() {
	// Create cache without compression
	cache, err := obcache.New(obcache.NewDefaultConfig().
		WithMaxEntries(100))
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Store some large data
	largeData := generateLargeJSON()
	fmt.Printf("📊 Original data size: %d bytes\n", len(largeData))

	// Store and retrieve
	cache.Set("large_data", largeData, 5*time.Minute)

	retrieved, found := cache.Get("large_data")
	if !found {
		log.Fatal("Failed to retrieve data")
	}

	fmt.Printf("✅ Data retrieved successfully, size: %d bytes\n", len(retrieved.(string)))
	fmt.Println("   No compression was applied")
}

func gzipCompressionExample() {
	// Create cache with gzip compression
	cache, err := obcache.New(obcache.NewDefaultConfig().
		WithMaxEntries(100).
		WithCompressionEnabled(true).
		WithCompressionAlgorithm(compression.CompressorGzip).
		WithCompressionMinSize(500). // Compress values larger than 500 bytes
		WithCompressionLevel(6))     // Balanced compression level
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Store some large data
	largeData := generateLargeJSON()
	user := generateLargeUser(1)

	fmt.Printf("📊 JSON data size: %d bytes\n", len(largeData))

	// Store and retrieve large JSON
	cache.Set("large_json", largeData, 5*time.Minute)
	cache.Set("large_user", user, 5*time.Minute)

	// Retrieve and verify
	retrievedJSON, found := cache.Get("large_json")
	if !found {
		log.Fatal("Failed to retrieve JSON data")
	}

	retrievedUser, found := cache.Get("large_user")
	if !found {
		log.Fatal("Failed to retrieve user data")
	}

	fmt.Printf("✅ JSON data retrieved successfully, size: %d bytes\n", len(retrievedJSON.(string)))

	// When using JSON serialization, complex types are deserialized as map[string]interface{}
	userMap, ok := retrievedUser.(map[string]interface{})
	if ok {
		fmt.Printf("✅ User data retrieved successfully: %s\n", userMap["name"])
	} else {
		fmt.Printf("✅ User data retrieved successfully: %+v\n", retrievedUser)
	}
	fmt.Printf("   Gzip compression was applied automatically\n")

	// Show stats if cache supports it
	stats := cache.Stats()
	fmt.Printf("📈 Cache stats - Hits: %d, Misses: %d, Hit Rate: %.1f%%\n",
		stats.Hits(), stats.Misses(), stats.HitRate())
}

func deflateCompressionExample() {
	// Create cache with deflate compression
	cache, err := obcache.New(obcache.NewDefaultConfig().
		WithMaxEntries(100).
		WithCompressionEnabled(true).
		WithCompressionAlgorithm(compression.CompressorDeflate).
		WithCompressionMinSize(1000). // Higher threshold
		WithCompressionLevel(9))      // Maximum compression
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test with very large repetitive data that should compress well
	repeatedText := strings.Repeat("This is a test string that repeats many times to demonstrate compression efficiency. ", 1000)
	fmt.Printf("📊 Repetitive text size: %d bytes\n", len(repeatedText))

	cache.Set("repetitive_text", repeatedText, 5*time.Minute)

	retrieved, found := cache.Get("repetitive_text")
	if !found {
		log.Fatal("Failed to retrieve repetitive text")
	}

	fmt.Printf("✅ Text retrieved successfully, size: %d bytes\n", len(retrieved.(string)))
	fmt.Println("   Deflate compression achieved high compression ratio")

	// Verify content integrity
	if retrieved.(string) == repeatedText {
		fmt.Println("✅ Content integrity verified - original and retrieved data match")
	} else {
		log.Fatal("❌ Content mismatch detected!")
	}
}

func compressionThresholdExample() {
	fmt.Println("Testing different compression thresholds...")

	testSizes := []int{100, 500, 1000, 2000}

	for _, minSize := range testSizes {
		fmt.Printf("\n  Testing with minimum size: %d bytes\n", minSize)

		cache, err := obcache.New(obcache.NewDefaultConfig().
			WithMaxEntries(50).
			WithCompressionEnabled(true).
			WithCompressionAlgorithm(compression.CompressorGzip).
			WithCompressionMinSize(minSize))
		if err != nil {
			log.Printf("Failed to create cache: %v", err)
			continue
		}

		// Test with small data (should not be compressed)
		smallData := strings.Repeat("small", 50) // ~250 bytes
		cache.Set("small_data", smallData, time.Minute)

		// Test with large data (should be compressed if above threshold)
		largeData := strings.Repeat("large data for compression testing ", 100) // ~3400 bytes
		cache.Set("large_data", largeData, time.Minute)

		// Retrieve both
		smallRetrieved, _ := cache.Get("small_data")
		largeRetrieved, _ := cache.Get("large_data")

		fmt.Printf("    Small data (250 bytes): %s\n",
			map[bool]string{true: "compressed", false: "not compressed"}[len(smallData) >= minSize])
		fmt.Printf("    Large data (3400 bytes): %s\n",
			map[bool]string{true: "compressed", false: "not compressed"}[len(largeData) >= minSize])

		// Verify data integrity
		if smallRetrieved.(string) == smallData && largeRetrieved.(string) == largeData {
			fmt.Printf("    ✅ Data integrity maintained\n")
		} else {
			fmt.Printf("    ❌ Data integrity failed\n")
		}

		cache.Close()
	}
}

func performanceComparisonExample() {
	fmt.Println("Comparing performance with and without compression...")

	// Generate test data
	testData := generateLargeJSON()
	iterations := 100

	// Test without compression
	fmt.Printf("  Testing %d operations without compression...\n", iterations)
	start := time.Now()

	uncompressedCache, _ := obcache.New(obcache.NewDefaultConfig().WithMaxEntries(200))
	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("test_%d", i)
		uncompressedCache.Set(key, testData, time.Hour)
		_, _ = uncompressedCache.Get(key)
	}
	uncompressedTime := time.Since(start)
	uncompressedCache.Close()

	// Test with compression
	fmt.Printf("  Testing %d operations with gzip compression...\n", iterations)
	start = time.Now()

	compressedCache, _ := obcache.New(obcache.NewDefaultConfig().
		WithMaxEntries(200).
		WithCompressionEnabled(true).
		WithCompressionAlgorithm(compression.CompressorGzip).
		WithCompressionMinSize(1000))

	for i := 0; i < iterations; i++ {
		key := fmt.Sprintf("test_%d", i)
		compressedCache.Set(key, testData, time.Hour)
		_, _ = compressedCache.Get(key)
	}
	compressedTime := time.Since(start)
	compressedCache.Close()

	// Results
	fmt.Printf("\n📊 Performance Results:\n")
	fmt.Printf("  Without compression: %v\n", uncompressedTime)
	fmt.Printf("  With compression:    %v\n", compressedTime)

	ratio := float64(compressedTime) / float64(uncompressedTime)
	fmt.Printf("  Compression overhead: %.2fx\n", ratio)

	if ratio < 1.5 {
		fmt.Println("  ✅ Reasonable compression overhead")
	} else {
		fmt.Println("  ⚠️  High compression overhead - consider adjusting settings")
	}
}
