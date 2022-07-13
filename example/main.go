package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/pyroscope-io/client/pyroscope"
	otelpyroscope "github.com/pyroscope-io/otel-profiling-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	appName           = "example-app"
	pyroscopeEndpoint = "http://localhost:4040"
)

func main() {
	tp := initTracer(appName, pyroscopeEndpoint)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	_, _ = pyroscope.Start(pyroscope.Config{
		ApplicationName: appName,
		ServerAddress:   pyroscopeEndpoint,
	})

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

func initTracer(appName, pyroscopeEndpoint string) *trace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(otelpyroscope.NewTracerProvider(tp,
		otelpyroscope.WithAppName(appName),
		otelpyroscope.WithPyroscopeURL(pyroscopeEndpoint),
		otelpyroscope.WithRootSpanOnly(true),
		otelpyroscope.WithAddSpanName(true),
		otelpyroscope.WithProfileURL(true),
		otelpyroscope.WithProfileBaselineURL(true),
	))
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp
}
