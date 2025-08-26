package metrics

import (
	"time"
)

// Exporter defines the interface for cache metrics exporters
// This abstraction allows supporting multiple observability systems
type Exporter interface {
	// ExportStats exports the current cache statistics
	ExportStats(stats Stats, labels Labels) error

	// RecordCacheOperation records individual cache operations with timing
	RecordCacheOperation(operation Operation, duration time.Duration, labels Labels) error

	// IncrementCounter increments a named counter with labels
	IncrementCounter(name string, labels Labels) error

	// RecordHistogram records a value in a named histogram
	RecordHistogram(name string, value float64, labels Labels) error

	// SetGauge sets a gauge value
	SetGauge(name string, value float64, labels Labels) error

	// Close shuts down the exporter and flushes any pending metrics
	Close() error
}

// Labels represents key-value pairs for metric labels/tags
type Labels map[string]string

// Stats interface defines the cache statistics that can be exported
// This allows the metrics package to work with any stats implementation
type Stats interface {
	Hits() int64
	Misses() int64
	Evictions() int64
	Invalidations() int64
	KeyCount() int64
	InFlight() int64
	HitRate() float64
}

// Operation represents different cache operations for metrics
type Operation string

const (
	// Cache operations
	OperationGet        Operation = "get"
	OperationSet        Operation = "set"
	OperationDelete     Operation = "delete"
	OperationInvalidate Operation = "invalidate"
	OperationEvict      Operation = "evict"
	OperationCleanup    Operation = "cleanup"

	// Function wrapping operations
	OperationFunctionCall Operation = "function_call"
)

// Result represents the result of a cache operation
type Result string

const (
	ResultHit   Result = "hit"
	ResultMiss  Result = "miss"
	ResultError Result = "error"
)

// MetricNames defines standard metric names used across exporters
type MetricNames struct {
	// Counters
	CacheHitsTotal          string
	CacheMissesTotal        string
	CacheEvictionsTotal     string
	CacheInvalidationsTotal string
	CacheOperationsTotal    string
	CacheErrorsTotal        string

	// Histograms
	CacheOperationDuration string
	CacheKeySize           string
	CacheValueSize         string

	// Gauges
	CacheKeysCount        string
	CacheInFlightRequests string
	CacheHitRate          string
}

// DefaultMetricNames returns the default metric names with proper namespacing
func DefaultMetricNames() MetricNames {
	return MetricNames{
		CacheHitsTotal:          "obcache_hits_total",
		CacheMissesTotal:        "obcache_misses_total",
		CacheEvictionsTotal:     "obcache_evictions_total",
		CacheInvalidationsTotal: "obcache_invalidations_total",
		CacheOperationsTotal:    "obcache_operations_total",
		CacheErrorsTotal:        "obcache_errors_total",
		CacheOperationDuration:  "obcache_operation_duration_seconds",
		CacheKeySize:            "obcache_key_size_bytes",
		CacheValueSize:          "obcache_value_size_bytes",
		CacheKeysCount:          "obcache_keys_count",
		CacheInFlightRequests:   "obcache_inflight_requests",
		CacheHitRate:            "obcache_hit_rate",
	}
}

// Config holds configuration for metrics exporters
type Config struct {
	// Enabled determines whether metrics collection is enabled
	Enabled bool

	// Namespace is prepended to all metric names
	Namespace string

	// Labels are default labels applied to all metrics
	Labels Labels

	// MetricNames allows customizing metric names
	MetricNames MetricNames

	// ReportingInterval determines how often to export stats (for push-based systems)
	ReportingInterval time.Duration

	// IncludeDetailedTimings enables detailed operation timing metrics
	IncludeDetailedTimings bool

	// IncludeKeyValueSizes enables key/value size metrics (may impact performance)
	IncludeKeyValueSizes bool
}

// NewDefaultConfig creates a default metrics configuration
func NewDefaultConfig() *Config {
	return &Config{
		Enabled:                true,
		Namespace:              "obcache",
		Labels:                 make(Labels),
		MetricNames:            DefaultMetricNames(),
		ReportingInterval:      30 * time.Second,
		IncludeDetailedTimings: false,
		IncludeKeyValueSizes:   false,
	}
}

// WithNamespace sets the metrics namespace
func (c *Config) WithNamespace(namespace string) *Config {
	c.Namespace = namespace
	return c
}

// WithLabels adds default labels to all metrics
func (c *Config) WithLabels(labels Labels) *Config {
	for k, v := range labels {
		c.Labels[k] = v
	}
	return c
}

// WithReportingInterval sets the reporting interval for push-based systems
func (c *Config) WithReportingInterval(interval time.Duration) *Config {
	c.ReportingInterval = interval
	return c
}

// WithDetailedTimings enables detailed operation timing metrics
func (c *Config) WithDetailedTimings(enabled bool) *Config {
	c.IncludeDetailedTimings = enabled
	return c
}

// WithKeyValueSizes enables key/value size metrics
func (c *Config) WithKeyValueSizes(enabled bool) *Config {
	c.IncludeKeyValueSizes = enabled
	return c
}

// MultiExporter allows using multiple exporters simultaneously
type MultiExporter struct {
	exporters []Exporter
}

// NewMultiExporter creates an exporter that writes to multiple backends
func NewMultiExporter(exporters ...Exporter) *MultiExporter {
	return &MultiExporter{
		exporters: exporters,
	}
}

// ExportStats exports to all configured exporters
func (m *MultiExporter) ExportStats(stats Stats, labels Labels) error {
	for _, exporter := range m.exporters {
		if err := exporter.ExportStats(stats, labels); err != nil {
			return err
		}
	}
	return nil
}

// RecordCacheOperation records to all configured exporters
func (m *MultiExporter) RecordCacheOperation(operation Operation, duration time.Duration, labels Labels) error {
	for _, exporter := range m.exporters {
		if err := exporter.RecordCacheOperation(operation, duration, labels); err != nil {
			return err
		}
	}
	return nil
}

// IncrementCounter increments on all configured exporters
func (m *MultiExporter) IncrementCounter(name string, labels Labels) error {
	for _, exporter := range m.exporters {
		if err := exporter.IncrementCounter(name, labels); err != nil {
			return err
		}
	}
	return nil
}

// RecordHistogram records to all configured exporters
func (m *MultiExporter) RecordHistogram(name string, value float64, labels Labels) error {
	for _, exporter := range m.exporters {
		if err := exporter.RecordHistogram(name, value, labels); err != nil {
			return err
		}
	}
	return nil
}

// SetGauge sets on all configured exporters
func (m *MultiExporter) SetGauge(name string, value float64, labels Labels) error {
	for _, exporter := range m.exporters {
		if err := exporter.SetGauge(name, value, labels); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all configured exporters
func (m *MultiExporter) Close() error {
	for _, exporter := range m.exporters {
		if err := exporter.Close(); err != nil {
			return err
		}
	}
	return nil
}

// NoOpExporter provides a no-op implementation for when metrics are disabled
type NoOpExporter struct{}

// NewNoOpExporter creates a no-op exporter
func NewNoOpExporter() *NoOpExporter {
	return &NoOpExporter{}
}

// ExportStats does nothing
func (n *NoOpExporter) ExportStats(Stats, Labels) error { return nil }

// RecordCacheOperation does nothing
func (n *NoOpExporter) RecordCacheOperation(Operation, time.Duration, Labels) error { return nil }

// IncrementCounter does nothing
func (n *NoOpExporter) IncrementCounter(string, Labels) error { return nil }

// RecordHistogram does nothing
func (n *NoOpExporter) RecordHistogram(string, float64, Labels) error { return nil }

// SetGauge does nothing
func (n *NoOpExporter) SetGauge(string, float64, Labels) error { return nil }

// Close does nothing
func (n *NoOpExporter) Close() error { return nil }

// Ensure interfaces are implemented
var (
	_ Exporter = (*MultiExporter)(nil)
	_ Exporter = (*NoOpExporter)(nil)
)
