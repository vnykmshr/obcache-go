#!/bin/bash

# Comprehensive Benchmarking Script for obcache-go
# This script runs all benchmarks and generates performance reports

set -e

echo "🚀 obcache-go Comprehensive Benchmark Suite"
echo "============================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create benchmark results directory
RESULTS_DIR="benchmark-results"
mkdir -p $RESULTS_DIR
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

echo -e "${BLUE}📊 Running Cache Operation Benchmarks${NC}"
echo "----------------------------------------"
go test -bench=BenchmarkCache -benchmem ./pkg/obcache/ | tee $RESULTS_DIR/cache_ops_$TIMESTAMP.txt

echo ""
echo -e "${BLUE}⚡ Running Function Wrapping Benchmarks${NC}"
echo "-------------------------------------------"
go test -bench=BenchmarkWrapped -benchmem ./pkg/obcache/ | tee $RESULTS_DIR/wrapping_$TIMESTAMP.txt

echo ""
echo -e "${BLUE}🔄 Running Direct vs Cached Comparison${NC}"
echo "----------------------------------------"
go test -bench=BenchmarkDirect -benchmem ./pkg/obcache/ | tee $RESULTS_DIR/direct_vs_cached_$TIMESTAMP.txt

echo ""
echo -e "${BLUE}🏃 Running Concurrent Access Benchmarks${NC}"
echo "-------------------------------------------"
go test -bench=BenchmarkConcurrent -benchmem ./pkg/obcache/ | tee $RESULTS_DIR/concurrent_$TIMESTAMP.txt

echo ""
echo -e "${BLUE}⭐ Running Singleflight Benchmarks${NC}"
echo "----------------------------------"
go test -bench=BenchmarkSingleflight -benchmem ./internal/singleflight/ | tee $RESULTS_DIR/singleflight_$TIMESTAMP.txt

echo ""
echo -e "${BLUE}🔑 Running Key Generation Benchmarks${NC}"
echo "------------------------------------"
go test -bench=BenchmarkDefaultKeyFunc -benchmem ./pkg/obcache/ | tee $RESULTS_DIR/keygen_$TIMESTAMP.txt
go test -bench=BenchmarkSimpleKeyFunc -benchmem ./pkg/obcache/ | tee -a $RESULTS_DIR/keygen_$TIMESTAMP.txt

echo ""
echo -e "${BLUE}📈 Running Comprehensive Comparison${NC}"
echo "-----------------------------------"
go test -bench=BenchmarkCacheEffectivenessComparison -benchmem ./pkg/obcache/ | tee $RESULTS_DIR/effectiveness_$TIMESTAMP.txt

echo ""
echo -e "${BLUE}🌐 Running Web Server Simulation${NC}"
echo "---------------------------------"
go test -bench=BenchmarkWebServerSimulation -benchmem ./pkg/obcache/ | tee $RESULTS_DIR/webserver_sim_$TIMESTAMP.txt

echo ""
echo -e "${BLUE}🎯 Running TTL Strategy Benchmarks${NC}"
echo "----------------------------------"
go test -bench=BenchmarkShortTTL -benchmem ./pkg/obcache/ | tee $RESULTS_DIR/ttl_strategies_$TIMESTAMP.txt
go test -bench=BenchmarkLongTTL -benchmem ./pkg/obcache/ | tee -a $RESULTS_DIR/ttl_strategies_$TIMESTAMP.txt

echo ""
echo -e "${GREEN}✅ All benchmarks completed!${NC}"
echo ""
echo -e "${YELLOW}📋 Results Summary:${NC}"
echo "- Results saved to: $RESULTS_DIR/"
echo "- Timestamp: $TIMESTAMP"
echo ""

# Generate a summary report
SUMMARY_FILE="$RESULTS_DIR/summary_$TIMESTAMP.md"
cat > $SUMMARY_FILE << EOF
# obcache-go Benchmark Results

Generated on: $(date)

## Overview

This report contains comprehensive benchmark results for obcache-go, measuring:
- Basic cache operations performance
- Function wrapping overhead and benefits
- Concurrent access patterns
- Singleflight effectiveness
- Key generation performance
- Real-world usage simulations

## How to Read Results

Benchmark results format:
\`\`\`
BenchmarkName-8    iterations    ns/op    B/op    allocs/op
\`\`\`

- **iterations**: Number of times the benchmark ran
- **ns/op**: Nanoseconds per operation (lower is better)
- **B/op**: Bytes allocated per operation (lower is better)
- **allocs/op**: Number of allocations per operation (lower is better)

## Key Findings

### Cache Effectiveness
- Compare BenchmarkDirect* vs BenchmarkWrapped* results
- Look for significant performance improvements in cached scenarios
- Note the trade-off between first call overhead and subsequent call speed

### Concurrency Performance
- Concurrent benchmarks show the library's thread-safety overhead
- Singleflight benchmarks demonstrate deduplication effectiveness

### Memory Usage
- Lower allocations/op indicates better memory efficiency
- Key generation strategy impacts memory usage

## Files Generated

EOF

# List all generated files
for file in $RESULTS_DIR/*_$TIMESTAMP.txt; do
    basename_file=$(basename "$file")
    echo "- $basename_file" >> $SUMMARY_FILE
done

echo "" >> $SUMMARY_FILE
echo "## Next Steps" >> $SUMMARY_FILE
echo "" >> $SUMMARY_FILE
echo "1. Review individual benchmark files for detailed results" >> $SUMMARY_FILE
echo "2. Compare results across different runs to identify performance regressions" >> $SUMMARY_FILE
echo "3. Use these benchmarks as baseline for performance optimizations" >> $SUMMARY_FILE

echo -e "${GREEN}📄 Summary report generated: $SUMMARY_FILE${NC}"
echo ""
echo -e "${BLUE}🔍 Quick Performance Check:${NC}"
echo "To run a quick performance comparison:"
echo "go test -bench=BenchmarkCacheEffectivenessComparison -benchmem ./pkg/obcache/"
echo ""
echo "To profile memory usage:"
echo "go test -bench=BenchmarkCacheMemoryUsage -benchmem -memprofile=mem.prof ./pkg/obcache/"
echo ""
echo "To profile CPU usage:"
echo "go test -bench=BenchmarkConcurrentWrappedFunction -benchmem -cpuprofile=cpu.prof ./pkg/obcache/"