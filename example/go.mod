module github.com/pyroscope-io/otelpyroscope/example

go 1.16

replace github.com/pyroscope-io/otelpyroscope => ../

require (
	github.com/pyroscope-io/client v0.2.1 // indirect
	github.com/pyroscope-io/otelpyroscope v0.1.0
	go.opentelemetry.io/otel v1.4.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.4.1
	go.opentelemetry.io/otel/sdk v1.4.1
)
