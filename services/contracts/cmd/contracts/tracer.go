package main

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// noopTracer satisfies trace.Tracer for use before the full OTel
// initialization is wired. Replaced by the real tracer in production.
type noopTracer struct{}

func (noopTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return noop.NewTracerProvider().Tracer("").Start(ctx, spanName, opts...)
}
