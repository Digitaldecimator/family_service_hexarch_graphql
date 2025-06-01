package di

import (
	"context"
	"fmt"

	"github.com/abitofhelp/family-service2/internal/adapters/mongodb"
	"github.com/abitofhelp/family-service2/internal/adapters/postgres"
	"github.com/abitofhelp/family-service2/internal/application"
	"github.com/abitofhelp/family-service2/internal/infrastructure/auth"
	"github.com/abitofhelp/family-service2/internal/infrastructure/config"
	"github.com/abitofhelp/family-service2/internal/infrastructure/logging"
	"github.com/abitofhelp/family-service2/internal/ports"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// Container is a dependency injection container for the application
type Container struct {
	ctx                  context.Context
	logger               *zap.Logger
	contextLogger        *logging.ContextLogger
	validator            *validator.Validate
	repositoryFactory    ports.RepositoryFactory
	familyService        ports.FamilyService
	authorizationService ports.AuthorizationService
	config               *config.Config
}

// NewContainer creates a new dependency injection container
func NewContainer(ctx context.Context, logger *zap.Logger, cfg *config.Config) (*Container, error) {
	container := &Container{
		ctx:    ctx,
		logger: logger,
		config: cfg,
	}

	// Initialize context logger
	container.contextLogger = logging.NewContextLogger(logger)

	// Initialize validator
	container.validator = validator.New()

	// Initialize repository factory based on database type
	dbType := cfg.Database.Type
	switch dbType {
	case "mongodb":
		// Use the config as a MongoDBConfig interface
		mongoFactory, err := mongodb.NewRepositoryFactory(
			ctx,
			logger,
			cfg, // The config implements the ports.MongoDBConfig interface
		)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MongoDB repository factory: %w", err)
		}
		container.repositoryFactory = mongoFactory
	case "postgres":
		// Always use the generic repository factory for PostgreSQL
		// as it provides better type safety and code reuse
		genericFactory, err := postgres.NewGenericRepositoryFactory(
			ctx,
			cfg.Database.Postgres.DSN,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Generic PostgreSQL repository factory: %w", err)
		}
		container.repositoryFactory = genericFactory
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Initialize authorization service
	authService := auth.NewAuthorizationService(logger)
	container.authorizationService = authService

	// Initialize family service
	container.familyService = application.NewFamilyService(
		container.repositoryFactory,
		container.validator,
		container.logger,
	)

	return container, nil
}

// GetContext returns the context
func (c *Container) GetContext() context.Context {
	return c.ctx
}

// GetLogger returns the logger
func (c *Container) GetLogger() *zap.Logger {
	return c.logger
}

// GetContextLogger returns the context logger
func (c *Container) GetContextLogger() *logging.ContextLogger {
	return c.contextLogger
}

// GetValidator returns the validator
func (c *Container) GetValidator() *validator.Validate {
	return c.validator
}

// GetRepositoryFactory returns the repository factory
func (c *Container) GetRepositoryFactory() ports.RepositoryFactory {
	return c.repositoryFactory
}

// GetFamilyService returns the family service
func (c *Container) GetFamilyService() ports.FamilyService {
	return c.familyService
}

// GetAuthorizationService returns the authorization service
func (c *Container) GetAuthorizationService() ports.AuthorizationService {
	return c.authorizationService
}

// Close closes all resources
func (c *Container) Close() error {
	var errs []error

	// Close MongoDB repository factory
	if mongoFactory, ok := c.repositoryFactory.(*mongodb.RepositoryFactory); ok {
		if err := mongoFactory.Close(c.ctx, c.config); err != nil {
			c.logger.Error("Failed to close MongoDB repository factory", zap.Error(err))
			errs = append(errs, err)
		}
	} else if closer, ok := c.repositoryFactory.(interface {
		Close(ctx context.Context) error
	}); ok {
		// For other repository factories that don't need config
		if err := closer.Close(c.ctx); err != nil {
			c.logger.Error("Failed to close repository factory", zap.Error(err))
			errs = append(errs, err)
		}
	}

	// Add more resource cleanup here as needed
	// For example:
	// if c.someOtherResource != nil {
	//     if err := c.someOtherResource.Close(); err != nil {
	//         c.logger.Error("Failed to close some other resource", zap.Error(err))
	//         errs = append(errs, err)
	//     }
	// }

	// Return a combined error if any occurred
	if len(errs) > 0 {
		errMsg := "failed to close one or more resources:"
		for _, err := range errs {
			errMsg += " " + err.Error()
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}
