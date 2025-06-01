package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/abitofhelp/family-service2/internal/adapters/mongodb/migrations"
	"github.com/abitofhelp/family-service2/internal/infrastructure/config"
	"github.com/abitofhelp/family-service2/internal/infrastructure/logging"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Database.MongoDB.MigrationTimeout)*time.Second)
	defer cancel()

	// Get MongoDB connection string and database name
	connString := cfg.Database.MongoDB.URI
	if connString == "" {
		logger.Fatal("MongoDB connection string not provided")
	}

	// Extract database name from URI
	dbName := extractDatabaseName(connString)
	if dbName == "" {
		logger.Fatal("Could not extract database name from MongoDB URI")
	}

	// Configure the client options with OpenTelemetry
	clientOptions := options.Client().
		ApplyURI(connString).
		SetMonitor(otelmongo.NewMonitor())

	// Create a new client and connect to the server
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Fatal("Failed to create MongoDB client", zap.Error(err))
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			logger.Error("Failed to disconnect MongoDB client", zap.Error(err))
		}
	}()

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		logger.Fatal("Failed to ping MongoDB", zap.Error(err))
	}

	// Get database
	db := client.Database(dbName)

	// Create migration registry
	registry := migrations.NewRegistry(db, logger)

	// Run migrations
	logger.Info("Running MongoDB migrations...")
	if err := registry.MigrateUp(ctx); err != nil {
		logger.Fatal("Failed to run MongoDB migrations", zap.Error(err))
	}
	logger.Info("MongoDB migrations completed successfully")
}

// extractDatabaseName extracts the database name from a MongoDB connection string
func extractDatabaseName(uri string) string {
	// Find the last '/' in the URI
	lastSlashIndex := strings.LastIndex(uri, "/")
	if lastSlashIndex == -1 || lastSlashIndex == len(uri)-1 {
		return ""
	}

	// Extract the part after the last '/'
	dbNameWithParams := uri[lastSlashIndex+1:]

	// If there are query parameters, remove them
	questionMarkIndex := strings.Index(dbNameWithParams, "?")
	if questionMarkIndex != -1 {
		return dbNameWithParams[:questionMarkIndex]
	}

	return dbNameWithParams
}
