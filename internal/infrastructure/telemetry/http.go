package telemetry

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// HTTPConfig holds configuration for HTTP instrumentation
type HTTPConfig struct {
	TracingEnabled bool
	// Add other HTTP instrumentation config options as needed
}

// NewHTTPConfig creates a new HTTP instrumentation configuration
func NewHTTPConfig() HTTPConfig {
	return HTTPConfig{
		TracingEnabled: true, // Default to enabled
	}
}

// InstrumentHandler wraps an http.Handler with OpenTelemetry instrumentation
func InstrumentHandler(handler http.Handler, operation string, opts ...otelhttp.Option) http.Handler {
	return otelhttp.NewHandler(handler, operation, opts...)
}

// InstrumentClient wraps an http.Client with OpenTelemetry instrumentation
func InstrumentClient(client *http.Client, opts ...otelhttp.Option) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}

	client.Transport = otelhttp.NewTransport(
		client.Transport,
		opts...,
	)

	return client
}

// NewHTTPMiddleware creates a new middleware for HTTP request tracing
func NewHTTPMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	tracer := otel.Tracer("infrastructure.telemetry.http")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
			defer span.End()

			// Add common attributes to the span
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.host", r.Host),
				attribute.String("http.user_agent", r.UserAgent()),
			)

			// Add trace ID to response headers for debugging
			traceID := span.SpanContext().TraceID().String()
			w.Header().Set("X-Trace-ID", traceID)

			// Log request with trace ID
			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("trace_id", traceID),
			)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// StartSpan is a helper function to start a new span from a context
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer("infrastructure.telemetry")
	return tracer.Start(ctx, name)
}

// AddSpanAttributes adds attributes to the current span in context
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordErrorSpan records an error on the current span in context
func RecordErrorSpan(ctx context.Context, err error, opts ...trace.EventOption) {
	if err != nil {
		span := trace.SpanFromContext(ctx)
		span.RecordError(err, opts...)
	}
}
