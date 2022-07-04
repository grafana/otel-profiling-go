package otelpyroscope

import (
	"context"
	"net/url"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	profileIDLabelName = "profile_id"
	spanNameLabelName  = "span_name"
)

var (
	profileIDSpanAttributeKey          = attribute.Key("pyroscope.profile.id")
	profileURLSpanAttributeKey         = attribute.Key("pyroscope.profile.url")
	profileBaselineURLSpanAttributeKey = attribute.Key("pyroscope.profile.baseline.url")
	profileDiffURLSpanAttributeKey     = attribute.Key("pyroscope.profile.diff.url")
)

type Config struct {
	AppName                   string
	PyroscopeURL              string
	IncludeProfileURL         bool
	IncludeProfileBaselineURL bool
	ProfileBaselineLabels     map[string]string

	RootOnly    bool
	AddSpanName bool
}

type Option func(*tracerProvider)

// WithAppName specifies the profiled application name.
// It should match the name specified in pyroscope configuration.
// Required, if profile URL or profile baseline URL is enabled.
func WithAppName(app string) Option {
	return func(tp *tracerProvider) {
		tp.config.AppName = app
	}
}

// WithRootSpanOnly indicates that only the root span is to be profiled.
// The profile includes samples captured during child span execution
// but the spans won't have their own profiles and won't be annotated
// with pyroscope.profile attributes.
// The option is enabled by default.
func WithRootSpanOnly(x bool) Option {
	return func(tp *tracerProvider) {
		tp.config.RootOnly = x
	}
}

// WithAddSpanName specifies whether the current span name should be added
// to the profile labels. N.B if the name is dynamic, or too many values
// are supposed, this may significantly deteriorate performance.
// By default, span name is not added to profile labels.
func WithAddSpanName(x bool) Option {
	return func(tp *tracerProvider) {
		tp.config.AddSpanName = x
	}
}

// WithPyroscopeURL provides a base URL for the profile and baseline URLs.
// Required, if profile URL or profile baseline URL is enabled.
func WithPyroscopeURL(addr string) Option {
	return func(tp *tracerProvider) {
		tp.config.PyroscopeURL = addr
	}
}

// WithProfileURL specifies whether to add the pyroscope.profile.url
// attribute with the URL to the span profile.
func WithProfileURL(x bool) Option {
	return func(tp *tracerProvider) {
		tp.config.IncludeProfileURL = x
	}
}

// WithProfileBaselineURL specifies whether to add the
// pyroscope.profile.baseline.url attribute with the URL
// to the baseline profile. See WithProfileBaselineLabels.
func WithProfileBaselineURL(x bool) Option {
	return func(tp *tracerProvider) {
		tp.config.IncludeProfileBaselineURL = x
	}
}

// WithProfileBaselineLabels provides a map of extra labels to be added to the
// baseline query alongside with pprof labels set in runtime. Typically,
// it should match the labels specified in the Pyroscope profiler config.
// Note that the map must not be modified.
func WithProfileBaselineLabels(x map[string]string) Option {
	return func(tp *tracerProvider) {
		tp.config.ProfileBaselineLabels = x
	}
}

// WithProfileURLBuilder specifies how profile URL is to be built.
// DEPRECATED: use WithProfileURL
func WithProfileURLBuilder(b func(_ string) string) Option {
	return func(tp *tracerProvider) {
		tp.config.IncludeProfileURL = true
	}
}

// WithDefaultProfileURLBuilder specifies the default profile URL builder.
// DEPRECATED: use WithProfileURL
func WithDefaultProfileURLBuilder(_, _ string) Option {
	return func(tp *tracerProvider) {
		tp.config.IncludeProfileURL = true
	}
}

// tracerProvider satisfies open telemetry TracerProvider interface.
type tracerProvider struct {
	tp     trace.TracerProvider
	config Config
}

// NewTracerProvider creates a new tracer provider that annotates pprof
// profiles with span ID tag. This allows to establish a relationship
// between pprof profiles and reported tracing spans.
func NewTracerProvider(tp trace.TracerProvider, options ...Option) trace.TracerProvider {
	p := tracerProvider{
		tp:     tp,
		config: Config{RootOnly: true},
	}
	for _, o := range options {
		o(&p)
	}
	return &p
}

func (w *tracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return &profileTracer{p: w, tr: w.tp.Tracer(name, opts...)}
}

type profileTracer struct {
	p  *tracerProvider
	tr trace.Tracer
}

func (w profileTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if w.p.config.RootOnly && !isRootSpan(trace.SpanContextFromContext(ctx)) {
		return w.tr.Start(ctx, spanName, opts...)
	}
	ctx, span := w.tr.Start(ctx, spanName, opts...)
	s := spanWrapper{
		Span:      span,
		profileID: trace.SpanContextFromContext(ctx).SpanID().String(),
		startTime: time.Now(),
		ctx:       ctx,
		p:         w.p,
	}

	labels := []string{profileIDLabelName, s.profileID}
	if w.p.config.AddSpanName && spanName != "" {
		labels = append(labels, spanNameLabelName, spanName)
	}

	ctx = pprof.WithLabels(ctx, pprof.Labels(labels...))
	pprof.SetGoroutineLabels(ctx)
	s.pprofCtx = ctx
	return ctx, &s
}

var emptySpanID trace.SpanID

func isRootSpan(s trace.SpanContext) bool {
	return s.IsRemote() || s.SpanID() == emptySpanID
}

type spanWrapper struct {
	trace.Span

	// Span context.
	ctx context.Context
	// Current pprof context with labels.
	pprofCtx  context.Context
	profileID string
	startTime time.Time

	p *tracerProvider
}

func (s spanWrapper) End(options ...trace.SpanEndOption) {
	// By this profiles can be easily associated with the corresponding spans.
	// We use span ID as a profile ID because it perfectly fits profiling scope.
	// In practice, a profile ID is an arbitrary string identifying the execution
	// scope that is associated with a tracing span.
	s.SetAttributes(profileIDSpanAttributeKey.String(s.profileID))
	// Optionally specify the profile URL.
	if s.p.config.IncludeProfileURL {
		s.setProfileURL()
	}
	if s.p.config.IncludeProfileBaselineURL {
		s.setBaselineURLs()
	}
	s.Span.End(options...)
	pprof.SetGoroutineLabels(s.ctx)
}

func (s spanWrapper) setProfileURL() {
	q := make(url.Values, 3)
	from := strconv.FormatInt(s.startTime.Unix(), 10)
	until := strconv.FormatInt(time.Now().Unix(), 10)
	q.Set("query", s.p.config.AppName+`.cpu{`+profileIDLabelName+`="`+s.profileID+`"}`)
	q.Set("from", from)
	q.Set("until", until)
	s.SetAttributes(profileURLSpanAttributeKey.String(s.p.config.PyroscopeURL + "/?" + q.Encode()))
}

func (s spanWrapper) setBaselineURLs() {
	var b strings.Builder
	pprof.ForLabels(s.pprofCtx, func(key, value string) bool {
		if key == profileIDLabelName {
			return true
		}
		if s.p.config.ProfileBaselineLabels != nil {
			if _, ok := s.p.config.ProfileBaselineLabels[key]; ok {
				return true
			}
		}
		writeLabel(&b, key, value)
		return true
	})
	for key, value := range s.p.config.ProfileBaselineLabels {
		if value != "" {
			writeLabel(&b, key, value)
		}
	}

	q := make(url.Values, 9)
	from := strconv.FormatInt(s.startTime.Unix()-3600, 10)
	until := strconv.FormatInt(time.Now().Unix(), 10)
	baselineQuery := s.p.config.AppName + `.cpu{` + b.String() + `}`

	q.Set("query", baselineQuery)
	q.Set("from", from)
	q.Set("until", until)

	q.Set("rightQuery", s.p.config.AppName+`.cpu{`+profileIDLabelName+`="`+s.profileID+`"}`)
	q.Set("rightFrom", from)
	q.Set("rightUntil", until)

	q.Set("leftQuery", baselineQuery)
	q.Set("leftFrom", from)
	q.Set("leftUntil", until)

	qs := q.Encode()
	s.SetAttributes(profileBaselineURLSpanAttributeKey.String(s.p.config.PyroscopeURL + "/comparison?" + qs))
	s.SetAttributes(profileDiffURLSpanAttributeKey.String(s.p.config.PyroscopeURL + "/comparison-diff?" + qs))
}

func writeLabel(b *strings.Builder, k, v string) {
	if b.Len() > 0 {
		b.WriteByte(',')
	}
	b.WriteString(k + `="` + v + `"`)
}
