# API Reference

Complete reference for the obcache-go library API.

## Package obcache

```go
import "github.com/vnykmshr/obcache-go/pkg/obcache"
```

---

## Core Types

### Cache

The main cache implementation with LRU eviction, TTL support, and comprehensive features.

```go
type Cache struct {
    // contains filtered or unexported fields
}
```

#### func New

```go
func New(config *Config) (*Cache, error)
```

Creates a new Cache instance with the given configuration.

**Example:**
```go
config := obcache.NewDefaultConfig().WithMaxEntries(1000)
cache, err := obcache.New(config)
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

---

### Config

Configuration for cache instances.

```go
type Config struct {
    StoreType                StoreType
    MaxEntries              int
    DefaultTTL              time.Duration
    CleanupInterval         time.Duration
    
    // Eviction Strategy
    EvictionType            eviction.EvictionType
    
    // Redis Configuration
    RedisAddr               string
    RedisPassword           string
    RedisDB                 int
    RedisKeyPrefix          string
    RedisClient             *redis.Client
    
    // Hooks
    Hooks                   *Hooks
    
    // Compression
    CompressionAlgorithm    string
    CompressionThreshold    int
    
    // Metrics
    MetricsExporter        metrics.Exporter
    MetricsLabels          metrics.Labels
    MetricsReportInterval  time.Duration
}
```

#### func NewDefaultConfig

```go
func NewDefaultConfig() *Config
```

Creates a new Config with sensible defaults:
- Memory store with 1000 max entries
- 1 hour default TTL
- 10 minute cleanup interval
- LRU eviction strategy

#### func NewRedisConfig

```go
func NewRedisConfig(addr string) *Config
```

Creates a new Config for Redis backend with the specified address.

**Example:**
```go
config := obcache.NewRedisConfig("localhost:6379").
    WithRedisAuth("password").
    WithRedisDB(1).
    WithDefaultTTL(30*time.Minute)
```

#### func NewRedisConfigWithClient

```go
func NewRedisConfigWithClient(client *redis.Client) *Config
```

Creates a new Config for Redis backend with a custom Redis client.

---

## Configuration Methods

### Store Configuration

#### func (*Config) WithMaxEntries

```go
func (c *Config) WithMaxEntries(maxEntries int) *Config
```

Sets the maximum number of entries for memory stores.

#### func (*Config) WithDefaultTTL

```go
func (c *Config) WithDefaultTTL(ttl time.Duration) *Config
```

Sets the default TTL for cache entries.

#### func (*Config) WithCleanupInterval

```go
func (c *Config) WithCleanupInterval(interval time.Duration) *Config
```

Sets the interval for TTL cleanup operations.

### Eviction Strategy Configuration

#### func (*Config) WithEvictionType

```go
func (c *Config) WithEvictionType(evictionType eviction.EvictionType) *Config
```

Sets the eviction strategy type.

#### func (*Config) WithLRUEviction

```go
func (c *Config) WithLRUEviction() *Config
```

Configures LRU (Least Recently Used) eviction strategy.

#### func (*Config) WithLFUEviction

```go
func (c *Config) WithLFUEviction() *Config
```

Configures LFU (Least Frequently Used) eviction strategy.

#### func (*Config) WithFIFOEviction

```go
func (c *Config) WithFIFOEviction() *Config
```

Configures FIFO (First In, First Out) eviction strategy.

### Redis Configuration

#### func (*Config) WithRedisAddr

```go
func (c *Config) WithRedisAddr(addr string) *Config
```

Sets the Redis server address.

#### func (*Config) WithRedisAuth

```go
func (c *Config) WithRedisAuth(password string) *Config
```

Sets the Redis authentication password.

#### func (*Config) WithRedisDB

```go
func (c *Config) WithRedisDB(db int) *Config
```

Sets the Redis database number.

#### func (*Config) WithRedisKeyPrefix

```go
func (c *Config) WithRedisKeyPrefix(prefix string) *Config
```

Sets a prefix for all Redis keys.

#### func (*Config) WithRedisClient

```go
func (c *Config) WithRedisClient(client *redis.Client) *Config
```

Sets a custom Redis client.

### Hooks Configuration

#### func (*Config) WithHooks

```go
func (c *Config) WithHooks(hooks *Hooks) *Config
```

Sets the hooks for cache events.

### Compression Configuration

#### func (*Config) WithCompression

```go
func (c *Config) WithCompression(algorithm string) *Config
```

Enables compression with the specified algorithm ("gzip", "lz4", etc.).

#### func (*Config) WithCompressionThreshold

```go
func (c *Config) WithCompressionThreshold(threshold int) *Config
```

Sets the size threshold for compression (in bytes).

### Metrics Configuration

#### func (*Config) WithMetricsExporter

```go
func (c *Config) WithMetricsExporter(exporter metrics.Exporter) *Config
```

Sets the metrics exporter.

#### func (*Config) WithMetricsLabels

```go
func (c *Config) WithMetricsLabels(labels metrics.Labels) *Config
```

Sets the metrics labels.

---

## Cache Operations

### Basic Operations

#### func (*Cache) Get

```go
func (c *Cache) Get(key string, options ...CacheOption) (any, bool)
```

Retrieves a value from the cache.

**Returns:**
- `any`: The cached value
- `bool`: Whether the key was found

**Example:**
```go
if value, found := cache.Get("user:123"); found {
    user := value.(*User)
    fmt.Printf("Found user: %s\n", user.Name)
}
```

#### func (*Cache) Set

```go
func (c *Cache) Set(key string, value any, ttl time.Duration, options ...CacheOption) error
```

Stores a value in the cache with the specified TTL.

**Parameters:**
- `key`: Cache key
- `value`: Value to store (any Go type)
- `ttl`: Time to live (0 = use default TTL)

**Example:**
```go
user := &User{Name: "John", Age: 30}
err := cache.Set("user:123", user, time.Hour)
if err != nil {
    log.Printf("Failed to set cache: %v", err)
}
```

#### func (*Cache) Invalidate

```go
func (c *Cache) Invalidate(key string, options ...CacheOption) error
```

Removes a specific key from the cache.

#### func (*Cache) InvalidateAll

```go
func (c *Cache) InvalidateAll(options ...CacheOption) error
```

Removes all entries from the cache.

### Advanced Operations

#### func (*Cache) GetWithTTL

```go
func (c *Cache) GetWithTTL(key string, options ...CacheOption) (value any, ttl time.Duration, found bool)
```

Retrieves a value along with its remaining TTL.

**Example:**
```go
value, remainingTTL, found := cache.GetWithTTL("user:123")
if found {
    fmt.Printf("Key expires in: %v\n", remainingTTL)
}
```

#### func (*Cache) Has

```go
func (c *Cache) Has(key string) bool
```

Checks if a key exists in the cache without retrieving the value.

#### func (*Cache) Keys

```go
func (c *Cache) Keys() []string
```

Returns all keys currently in the cache.

**Note:** Use with caution on large caches as this operation can be expensive.

#### func (*Cache) Len

```go
func (c *Cache) Len() int
```

Returns the current number of entries in the cache.

### Warmup Operations

#### func (*Cache) Warmup

```go
func (c *Cache) Warmup(key string, value any) error
```

Pre-populates the cache with a value using the default TTL.

#### func (*Cache) WarmupWithTTL

```go
func (c *Cache) WarmupWithTTL(key string, value any, ttl time.Duration) error
```

Pre-populates the cache with a value using a specific TTL.

### Maintenance Operations

#### func (*Cache) Cleanup

```go
func (c *Cache) Cleanup() int
```

Manually triggers cleanup of expired entries. Returns the number of entries removed.

#### func (*Cache) Close

```go
func (c *Cache) Close() error
```

Closes the cache and cleans up resources. Always call this when done with the cache.

### Statistics

#### func (*Cache) Stats

```go
func (c *Cache) Stats() *Stats
```

Returns comprehensive cache statistics.

**Example:**
```go
stats := cache.Stats()
fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate())
fmt.Printf("Total Hits: %d\n", stats.Hits())
fmt.Printf("Total Misses: %d\n", stats.Misses())
```

---

## Function Wrapping

### Core Wrapping Function

#### func Wrap

```go
func Wrap[T any](cache *Cache, fn T, options ...WrapOption) T
```

Wraps any function to automatically cache its results. Uses Go generics for type safety.

**Example:**
```go
// Original function
getUserFromDB := func(userID int) (*User, error) {
    return database.GetUser(userID)
}

// Wrap it
cachedGetUser := obcache.Wrap(cache, getUserFromDB,
    obcache.WithTTL(time.Hour),
    obcache.WithKeyFunc(func(args []any) string {
        return fmt.Sprintf("user:%d", args[0].(int))
    }))

// Use it like the original function
user, err := cachedGetUser(123)
```

### Specialized Wrapping Functions

#### func WrapSimple

```go
func WrapSimple[T any, R any](cache *Cache, fn func(T) R, options ...WrapOption) func(T) R
```

Wraps a function with one parameter and one return value.

#### func WrapWithError

```go
func WrapWithError[T any, R any](cache *Cache, fn func(T) (R, error), options ...WrapOption) func(T) (R, error)
```

Wraps a function with one parameter and return value + error.

#### func WrapFunc0

```go
func WrapFunc0[R any](cache *Cache, fn func() R, options ...WrapOption) func() R
```

Wraps a function with no parameters.

#### func WrapFunc1

```go
func WrapFunc1[T any, R any](cache *Cache, fn func(T) R, options ...WrapOption) func(T) R
```

Wraps a function with one parameter.

#### func WrapFunc2

```go
func WrapFunc2[T1, T2, R any](cache *Cache, fn func(T1, T2) R, options ...WrapOption) func(T1, T2) R
```

Wraps a function with two parameters.

### Wrap Options

#### func WithTTL

```go
func WithTTL(ttl time.Duration) WrapOption
```

Sets the TTL for cached function results.

#### func WithKeyFunc

```go
func WithKeyFunc(keyFunc KeyGenFunc) WrapOption
```

Sets a custom key generation function.

**Example:**
```go
wrapped := obcache.Wrap(cache, expensiveFunc,
    obcache.WithKeyFunc(func(args []any) string {
        return fmt.Sprintf("expensive:%v", args)
    }))
```

#### func WithErrorCaching

```go
func WithErrorCaching() WrapOption
```

Enables caching of error results.

#### func WithErrorTTL

```go
func WithErrorTTL(ttl time.Duration) WrapOption
```

Sets a separate TTL for cached errors.

#### func WithoutCache

```go
func WithoutCache() WrapOption
```

Disables caching (useful for testing or conditional caching).

---

## Key Generation

### KeyGenFunc

```go
type KeyGenFunc func(args []any) string
```

Function type for generating cache keys from function arguments.

#### func DefaultKeyFunc

```go
func DefaultKeyFunc(args []any) string
```

Default key generation using JSON serialization. Handles most Go types including structs, maps, and slices.

#### func SimpleKeyFunc

```go
func SimpleKeyFunc(args []any) string
```

Simple key generation using fmt.Sprintf. Faster but less reliable for complex types.

**Example:**
```go
// Custom key function for specific use case
userKeyFunc := func(args []any) string {
    userID := args[0].(int)
    includeProfile := args[1].(bool)
    return fmt.Sprintf("user:%d:profile:%t", userID, includeProfile)
}

wrapped := obcache.Wrap(cache, getUserWithProfile, 
    obcache.WithKeyFunc(userKeyFunc))
```

---

## Statistics

### Stats

```go
type Stats struct {
    // contains filtered or unexported fields
}
```

Comprehensive cache statistics.

#### func (*Stats) Hits

```go
func (s *Stats) Hits() int64
```

Returns the total number of cache hits.

#### func (*Stats) Misses

```go
func (s *Stats) Misses() int64
```

Returns the total number of cache misses.

#### func (*Stats) HitRate

```go
func (s *Stats) HitRate() float64
```

Returns the cache hit rate as a percentage (0-100).

#### func (*Stats) Evictions

```go
func (s *Stats) Evictions() int64
```

Returns the total number of evictions.

#### func (*Stats) KeyCount

```go
func (s *Stats) KeyCount() int64
```

Returns the current number of keys in the cache.

---

## Hooks System

### Hooks

```go
type Hooks struct {
    OnHit         []OnHitHook
    OnMiss        []OnMissHook
    OnEvict       []OnEvictHook
    OnInvalidate  []OnInvalidateHook
    // ... additional hook types
}
```

Event hooks for cache operations.

#### Hook Types

```go
type OnHitHook func(key string, value any)
type OnMissHook func(key string)
type OnEvictHook func(key string, value any, reason EvictReason)
type OnInvalidateHook func(key string)
```

#### func (*Hooks) AddOnHit

```go
func (h *Hooks) AddOnHit(hook OnHitHook)
```

Adds a cache hit event handler.

#### func (*Hooks) AddOnMiss

```go
func (h *Hooks) AddOnMiss(hook OnMissHook)
```

Adds a cache miss event handler.

#### func (*Hooks) AddOnEvict

```go
func (h *Hooks) AddOnEvict(hook OnEvictHook)
```

Adds an eviction event handler.

#### func (*Hooks) AddOnInvalidate

```go
func (h *Hooks) AddOnInvalidate(hook OnInvalidateHook)
```

Adds an invalidation event handler.

### Context-Aware Hooks

Advanced hooks that receive context and arguments:

```go
type OnHitHookCtx func(ctx context.Context, key string, value any, args []any)
type OnMissHookCtx func(ctx context.Context, key string, args []any)
```

#### func (*Hooks) AddOnHitCtx

```go
func (h *Hooks) AddOnHitCtx(hook OnHitHookCtx)
```

#### func (*Hooks) AddOnMissCtx

```go
func (h *Hooks) AddOnMissCtx(hook OnMissHookCtx)
```

### Priority Hooks

Hooks with execution priority:

#### func (*Hooks) AddOnHitWithPriority

```go
func (h *Hooks) AddOnHitWithPriority(hook OnHitHook, priority HookPriority)
```

#### Hook Priorities

```go
type HookPriority int

const (
    HookPriorityLow    HookPriority = 1
    HookPriorityMedium HookPriority = 5
    HookPriorityHigh   HookPriority = 10
)
```

### Conditional Hooks

Hooks that only execute when conditions are met:

#### func (*Hooks) AddOnHitCtxIf

```go
func (h *Hooks) AddOnHitCtxIf(hook OnHitHookCtx, condition HookCondition)
```

#### Condition Functions

```go
type HookCondition func(ctx context.Context, key string, args []any) bool
```

#### func KeyPrefixCondition

```go
func KeyPrefixCondition(prefix string) HookCondition
```

Creates a condition that matches keys with a specific prefix.

#### func ContextValueCondition

```go
func ContextValueCondition(contextKey, expectedValue any) HookCondition
```

Creates a condition that matches context values.

#### func AndCondition

```go
func AndCondition(conditions ...HookCondition) HookCondition
```

Combines multiple conditions with logical AND.

#### func OrCondition

```go
func OrCondition(conditions ...HookCondition) HookCondition
```

Combines multiple conditions with logical OR.

---

## Eviction

### EvictReason

```go
type EvictReason string

const (
    EvictReasonCapacity EvictReason = "capacity"
    EvictReasonTTL      EvictReason = "ttl"
    EvictReasonLRU      EvictReason = "lru"
)
```

Reasons for cache entry eviction.

### Eviction Types

```go
type EvictionType string

const (
    LRU  EvictionType = "lru"   // Least Recently Used
    LFU  EvictionType = "lfu"   // Least Frequently Used  
    FIFO EvictionType = "fifo"  // First In, First Out
)
```

---

## Store Types

### StoreType

```go
type StoreType string

const (
    StoreTypeMemory StoreType = "memory"
    StoreTypeRedis  StoreType = "redis"
)
```

Available backend store types.

---

## Context Support

### Cache Options

#### func WithContext

```go
func WithContext(ctx context.Context) CacheOption
```

Adds context to cache operations.

#### func WithArgs

```go
func WithArgs(args []any) CacheOption
```

Adds argument metadata to cache operations.

### Context Utilities

#### func CacheTagsFromContext

```go
func CacheTagsFromContext(ctx context.Context) []string
```

Extracts cache tags from context.

#### func WithCacheTags

```go
func WithCacheTags(ctx context.Context, tags []string) context.Context
```

Adds cache tags to context.

**Example:**
```go
ctx := obcache.WithCacheTags(context.Background(), []string{"user", "profile"})
value, found := cache.Get("user:123", obcache.WithContext(ctx))
```

---

## Debug and Monitoring

### Debug Handler

#### func (*Cache) DebugHandler

```go
func (c *Cache) DebugHandler() http.Handler
```

Returns an HTTP handler for cache debugging and inspection.

**Example:**
```go
http.Handle("/debug/cache", cache.DebugHandler())
log.Fatal(http.ListenAndServe(":8080", nil))
```

#### func (*Cache) NewDebugServer

```go
func (c *Cache) NewDebugServer(addr string) *http.Server
```

Creates a standalone HTTP server for cache debugging.

---

## Error Handling

### Common Errors

The library returns standard Go errors. Common error scenarios:

```go
// Cache creation errors
cache, err := obcache.New(config)
if err != nil {
    // Handle configuration or backend connection errors
}

// Operation errors  
err := cache.Set("key", "value", time.Hour)
if err != nil {
    // Handle storage errors (e.g., Redis connection issues)
}

// Resource cleanup errors
err := cache.Close()
if err != nil {
    // Handle cleanup errors
}
```

---

## Thread Safety

All cache operations are thread-safe and can be called concurrently from multiple goroutines. The library uses appropriate locking mechanisms internally.

```go
// Safe to call from multiple goroutines
go func() {
    cache.Set("key1", "value1", time.Hour)
}()

go func() {
    value, found := cache.Get("key1")
    // ... use value
}()
```

---

## Best Practices

### 1. Resource Management

```go
// Always close caches
cache, err := obcache.New(config)
if err != nil {
    return err
}
defer cache.Close() // Important!
```

### 2. Error Handling

```go
// Check all errors
if err := cache.Set("key", value, time.Hour); err != nil {
    log.Printf("Cache set failed: %v", err)
    // Handle fallback logic
}
```

### 3. Key Design

```go
// Use structured keys
userKey := fmt.Sprintf("user:%d:profile", userID)
sessionKey := fmt.Sprintf("session:%s:data", sessionID)

// Avoid special characters
// Good: "user:123:profile"
// Bad: "user/123/profile with spaces"
```

### 4. TTL Strategy

```go
// Short TTL for frequently changing data
cache.Set("stock:AAPL", price, 5*time.Second)

// Long TTL for stable data
cache.Set("user:profile", profile, 24*time.Hour)

// Use appropriate defaults
config := obcache.NewDefaultConfig().
    WithDefaultTTL(time.Hour) // Reasonable default
```

### 5. Function Wrapping

```go
// Wrap pure functions (no side effects)
cachedGetUser := obcache.Wrap(cache, getUserFromDB)

// Don't wrap functions with side effects
// Bad: wrapping a function that sends emails
```

This API reference covers all public APIs in the obcache-go library. For more examples and use cases, see the [examples directory](../examples/) and the [migration guide](migration-guide.md).