module github.com/pyroscope-io/otel-profiling-go/example

go 1.24.0

require (
	github.com/grafana/otel-profiling-go v0.5.0
	github.com/grafana/pyroscope-go v1.3.0
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.4.1
	go.opentelemetry.io/otel/sdk v1.21.0
)

require (
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/grafana/pyroscope-go/godeltaprof v0.1.10 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	go.opentelemetry.io/otel/metric v1.21.0 // indirect
	go.opentelemetry.io/otel/trace v1.21.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
)

replace github.com/grafana/otel-profiling-go => ../
