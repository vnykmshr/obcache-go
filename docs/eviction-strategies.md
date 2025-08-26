# Eviction Strategies Guide

ObCache supports multiple eviction strategies to handle cache capacity limits effectively. This guide covers the available strategies, their characteristics, and usage patterns.

## Supported Eviction Strategies

### LRU (Least Recently Used) - Default

**Best for**: General-purpose caching, temporal locality access patterns

**Characteristics**:
- Evicts the least recently accessed entry when capacity is reached
- Maintains access order, promoting frequently accessed items
- O(1) operations for get/set/evict
- Memory efficient using linked hash map structure

**Use Cases**:
- Web page caching
- Database query result caching
- General application caching where recent access indicates future access

```go
// Default configuration uses LRU
config := obcache.NewDefaultConfig().WithMaxEntries(1000)

// Explicitly configure LRU
config := obcache.NewDefaultConfig().
    WithMaxEntries(1000).
    WithLRUEviction()
```

### LFU (Least Frequently Used)

**Best for**: Data with clear frequency patterns, analytical workloads

**Characteristics**:
- Evicts the least frequently accessed entry when capacity is reached
- Maintains frequency counters for each entry
- Better for workloads with stable access patterns
- Higher memory overhead due to frequency tracking

**Use Cases**:
- Analytics caching (popular reports, dashboards)
- Content delivery (popular files, images)
- Scientific computing (frequently used datasets)

```go
config := obcache.NewDefaultConfig().
    WithMaxEntries(1000).
    WithLFUEviction()

cache, err := obcache.New(config)
```

### FIFO (First In, First Out)

**Best for**: Simple eviction needs, streaming data, time-based ordering

**Characteristics**:
- Evicts the oldest entry regardless of access patterns
- Maintains insertion order only
- Lower computational overhead
- Predictable eviction behavior

**Use Cases**:
- Log message caching
- Event stream processing
- Simple buffer implementations
- When access patterns don't matter

```go
config := obcache.NewDefaultConfig().
    WithMaxEntries(1000).
    WithFIFOEviction()

cache, err := obcache.New(config)
```

## Configuration Examples

### Basic Configuration

```go
import "github.com/vnykmshr/obcache-go/pkg/obcache"

// LRU (default)
cache, err := obcache.New(obcache.NewDefaultConfig().WithMaxEntries(500))

// LFU
cache, err := obcache.New(
    obcache.NewDefaultConfig().
        WithMaxEntries(500).
        WithLFUEviction(),
)

// FIFO
cache, err := obcache.New(
    obcache.NewDefaultConfig().
        WithMaxEntries(500).
        WithFIFOEviction(),
)
```

### Advanced Configuration with TTL and Cleanup

```go
config := obcache.NewDefaultConfig().
    WithMaxEntries(1000).
    WithLFUEviction().
    WithDefaultTTL(30 * time.Minute).
    WithCleanupInterval(5 * time.Minute)

cache, err := obcache.New(config)
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

### Environment-Based Strategy Selection

```go
func createCacheWithStrategy() (*obcache.Cache, error) {
    config := obcache.NewDefaultConfig().WithMaxEntries(1000)
    
    switch os.Getenv("CACHE_EVICTION_STRATEGY") {
    case "lfu":
        config = config.WithLFUEviction()
    case "fifo":
        config = config.WithFIFOEviction()
    default:
        config = config.WithLRUEviction() // default
    }
    
    return obcache.New(config)
}
```

## Performance Characteristics

| Strategy | Get Operation | Set Operation | Memory Overhead | Use Case Fit |
|----------|---------------|---------------|-----------------|--------------|
| LRU      | O(1)          | O(1)          | Low             | General purpose |
| LFU      | O(1)          | O(1)          | Medium          | Frequency-based |
| FIFO     | O(1)          | O(1)          | Low             | Simple/streaming |

## Eviction Strategy with Function Wrapping

```go
// Configure cache with LFU for analytics function
config := obcache.NewDefaultConfig().
    WithMaxEntries(100).
    WithLFUEviction()

cache, err := obcache.New(config)
if err != nil {
    log.Fatal(err)
}

// Wrap expensive analytics function
analyticsFunc := func(userId int, timeRange string) (*Report, error) {
    // Expensive computation
    return generateReport(userId, timeRange)
}

// Cached version will use LFU eviction
cachedAnalytics := obcache.Wrap(cache, analyticsFunc)

// Frequently called reports stay in cache longer
report1, _ := cachedAnalytics(123, "last-month") // First call
report2, _ := cachedAnalytics(456, "last-week")  // First call
report3, _ := cachedAnalytics(123, "last-month") // Cache hit, frequency++
```

## Eviction Callbacks and Monitoring

```go
evictedItems := make([]string, 0)

config := obcache.NewDefaultConfig().
    WithMaxEntries(5).
    WithFIFOEviction().
    WithHooks(&obcache.Hooks{
        OnEvict: []obcache.OnEvictHook{
            func(key string, value any, reason obcache.EvictReason) {
                log.Printf("Evicted: key=%s, reason=%s", key, reason)
                
                if reason == obcache.EvictReasonCapacity {
                    evictedItems = append(evictedItems, key)
                }
            },
        },
    })

cache, err := obcache.New(config)
```

## Strategy Selection Guidelines

### Choose LRU when:
- ✅ General-purpose caching needs
- ✅ Temporal locality in access patterns
- ✅ Mixed read/write workloads
- ✅ Unknown or variable access patterns
- ✅ Memory efficiency is important

### Choose LFU when:
- ✅ Clear frequency-based access patterns
- ✅ Read-heavy workloads
- ✅ Long-running applications with stable patterns
- ✅ Analytics/reporting applications
- ❌ Avoid for rapidly changing access patterns

### Choose FIFO when:
- ✅ Simple buffering needs
- ✅ Time-ordered data processing
- ✅ Streaming data applications
- ✅ Predictable eviction behavior required
- ✅ Minimal computational overhead needed
- ❌ Avoid when access patterns matter for performance

## Advanced Usage Patterns

### Hybrid Caching Strategy

```go
// Use different strategies for different cache instances
type CacheManager struct {
    userCache     *obcache.Cache // LRU for user sessions
    contentCache  *obcache.Cache // LFU for popular content
    logCache      *obcache.Cache // FIFO for log entries
}

func NewCacheManager() (*CacheManager, error) {
    userCache, err := obcache.New(
        obcache.NewDefaultConfig().
            WithMaxEntries(1000).
            WithLRUEviction().
            WithDefaultTTL(30 * time.Minute),
    )
    if err != nil {
        return nil, err
    }

    contentCache, err := obcache.New(
        obcache.NewDefaultConfig().
            WithMaxEntries(500).
            WithLFUEviction().
            WithDefaultTTL(2 * time.Hour),
    )
    if err != nil {
        return nil, err
    }

    logCache, err := obcache.New(
        obcache.NewDefaultConfig().
            WithMaxEntries(10000).
            WithFIFOEviction().
            WithDefaultTTL(10 * time.Minute),
    )
    if err != nil {
        return nil, err
    }

    return &CacheManager{
        userCache:    userCache,
        contentCache: contentCache,
        logCache:     logCache,
    }, nil
}
```

### Dynamic Strategy Switching

```go
type AdaptiveCache struct {
    cache    *obcache.Cache
    strategy string
    metrics  *CacheMetrics
}

func (a *AdaptiveCache) maybeRebuildCache() {
    hitRate := a.metrics.HitRate()
    
    // Switch to LFU if hit rate is low and access patterns are stable
    if hitRate < 0.3 && a.strategy != "lfu" && a.metrics.IsStablePattern() {
        a.rebuildWithStrategy("lfu")
    }
    
    // Switch to LRU for general workloads
    if hitRate > 0.7 && a.strategy != "lru" {
        a.rebuildWithStrategy("lru")
    }
}
```

## Migration Between Strategies

```go
func migrateEvictionStrategy(oldCache *obcache.Cache, 
                           newStrategy eviction.EvictionType) (*obcache.Cache, error) {
    // Create new cache with different strategy
    newConfig := obcache.NewDefaultConfig().
        WithMaxEntries(oldCache.Capacity()).
        WithEvictionType(newStrategy)
    
    newCache, err := obcache.New(newConfig)
    if err != nil {
        return nil, err
    }
    
    // Migrate existing entries
    for _, key := range oldCache.Keys() {
        if value, found := oldCache.Get(key); found {
            newCache.Set(key, value, time.Hour) // Use appropriate TTL
        }
    }
    
    return newCache, nil
}
```

## Testing Different Strategies

```go
func BenchmarkEvictionStrategies(b *testing.B) {
    strategies := []struct {
        name   string
        config func() *obcache.Config
    }{
        {"LRU", func() *obcache.Config {
            return obcache.NewDefaultConfig().WithLRUEviction().WithMaxEntries(100)
        }},
        {"LFU", func() *obcache.Config {
            return obcache.NewDefaultConfig().WithLFUEviction().WithMaxEntries(100)
        }},
        {"FIFO", func() *obcache.Config {
            return obcache.NewDefaultConfig().WithFIFOEviction().WithMaxEntries(100)
        }},
    }
    
    for _, strategy := range strategies {
        b.Run(strategy.name, func(b *testing.B) {
            cache, _ := obcache.New(strategy.config())
            defer cache.Close()
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                key := fmt.Sprintf("key-%d", i%150) // Force evictions
                cache.Set(key, i, time.Hour)
                cache.Get(key)
            }
        })
    }
}
```

## Best Practices

1. **Start with LRU**: It's the most versatile strategy for general use cases
2. **Profile Your Workload**: Use metrics to understand access patterns before optimizing
3. **Consider Memory Overhead**: LFU uses more memory for frequency tracking
4. **Monitor Hit Rates**: Different strategies will have different hit rates for your workload
5. **Use Appropriate Capacity**: Size your cache based on working set and eviction behavior
6. **Test with Real Data**: Synthetic benchmarks may not reflect real-world performance
7. **Consider TTL Interaction**: Eviction strategy works alongside TTL expiration
8. **Handle Eviction Callbacks**: Use them for cleanup, logging, or cache warming

## Troubleshooting

### Low Hit Rates
- **LRU**: Consider if your access patterns have temporal locality
- **LFU**: Check if frequency patterns are stable over time
- **FIFO**: Verify that eviction order matches your data lifecycle

### High Memory Usage
- Reduce cache capacity
- Consider FIFO over LFU for lower memory overhead
- Implement more aggressive TTL policies

### Performance Issues
- Profile eviction overhead for your workload
- Consider async eviction callbacks for expensive cleanup
- Monitor eviction frequency vs cache size

This comprehensive guide should help you choose and configure the right eviction strategy for your specific use case.