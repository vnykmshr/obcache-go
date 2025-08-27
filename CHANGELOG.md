# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2025-08-27

### Initial Release

High-performance, thread-safe caching library for Go.

**Core Features:**
- Function wrapping with `obcache.Wrap()`
- TTL support with automatic cleanup
- LRU/LFU/FIFO eviction strategies
- Thread-safe operations
- Redis backend support
- Compression (gzip/deflate)
- Statistics and monitoring hooks

**API:**
- `cache.Get(key)` / `cache.Set(key, value, ttl)`
- `obcache.Wrap(cache, function, options...)`
- Memory and Redis backends
- Prometheus/OpenTelemetry metrics

See [README.md](README.md) for usage examples.