# Observability Quick Start Guide

This guide provides practical, ready-to-use examples for monitoring obcache-go in production environments.

## Built-in Debug Handler (Simplest)

The easiest way to get started with observability is using the built-in debug handler:

```go
package main

import (
    "log"
    "net/http"
    "time"
    
    "github.com/vnykmshr/obcache-go/pkg/obcache"
)

func main() {
    // Create cache
    cache, _ := obcache.New(obcache.NewDefaultConfig())
    
    // Add some data
    _ = cache.Set("user:1", "Alice", time.Hour)
    _ = cache.Set("user:2", "Bob", time.Hour)
    
    // Start debug server
    server := cache.NewDebugServer(":8080")
    log.Printf("Debug server running on http://localhost:8080")
    
    // Endpoints:
    // GET /stats - JSON stats only
    // GET /keys  - JSON stats + all keys 
    // GET /      - Same as /keys
    
    log.Fatal(server.ListenAndServe())
}
```

**Example Response from `/stats`:**
```json
{
  "stats": {
    "hits": 15,
    "misses": 3,
    "hitRate": 83.33,
    "keyCount": 42,
    "evictions": 2,
    "invalidations": 1,
    "config": {
      "maxEntries": 1000,
      "defaultTTL": "5m0s",
      "cleanupInterval": "1m0s"
    }
  }
}
```

## Prometheus Integration with Hooks

For production monitoring, integrate with Prometheus using cache hooks:

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/vnykmshr/obcache-go/pkg/obcache"
)

// Prometheus metrics
var (
    cacheHits = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cache_hits_total",
        Help: "Total cache hits",
    })
    
    cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cache_misses_total", 
        Help: "Total cache misses",
    })
    
    cacheHitRate = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "cache_hit_rate_percent",
        Help: "Current hit rate percentage",
    })
)

func main() {
    // Create hooks for metrics
    hooks := &obcache.Hooks{}
    hooks.AddOnHit(func(key string, value any) {
        cacheHits.Inc()
    })
    hooks.AddOnMiss(func(key string) {
        cacheMisses.Inc()
    })
    
    // Create cache with hooks
    config := obcache.NewDefaultConfig().WithHooks(hooks)
    cache, _ := obcache.New(config)
    
    // Update hit rate periodically
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        for range ticker.C {
            cacheHitRate.Set(cache.Stats().HitRate())
        }
    }()
    
    // Expose metrics
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":2112", nil)
}
```

**Key Prometheus Queries:**
```promql
# Hit rate over time
cache_hit_rate_percent

# Request rate (ops/sec)  
rate(cache_hits_total[1m]) + rate(cache_misses_total[1m])

# Miss rate
rate(cache_misses_total[1m]) / (rate(cache_hits_total[1m]) + rate(cache_misses_total[1m]))
```

## Structured Logging with Hooks

Add structured logging for cache events:

```go
package main

import (
    "log/slog"
    "os"
    "time"
    
    "github.com/vnykmshr/obcache-go/pkg/obcache"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
    
    hooks := &obcache.Hooks{}
    
    // Log cache hits (debug level)
    hooks.AddOnHit(func(key string, value any) {
        logger.Debug("cache hit", 
            slog.String("key", key),
            slog.String("value_type", fmt.Sprintf("%T", value)),
        )
    })
    
    // Log cache misses (debug level)
    hooks.AddOnMiss(func(key string) {
        logger.Debug("cache miss", slog.String("key", key))
    })
    
    // Log evictions (info level)
    hooks.AddOnEvict(func(key string, value any, reason obcache.EvictReason) {
        logger.Info("cache eviction",
            slog.String("key", key), 
            slog.String("reason", reason.String()),
        )
    })
    
    config := obcache.NewDefaultConfig().WithHooks(hooks)
    cache, _ := obcache.New(config)
    
    // Use cache...
    _ = cache.Set("test", "value", time.Hour)
    _, _ = cache.Get("test")      // Logs: cache hit
    _, _ = cache.Get("missing")   // Logs: cache miss
}
```

## Health Check Endpoint

Create a health check for your cache:

```go
func cacheHealthCheck(cache *obcache.Cache) error {
    testKey := "_health_check_" + time.Now().Format("20060102150405")
    testValue := "ok"
    
    // Test Set
    if err := cache.Set(testKey, testValue, time.Minute); err != nil {
        return fmt.Errorf("cache set failed: %w", err)
    }
    
    // Test Get
    if value, found := cache.Get(testKey); !found || value != testValue {
        return fmt.Errorf("cache get failed")
    }
    
    // Cleanup
    _ = cache.Invalidate(testKey)
    return nil
}

// HTTP handler
func healthHandler(cache *obcache.Cache) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := cacheHealthCheck(cache); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
            return
        }
        
        stats := cache.Stats()
        fmt.Fprintf(w, `{
  "status": "healthy",
  "stats": {
    "hits": %d,
    "misses": %d,
    "hit_rate": %.2f,
    "keys": %d
  }
}`, stats.Hits(), stats.Misses(), stats.HitRate(), stats.KeyCount())
    }
}
```

## Essential Production Metrics

Monitor these key metrics in production:

### Performance Metrics
- **Hit Rate** (`cache_hit_rate_percent`): Target >80% for most applications
- **Response Time** (`cache_operation_duration_seconds`): Target <1ms for cache ops
- **Request Rate** (`rate(cache_hits_total[1m] + cache_misses_total[1m])`): Ops/second

### Capacity Metrics  
- **Key Count** (`cache_keys_current`): Current vs maximum entries
- **Eviction Rate** (`rate(cache_evictions_total[1m])`): Should be low and stable
- **Memory Usage**: Track if available from your deployment platform

### Health Metrics
- **Error Rate**: Track failed cache operations
- **Availability**: Cache service uptime and responsiveness

## Alerting Rules (Prometheus)

```yaml
groups:
- name: obcache-alerts
  rules:
  # Hit rate too low
  - alert: CacheHitRateLow
    expr: cache_hit_rate_percent < 50
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Cache hit rate is {{ $value }}% (below 50%)"
      
  # High eviction rate
  - alert: CacheEvictionsHigh
    expr: rate(cache_evictions_total[5m]) > 10
    for: 2m  
    labels:
      severity: warning
    annotations:
      summary: "Cache eviction rate is {{ $value }}/sec"
      
  # Cache near capacity
  - alert: CacheNearCapacity
    expr: cache_keys_current / cache_max_entries > 0.9
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Cache is {{ $value | humanizePercentage }} full"
```

## Grafana Dashboard Panels

Essential dashboard panels for cache monitoring:

1. **Hit Rate Over Time** (Stat + Graph)
   - Query: `cache_hit_rate_percent`
   - Show current value and trend

2. **Request Volume** (Graph)
   - Query: `rate(cache_hits_total[1m]) + rate(cache_misses_total[1m])`
   - Unit: requests/sec

3. **Evictions by Reason** (Stacked Bar Chart)
   - Query: `rate(cache_evictions_total[1m]) by (reason)`
   - Group by eviction reason

4. **Cache Capacity** (Gauge)
   - Query: `cache_keys_current / cache_max_entries * 100`
   - Show percentage full

5. **Operation Latency** (Heatmap)
   - Query: `cache_operation_duration_seconds_bucket`
   - Show P50, P95, P99 percentiles

## Testing Your Observability

Verify your monitoring setup:

```go
// Generate test load
func generateTestLoad(cache *obcache.Cache) {
    keys := []string{"user:1", "user:2", "product:100"}
    
    for i := 0; i < 1000; i++ {
        // Mix of hits and misses
        key := keys[rand.Intn(len(keys))]
        if rand.Float64() < 0.8 { // 80% hit rate
            _, _ = cache.Get(key)
        } else {
            _, _ = cache.Get(fmt.Sprintf("missing:%d", rand.Intn(100)))
        }
        
        // Occasional sets
        if rand.Float64() < 0.1 {
            _ = cache.Set(key, "value", time.Hour)
        }
        
        time.Sleep(10 * time.Millisecond)
    }
}
```

This should generate metrics you can verify in your monitoring dashboards.

## Next Steps

1. **Start with the debug handler** for development and testing
2. **Add Prometheus metrics** for production monitoring  
3. **Set up alerting** on key metrics (hit rate, evictions)
4. **Create dashboards** for operational visibility
5. **Implement health checks** for service monitoring
6. **Add structured logging** for debugging and audit trails

For more advanced patterns, see the full [observability.md](./observability.md) documentation.