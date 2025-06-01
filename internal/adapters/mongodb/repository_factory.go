package mongodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.uber.org/zap"
)

// RepositoryFactory implements the ports.RepositoryFactory interface for MongoDB
type RepositoryFactory struct {
	client             *mongo.Client
	db                 *mongo.Database
	logger             *zap.Logger
	transactionManager *TransactionManager
	parentRepository   *ParentRepository
	childRepository    *ChildRepository
}

// NewRepositoryFactory creates a new MongoDB repository factory
func NewRepositoryFactory(ctx context.Context, logger *zap.Logger, config ports.MongoDBConfig) (*RepositoryFactory, error) {
	// Validate context
	if ctx == nil {
		ctx = context.Background()
		logger.Warn("Nil context provided to NewRepositoryFactory, using background context")
	}

	// Add timeout for connection operations
	connectionTimeout := config.GetConnectionTimeout()
	connCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	// Get the connection URI from the config
	connString := config.GetURI()

	// Configure the client options with OpenTelemetry
	clientOptions := options.Client().
		ApplyURI(connString).
		SetMonitor(otelmongo.NewMonitor())

	// Authentication credentials are included in the connection string

	// Create a new client and connect to the server
	client, err := mongo.Connect(connCtx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Ping the database to verify connection
	pingTimeout := config.GetPingTimeout()
	pingCtx, pingCancel := context.WithTimeout(ctx, pingTimeout)
	defer pingCancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Extract database name from the connection string
	// The database name is the part after the last slash in the URI
	dbName := "family_service" // Default database name
	if connString != "" {
		// Find the last occurrence of '/' before any query parameters
		lastSlashIndex := strings.LastIndex(connString, "/")
		if lastSlashIndex != -1 && lastSlashIndex < len(connString)-1 {
			// Extract everything after the last slash
			remainingPart := connString[lastSlashIndex+1:]
			// If there are query parameters, extract only the database name
			if queryIndex := strings.Index(remainingPart, "?"); queryIndex != -1 {
				dbName = remainingPart[:queryIndex]
			} else {
				dbName = remainingPart
			}
			// Log the extracted database name for debugging
			logger.Debug("Extracted database name from connection string", zap.String("database", dbName))
		}
	}

	// Get database
	db := client.Database(dbName)

	// Create transaction manager
	transactionManager := NewTransactionManager(client, logger)

	// Create repositories
	parentRepository := NewParentRepository(ctx, db, logger, config)
	childRepository := NewChildRepository(ctx, db, logger, config)

	return &RepositoryFactory{
		client:             client,
		db:                 db,
		logger:             logger,
		transactionManager: transactionManager,
		parentRepository:   parentRepository,
		childRepository:    childRepository,
	}, nil
}

// NewParentRepository returns a parent repository
func (f *RepositoryFactory) NewParentRepository() ports.ParentRepository {
	return f.parentRepository
}

// NewChildRepository returns a child repository
func (f *RepositoryFactory) NewChildRepository() ports.ChildRepository {
	return f.childRepository
}

// GetTransactionManager returns the transaction manager
func (f *RepositoryFactory) GetTransactionManager() ports.TransactionManager {
	return f.transactionManager
}

// Close closes the MongoDB client connection
func (f *RepositoryFactory) Close(ctx context.Context, config ports.MongoDBConfig) error {
	// Validate context
	if ctx == nil {
		ctx = context.Background()
		f.logger.Warn("Nil context provided to Close, using background context")
	}

	// Add timeout for disconnection operation
	disconnectTimeout := config.GetDisconnectTimeout()
	disconnectCtx, cancel := context.WithTimeout(ctx, disconnectTimeout)
	defer cancel()

	var errs []error

	// Disconnect MongoDB client
	if f.client != nil {
		if err := f.client.Disconnect(disconnectCtx); err != nil {
			f.logger.Error("Failed to disconnect MongoDB client", zap.Error(err))
			errs = append(errs, fmt.Errorf("failed to disconnect MongoDB client: %w", err))
		}
	}

	// Add cleanup for any other resources here
	// For example:
	// if f.someOtherResource != nil {
	//     if err := f.someOtherResource.Close(); err != nil {
	//         f.logger.Error("Failed to close some other resource", zap.Error(err))
	//         errs = append(errs, fmt.Errorf("failed to close some other resource: %w", err))
	//     }
	// }

	// Return a combined error if any occurred
	if len(errs) > 0 {
		errMsg := "failed to close one or more MongoDB resources:"
		for _, err := range errs {
			errMsg += " " + err.Error()
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

// InitSchema initializes the database schema
func (f *RepositoryFactory) InitSchema(ctx context.Context) error {
	// MongoDB is schemaless, but we can create indexes
	// The indexes are already created in the repository constructors
	return nil
}

// Ensure RepositoryFactory implements ports.RepositoryFactory
var _ ports.RepositoryFactory = (*RepositoryFactory)(nil)
