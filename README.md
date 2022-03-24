# Profiling Instrumentation

**NOTE**: Tracing integration is supported in [Pyroscope](https://pyroscope.io) starting from v0.14.0.

The package provides means to integrate tracing with profiling. More specifically, a `TracerProvider` implementation,
that annotates profiling data with span IDs: when a new trace span emerges, the tracer adds a `profile_id` [pprof tag](https://github.com/google/pprof/blob/master/doc/README.md#tag-filtering)
that points to the span. This makes it possible to filter out a profile of a particular trace span in [Pyroscope](https://pyroscope.io).
You can find a full example in the [example](/example) directory or in the [Pyroscope repository](https://github.com/pyroscope-io/pyroscope/tree/main/examples/tracing).

Note that the module does not control `pprof` profiler itself – it still needs to be started for profiles to be
collected. This can be done either via `runtime/pprof` package, or using the [Pyroscope client](https://github.com/pyroscope-io/client).

By default, only the root span gets annotated (the first span created locally), this is done to circumvent the fact that
the profiler records only the time spent on CPU. Otherwise, all the children profiles should be merged to get the full
representation of the root span profile.

There are number of limitations:
 - Only Go CPU profiling is fully supported at the moment.
 - Due to the very idea of the sampling profilers, spans shorter than the sample interval may not be captured. For example, Go CPU profiler probes stack traces 100 times per second, meaning that spans shorter than 10ms may not be captured.
