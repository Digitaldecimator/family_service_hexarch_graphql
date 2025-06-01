// Package main is the entry point for the family service application.
// It initializes the application, sets up the HTTP server, and handles graceful shutdown.
// The application follows a hexagonal architecture pattern, with the main package
// serving as the primary adapter that connects the application to the outside world.
package main

import (
	"context"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/abitofhelp/family-service2/internal/adapters/graphql"
	"github.com/abitofhelp/family-service2/internal/infrastructure/config"
	"github.com/abitofhelp/family-service2/internal/infrastructure/di"
	"github.com/abitofhelp/family-service2/internal/infrastructure/health"
	"github.com/abitofhelp/family-service2/internal/infrastructure/logging"
	"github.com/abitofhelp/family-service2/internal/infrastructure/server"
	"github.com/abitofhelp/family-service2/internal/infrastructure/shutdown"
	"github.com/abitofhelp/family-service2/internal/infrastructure/telemetry"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
)

// containerAdapter adapts the di.Container to implement health.HealthCheckProvider
type containerAdapter struct {
	*di.Container
}

// GetRepositoryFactory returns the repository factory as an interface{}
func (a *containerAdapter) GetRepositoryFactory() any {
	return a.Container.GetRepositoryFactory()
}

// No global variables needed

func main() {
	// Create a root context with cancellation
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel() // Ensure all resources are cleaned up when main exits

	// Initialize configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load application configuration: %v", err)
	}

	// Initialize logger
	logger, err := logging.NewLogger(cfg.Log.Level, cfg.Log.Development)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize dependency injection container
	container, err := di.NewContainer(rootCtx, logger, cfg)
	if err != nil {
		logger.Fatal("Failed to initialize dependency injection container", zap.Error(err))
	}
	defer func() {
		if err := container.Close(); err != nil {
			logger.Error("Error closing container", zap.Error(err))
		}
	}()

	// Create a container adapter for health checks
	adapter := &containerAdapter{Container: container}

	// Create a ServeMux for routing
	mux := http.NewServeMux()

	// Set up health check endpoint
	healthEndpoint := cfg.Server.HealthEndpoint
	mux.Handle(healthEndpoint, health.NewHandler(adapter, logger, cfg))

	// Set up metrics endpoint
	if cfg.Telemetry.Exporters.Metrics.Prometheus.Enabled {
		metricsPath := cfg.Telemetry.Exporters.Metrics.Prometheus.Path
		logger.Info("Setting up Prometheus metrics endpoint", 
			zap.String("path", metricsPath),
			zap.String("listen", cfg.Telemetry.Exporters.Metrics.Prometheus.Listen))
		mux.Handle(metricsPath, telemetry.CreatePrometheusHandler())
	}

	// Set up GraphQL endpoint
	resolver := graphql.NewResolver(container.GetFamilyService(), container.GetAuthorizationService(), logger)
	gqlServer := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{
		Resolvers: resolver,
	}))
	mux.HandleFunc("/graphql", gqlServer.ServeHTTP)

	// Create context logger
	contextLogger := logging.NewContextLogger(logger)

	// Create and start the server
	serverConfig := server.NewConfig(
		cfg.Server.Port,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		cfg.Server.IdleTimeout,
		cfg.Server.ShutdownTimeout,
	)
	srv := server.New(serverConfig, mux, logger, contextLogger)
	srv.Start()

	// Set up graceful shutdown
	shutdownFunc := func() error {
		// Cancel the root context to signal all operations to stop
		rootCancel()

		// Create a separate context for server shutdown with a timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer shutdownCancel()

		// Shutdown the server
		return srv.Shutdown(shutdownCtx)
	}

	// Wait for shutdown signal
	if err := shutdown.GracefulShutdown(rootCtx, logger, shutdownFunc); err != nil {
		logger.Error("Failed to shutdown gracefully", zap.Error(err))
		os.Exit(1)
	}
}
