package middleware

import (
	"context"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recovery is a middleware that recovers from panics and logs the error
func Recovery(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the error and stack trace
					logger.Error("Panic recovered",
						zap.Any("error", err),
						zap.String("stack", string(debug.Stack())),
						zap.String("url", r.URL.String()),
						zap.String("method", r.Method),
					)

					// Return a 500 Internal Server Error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, writeErr := w.Write([]byte(`{"error":"Internal Server Error"}`))
					if writeErr != nil {
						logger.Error("Failed to write error response", zap.Error(writeErr))
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// ContextCancellation is a middleware that checks for context cancellation
func ContextCancellation(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a done channel to signal when the handler is complete
			done := make(chan struct{})

			// Create a copy of the request with a context that we can check
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()
			r = r.WithContext(ctx)

			go func() {
				next.ServeHTTP(w, r)
				close(done)
			}()

			select {
			case <-done:
				// Request completed normally
				return
			case <-r.Context().Done():
				// Client disconnected or request was cancelled
				logger.Info("Request cancelled by client",
					zap.String("url", r.URL.String()),
					zap.String("method", r.Method),
					zap.Error(r.Context().Err()),
				)
				// We don't need to send a response as the client has disconnected
				cancel()
				return
			}
		})
	}
}
