package otelpyroscope

import (
	"context"
	"runtime/pprof"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func Test_tracerProvider(t *testing.T) {
	otel.SetTracerProvider(NewTracerProvider(trace.NewTracerProvider()))

	tracer := otel.Tracer("")
	labels := make(map[string]string)

	ctx, spanR := tracer.Start(context.Background(), "RootSpan")
	pprof.ForLabels(ctx, func(key, value string) bool {
		labels[key] = value
		return true
	})
	spanID, ok := labels[spanIDLabelName]
	if !ok {
		t.Fatal("span ID label not found")
	}
	if len(spanID) != 16 {
		t.Fatalf("invalid span ID: %q", spanID)
	}
	name, ok := labels[spanNameLabelName]
	if !ok {
		t.Fatal("span name label not found")
	}
	if name != "RootSpan" {
		t.Fatalf("invalid span name: %q", name)
	}

	// Nested child span has the same labels.
	ctx, spanA := tracer.Start(ctx, "SpanA")
	pprof.ForLabels(ctx, func(key, value string) bool {
		if v, ok := labels[key]; !ok || v != value {
			t.Fatalf("nested span labels mismatch: %q=%q", key, value)
		}
		return true
	})

	spanA.End()
	spanR.End()

	// Child span created after the root span end using its context.
	ctx, spanB := tracer.Start(ctx, "SpanB")
	pprof.ForLabels(ctx, func(key, value string) bool {
		if v, ok := labels[key]; !ok || v != value {
			t.Fatalf("nested span labels mismatch: %q=%q", key, value)
		}
		return true
	})
	spanB.End()

	// A new root span.
	ctx, spanC := tracer.Start(context.Background(), "SpanC")
	pprof.ForLabels(ctx, func(key, value string) bool {
		if v, ok := labels[key]; !ok || v == value {
			t.Fatalf("unexpected match: %q=%q", key, value)
		}
		return true
	})
	spanC.End()
}

func Test_tracerProvider_WithTraceID(t *testing.T) {
	otel.SetTracerProvider(NewTracerProvider(trace.NewTracerProvider(), WithTraceID()))

	tracer := otel.Tracer("")
	labels := make(map[string]string)

	ctx, spanR := tracer.Start(context.Background(), "RootSpan")
	pprof.ForLabels(ctx, func(key, value string) bool {
		labels[key] = value
		return true
	})

	// Verify span_id is present
	spanID, ok := labels[spanIDLabelName]
	if !ok {
		t.Fatal("span ID label not found")
	}
	if len(spanID) != 16 {
		t.Fatalf("invalid span ID: %q", spanID)
	}

	// Verify span_name is present
	name, ok := labels[spanNameLabelName]
	if !ok {
		t.Fatal("span name label not found")
	}
	if name != "RootSpan" {
		t.Fatalf("invalid span name: %q", name)
	}

	// Verify trace_id is present
	traceID, ok := labels[traceIDLabelName]
	if !ok {
		t.Fatal("trace ID label not found")
	}
	if len(traceID) != 32 {
		t.Fatalf("invalid trace ID length: expected 32, got %d for %q", len(traceID), traceID)
	}

	// Nested child span should have the same trace_id
	ctx, spanA := tracer.Start(ctx, "SpanA")
	childLabels := make(map[string]string)
	pprof.ForLabels(ctx, func(key, value string) bool {
		childLabels[key] = value
		return true
	})

	childTraceID, ok := childLabels[traceIDLabelName]
	if !ok {
		t.Fatal("trace ID label not found in child span")
	}
	if childTraceID != traceID {
		t.Fatalf("child span trace ID mismatch: expected %q, got %q", traceID, childTraceID)
	}

	spanA.End()
	spanR.End()

	// A new root span should have a different trace_id
	ctx, spanB := tracer.Start(context.Background(), "SpanB")
	newLabels := make(map[string]string)
	pprof.ForLabels(ctx, func(key, value string) bool {
		newLabels[key] = value
		return true
	})

	newTraceID, ok := newLabels[traceIDLabelName]
	if !ok {
		t.Fatal("trace ID label not found in new root span")
	}
	if newTraceID == traceID {
		t.Fatalf("new root span should have different trace ID, but got same: %q", newTraceID)
	}
	if len(newTraceID) != 32 {
		t.Fatalf("invalid trace ID length: expected 32, got %d for %q", len(newTraceID), newTraceID)
	}

	spanB.End()
}
