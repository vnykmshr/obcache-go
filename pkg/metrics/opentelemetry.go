package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OpenTelemetryExporter implements the Exporter interface for OpenTelemetry metrics
type OpenTelemetryExporter struct {
	config *Config
	meter  metric.Meter
	ctx    context.Context

	// Standard metrics instruments
	hitsCounter         metric.Int64Counter
	missesCounter       metric.Int64Counter
	evictionsCounter    metric.Int64Counter
	invalidationsCounter metric.Int64Counter
	operationsCounter   metric.Int64Counter
	errorsCounter       metric.Int64Counter

	operationDuration  metric.Float64Histogram
	keySize           metric.Int64Histogram
	valueSize         metric.Int64Histogram

	keysGauge          metric.Int64Gauge
	inFlightGauge      metric.Int64Gauge
	hitRateGauge       metric.Float64Gauge

	// Custom metrics (for IncrementCounter, etc.)
	customCounters   map[string]metric.Int64Counter
	customHistograms map[string]metric.Float64Histogram
	customGauges     map[string]metric.Float64Gauge
	mu               sync.RWMutex
}

// OpenTelemetryConfig holds OpenTelemetry-specific configuration
type OpenTelemetryConfig struct {
	// Meter is the OpenTelemetry meter to use
	Meter metric.Meter

	// Context is the context to use for metric operations
	Context context.Context

	// DefaultAttributes are applied to all metrics
	DefaultAttributes []attribute.KeyValue
}

// NewOpenTelemetryExporter creates a new OpenTelemetry metrics exporter
func NewOpenTelemetryExporter(config *Config, otelConfig *OpenTelemetryConfig) (*OpenTelemetryExporter, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	if otelConfig == nil {
		return nil, fmt.Errorf("OpenTelemetry configuration is required")
	}

	if otelConfig.Meter == nil {
		return nil, fmt.Errorf("OpenTelemetry meter is required")
	}

	ctx := otelConfig.Context
	if ctx == nil {
		ctx = context.Background()
	}

	exporter := &OpenTelemetryExporter{
		config:           config,
		meter:            otelConfig.Meter,
		ctx:              ctx,
		customCounters:   make(map[string]metric.Int64Counter),
		customHistograms: make(map[string]metric.Float64Histogram),
		customGauges:     make(map[string]metric.Float64Gauge),
	}

	// Create standard metrics
	if err := exporter.createStandardMetrics(); err != nil {
		return nil, fmt.Errorf("failed to create standard metrics: %w", err)
	}

	return exporter, nil
}

// createStandardMetrics creates all the standard cache metrics
func (o *OpenTelemetryExporter) createStandardMetrics() error {
	var err error

	// Counters
	o.hitsCounter, err = o.meter.Int64Counter(
		o.config.MetricNames.CacheHitsTotal,
		metric.WithDescription("Total number of cache hits"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create hits counter: %w", err)
	}

	o.missesCounter, err = o.meter.Int64Counter(
		o.config.MetricNames.CacheMissesTotal,
		metric.WithDescription("Total number of cache misses"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create misses counter: %w", err)
	}

	o.evictionsCounter, err = o.meter.Int64Counter(
		o.config.MetricNames.CacheEvictionsTotal,
		metric.WithDescription("Total number of cache evictions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create evictions counter: %w", err)
	}

	o.invalidationsCounter, err = o.meter.Int64Counter(
		o.config.MetricNames.CacheInvalidationsTotal,
		metric.WithDescription("Total number of cache invalidations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create invalidations counter: %w", err)
	}

	o.operationsCounter, err = o.meter.Int64Counter(
		o.config.MetricNames.CacheOperationsTotal,
		metric.WithDescription("Total number of cache operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create operations counter: %w", err)
	}

	o.errorsCounter, err = o.meter.Int64Counter(
		o.config.MetricNames.CacheErrorsTotal,
		metric.WithDescription("Total number of cache errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create errors counter: %w", err)
	}

	// Histograms
	if o.config.IncludeDetailedTimings {
		o.operationDuration, err = o.meter.Float64Histogram(
			o.config.MetricNames.CacheOperationDuration,
			metric.WithDescription("Cache operation duration"),
			metric.WithUnit("s"),
		)
		if err != nil {
			return fmt.Errorf("failed to create operation duration histogram: %w", err)
		}
	}

	if o.config.IncludeKeyValueSizes {
		o.keySize, err = o.meter.Int64Histogram(
			o.config.MetricNames.CacheKeySize,
			metric.WithDescription("Cache key size in bytes"),
			metric.WithUnit("By"),
		)
		if err != nil {
			return fmt.Errorf("failed to create key size histogram: %w", err)
		}

		o.valueSize, err = o.meter.Int64Histogram(
			o.config.MetricNames.CacheValueSize,
			metric.WithDescription("Cache value size in bytes"),
			metric.WithUnit("By"),
		)
		if err != nil {
			return fmt.Errorf("failed to create value size histogram: %w", err)
		}
	}

	// Gauges
	o.keysGauge, err = o.meter.Int64Gauge(
		o.config.MetricNames.CacheKeysCount,
		metric.WithDescription("Current number of keys in cache"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create keys gauge: %w", err)
	}

	o.inFlightGauge, err = o.meter.Int64Gauge(
		o.config.MetricNames.CacheInFlightRequests,
		metric.WithDescription("Current number of in-flight requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create in-flight gauge: %w", err)
	}

	o.hitRateGauge, err = o.meter.Float64Gauge(
		o.config.MetricNames.CacheHitRate,
		metric.WithDescription("Cache hit rate as a percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return fmt.Errorf("failed to create hit rate gauge: %w", err)
	}

	return nil
}

// ExportStats exports the current cache statistics to OpenTelemetry
func (o *OpenTelemetryExporter) ExportStats(stats Stats, labels Labels) error {
	attrs := o.convertLabels(labels)

	// Record counters (using the current values as incremental updates)
	o.hitsCounter.Add(o.ctx, stats.Hits(), metric.WithAttributes(attrs...))
	o.missesCounter.Add(o.ctx, stats.Misses(), metric.WithAttributes(attrs...))
	o.evictionsCounter.Add(o.ctx, stats.Evictions(), metric.WithAttributes(attrs...))
	o.invalidationsCounter.Add(o.ctx, stats.Invalidations(), metric.WithAttributes(attrs...))

	// Record gauges
	o.keysGauge.Record(o.ctx, stats.KeyCount(), metric.WithAttributes(attrs...))
	o.inFlightGauge.Record(o.ctx, stats.InFlight(), metric.WithAttributes(attrs...))
	o.hitRateGauge.Record(o.ctx, stats.HitRate(), metric.WithAttributes(attrs...))

	return nil
}

// RecordCacheOperation records a cache operation with timing
func (o *OpenTelemetryExporter) RecordCacheOperation(operation Operation, duration time.Duration, labels Labels) error {
	attrs := o.convertLabels(labels)
	
	// Add operation to attributes
	opAttrs := make([]attribute.KeyValue, len(attrs)+1)
	copy(opAttrs, attrs)
	opAttrs[len(attrs)] = attribute.String("operation", string(operation))

	// Record operation counter
	o.operationsCounter.Add(o.ctx, 1, metric.WithAttributes(opAttrs...))

	// Record operation timing if enabled
	if o.operationDuration != nil {
		o.operationDuration.Record(o.ctx, duration.Seconds(), metric.WithAttributes(opAttrs...))
	}

	return nil
}

// IncrementCounter increments a custom counter
func (o *OpenTelemetryExporter) IncrementCounter(name string, labels Labels) error {
	o.mu.Lock()
	counter, exists := o.customCounters[name]
	if !exists {
		var err error
		counter, err = o.meter.Int64Counter(
			name,
			metric.WithDescription(fmt.Sprintf("Custom counter: %s", name)),
			metric.WithUnit("1"),
		)
		if err != nil {
			o.mu.Unlock()
			return fmt.Errorf("failed to create counter %s: %w", name, err)
		}
		o.customCounters[name] = counter
	}
	o.mu.Unlock()

	attrs := o.convertLabels(labels)
	counter.Add(o.ctx, 1, metric.WithAttributes(attrs...))
	return nil
}

// RecordHistogram records a value in a custom histogram
func (o *OpenTelemetryExporter) RecordHistogram(name string, value float64, labels Labels) error {
	o.mu.Lock()
	histogram, exists := o.customHistograms[name]
	if !exists {
		var err error
		histogram, err = o.meter.Float64Histogram(
			name,
			metric.WithDescription(fmt.Sprintf("Custom histogram: %s", name)),
			metric.WithUnit("1"),
		)
		if err != nil {
			o.mu.Unlock()
			return fmt.Errorf("failed to create histogram %s: %w", name, err)
		}
		o.customHistograms[name] = histogram
	}
	o.mu.Unlock()

	attrs := o.convertLabels(labels)
	histogram.Record(o.ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// SetGauge sets a custom gauge value
func (o *OpenTelemetryExporter) SetGauge(name string, value float64, labels Labels) error {
	o.mu.Lock()
	gauge, exists := o.customGauges[name]
	if !exists {
		var err error
		gauge, err = o.meter.Float64Gauge(
			name,
			metric.WithDescription(fmt.Sprintf("Custom gauge: %s", name)),
			metric.WithUnit("1"),
		)
		if err != nil {
			o.mu.Unlock()
			return fmt.Errorf("failed to create gauge %s: %w", name, err)
		}
		o.customGauges[name] = gauge
	}
	o.mu.Unlock()

	attrs := o.convertLabels(labels)
	gauge.Record(o.ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// Close shuts down the exporter
func (o *OpenTelemetryExporter) Close() error {
	// OpenTelemetry metrics don't need explicit cleanup
	return nil
}

// Helper methods

func (o *OpenTelemetryExporter) convertLabels(labels Labels) []attribute.KeyValue {
	if labels == nil {
		return []attribute.KeyValue{}
	}
	
	attrs := make([]attribute.KeyValue, 0, len(labels)+len(o.config.Labels))
	
	// Add config labels first
	for k, v := range o.config.Labels {
		attrs = append(attrs, attribute.String(k, v))
	}
	
	// Add provided labels
	for k, v := range labels {
		attrs = append(attrs, attribute.String(k, v))
	}
	
	return attrs
}

// Ensure interface is implemented
var _ Exporter = (*OpenTelemetryExporter)(nil)