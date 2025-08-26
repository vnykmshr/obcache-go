// Package obcache provides a high-performance, thread-safe, in-memory cache with TTL support,
// LRU eviction, function memoization, and advanced hook system for observability.
//
// # Overview
//
// obcache is designed for high-throughput applications requiring fast, reliable caching
// with comprehensive observability and flexible configuration options. It supports both
// direct cache operations and transparent function memoization through its Wrap functionality.
//
// # Key Features
//
//   - Thread-safe concurrent access with minimal lock contention
//   - Time-to-live (TTL) expiration with automatic cleanup
//   - LRU (Least Recently Used) eviction when capacity limits are reached
//   - Function memoization with customizable key generation
//   - Advanced hook system with priority-based and conditional execution
//   - Built-in statistics and performance monitoring
//   - Redis backend support for distributed caching
//   - Compression support for large values
//   - Prometheus and OpenTelemetry integration
//   - Singleflight pattern to prevent cache stampedes
//
// # Basic Usage
//
// Create a cache and perform basic operations:
//
//	cache, err := obcache.New(obcache.NewDefaultConfig())
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Store a value with 1-hour TTL
//	err = cache.Set("user:123", userData, time.Hour)
//	if err != nil {
//	    log.Printf("Failed to set cache: %v", err)
//	}
//
//	// Retrieve a value
//	value, found := cache.Get("user:123")
//	if found {
//	    user := value.(UserData)
//	    fmt.Printf("Found user: %+v\n", user)
//	}
//
//	// Check statistics
//	stats := cache.Stats()
//	fmt.Printf("Hit rate: %.2f%%\n", stats.HitRate())
//
// # Function Memoization
//
// Cache expensive function calls automatically:
//
//	// Original expensive function
//	func fetchUser(userID int) (*User, error) {
//	    // Expensive database query
//	    return queryDatabase(userID)
//	}
//
//	// Wrap with caching
//	cache, _ := obcache.New(obcache.NewDefaultConfig())
//	cachedFetchUser := obcache.Wrap(cache, fetchUser, obcache.WithTTL(5*time.Minute))
//
//	// Use exactly like the original function - caching is transparent
//	user1, err := cachedFetchUser(123)  // Database query
//	user2, err := cachedFetchUser(123)  // Cache hit
//
// # Configuration
//
// Customize cache behavior with fluent configuration:
//
//	config := obcache.NewDefaultConfig().
//	    WithMaxEntries(10000).
//	    WithDefaultTTL(30*time.Minute).
//	    WithCleanupInterval(5*time.Minute).
//	    WithKeyGenFunc(obcache.SimpleKeyFunc)
//
//	cache, err := obcache.New(config)
//
// # Advanced Hook System
//
// Monitor cache operations with priority-based and conditional hooks:
//
//	hooks := &obcache.Hooks{}
//
//	// High priority: metrics collection (executes first)
//	hooks.AddOnHitWithPriority(func(key string, value any) {
//	    metrics.IncrementCounter("cache.hits")
//	}, obcache.HookPriorityHigh)
//
//	// Low priority: logging (executes after metrics)
//	hooks.AddOnHitWithPriority(func(key string, value any) {
//	    log.Printf("Cache hit: %s", key)
//	}, obcache.HookPriorityLow)
//
//	// Conditional hook: only for specific key patterns
//	hooks.AddOnMissCtxIf(func(ctx context.Context, key string, args []any) {
//	    alerts.SendAlert("Critical cache miss: " + key)
//	}, obcache.AndCondition(
//	    obcache.KeyPrefixCondition("critical:"),
//	    obcache.ContextValueCondition("env", "production"),
//	))
//
//	cache, _ := obcache.New(obcache.NewDefaultConfig().WithHooks(hooks))
//
// # Redis Backend
//
// Use Redis for distributed caching:
//
//	config := obcache.NewRedisConfig("localhost:6379").
//	    WithDefaultTTL(time.Hour).
//	    WithHooks(observabilityHooks)
//
//	cache, err := obcache.New(config)
//	// All operations now use Redis instead of local memory
//
// # Compression
//
// Enable compression for large values:
//
//	config := obcache.NewDefaultConfig().
//	    WithCompression(obcache.CompressionGzip, 1024) // Compress values > 1KB
//
//	cache, err := obcache.New(config)
//
// # Metrics Integration
//
// Export metrics to Prometheus:
//
//	import "github.com/vnykmshr/obcache-go/pkg/metrics"
//
//	prometheusExporter := metrics.NewPrometheusExporter("my_app")
//	config := obcache.NewDefaultConfig().
//	    WithMetricsExporter(prometheusExporter, map[string]string{
//	        "service": "user-api",
//	        "version": "1.0.0",
//	    })
//
//	cache, _ := obcache.New(config)
//
// # Performance Considerations
//
//   - Use appropriate cache sizes based on available memory
//   - Set reasonable TTL values to balance freshness with performance
//   - Consider using Redis backend for multi-instance deployments
//   - Enable compression for large values to reduce memory usage
//   - Use hooks judiciously to avoid performance overhead
//   - Monitor hit rates and adjust cache policies accordingly
//
// # Thread Safety
//
// All cache operations are thread-safe and can be called concurrently from multiple
// goroutines without additional synchronization. The cache uses fine-grained locking
// and atomic operations to minimize contention.
//
// # Error Handling
//
// The cache is designed to degrade gracefully:
//   - Set operations may fail due to capacity or backend issues
//   - Get operations never fail - they return (nil, false) for missing/error cases
//   - Hook execution errors are logged but don't affect cache operations
//   - Backend connectivity issues fall back to cache misses where possible
//
// # Best Practices
//
//   - Use meaningful cache keys with consistent naming patterns
//   - Set appropriate TTL values based on data freshness requirements
//   - Monitor cache performance using built-in statistics
//   - Use function wrapping for transparent caching of expensive operations
//   - Implement proper error handling for critical cache operations
//   - Use hooks for observability and debugging, not business logic
//   - Test cache behavior under various load conditions
//
// # Examples
//
// See the examples directory for complete, runnable examples including:
//   - Basic cache usage patterns
//   - Advanced hook configurations
//   - Redis integration
//   - Metrics collection
//   - Compression usage
//
// For more detailed documentation and examples, visit:
// https://github.com/vnykmshr/obcache-go
package obcache