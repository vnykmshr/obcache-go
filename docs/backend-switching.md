# Backend Switching Guide

ObCache supports multiple storage backends to meet different scalability and deployment requirements. This guide covers how to configure and switch between available backends.

## Supported Backends

### Memory Backend (Default)
- **Best for**: Single-instance applications, development, testing
- **Characteristics**: Fast, low latency, process-local storage
- **Limitations**: Data is lost on restart, not shared across instances

### Redis Backend
- **Best for**: Distributed applications, multiple instances, persistence
- **Characteristics**: Shared across instances, persistent, battle-tested
- **Requirements**: Redis server (v6.0+)

## Configuration Examples

### Memory Backend (Default)

```go
import "github.com/vnykmshr/obcache-go/pkg/obcache"

// Default configuration uses memory backend
config := obcache.NewDefaultConfig().
    WithMaxEntries(10000).
    WithDefaultTTL(30 * time.Minute)

cache, err := obcache.New(config)
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

### Redis Backend - Simple Configuration

```go
// Connect to Redis with default settings
config := obcache.NewRedisConfig("localhost:6379").
    WithDefaultTTL(time.Hour).
    WithRedisKeyPrefix("myapp:")

cache, err := obcache.New(config)
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

### Redis Backend - Advanced Configuration

```go
// Advanced Redis configuration
config := obcache.NewDefaultConfig().
    WithRedisAddr("redis.example.com:6379").
    WithRedisAuth("your-password").
    WithRedisDB(1).
    WithRedisKeyPrefix("myapp:cache:").
    WithDefaultTTL(2 * time.Hour)

cache, err := obcache.New(config)
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

### Redis Backend - Custom Client

```go
import "github.com/redis/go-redis/v9"

// Create custom Redis client with advanced settings
redisClient := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    Password:     "your-password",
    DB:           0,
    PoolSize:     10,
    ReadTimeout:  time.Second * 3,
    WriteTimeout: time.Second * 3,
})

// Use custom client
config := obcache.NewRedisConfigWithClient(redisClient).
    WithRedisKeyPrefix("myapp:").
    WithDefaultTTL(time.Hour)

cache, err := obcache.New(config)
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

## Environment-Based Backend Selection

```go
func createCache() (*obcache.Cache, error) {
    var config *obcache.Config
    
    switch os.Getenv("CACHE_BACKEND") {
    case "redis":
        redisAddr := os.Getenv("REDIS_ADDR")
        if redisAddr == "" {
            redisAddr = "localhost:6379"
        }
        
        config = obcache.NewRedisConfig(redisAddr).
            WithRedisAuth(os.Getenv("REDIS_PASSWORD")).
            WithRedisKeyPrefix(os.Getenv("CACHE_PREFIX"))
            
    default:
        config = obcache.NewDefaultConfig().
            WithMaxEntries(1000)
    }
    
    return obcache.New(config)
}
```

## Backend-Specific Considerations

### Memory Backend

**Advantages:**
- Extremely fast (nanosecond access times)
- No network latency
- No external dependencies

**Limitations:**
- Memory usage grows with cache size
- Data lost on process restart
- Not shared across application instances
- LRU eviction when `MaxEntries` reached

**Best Practices:**
```go
config := obcache.NewDefaultConfig().
    WithMaxEntries(10000).           // Prevent unbounded growth
    WithCleanupInterval(time.Minute) // Regular TTL cleanup
```

### Redis Backend

**Advantages:**
- Shared across multiple instances
- Data persistence options
- Built-in TTL management
- Horizontal scalability

**Considerations:**
- Network latency (typically 1-5ms)
- Redis server dependency
- Serialization overhead
- Network bandwidth usage

**Best Practices:**
```go
config := obcache.NewRedisConfig("localhost:6379").
    WithRedisKeyPrefix("myapp:cache:"). // Avoid key conflicts
    WithDefaultTTL(time.Hour)           // Reasonable default TTL
```

## Migration Between Backends

### Memory to Redis Migration

```go
// 1. Create both caches
memoryCache, _ := obcache.New(obcache.NewDefaultConfig())
redisCache, _ := obcache.New(obcache.NewRedisConfig("localhost:6379"))

// 2. Optional: Migrate existing data
for _, key := range memoryCache.Keys() {
    if value, found := memoryCache.Get(key); found {
        _ = redisCache.Set(key, value, time.Hour) // Set appropriate TTL
    }
}

// 3. Switch to Redis cache
memoryCache.Close()
cache = redisCache
```

### Gradual Migration Strategy

```go
type DualCache struct {
    primary   *obcache.Cache
    secondary *obcache.Cache
}

func (d *DualCache) Get(key string) (any, bool) {
    // Try primary first
    if value, found := d.primary.Get(key); found {
        return value, true
    }
    
    // Fall back to secondary
    if value, found := d.secondary.Get(key); found {
        // Optionally warm primary cache
        _ = d.primary.Set(key, value, time.Hour)
        return value, true
    }
    
    return nil, false
}

func (d *DualCache) Set(key string, value any, ttl time.Duration) error {
    // Write to both
    _ = d.primary.Set(key, value, ttl)
    return d.secondary.Set(key, value, ttl)
}
```

## Performance Characteristics

| Backend | Read Latency | Write Latency | Memory Usage | Persistence |
|---------|-------------|---------------|--------------|-------------|
| Memory  | ~100ns      | ~200ns        | High         | None        |
| Redis   | ~1-5ms      | ~1-5ms        | Low          | Optional    |

## Monitoring and Troubleshooting

### Memory Backend Monitoring

```go
stats := cache.Stats()
fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate())
fmt.Printf("Memory Usage: %d entries\n", stats.KeyCount())
```

### Redis Backend Monitoring

```go
// Check Redis connection
if redisStore, ok := cache.Store().(*redis.Store); ok {
    ctx := context.Background()
    if err := redisStore.Client().Ping(ctx).Err(); err != nil {
        log.Printf("Redis connection error: %v", err)
    }
}

// Monitor Redis memory usage
info, err := redisClient.Info(ctx, "memory").Result()
if err == nil {
    log.Printf("Redis memory info: %s", info)
}
```

## Common Patterns

### Development vs Production

```go
func createCacheForEnvironment(env string) (*obcache.Cache, error) {
    switch env {
    case "development", "test":
        return obcache.New(obcache.NewDefaultConfig().WithMaxEntries(100))
    case "production":
        return obcache.New(obcache.NewRedisConfig(os.Getenv("REDIS_URL")))
    default:
        return nil, fmt.Errorf("unknown environment: %s", env)
    }
}
```

### High Availability Setup

```go
// Redis Cluster or Sentinel setup
clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{"redis1:6379", "redis2:6379", "redis3:6379"},
})

config := obcache.NewRedisConfigWithClient(clusterClient)
cache, err := obcache.New(config)
```

## Best Practices Summary

1. **Choose the right backend**: Memory for speed, Redis for distribution
2. **Set appropriate TTLs**: Prevent unbounded growth
3. **Use key prefixes**: Avoid conflicts in shared Redis instances
4. **Monitor performance**: Track hit rates and latency
5. **Handle failures gracefully**: Implement fallback strategies
6. **Test both backends**: Ensure your application works with either backend
7. **Consider migration paths**: Plan for backend changes as you scale