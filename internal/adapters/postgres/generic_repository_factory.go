package postgres

import (
	"context"
	"fmt"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/abitofhelp/family-service2/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// GenericRepositoryFactory implements the ports.RepositoryFactory interface using generic repositories
type GenericRepositoryFactory struct {
	pool               *pgxpool.Pool
	logger             *zap.Logger
	transactionManager *TransactionManager
	parentRepository   *GenericParentRepository
	childRepository    *GenericChildRepository
}

// NewGenericRepositoryFactory creates a new generic repository factory
func NewGenericRepositoryFactory(ctx context.Context, connString string, logger *zap.Logger) (*GenericRepositoryFactory, error) {
	// Validate context
	if ctx == nil {
		ctx = context.Background()
		logger.Warn("Nil context provided to NewGenericRepositoryFactory, using background context")
	}

	// Create connection pool
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Ping database to verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create transaction manager
	transactionManager := NewTransactionManager(pool, logger)

	// Create repositories
	parentRepository := NewGenericParentRepository(pool, logger)
	childRepository := NewGenericChildRepository(pool, logger)

	return &GenericRepositoryFactory{
		pool:               pool,
		logger:             logger,
		transactionManager: transactionManager,
		parentRepository:   parentRepository,
		childRepository:    childRepository,
	}, nil
}

// NewParentRepository returns a parent repository
func (f *GenericRepositoryFactory) NewParentRepository() ports.ParentRepository {
	return f.parentRepository
}

// NewChildRepository returns a child repository
func (f *GenericRepositoryFactory) NewChildRepository() ports.ChildRepository {
	return f.childRepository
}

// GetTransactionManager returns the transaction manager
func (f *GenericRepositoryFactory) GetTransactionManager() ports.TransactionManager {
	return f.transactionManager
}

// Close closes the connection pool
func (f *GenericRepositoryFactory) Close(ctx context.Context) error {
	// Validate context
	if ctx == nil {
		ctx = context.Background()
		f.logger.Warn("Nil context provided to Close, using background context")
	}

	if f.pool != nil {
		f.pool.Close()
	}
	return nil
}

// InitSchema initializes the database schema
func (f *GenericRepositoryFactory) InitSchema(ctx context.Context) error {
	// Create tables if they don't exist
	schema := `
		CREATE TABLE IF NOT EXISTS parents (
			id UUID PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT NOT NULL,
			birth_date TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			deleted_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS children (
			id UUID PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			birth_date TIMESTAMP NOT NULL,
			parent_id UUID NOT NULL REFERENCES parents(id),
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			deleted_at TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_parents_deleted_at ON parents(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_children_deleted_at ON children(deleted_at);
		CREATE INDEX IF NOT EXISTS idx_children_parent_id ON children(parent_id);
	`

	_, err := f.pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

// GetGenericParentRepository returns the generic parent repository
func (f *GenericRepositoryFactory) GetGenericParentRepository() ports.Repository[*domain.Parent] {
	return f.parentRepository
}

// GetGenericChildRepository returns the generic child repository
func (f *GenericRepositoryFactory) GetGenericChildRepository() ports.Repository[*domain.Child] {
	return f.childRepository
}

// Ensure GenericRepositoryFactory implements ports.RepositoryFactory
var _ ports.RepositoryFactory = (*GenericRepositoryFactory)(nil)
