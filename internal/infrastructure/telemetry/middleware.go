package telemetry

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingMiddleware provides middleware for tracing HTTP requests
type TracingMiddleware struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	logger     *zap.Logger
}

// NewTracingMiddleware creates a new tracing middleware
func NewTracingMiddleware(logger *zap.Logger) *TracingMiddleware {
	return &TracingMiddleware{
		tracer:     otel.Tracer("infrastructure.telemetry.middleware"),
		propagator: otel.GetTextMapPropagator(),
		logger:     logger,
	}
}

// Middleware returns an http.Handler middleware function
func (m *TracingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract context from the incoming request
		ctx := m.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		
		// Start a new span
		operationName := r.Method + " " + r.URL.Path
		ctx, span := m.tracer.Start(ctx, operationName)
		defer span.End()
		
		// Add basic span attributes
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.host", r.Host),
			attribute.String("http.user_agent", r.UserAgent()),
			attribute.String("http.flavor", r.Proto),
			attribute.String("http.scheme", getScheme(r)),
			attribute.String("http.target", r.URL.Path),
		)
		
		// Add trace ID to response headers for debugging
		traceID := span.SpanContext().TraceID().String()
		w.Header().Set("X-Trace-ID", traceID)
		
		// Create a wrapped response writer to capture status code
		wrw := &wrappedResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200 OK
		}
		
		// Record start time
		startTime := time.Now()
		
		// Call the next handler with the context containing the span
		next.ServeHTTP(wrw, r.WithContext(ctx))
		
		// Record duration
		duration := time.Since(startTime)
		
		// Add response attributes to the span
		span.SetAttributes(
			attribute.Int("http.status_code", wrw.statusCode),
			attribute.Int64("http.response_content_length", wrw.contentLength),
			attribute.Int64("http.duration_ms", duration.Milliseconds()),
		)
		
		// Log the request with tracing information
		m.logger.Info("HTTP request completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrw.statusCode),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.String("trace_id", traceID),
		)
	})
}

// wrappedResponseWriter is a wrapper for http.ResponseWriter that captures status code and content length
type wrappedResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	contentLength int64
}

// WriteHeader captures the status code
func (w *wrappedResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the content length
func (w *wrappedResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.contentLength += int64(n)
	return n, err
}

// getScheme returns the request scheme (http or https)
func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}