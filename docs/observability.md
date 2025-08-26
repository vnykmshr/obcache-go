# Observability and Monitoring

This document describes how to integrate obcache-go with popular observability and monitoring tools like Prometheus, OpenTelemetry, and custom monitoring solutions.

## Table of Contents

- [Built-in Observability Features](#built-in-observability-features)
- [Prometheus Integration](#prometheus-integration)
- [OpenTelemetry Integration](#opentelemetry-integration)
- [Custom Monitoring with Hooks](#custom-monitoring-with-hooks)
- [Debug Handler](#debug-handler)
- [Production Monitoring Best Practices](#production-monitoring-best-practices)

## Built-in Observability Features

obcache-go provides several built-in observability features:

### Statistics API

Access real-time cache statistics programmatically:

```go
cache, _ := obcache.New(obcache.NewDefaultConfig())

// Get statistics
stats := cache.Stats()
fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate())
fmt.Printf("Total Requests: %d\n", stats.Total())
fmt.Printf("Current Keys: %d\n", stats.KeyCount())
```

### HTTP Debug Handler

Expose cache metrics and keys via HTTP endpoints:

```go
// Simple debug server
server := cache.NewDebugServer(":8080")
go server.ListenAndServe()

// Custom integration
mux := http.NewServeMux()
mux.Handle("/debug/cache/", cache.DebugHandler())
```

Endpoints:
- `GET /stats` - Cache statistics only
- `GET /keys` - Statistics + key metadata  
- `GET /` - Same as `/keys`

## Prometheus Integration

### Basic Metrics Collection

Create Prometheus metrics for cache statistics:

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

var (
    cacheHits = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cache_hits_total",
        Help: "The total number of cache hits",
    })
    
    cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cache_misses_total", 
        Help: "The total number of cache misses",
    })
    
    cacheHitRate = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "cache_hit_rate_percent",
        Help: "Current cache hit rate as percentage",
    })
    
    cacheKeyCount = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "cache_keys_current",
        Help: "Current number of keys in cache",
    })
    
    cacheEvictions = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cache_evictions_total",
        Help: "Total number of cache evictions",
    })
)

func main() {
    cache, _ := obcache.New(obcache.NewDefaultConfig())
    
    // Update Prometheus metrics periodically
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()
        
        var lastHits, lastMisses, lastEvictions int64
        
        for range ticker.C {
            stats := cache.Stats()
            
            // Update counters with deltas
            newHits := stats.Hits()
            newMisses := stats.Misses()
            newEvictions := stats.Evictions()
            
            cacheHits.Add(float64(newHits - lastHits))
            cacheMisses.Add(float64(newMisses - lastMisses))
            cacheEvictions.Add(float64(newEvictions - lastEvictions))
            
            // Update gauges with current values
            cacheHitRate.Set(stats.HitRate())
            cacheKeyCount.Set(float64(stats.KeyCount()))
            
            lastHits = newHits
            lastMisses = newMisses
            lastEvictions = newEvictions
        }
    }()
    
    // Expose metrics endpoint
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":2112", nil)
}
```

### Hook-Based Metrics Collection

Use cache hooks for real-time metric updates:

```go
func setupPrometheusHooks(cache *obcache.Cache) {
    hooks := &obcache.Hooks{}
    
    // Track hits
    hooks.AddOnHit(func(key string, value any) {
        cacheHits.Inc()
    })
    
    // Track misses  
    hooks.AddOnMiss(func(key string) {
        cacheMisses.Inc()
    })
    
    // Track evictions by reason
    hooks.AddOnEvict(func(key string, value any, reason obcache.EvictReason) {
        cacheEvictions.With(prometheus.Labels{
            "reason": reason.String(),
        }).Inc()
    })
    
    // Track invalidations
    hooks.AddOnInvalidate(func(key string) {
        cacheInvalidations.Inc()
    })
    
    // Apply hooks to existing cache (if supported) or create new cache with hooks
    config := obcache.NewDefaultConfig().WithHooks(hooks)
    cache, _ = obcache.New(config)
}
```

### Advanced Prometheus Metrics

```go
var (
    // Histogram for operation latencies
    cacheOperationDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "cache_operation_duration_seconds",
            Help: "Time spent on cache operations",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation", "status"},
    )
    
    // Counter for operations by key pattern
    cacheOperationsByPattern = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_operations_by_pattern_total",
            Help: "Cache operations by key pattern",
        },
        []string{"pattern", "operation"},
    )
)

// Instrumented cache wrapper
type InstrumentedCache struct {
    cache *obcache.Cache
}

func (ic *InstrumentedCache) Get(key string) (any, bool) {
    start := time.Now()
    
    value, found := ic.cache.Get(key)
    
    status := "miss"
    if found {
        status = "hit"
    }
    
    cacheOperationDuration.WithLabelValues("get", status).Observe(
        time.Since(start).Seconds(),
    )
    
    pattern := extractKeyPattern(key)
    cacheOperationsByPattern.WithLabelValues(pattern, "get").Inc()
    
    return value, found
}

func extractKeyPattern(key string) string {
    // Extract pattern from key, e.g., "user:123" -> "user:*"
    parts := strings.Split(key, ":")
    if len(parts) > 1 {
        return parts[0] + ":*"
    }
    return "other"
}
```

## OpenTelemetry Integration

### Basic Tracing

```go
package main

import (
    "context"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
    "github.com/vnykmshr/obcache-go/pkg/obcache"
)

var (
    tracer = otel.Tracer("obcache-go")
    meter  = otel.Meter("obcache-go")
)

type TracedCache struct {
    cache *obcache.Cache
    
    // Metrics
    hitCounter    metric.Int64Counter
    missCounter   metric.Int64Counter
    keyCountGauge metric.Int64UpDownCounter
}

func NewTracedCache(config *obcache.Config) (*TracedCache, error) {
    cache, err := obcache.New(config)
    if err != nil {
        return nil, err
    }
    
    // Initialize metrics
    hitCounter, _ := meter.Int64Counter(
        "cache_hits_total",
        metric.WithDescription("Total number of cache hits"),
    )
    
    missCounter, _ := meter.Int64Counter(
        "cache_misses_total", 
        metric.WithDescription("Total number of cache misses"),
    )
    
    keyCountGauge, _ := meter.Int64UpDownCounter(
        "cache_keys_current",
        metric.WithDescription("Current number of keys in cache"),
    )
    
    tc := &TracedCache{
        cache:         cache,
        hitCounter:    hitCounter,
        missCounter:   missCounter,
        keyCountGauge: keyCountGauge,
    }
    
    // Set up hooks for automatic metric collection
    tc.setupHooks()
    
    return tc, nil
}

func (tc *TracedCache) Get(ctx context.Context, key string) (any, bool) {
    ctx, span := tracer.Start(ctx, "cache.Get",
        trace.WithAttributes(
            attribute.String("cache.key", key),
        ),
    )
    defer span.End()
    
    start := time.Now()
    value, found := tc.cache.Get(key)
    duration := time.Since(start)
    
    // Add attributes based on result
    span.SetAttributes(
        attribute.Bool("cache.hit", found),
        attribute.Int64("cache.duration_ns", duration.Nanoseconds()),
    )
    
    if found {
        span.SetAttributes(attribute.String("cache.status", "hit"))
    } else {
        span.SetAttributes(attribute.String("cache.status", "miss"))
    }
    
    return value, found
}

func (tc *TracedCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
    ctx, span := tracer.Start(ctx, "cache.Set",
        trace.WithAttributes(
            attribute.String("cache.key", key),
            attribute.Int64("cache.ttl_seconds", int64(ttl.Seconds())),
        ),
    )
    defer span.End()
    
    err := tc.cache.Set(key, value, ttl)
    if err != nil {
        span.RecordError(err)
    }
    
    return err
}

func (tc *TracedCache) setupHooks() {
    hooks := &obcache.Hooks{}
    
    hooks.AddOnHit(func(key string, value any) {
        tc.hitCounter.Add(context.Background(), 1,
            metric.WithAttributes(attribute.String("key_pattern", extractKeyPattern(key))),
        )
    })
    
    hooks.AddOnMiss(func(key string) {
        tc.missCounter.Add(context.Background(), 1,
            metric.WithAttributes(attribute.String("key_pattern", extractKeyPattern(key))),
        )
    })
    
    // Apply hooks (would need to be done during cache creation)
}
```

### Metrics and Tracing Combined

```go
func (tc *TracedCache) WrapFunction(ctx context.Context, fn func() (any, error), key string, ttl time.Duration) (any, error) {
    ctx, span := tracer.Start(ctx, "cache.WrapFunction",
        trace.WithAttributes(
            attribute.String("cache.key", key),
            attribute.String("cache.operation", "wrap"),
        ),
    )
    defer span.End()
    
    // Check cache first
    if value, found := tc.Get(ctx, key); found {
        span.SetAttributes(
            attribute.Bool("cache.hit", true),
            attribute.String("cache.source", "cache"),
        )
        return value, nil
    }
    
    // Execute function and cache result
    span.SetAttributes(attribute.String("cache.source", "function"))
    
    start := time.Now()
    value, err := fn()
    duration := time.Since(start)
    
    span.SetAttributes(
        attribute.Int64("function.duration_ns", duration.Nanoseconds()),
        attribute.Bool("function.error", err != nil),
    )
    
    if err != nil {
        span.RecordError(err)
        return value, err
    }
    
    // Cache successful result
    if setErr := tc.Set(ctx, key, value, ttl); setErr != nil {
        span.SetAttributes(attribute.Bool("cache.set_error", true))
        // Don't return the set error, return the function result
    }
    
    return value, nil
}
```

## Custom Monitoring with Hooks

### Structured Logging

```go
package main

import (
    "log/slog"
    "time"
    
    "github.com/vnykmshr/obcache-go/pkg/obcache"
)

func setupLoggingHooks(cache *obcache.Cache, logger *slog.Logger) {
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
        logger.Debug("cache miss",
            slog.String("key", key),
        )
    })
    
    // Log evictions (info level)
    hooks.AddOnEvict(func(key string, value any, reason obcache.EvictReason) {
        logger.Info("cache eviction",
            slog.String("key", key),
            slog.String("reason", reason.String()),
            slog.String("value_type", fmt.Sprintf("%T", value)),
        )
    })
    
    // Log invalidations (info level)
    hooks.AddOnInvalidate(func(key string) {
        logger.Info("cache invalidation",
            slog.String("key", key),
        )
    })
}
```

### Custom Metrics Collection

```go
type CustomMetrics struct {
    hits           int64
    misses         int64
    evictions      map[string]int64 // by reason
    keyPatterns    map[string]int64 // by pattern
    lastStatsReset time.Time
}

func setupCustomMetrics(cache *obcache.Cache) *CustomMetrics {
    metrics := &CustomMetrics{
        evictions:      make(map[string]int64),
        keyPatterns:    make(map[string]int64),
        lastStatsReset: time.Now(),
    }
    
    hooks := &obcache.Hooks{}
    
    hooks.AddOnHit(func(key string, value any) {
        atomic.AddInt64(&metrics.hits, 1)
        pattern := extractKeyPattern(key)
        // Note: This has race conditions, use sync.Map for production
        metrics.keyPatterns[pattern]++
    })
    
    hooks.AddOnMiss(func(key string) {
        atomic.AddInt64(&metrics.misses, 1)
    })
    
    hooks.AddOnEvict(func(key string, value any, reason obcache.EvictReason) {
        reasonStr := reason.String()
        // Note: This has race conditions, use sync.Map for production
        metrics.evictions[reasonStr]++
    })
    
    return metrics
}

func (m *CustomMetrics) GetSummary() map[string]interface{} {
    hits := atomic.LoadInt64(&m.hits)
    misses := atomic.LoadInt64(&m.misses)
    total := hits + misses
    
    hitRate := 0.0
    if total > 0 {
        hitRate = float64(hits) / float64(total) * 100
    }
    
    return map[string]interface{}{
        "hits":              hits,
        "misses":            misses,
        "hit_rate_percent":  hitRate,
        "evictions_by_reason": m.evictions,
        "requests_by_pattern": m.keyPatterns,
        "uptime_seconds":    time.Since(m.lastStatsReset).Seconds(),
    }
}
```

## Production Monitoring Best Practices

### 1. Essential Metrics to Monitor

**Performance Metrics:**
- Hit rate percentage (target: >80% for most use cases)
- Average response time (target: <1ms for cache operations)
- P95/P99 latency percentiles

**Capacity Metrics:**
- Current key count vs max entries
- Memory usage (if available)
- Eviction rate by reason

**Health Metrics:**
- Error rates for cache operations
- Connection/availability status
- In-flight request count

### 2. Alerting Thresholds

```yaml
# Example Prometheus alerting rules
groups:
- name: obcache-alerts
  rules:
  - alert: CacheHitRateLow
    expr: cache_hit_rate_percent < 50
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Cache hit rate is below 50%"
      
  - alert: CacheEvictionHigh  
    expr: rate(cache_evictions_total[5m]) > 10
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High cache eviction rate detected"
      
  - alert: CacheNearCapacity
    expr: cache_keys_current / cache_max_entries > 0.9
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Cache is near capacity"
```

### 3. Dashboard Examples

**Grafana Dashboard Panels:**
- Hit rate over time (line chart)
- Request volume (hits + misses) over time
- Evictions by reason (stacked bar chart) 
- Key count vs capacity (gauge)
- Operation latency distribution (histogram)
- Top cache key patterns by volume

### 4. Log Aggregation

Structure cache logs for easy analysis:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "info",
  "component": "cache",
  "operation": "eviction",
  "key": "user:12345",
  "reason": "LRU",
  "ttl_remaining": "45s",
  "key_pattern": "user:*",
  "cache_size": 950
}
```

### 5. Health Checks

```go
func cacheHealthCheck(cache *obcache.Cache) error {
    // Check if cache is responsive
    testKey := fmt.Sprintf("_health_check_%d", time.Now().Unix())
    testValue := "ok"
    
    if err := cache.Set(testKey, testValue, time.Minute); err != nil {
        return fmt.Errorf("cache set failed: %w", err)
    }
    
    if value, found := cache.Get(testKey); !found || value != testValue {
        return fmt.Errorf("cache get failed: expected %v, got %v (found: %v)", 
            testValue, value, found)
    }
    
    if err := cache.Invalidate(testKey); err != nil {
        return fmt.Errorf("cache invalidate failed: %w", err)
    }
    
    return nil
}
```

This comprehensive observability setup ensures you have full visibility into your cache performance and can quickly identify and resolve issues in production environments.