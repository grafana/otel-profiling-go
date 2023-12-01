module github.com/pyroscope-io/otel-profiling-go/example

go 1.16

require (
	github.com/grafana/otel-profiling-go v0.5.0
	github.com/grafana/pyroscope-go v1.0.4
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.4.1
	go.opentelemetry.io/otel/sdk v1.21.0
)

replace github.com/grafana/otel-profiling-go => ../
