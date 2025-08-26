package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusExporter implements the Exporter interface for Prometheus metrics
type PrometheusExporter struct {
	config   *Config
	registry prometheus.Registerer

	// Counters
	hitsTotal         *prometheus.CounterVec
	missesTotal       *prometheus.CounterVec
	evictionsTotal    *prometheus.CounterVec
	invalidationsTotal *prometheus.CounterVec
	operationsTotal   *prometheus.CounterVec
	errorsTotal       *prometheus.CounterVec

	// Histograms
	operationDuration *prometheus.HistogramVec
	keySize          *prometheus.HistogramVec
	valueSize        *prometheus.HistogramVec

	// Gauges
	keysCount        *prometheus.GaugeVec
	inFlightRequests *prometheus.GaugeVec
	hitRate          *prometheus.GaugeVec

	// Custom metrics (for IncrementCounter, etc.)
	customCounters   map[string]*prometheus.CounterVec
	customHistograms map[string]*prometheus.HistogramVec
	customGauges     map[string]*prometheus.GaugeVec
	mu               sync.RWMutex
}

// PrometheusConfig holds Prometheus-specific configuration
type PrometheusConfig struct {
	// Registry is the Prometheus registry to use (optional, uses default if nil)
	Registry prometheus.Registerer

	// DefaultLabels are applied to all metrics
	DefaultLabels prometheus.Labels

	// Buckets for histogram metrics
	DurationBuckets []float64
	SizeBuckets     []float64
}

// NewPrometheusExporter creates a new Prometheus metrics exporter
func NewPrometheusExporter(config *Config, promConfig *PrometheusConfig) (*PrometheusExporter, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	if promConfig == nil {
		promConfig = &PrometheusConfig{}
	}

	registry := promConfig.Registry
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	// Default histogram buckets
	durationBuckets := promConfig.DurationBuckets
	if durationBuckets == nil {
		durationBuckets = []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
	}

	sizeBuckets := promConfig.SizeBuckets
	if sizeBuckets == nil {
		sizeBuckets = []float64{64, 256, 1024, 4096, 16384, 65536, 262144, 1048576}
	}

	// Convert config labels to prometheus labels
	var defaultLabels prometheus.Labels
	if promConfig.DefaultLabels != nil {
		defaultLabels = promConfig.DefaultLabels
	} else {
		defaultLabels = make(prometheus.Labels)
	}

	// Add config labels to default labels
	for k, v := range config.Labels {
		defaultLabels[k] = v
	}

	exporter := &PrometheusExporter{
		config:           config,
		registry:         registry,
		customCounters:   make(map[string]*prometheus.CounterVec),
		customHistograms: make(map[string]*prometheus.HistogramVec),
		customGauges:     make(map[string]*prometheus.GaugeVec),
	}

	// Create standard metrics
	if err := exporter.createStandardMetrics(defaultLabels, durationBuckets, sizeBuckets); err != nil {
		return nil, fmt.Errorf("failed to create standard metrics: %w", err)
	}

	return exporter, nil
}

// createStandardMetrics creates all the standard cache metrics
func (p *PrometheusExporter) createStandardMetrics(defaultLabels prometheus.Labels, durationBuckets, sizeBuckets []float64) error {
	var err error

	// Use a consistent set of base labels for all metrics
	baseLabels := []string{"cache_name"}

	// Counters
	p.hitsTotal, err = p.createCounterVec(p.config.MetricNames.CacheHitsTotal, "Total number of cache hits", baseLabels, defaultLabels)
	if err != nil {
		return err
	}

	p.missesTotal, err = p.createCounterVec(p.config.MetricNames.CacheMissesTotal, "Total number of cache misses", baseLabels, defaultLabels)
	if err != nil {
		return err
	}

	p.evictionsTotal, err = p.createCounterVec(p.config.MetricNames.CacheEvictionsTotal, "Total number of cache evictions", append(baseLabels, "reason"), defaultLabels)
	if err != nil {
		return err
	}

	p.invalidationsTotal, err = p.createCounterVec(p.config.MetricNames.CacheInvalidationsTotal, "Total number of cache invalidations", baseLabels, defaultLabels)
	if err != nil {
		return err
	}

	p.operationsTotal, err = p.createCounterVec(p.config.MetricNames.CacheOperationsTotal, "Total number of cache operations", append(baseLabels, "operation", "result"), defaultLabels)
	if err != nil {
		return err
	}

	p.errorsTotal, err = p.createCounterVec(p.config.MetricNames.CacheErrorsTotal, "Total number of cache errors", append(baseLabels, "operation"), defaultLabels)
	if err != nil {
		return err
	}

	// Histograms
	if p.config.IncludeDetailedTimings {
		p.operationDuration, err = p.createHistogramVec(p.config.MetricNames.CacheOperationDuration, "Cache operation duration in seconds", append(baseLabels, "operation"), defaultLabels, durationBuckets)
		if err != nil {
			return err
		}
	}

	if p.config.IncludeKeyValueSizes {
		p.keySize, err = p.createHistogramVec(p.config.MetricNames.CacheKeySize, "Cache key size in bytes", baseLabels, defaultLabels, sizeBuckets)
		if err != nil {
			return err
		}

		p.valueSize, err = p.createHistogramVec(p.config.MetricNames.CacheValueSize, "Cache value size in bytes", baseLabels, defaultLabels, sizeBuckets)
		if err != nil {
			return err
		}
	}

	// Gauges
	p.keysCount, err = p.createGaugeVec(p.config.MetricNames.CacheKeysCount, "Current number of keys in cache", baseLabels, defaultLabels)
	if err != nil {
		return err
	}

	p.inFlightRequests, err = p.createGaugeVec(p.config.MetricNames.CacheInFlightRequests, "Current number of in-flight requests", baseLabels, defaultLabels)
	if err != nil {
		return err
	}

	p.hitRate, err = p.createGaugeVec(p.config.MetricNames.CacheHitRate, "Cache hit rate as a percentage", baseLabels, defaultLabels)
	if err != nil {
		return err
	}

	return nil
}

// ExportStats exports the current cache statistics to Prometheus
func (p *PrometheusExporter) ExportStats(stats Stats, labels Labels) error {
	// Extract only cache_name for basic metrics
	baseLabels := prometheus.Labels{}
	if cacheName, exists := labels["cache_name"]; exists {
		baseLabels["cache_name"] = cacheName
	}

	// Update counters that only need cache_name
	p.hitsTotal.With(baseLabels).Add(float64(stats.Hits()))
	p.missesTotal.With(baseLabels).Add(float64(stats.Misses()))
	p.invalidationsTotal.With(baseLabels).Add(float64(stats.Invalidations()))

	// For evictions, we need to add the reason label
	evictionLabels := make(prometheus.Labels)
	for k, v := range baseLabels {
		evictionLabels[k] = v
	}
	evictionLabels["reason"] = "capacity" // Default reason for stats export
	p.evictionsTotal.With(evictionLabels).Add(float64(stats.Evictions()))

	// Update gauges
	p.keysCount.With(baseLabels).Set(float64(stats.KeyCount()))
	p.inFlightRequests.With(baseLabels).Set(float64(stats.InFlight()))
	p.hitRate.With(baseLabels).Set(stats.HitRate())

	return nil
}

// RecordCacheOperation records a cache operation with timing
func (p *PrometheusExporter) RecordCacheOperation(operation Operation, duration time.Duration, labels Labels) error {
	// Extract only cache_name for basic operations
	baseLabels := prometheus.Labels{}
	if cacheName, exists := labels["cache_name"]; exists {
		baseLabels["cache_name"] = cacheName
	}

	// Add operation to labels
	opLabels := prometheus.Labels{}
	for k, v := range baseLabels {
		opLabels[k] = v
	}
	opLabels["operation"] = string(operation)

	// Record operation timing if enabled
	if p.operationDuration != nil {
		p.operationDuration.With(opLabels).Observe(duration.Seconds())
	}

	return nil
}

// IncrementCounter increments a custom counter
func (p *PrometheusExporter) IncrementCounter(name string, labels Labels) error {
	p.mu.Lock()
	counter, exists := p.customCounters[name]
	if !exists {
		labelNames := p.getLabelNames(labels)
		defaultLabels := p.convertLabelsToPromLabels(p.config.Labels)
		
		var err error
		counter, err = p.createCounterVec(name, fmt.Sprintf("Custom counter: %s", name), labelNames, defaultLabels)
		if err != nil {
			p.mu.Unlock()
			return fmt.Errorf("failed to create counter %s: %w", name, err)
		}
		p.customCounters[name] = counter
	}
	p.mu.Unlock()

	promLabels := p.convertLabels(labels)
	counter.With(promLabels).Inc()
	return nil
}

// RecordHistogram records a value in a custom histogram
func (p *PrometheusExporter) RecordHistogram(name string, value float64, labels Labels) error {
	p.mu.Lock()
	histogram, exists := p.customHistograms[name]
	if !exists {
		labelNames := p.getLabelNames(labels)
		defaultLabels := p.convertLabelsToPromLabels(p.config.Labels)
		buckets := prometheus.DefBuckets
		
		var err error
		histogram, err = p.createHistogramVec(name, fmt.Sprintf("Custom histogram: %s", name), labelNames, defaultLabels, buckets)
		if err != nil {
			p.mu.Unlock()
			return fmt.Errorf("failed to create histogram %s: %w", name, err)
		}
		p.customHistograms[name] = histogram
	}
	p.mu.Unlock()

	promLabels := p.convertLabels(labels)
	histogram.With(promLabels).Observe(value)
	return nil
}

// SetGauge sets a custom gauge value
func (p *PrometheusExporter) SetGauge(name string, value float64, labels Labels) error {
	p.mu.Lock()
	gauge, exists := p.customGauges[name]
	if !exists {
		labelNames := p.getLabelNames(labels)
		defaultLabels := p.convertLabelsToPromLabels(p.config.Labels)
		
		var err error
		gauge, err = p.createGaugeVec(name, fmt.Sprintf("Custom gauge: %s", name), labelNames, defaultLabels)
		if err != nil {
			p.mu.Unlock()
			return fmt.Errorf("failed to create gauge %s: %w", name, err)
		}
		p.customGauges[name] = gauge
	}
	p.mu.Unlock()

	promLabels := p.convertLabels(labels)
	gauge.With(promLabels).Set(value)
	return nil
}

// Close shuts down the exporter
func (p *PrometheusExporter) Close() error {
	// Prometheus metrics don't need explicit cleanup
	return nil
}

// Helper methods

func (p *PrometheusExporter) createCounterVec(name, help string, labelNames []string, defaultLabels prometheus.Labels) (*prometheus.CounterVec, error) {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        name,
			Help:        help,
			ConstLabels: defaultLabels,
		},
		labelNames,
	)

	if err := p.registry.Register(counter); err != nil {
		return nil, err
	}

	return counter, nil
}

func (p *PrometheusExporter) createHistogramVec(name, help string, labelNames []string, defaultLabels prometheus.Labels, buckets []float64) (*prometheus.HistogramVec, error) {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        name,
			Help:        help,
			ConstLabels: defaultLabels,
			Buckets:     buckets,
		},
		labelNames,
	)

	if err := p.registry.Register(histogram); err != nil {
		return nil, err
	}

	return histogram, nil
}

func (p *PrometheusExporter) createGaugeVec(name, help string, labelNames []string, defaultLabels prometheus.Labels) (*prometheus.GaugeVec, error) {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        name,
			Help:        help,
			ConstLabels: defaultLabels,
		},
		labelNames,
	)

	if err := p.registry.Register(gauge); err != nil {
		return nil, err
	}

	return gauge, nil
}

func (p *PrometheusExporter) convertLabels(labels Labels) prometheus.Labels {
	if labels == nil {
		return prometheus.Labels{}
	}
	
	promLabels := make(prometheus.Labels)
	for k, v := range labels {
		promLabels[k] = v
	}
	return promLabels
}

func (p *PrometheusExporter) convertLabelsToPromLabels(labels Labels) prometheus.Labels {
	if labels == nil {
		return prometheus.Labels{}
	}
	
	promLabels := make(prometheus.Labels)
	for k, v := range labels {
		promLabels[k] = v
	}
	return promLabels
}

func (p *PrometheusExporter) getLabelNames(labels Labels) []string {
	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	return names
}

// Ensure interface is implemented
var _ Exporter = (*PrometheusExporter)(nil)