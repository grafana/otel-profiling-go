package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/grafana/pyroscope-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"

	otelpyroscope "github.com/grafana/otel-profiling-go"
)

func main() {
	// Initialize your tracer provider as usual.
	tp := initTracer()

	// Wrap it with otelpyroscope tracer provider.
	otel.SetTracerProvider(otelpyroscope.NewTracerProvider(tp))

	// Initialize pyroscope profiler.
	_, _ = pyroscope.Start(pyroscope.Config{
		ApplicationName: "my-service",
		ServerAddress:   "http://localhost:4040",
	})

	log.Println("starting listening")
	err := http.ListenAndServe(":5000", http.HandlerFunc(cpuBoundHandler))
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func cpuBoundHandler(_ http.ResponseWriter, r *http.Request) {
	tracer := otel.GetTracerProvider().Tracer("")
	_, span := tracer.Start(r.Context(), "cpuBoundHandler")
	defer span.End()

	var i int64 = 0
	st := time.Now().Unix()
	for (time.Now().Unix() - st) < 2 {
		i++
	}
}

func initTracer() *trace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(exporter),
	)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp
}
