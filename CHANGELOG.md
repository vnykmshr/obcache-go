# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-27

### 🎉 Initial Release

The first stable release of obcache-go, a high-performance, feature-rich caching library for Go.

### ✨ Core Features

#### Cache Operations
- **Thread-safe operations** with optimized locking
- **Generic function wrapping** with type safety
- **TTL support** with configurable cleanup intervals
- **Comprehensive statistics** with hit/miss tracking
- **Multiple backends**: Memory and Redis support

#### Eviction Strategies
- **LRU (Least Recently Used)** - Default strategy, optimal for most use cases
- **LFU (Least Frequently Used)** - Best for frequency-based access patterns
- **FIFO (First In, First Out)** - Simple time-based eviction
- **Pluggable architecture** for custom eviction strategies

#### Function Wrapping
- **Automatic caching** of function results with `obcache.Wrap()`
- **Type-safe generics** support for all Go function signatures
- **Singleflight pattern** prevents duplicate concurrent function calls
- **Configurable error caching** with separate TTL settings
- **Custom key generation** functions

#### Advanced Features
- **Hook system** with priority and conditional execution
- **Context-aware operations** with metadata support
- **Compression support** (gzip, lz4) with configurable thresholds
- **Metrics integration** (Prometheus, OpenTelemetry)
- **Debug HTTP handler** for cache inspection and monitoring

### 📦 Backend Support

#### Memory Backend
- **High performance** with nanosecond access times
- **Configurable capacity** with automatic eviction
- **Multiple eviction strategies** (LRU/LFU/FIFO)
- **TTL cleanup** with configurable intervals

#### Redis Backend
- **Distributed caching** across multiple application instances
- **Persistent storage** with Redis durability guarantees
- **Custom Redis client** support for advanced configurations
- **Key prefixing** to avoid conflicts in shared Redis instances

### 🎣 Hook System

#### Basic Hooks
- `OnHit` - Cache hit events
- `OnMiss` - Cache miss events  
- `OnEvict` - Entry eviction events
- `OnInvalidate` - Manual invalidation events

#### Advanced Hooks
- **Context-aware hooks** with function arguments
- **Priority-based execution** (High/Medium/Low)
- **Conditional execution** with custom conditions
- **Built-in conditions**: Key prefix, context values, logical combinations

### 📊 Statistics & Monitoring

#### Built-in Metrics
- Hit/miss counts and rates
- Eviction counts by reason
- Key count and capacity utilization
- Operation timing and performance

#### Extensible Monitoring
- **Prometheus integration** with standard metrics
- **OpenTelemetry support** for distributed tracing
- **Custom exporters** for third-party monitoring systems
- **Real-time dashboard** via debug HTTP handler

### 🔧 Configuration

#### Fluent Configuration API
```go
config := obcache.NewDefaultConfig().
    WithMaxEntries(10000).
    WithLFUEviction().
    WithDefaultTTL(time.Hour).
    WithCompress("gzip").
    WithMetricsExporter(prometheusExporter)
```

#### Environment-based Configuration
- Redis connection from environment variables
- Configurable defaults for different environments
- Production-ready configuration templates

### 📚 Documentation & Examples

#### Comprehensive Documentation
- [**Getting Started Guide**](docs/getting-started.md)
- [**Eviction Strategies Guide**](docs/eviction-strategies.md)
- [**Backend Switching Guide**](docs/backend-switching.md)
- [**Migration Guide**](docs/migration-guide.md) from popular libraries
- [**Complete API Reference**](docs/api-reference.md)

#### Real-world Examples
- [**Basic Usage**](examples/basic/) - Simple cache operations
- [**Gin Web Server**](examples/gin-web-server/) - REST API caching
- [**Echo Web Server**](examples/echo-web-server/) - Multi-layer caching
- [**Batch Processing**](examples/batch-processing/) - High-throughput scenarios
- [**Redis Integration**](examples/redis-cache/) - Distributed caching
- [**Metrics & Monitoring**](examples/prometheus/) - Observability setup

### 🚀 Performance

#### Benchmark Results
```
BenchmarkCacheGet-8              20000000    95.2 ns/op    16 B/op   1 allocs/op
BenchmarkCacheSet-8              10000000   148.2 ns/op    64 B/op   2 allocs/op
BenchmarkWrappedFunction-8        5000000   245.3 ns/op    96 B/op   3 allocs/op
BenchmarkConcurrentAccess-8      50000000    45.1 ns/op     8 B/op   0 allocs/op
```

#### Performance Features
- **Lock-free reads** where possible
- **Batch operations** for bulk updates
- **Memory-efficient storage** with object pooling
- **Optimized serialization** for complex data types

### 🧪 Testing & Quality

#### Comprehensive Test Suite
- **Unit tests** with >95% coverage
- **Integration tests** for all backends
- **Stress tests** for high-concurrency scenarios
- **Fuzz testing** for edge cases and input validation
- **Benchmark tests** for performance regression detection

#### Code Quality
- **golangci-lint** with strict rules
- **Go vet** and race condition detection
- **Continuous integration** with GitHub Actions
- **Semantic versioning** for stable releases

### 🛠 Developer Experience

#### Modern Go Features
- **Generics support** (Go 1.18+) for type safety
- **Context integration** throughout the API
- **Error wrapping** with detailed error messages
- **Resource management** with proper cleanup

#### IDE Integration
- **Complete godoc** documentation
- **Type hints** and autocompletion
- **Example code** in documentation
- **Comprehensive error messages**

### 📋 Migration Support

#### Supported Libraries
- Migration from `patrickmn/go-cache`
- Migration from `go-redis/cache`
- Migration from `allegro/bigcache`
- Migration from `hashicorp/golang-lru`

#### Migration Tools
- **API compatibility layers** for common patterns
- **Performance comparison** benchmarks
- **Feature mapping** guides
- **Gradual migration** strategies

---

## Development History

This changelog represents the culmination of extensive development phases:

### Phase 1: MVP (Core Memory Cache & API)
- Basic cache operations (Get/Set/Invalidate)
- Memory backend with LRU eviction
- TTL support with cleanup
- Thread-safe operations
- Function wrapping foundation

### Phase 2: Observability & Debugging  
- Comprehensive hook system
- Statistics and metrics
- Debug HTTP handler
- Logging integration
- Performance monitoring

### Phase 3: Backend Extensibility
- Redis backend implementation
- Backend abstraction layer
- Configuration system
- Environment-based setup
- Connection management

### Phase 4: Advanced Features
- **Configurable error caching** for robust function wrapping
- **Pluggable eviction strategies** (LFU, FIFO, custom)
- **Enhanced hooks** with priority and conditions
- **Compression support** for large values
- **Metrics integration** with popular systems

### Phase 5: Production Hardening
- **Stress testing** and fuzz testing
- **Real-world examples** for common use cases
- **Migration guides** from popular libraries
- **Performance optimization** and benchmarking
- **Documentation completeness**

---

## Versioning Strategy

obcache-go follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version when making incompatible API changes
- **MINOR** version when adding functionality in a backwards compatible manner  
- **PATCH** version when making backwards compatible bug fixes

## Compatibility Promise

- **Go Version**: Supports Go 1.18+ (required for generics)
- **API Stability**: Public API is stable and follows semantic versioning
- **Backend Compatibility**: Redis 6.0+ recommended, 5.0+ supported
- **Platform Support**: Linux, macOS, Windows (wherever Go runs)

## Getting Started

```go
go get github.com/vnykmshr/obcache-go
```

```go
package main

import (
    "fmt"
    "time"
    "github.com/vnykmshr/obcache-go/pkg/obcache"
)

func main() {
    // Create cache
    cache, err := obcache.New(obcache.NewDefaultConfig())
    if err != nil {
        panic(err)
    }
    defer cache.Close()

    // Basic operations
    cache.Set("key", "value", time.Hour)
    
    if value, found := cache.Get("key"); found {
        fmt.Printf("Found: %s\n", value)
    }

    // Function wrapping
    expensiveFunc := func(n int) int {
        time.Sleep(100 * time.Millisecond)
        return n * 2
    }
    
    cachedFunc := obcache.Wrap(cache, expensiveFunc)
    result := cachedFunc(42) // Cached automatically!
    
    fmt.Printf("Result: %d\n", result)
}
```

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by battle-tested libraries like `go-cache`, `bigcache`, and `golang-lru`
- Built with modern Go features and best practices
- Designed for production use at scale

---

**Full Changelog**: https://github.com/vnykmshr/obcache-go/commits/v1.0.0