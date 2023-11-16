module github.com/pyroscope-io/otel-profiling-go/example

go 1.16

require (
	github.com/grafana/pyroscope-go v1.0.4 // indirect
	github.com/pyroscope-io/otel-profiling-go v0.4.0
	go.opentelemetry.io/otel v1.20.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.4.1
	go.opentelemetry.io/otel/sdk v1.20.0
)

replace github.com/pyroscope-io/otel-profiling-go => ../
