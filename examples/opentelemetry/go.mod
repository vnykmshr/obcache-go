module opentelemetry-example

go 1.21

replace github.com/vnykmshr/obcache-go => ../..

require (
	github.com/prometheus/client_golang v1.17.0
	github.com/vnykmshr/obcache-go v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/prometheus v0.44.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.21.0
	go.opentelemetry.io/otel/metric v1.21.0
	go.opentelemetry.io/otel/propagation v1.21.0
	go.opentelemetry.io/otel/sdk v1.21.0
	go.opentelemetry.io/otel/sdk/metric v1.21.0
	go.opentelemetry.io/otel/trace v1.21.0
)