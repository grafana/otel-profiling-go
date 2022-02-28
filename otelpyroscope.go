package otelpyroscope

import (
	"context"
	"runtime/pprof"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const profileIDLabelName = "profile_id"

var (
	profileIDSpanAttributeKey  = attribute.Key("pyroscope.profile.id")
	profileURLSpanAttributeKey = attribute.Key("pyroscope.profile.url")
)

// tracerProvider satisfies open telemetry TracerProvider interface.
type tracerProvider struct {
	tp trace.TracerProvider

	rootOnly bool
	buildURL func(string) string
}

// NewTracerProvider creates a new tracer provider that annotates pprof
// profiles with span ID tag. This allows to establish a relationship
// between pprof profiles and reported tracing spans.
func NewTracerProvider(tp trace.TracerProvider, options ...Option) trace.TracerProvider {
	p := tracerProvider{
		tp:       tp,
		rootOnly: true,
	}
	for _, o := range options {
		o(&p)
	}
	return &p
}

type Option func(*tracerProvider)

// WithRootSpanOnly indicates that only the root span is to be profiled.
// The profile includes samples captured during child span execution
// but the spans won't have their own profiles and won't be annotated
// with pyroscope.profile attributes.
func WithRootSpanOnly(rootOnly bool) Option {
	return func(tp *tracerProvider) {
		tp.rootOnly = rootOnly
	}
}

// WithProfileURLBuilder specifies how profile URL is to be built. Optional.
func WithProfileURLBuilder(b func(profileID string) string) Option {
	return func(tp *tracerProvider) {
		tp.buildURL = b
	}
}

func WithDefaultProfileURLBuilder(addr string, app string) Option {
	return func(tp *tracerProvider) {
		tp.buildURL = DefaultProfileURLBuilder(addr, app)
	}
}

func DefaultProfileURLBuilder(addr string, app string) func(string) string {
	return func(id string) string {
		return addr + "?query=" + app + ".cpu%7Bprofile_id%3D%22" + id + "%22%7D"
	}
}

func (w tracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return &profileTracer{p: w, tr: w.tp.Tracer(name, opts...)}
}

type profileTracer struct {
	p  tracerProvider
	tr trace.Tracer
}

func (w profileTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if w.p.rootOnly && !isRootSpan(trace.SpanContextFromContext(ctx)) {
		return w.tr.Start(ctx, spanName, opts...)
	}
	ctx, span := w.tr.Start(ctx, spanName, opts...)
	s := spanWrapper{
		profileID: trace.SpanContextFromContext(ctx).SpanID().String(),
		Span:      span,
		ctx:       ctx,
		p:         w.p,
	}
	ctx = pprof.WithLabels(ctx, pprof.Labels(profileIDLabelName, s.profileID))
	pprof.SetGoroutineLabels(ctx)
	return ctx, &s
}

var emptySpanID trace.SpanID

func isRootSpan(s trace.SpanContext) bool {
	return s.IsRemote() || s.SpanID() == emptySpanID
}

type spanWrapper struct {
	trace.Span
	ctx context.Context

	profileID string
	p         tracerProvider
}

func (s spanWrapper) End(options ...trace.SpanEndOption) {
	// By this profiles can be easily associated with the corresponding spans.
	// We use span ID as a profile ID because it perfectly fits profiling scope.
	// In practice, a profile ID is an arbitrary string identifying the execution
	// scope that is associated with a tracing span.
	s.SetAttributes(profileIDSpanAttributeKey.String(s.profileID))
	// Optionally specify the profile URL.
	if s.p.buildURL != nil {
		s.SetAttributes(profileURLSpanAttributeKey.String(s.p.buildURL(s.profileID)))
	}
	s.Span.End(options...)
	pprof.SetGoroutineLabels(s.ctx)
}
