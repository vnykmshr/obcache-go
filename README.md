# obcache-go

A high-performance, thread-safe caching library for Go with automatic function wrapping, TTL support, and LRU eviction.

[![Go Reference](https://pkg.go.dev/badge/github.com/vnykmshr/obcache-go.svg)](https://pkg.go.dev/github.com/vnykmshr/obcache-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/vnykmshr/obcache-go)](https://goreportcard.com/report/github.com/vnykmshr/obcache-go)

## Features

- **🚀 High Performance**: Built with performance in mind, supporting concurrent access
- **⚡ Function Wrapping**: Automatically cache expensive function calls with Go generics
- **🔒 Thread Safe**: All operations are thread-safe with optimized locking
- **⏰ TTL Support**: Time-based expiration with configurable cleanup intervals
- **📊 LRU Eviction**: Automatic eviction of least recently used entries
- **🎯 Singleflight**: Prevents duplicate concurrent calls to the same function
- **📈 Statistics**: Built-in metrics for hits, misses, evictions, and more
- **🎣 Hooks**: Event callbacks for cache operations (hit, miss, evict, invalidate)
- **🔑 Smart Key Generation**: Automatic key generation from function arguments
- **⚙️ Flexible Configuration**: Extensive customization options

## Installation

```bash
go get github.com/vnykmshr/obcache-go
```

## Quick Start

### Basic Cache Operations

```go
package main

import (
    "fmt"
    "time"
    "github.com/vnykmshr/obcache-go/pkg/obcache"
)

func main() {
    // Create a cache with default configuration
    cache, err := obcache.New(obcache.NewDefaultConfig())
    if err != nil {
        panic(err)
    }

    // Set a value with TTL
    cache.Set("user:123", "John Doe", time.Hour)

    // Get a value
    if value, found := cache.Get("user:123"); found {
        fmt.Println("Found:", value) // Found: John Doe
    }

    // Check cache statistics
    stats := cache.Stats()
    fmt.Printf("Hits: %d, Misses: %d, Hit Rate: %.1f%%\n", 
        stats.Hits(), stats.Misses(), stats.HitRate())
}
```

### Function Wrapping (Recommended)

The most powerful feature of obcache-go is automatic function wrapping:

```go
package main

import (
    "fmt"
    "time"
    "github.com/vnykmshr/obcache-go/pkg/obcache"
)

// Expensive database query
func getUserFromDB(userID int) (string, error) {
    // Simulate expensive operation
    time.Sleep(100 * time.Millisecond)
    return fmt.Sprintf("User-%d", userID), nil
}

func main() {
    cache, _ := obcache.New(obcache.NewDefaultConfig())

    // Wrap the function with caching
    cachedGetUser := obcache.Wrap(cache, getUserFromDB)

    // First call: cache miss, executes function
    start := time.Now()
    user1, err := cachedGetUser(123)
    fmt.Printf("First call: %s (took %v)\n", user1, time.Since(start))

    // Second call: cache hit, returns immediately
    start = time.Now()
    user2, err := cachedGetUser(123)
    fmt.Printf("Second call: %s (took %v)\n", user2, time.Since(start))

    if err != nil {
        panic(err)
    }
}
```

## Configuration

### Custom Configuration

```go
config := obcache.NewDefaultConfig().
    WithMaxEntries(1000).                    // Maximum 1000 entries
    WithDefaultTTL(30 * time.Minute).        // 30-minute default TTL
    WithCleanupInterval(5 * time.Minute).    // Cleanup every 5 minutes
    WithKeyGenFunc(obcache.SimpleKeyFunc).   // Use simple key generation
    WithHooks(&obcache.Hooks{
        OnHit: []obcache.OnHitHook{
            func(key string, value any) {
                fmt.Printf("Cache hit: %s\n", key)
            },
        },
    })

cache, err := obcache.New(config)
```

### Redis Configuration (Distributed Caching)

obcache-go supports Redis as a backend for distributed caching scenarios:

```go
// Simple Redis configuration
config := obcache.NewRedisConfig("localhost:6379").
    WithDefaultTTL(30 * time.Minute).
    WithRedisKeyPrefix("myapp:")

cache, err := obcache.New(config)

// Advanced Redis configuration
client := redis.NewClient(&redis.Options{
    Addr:         "redis-cluster:6379",
    Password:     "secret",
    DB:           1,
    PoolSize:     10,
    MinIdleConns: 5,
})

config := obcache.NewRedisConfigWithClient(client).
    WithRedisKeyPrefix("distributed:").
    WithDefaultTTL(1 * time.Hour)

cache, err := obcache.New(config)

// Or convert existing memory config to Redis
config := obcache.NewDefaultConfig().
    WithRedisAddr("localhost:6379").
    WithRedisAuth("password").
    WithRedisDB(2)
```

**Redis Features:**
- **Distributed Caching**: Share cache across multiple application instances
- **Persistence**: Cache survives application restarts
- **Automatic TTL**: Redis handles expiration automatically  
- **Connection Pooling**: Built-in connection pooling and retry logic
- **Clustering**: Works with Redis clusters and Sentinel

**Note**: Redis adapter uses JSON serialization. For production workloads with complex types, consider implementing custom serialization.

### Function Wrapping Options

```go
// Custom TTL for this function
wrapped := obcache.Wrap(cache, expensiveFunc, 
    obcache.WithTTL(time.Hour),
)

// Custom key generation
wrapped := obcache.Wrap(cache, expensiveFunc,
    obcache.WithKeyFunc(func(args []any) string {
        return fmt.Sprintf("custom:%v", args[0])
    }),
)

// Disable caching (useful for testing)
wrapped := obcache.Wrap(cache, expensiveFunc, 
    obcache.WithoutCache(),
)
```

## Advanced Usage

### Working with Hooks

Hooks provide visibility into cache operations:

```go
hooks := &obcache.Hooks{
    OnHit: []obcache.OnHitHook{
        func(key string, value any) {
            fmt.Printf("Cache hit: %s\n", key)
        },
    },
    OnMiss: []obcache.OnMissHook{
        func(key string) {
            fmt.Printf("Cache miss: %s\n", key)
        },
    },
    OnEvict: []obcache.OnEvictHook{
        func(key string, value any, reason obcache.EvictReason) {
            fmt.Printf("Evicted: %s (reason: %v)\n", key, reason)
        },
    },
    OnInvalidate: []obcache.OnInvalidateHook{
        func(key string) {
            fmt.Printf("Invalidated: %s\n", key)
        },
    },
}

config := obcache.NewDefaultConfig().WithHooks(hooks)
cache, _ := obcache.New(config)
```

### Complex Function Signatures

obcache-go supports any function signature with Go generics:

```go
// Functions with multiple parameters and return values
func complexQuery(userID int, includeDeleted bool) ([]User, int, error) {
    // ... implementation
}

wrappedQuery := obcache.Wrap(cache, complexQuery)
users, count, err := wrappedQuery(123, false)

// Functions with no error return
func computeHash(data string) string {
    // ... implementation
}

wrappedHash := obcache.Wrap(cache, computeHash)
hash := wrappedHash("some data")
```

### Convenience Functions

For better type safety and readability:

```go
// Type-safe wrappers
wrapped := obcache.WrapFunc1WithError(cache, 
    func(id int) (User, error) {
        return getUserFromDB(id)
    },
)

user, err := wrapped(123)
```

### Cache Warmup

Pre-populate the cache with known values:

```go
// Warm up the cache
cache.Warmup("user:123", User{ID: 123, Name: "John"})

// Warm up with custom TTL
cache.WarmupWithTTL("config:app", appConfig, 24*time.Hour)
```

## API Reference

### Cache Operations

```go
// Create cache
cache, err := obcache.New(config)

// Basic operations
err = cache.Set(key string, value any, ttl time.Duration)
value, found := cache.Get(key string)
err = cache.Invalidate(key string)
err = cache.Warmup(key string, value any)
err = cache.WarmupWithTTL(key string, value any, ttl time.Duration)

// Statistics
stats := cache.Stats()
hits := stats.Hits()
misses := stats.Misses()
hitRate := stats.HitRate()
evictions := stats.Evictions()
keyCount := stats.KeyCount()
inFlight := stats.InFlight()
```

### Function Wrapping

```go
// Generic wrapping (recommended)
wrapped := obcache.Wrap(cache, function, options...)

// Type-specific wrappers
wrapped := obcache.WrapFunc1(cache, func(T) R)
wrapped := obcache.WrapFunc2(cache, func(T1, T2) R)
wrapped := obcache.WrapFunc1WithError(cache, func(T) (R, error))

// Convenience functions
wrapped := obcache.WrapSimple(cache, func(T) R)
wrapped := obcache.WrapWithError(cache, func(T) (R, error))
```

### Configuration Options

```go
// Memory cache configuration
config := obcache.NewDefaultConfig()
config.WithMaxEntries(n int)              // Set max entries (default: 1000)
config.WithDefaultTTL(d time.Duration)    // Set default TTL (default: 5min)
config.WithCleanupInterval(d time.Duration) // Set cleanup interval (default: 1min)
config.WithKeyGenFunc(f KeyGenFunc)       // Set key generation function
config.WithHooks(h *Hooks)                // Set event hooks

// Redis cache configuration
config := obcache.NewRedisConfig(addr string)           // Create Redis config
config := obcache.NewRedisConfigWithClient(client)      // Use existing client
config.WithRedisAddr(addr string)         // Set Redis address
config.WithRedisAuth(password string)     // Set Redis password
config.WithRedisDB(db int)                // Set Redis database
config.WithRedisKeyPrefix(prefix string)  // Set key prefix
```

### Wrap Options

```go
obcache.WithTTL(d time.Duration)          // Custom TTL for wrapped function
obcache.WithKeyFunc(f KeyGenFunc)         // Custom key generation
obcache.WithoutCache()                    // Disable caching
```

## Performance

obcache-go is designed for high performance:

- **Concurrent Access**: Optimized locking strategy for concurrent reads/writes
- **Memory Efficient**: LRU eviction prevents unbounded memory growth
- **Singleflight**: Prevents thundering herd problems
- **Fast Key Generation**: Efficient key generation with hashing for long keys

### Benchmarks

```go
// Run benchmarks
go test -bench=. ./pkg/obcache/...
```

Example benchmark results:
```
BenchmarkCacheGet-8           	10000000	       150 ns/op
BenchmarkCacheSet-8           	 5000000	       280 ns/op
BenchmarkWrapFunction-8       	 3000000	       450 ns/op
BenchmarkDefaultKeyFunc-8     	 2000000	       650 ns/op
```

## Best Practices

### 1. Choose Appropriate TTL

```go
// Short TTL for frequently changing data
userCache := obcache.Wrap(cache, getUserData, 
    obcache.WithTTL(5*time.Minute))

// Long TTL for stable data
configCache := obcache.Wrap(cache, getConfig, 
    obcache.WithTTL(time.Hour))
```

### 2. Monitor Cache Performance

```go
// Regularly check cache statistics
stats := cache.Stats()
if stats.HitRate() < 50.0 {
    log.Warn("Low cache hit rate: %.1f%%", stats.HitRate())
}
```

### 3. Use Appropriate Key Generation

```go
// Simple keys for better performance
wrapped := obcache.Wrap(cache, func, 
    obcache.WithKeyFunc(obcache.SimpleKeyFunc))

// Default keys for complex types
wrapped := obcache.Wrap(cache, func, 
    obcache.WithKeyFunc(obcache.DefaultKeyFunc))
```

### 4. Handle Errors Appropriately

```go
// Errors are not cached by default (good for transient errors)
user, err := cachedGetUser(123)
if err != nil {
    // Handle error - function will be retried on next call
}
```

## Testing

The library includes comprehensive tests:

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

## Examples

See the [examples/](examples/) directory for complete working examples:

- [Basic Usage](examples/basic/main.go) - Simple cache operations
- [Advanced Features](examples/advanced/main.go) - Context-aware hooks and service integration
- [Redis Distributed Caching](examples/redis-cache/main.go) - Redis backend for distributed scenarios

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
git clone https://github.com/vnykmshr/obcache-go.git
cd obcache-go
go mod download
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.

---

**Made with ❤️ for the Go community**