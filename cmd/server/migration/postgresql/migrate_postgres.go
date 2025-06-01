package main

import (
	"context"
	"log"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/adapters/postgres/migrations"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/infrastructure/config"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/infrastructure/logging"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	// Initialize configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := logging.NewLogger(cfg.Log.Level, cfg.Log.Development)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Database.Postgres.MigrationTimeout)*time.Second)
	defer cancel()

	// Get PostgreSQL connection string
	connString := cfg.Database.Postgres.DSN
	if connString == "" {
		logger.Fatal("PostgreSQL connection string not provided")
	}

	// Create connection pool
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		logger.Fatal("Failed to parse connection string", zap.Error(err))
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		logger.Fatal("Failed to create connection pool", zap.Error(err))
	}
	defer pool.Close()

	// Ping database to verify connection
	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	// Create migration registry
	registry := migrations.NewRegistry(pool, logger)

	// Run migrations
	logger.Info("Running PostgreSQL migrations...")
	if err := registry.MigrateUp(ctx); err != nil {
		logger.Fatal("Failed to run PostgreSQL migrations", zap.Error(err))
	}
	logger.Info("PostgreSQL migrations completed successfully")
}
