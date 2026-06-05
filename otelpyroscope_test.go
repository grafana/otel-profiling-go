package otelpyroscope

import (
	"context"
	"runtime/pprof"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func collectLabels(ctx context.Context) map[string]string {
	m := make(map[string]string)
	pprof.ForLabels(ctx, func(key, value string) bool {
		m[key] = value
		return true
	})
	return m
}

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

func Test_traceIDLabel_defaultEnabled(t *testing.T) {
	tracer := NewTracerProvider(trace.NewTracerProvider()).Tracer("")

	ctx, root := tracer.Start(context.Background(), "Root")
	rootLabels := collectLabels(ctx)

	want := root.SpanContext().TraceID().String()
	if got := rootLabels[traceIDLabelName]; got != want {
		t.Fatalf("root trace_id = %q, want %q", got, want)
	}
	if len(want) != 32 {
		t.Fatalf("expected 32-char hex trace ID, got %q", want)
	}

	childCtx, child := tracer.Start(ctx, "Child")
	if got := collectLabels(childCtx)[traceIDLabelName]; got != want {
		t.Fatalf("child trace_id = %q, want %q (inherited)", got, want)
	}

	grandchildCtx, grandchild := tracer.Start(childCtx, "Grandchild")
	if got := collectLabels(grandchildCtx)[traceIDLabelName]; got != want {
		t.Fatalf("grandchild trace_id = %q, want %q", got, want)
	}

	grandchild.End()
	child.End()
	root.End()
}

func Test_traceIDLabel_unsampledSpan(t *testing.T) {
	tp := trace.NewTracerProvider(trace.WithSampler(trace.NeverSample()))
	tracer := NewTracerProvider(tp).Tracer("")

	ctx, span := tracer.Start(context.Background(), "Root")
	defer span.End()

	if span.SpanContext().IsSampled() {
		t.Fatal("expected unsampled span")
	}
	labels := collectLabels(ctx)
	if v, ok := labels[traceIDLabelName]; ok {
		t.Fatalf("trace_id should be absent on unsampled span, got %q", v)
	}
	if v, ok := labels[spanIDLabelName]; ok {
		t.Fatalf("span_id should be absent on unsampled span, got %q", v)
	}
}

func Test_traceIDLabel_inheritsAcrossScopeAllSpans(t *testing.T) {
	// trace_id must stay consistent across spans even when span_id uses
	// ScopeAllSpans (each span emits its own span_id).
	tracer := NewTracerProvider(
		trace.NewTracerProvider(),
		WithSpanIDLabelScope(ScopeAllSpans),
	).Tracer("")

	ctx, root := tracer.Start(context.Background(), "Root")
	defer root.End()
	want := root.SpanContext().TraceID().String()
	rootSpanID := root.SpanContext().SpanID().String()

	if got := collectLabels(ctx)[traceIDLabelName]; got != want {
		t.Fatalf("root trace_id = %q, want %q", got, want)
	}
	if got := collectLabels(ctx)[spanIDLabelName]; got != rootSpanID {
		t.Fatalf("root span_id = %q, want %q", got, rootSpanID)
	}

	childCtx, child := tracer.Start(ctx, "Child")
	defer child.End()
	childSpanID := child.SpanContext().SpanID().String()
	childLabels := collectLabels(childCtx)

	if got := childLabels[traceIDLabelName]; got != want {
		t.Fatalf("child trace_id = %q, want %q (constant across trace)", got, want)
	}
	if got := childLabels[spanIDLabelName]; got != childSpanID {
		t.Fatalf("child span_id = %q, want child's own %q under ScopeAllSpans", got, childSpanID)
	}
}

// Regression: WithSpanIDLabelScope used to write to spanNameScope.
func Test_WithSpanIDLabelScope_typoRegression(t *testing.T) {
	tracer := NewTracerProvider(
		trace.NewTracerProvider(),
		WithSpanIDLabelScope(ScopeAllSpans),
		WithSpanNameLabelScope(ScopeRootSpan),
	).Tracer("")

	ctx, root := tracer.Start(context.Background(), "Root")
	defer root.End()
	rootSpanID := root.SpanContext().SpanID().String()

	childCtx, child := tracer.Start(ctx, "Child")
	defer child.End()
	childSpanID := child.SpanContext().SpanID().String()
	if rootSpanID == childSpanID {
		t.Fatal("test setup: root and child span IDs should differ")
	}

	childLabels := collectLabels(childCtx)
	if got := childLabels[spanIDLabelName]; got != childSpanID {
		t.Fatalf("child span_id = %q, want %q", got, childSpanID)
	}
	if got := childLabels[spanNameLabelName]; got != "Root" {
		t.Fatalf("child span_name = %q, want %q", got, "Root")
	}
}
