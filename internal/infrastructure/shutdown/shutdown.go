// Package shutdown provides functionality for graceful application shutdown.
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// GracefulShutdown waits for termination signals and calls the provided shutdown function.
// It handles OS signals (SIGINT, SIGTERM, SIGHUP) and context cancellation to trigger
// graceful shutdown. It also handles multiple signals, forcing exit if a second signal
// is received during shutdown.
//
// Parameters:
//   - ctx: Context that can be cancelled to trigger shutdown
//   - logger: Logger for recording shutdown events
//   - shutdownFunc: Function to execute during shutdown
//
// Returns:
//   - The error from the shutdown function, if any
func GracefulShutdown(ctx context.Context, logger *zap.Logger, shutdownFunc func() error) error {
	// Create a channel to receive OS signals with buffer size 2 to avoid missing signals
	quit := make(chan os.Signal, 2)

	// Register for SIGINT, SIGTERM, and SIGHUP
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Make sure to stop signal handling when we're done
	defer signal.Stop(quit)

	// Create a channel to handle multiple signals
	done := make(chan struct{})
	var shutdownErr error

	// Start a goroutine to handle signals
	go func() {
		defer close(done)

		// Wait for either interrupt signal or context cancellation
		select {
		case sig := <-quit:
			logger.Info("Received termination signal", 
				zap.String("signal", sig.String()),
				zap.String("type", "first"))

			// Call the shutdown function
			logger.Info("Executing shutdown function")
			shutdownErr = shutdownFunc()
			if shutdownErr != nil {
				logger.Error("Error during shutdown", zap.Error(shutdownErr))
			} else {
				logger.Info("Graceful shutdown completed successfully")
			}

			// Handle additional signals during shutdown
			select {
			case sig := <-quit:
				logger.Warn("Received second termination signal during shutdown, forcing exit", 
					zap.String("signal", sig.String()),
					zap.String("type", "second"))
				os.Exit(1)
			default:
				// No second signal, continue normal shutdown
			}

		case <-ctx.Done():
			logger.Info("Context cancelled, shutting down", zap.Error(ctx.Err()))

			// Call the shutdown function
			logger.Info("Executing shutdown function")
			shutdownErr = shutdownFunc()
			if shutdownErr != nil {
				logger.Error("Error during shutdown", zap.Error(shutdownErr))
			} else {
				logger.Info("Graceful shutdown completed successfully")
			}

			// Handle signals during context-initiated shutdown
			select {
			case sig := <-quit:
				logger.Warn("Received termination signal during context-initiated shutdown, forcing exit", 
					zap.String("signal", sig.String()))
				os.Exit(1)
			default:
				// No signal, continue normal shutdown
			}
		}
	}()

	// Wait for shutdown to complete
	<-done
	return shutdownErr
}

// SetupGracefulShutdown sets up a goroutine that will handle graceful shutdown.
// It creates a new context with cancellation and starts a background goroutine
// that calls GracefulShutdown. This allows for both signal-based and programmatic
// shutdown initiation.
//
// Parameters:
//   - ctx: Parent context for the shutdown context
//   - logger: Logger for recording shutdown events
//   - shutdownFunc: Function to execute during shutdown
//
// Returns:
//   - A cancel function that can be called to trigger shutdown programmatically
func SetupGracefulShutdown(ctx context.Context, logger *zap.Logger, shutdownFunc func() error) context.CancelFunc {
	// Create a context with cancellation
	shutdownCtx, cancel := context.WithCancel(ctx)

	// Start a goroutine to handle shutdown
	go func() {
		// If an error occurs during shutdown, log it but don't propagate it
		// since this is a background goroutine
		err := GracefulShutdown(shutdownCtx, logger, shutdownFunc)
		if err != nil {
			logger.Error("Error during background graceful shutdown", zap.Error(err))
		}

		// Ensure context is cancelled when shutdown is complete
		cancel()
	}()

	return cancel
}
