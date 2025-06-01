// Package server provides functionality for HTTP server management.
package server

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/abitofhelp/family-service2/internal/infrastructure/logging"
	"go.uber.org/zap"
)

// Server represents an HTTP server with additional functionality.
// It extends the standard http.Server with logging, context-aware logging,
// and graceful shutdown capabilities.
type Server struct {
	*http.Server                      // Embedded standard HTTP server
	logger          *zap.Logger       // Logger for server events
	contextLogger   *logging.ContextLogger // Context-aware logger
	shutdownTimeout time.Duration     // Maximum time to wait for server shutdown
}

// Config contains server configuration parameters.
// It defines all the timeouts and connection settings for the HTTP server.
type Config struct {
	Port            string        // Port on which the server will listen
	ReadTimeout     time.Duration // Maximum duration for reading the entire request
	WriteTimeout    time.Duration // Maximum duration before timing out writes of the response
	IdleTimeout     time.Duration // Maximum amount of time to wait for the next request
	ShutdownTimeout time.Duration // Maximum time to wait for server shutdown
}

// NewConfig creates a new server configuration from the provided values.
// It initializes a Config struct with the specified parameters.
//
// Parameters:
//   - port: The port on which the server will listen
//   - readTimeout: Maximum duration for reading the entire request
//   - writeTimeout: Maximum duration before timing out writes of the response
//   - idleTimeout: Maximum amount of time to wait for the next request
//   - shutdownTimeout: Maximum time to wait for server shutdown
//
// Returns:
//   - A Config struct initialized with the provided values
func NewConfig(port string, readTimeout, writeTimeout, idleTimeout, shutdownTimeout time.Duration) Config {
	return Config{
		Port:            port,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ShutdownTimeout: shutdownTimeout,
	}
}

// New creates a new server with the given configuration and handler.
// It applies middleware to the handler and initializes the HTTP server.
//
// Parameters:
//   - cfg: Server configuration parameters
//   - handler: HTTP handler for processing requests
//   - logger: Logger for server events
//   - contextLogger: Context-aware logger for request-scoped logging
//
// Returns:
//   - A pointer to a new Server instance
func New(cfg Config, handler http.Handler, logger *zap.Logger, contextLogger *logging.ContextLogger) *Server {
	// Apply middleware to the handler
	wrappedHandler := applyMiddleware(handler, logger)

	return &Server{
		Server: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      wrappedHandler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger:          logger,
		contextLogger:   contextLogger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

// applyMiddleware wraps the handler with middleware for panic recovery and context cancellation handling.
// It creates a middleware chain that ensures proper error handling and request lifecycle management.
//
// Parameters:
//   - handler: The HTTP handler to wrap with middleware
//   - logger: Logger for recording middleware events
//
// Returns:
//   - A wrapped HTTP handler with middleware applied
func applyMiddleware(handler http.Handler, logger *zap.Logger) http.Handler {
	// Import the middleware package
	// This is done here to avoid circular dependencies
	middleware := func(next http.Handler) http.Handler {
		// Recover from panics
		recoveryMiddleware := func(next http.Handler) http.Handler {
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

		// Check for context cancellation
		contextCancellationMiddleware := func(next http.Handler) http.Handler {
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

		// Apply middleware in order (outermost first)
		handler := recoveryMiddleware(next)
		handler = contextCancellationMiddleware(handler)
		return handler
	}

	return middleware(handler)
}

// Start starts the server in a goroutine.
// It begins listening for HTTP requests in a non-blocking manner.
// If the server fails to start, it logs a fatal error and terminates the application.
func (s *Server) Start() {
	s.contextLogger.Info(context.Background(), "Starting server", zap.String("address", s.Addr))
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.contextLogger.Fatal(context.Background(), "Server failed to start", zap.Error(err))
		}
	}()
}

// Shutdown gracefully shuts down the server.
// It stops accepting new connections and waits for existing connections to complete
// processing, up to the configured shutdown timeout.
//
// Parameters:
//   - ctx: Context for the shutdown operation, which may be cancelled
//
// Returns:
//   - An error if the shutdown fails, or nil on success
func (s *Server) Shutdown(ctx context.Context) error {
	s.contextLogger.Info(ctx, "Shutting down server...")

	// Create a deadline to wait for server shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer shutdownCancel()

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline
	if err := s.Server.Shutdown(shutdownCtx); err != nil {
		s.contextLogger.Error(ctx, "Server forced to shutdown", zap.Error(err))
		return err
	}

	s.contextLogger.Info(ctx, "Server exited properly")
	return nil
}
