# Generic Repository Pattern with Go Generics

This package implements a generic repository pattern using Go's generics feature. The implementation provides a more type-safe and reusable approach to data access compared to the traditional repository pattern.

## Overview

The generic repository pattern in this package consists of the following components:

1. **Entity Interface**: A common interface that all domain entities must implement, providing methods for ID access, timestamps, and soft deletion.

2. **Generic Repository Interface**: A generic interface that defines common CRUD operations for any entity type.

3. **Base Repository**: A generic implementation of the repository interface that provides common functionality for all entity types.

4. **Entity-Specific Repositories**: Repositories that extend the base repository and add entity-specific functionality.

5. **Repository Factory**: A factory that creates and manages repository instances.

## Benefits of Using Generics

Using generics in the repository pattern provides several benefits:

1. **Type Safety**: The compiler ensures that the correct entity types are used with the correct repositories, reducing runtime errors.

2. **Code Reuse**: Common functionality is implemented once in the base repository and reused across all entity types.

3. **Reduced Boilerplate**: Less code duplication for common operations like GetByID, Delete, List, and Count.

4. **Consistency**: Ensures consistent behavior across different repositories.

5. **Flexibility**: Allows for entity-specific customization while maintaining a common interface.

## Usage

### Creating a New Entity Type

1. Define your entity struct in the domain package.
2. Implement the `Entity` interface for your entity.

```go
// Example: Implementing the Entity interface for a new entity
func (e *MyEntity) GetID() uuid.UUID {
    return e.ID
}

func (e *MyEntity) IsDeleted() bool {
    return e.DeletedAt != nil
}

// ... implement other required methods
```

### Creating a New Repository

1. Create a new repository struct that embeds the BaseRepository.
2. Implement entity-specific methods like Create and Update.
3. Provide a scan function that knows how to scan a database row into your entity.
4. Provide a buildListQuery function that knows how to build SQL queries for your entity.

```go
// Example: Creating a new repository for MyEntity
type MyEntityRepository struct {
    *BaseRepository[*domain.MyEntity]
    pool   *pgxpool.Pool
    logger *zap.Logger
}

func NewMyEntityRepository(pool *pgxpool.Pool, logger *zap.Logger) *MyEntityRepository {
    repo := &MyEntityRepository{
        pool:   pool,
        logger: logger,
    }

    baseRepo := NewBaseRepository[*domain.MyEntity](
        pool,
        logger,
        "postgres.my_entity_repository",
        "my_entities",
        repo.scanMyEntity,
        repo.buildListQuery,
    )

    repo.BaseRepository = baseRepo
    return repo
}

// Implement entity-specific methods...
```

### Using the Repository Factory

The repository factory creates and manages repository instances. You can get repositories either as their specific types or as generic repositories:

```go
// Get a repository as its specific type
parentRepo := factory.NewParentRepository()

// Get a repository as a generic repository
genericParentRepo := factory.GetGenericParentRepository()
```

## Testing

This package includes test helpers that simplify integration testing of repositories. The test helpers are designed to follow best practices for testing in a hexagonal architecture:

1. **Separation of Concerns**: Tests focus on the adapter implementation, not the application logic.
2. **Real Database Testing**: Integration tests use a real PostgreSQL database to ensure the adapter works correctly with the actual database.
3. **Reusable Setup**: Common setup code is extracted into helper functions to reduce duplication.
4. **Proper Cleanup**: Resources are properly cleaned up after tests to avoid interference between tests.

### Using Test Helpers

The package provides the following test helpers:

1. **SetupTestDatabase**: Sets up a PostgreSQL connection for testing and returns a connection pool, context, logger, and cleanup function.
2. **SetupTestRepositories**: Sets up repositories for testing and returns a repository factory, context, and cleanup function.

> **Important**: Before running integration tests, ensure that a PostgreSQL server is running and accessible. The tests expect a PostgreSQL server to be running on localhost:5432 by default. You can use the provided Docker Compose file to start a PostgreSQL server:
> ```bash
> docker-compose up -d postgresql
> ```

Example usage:

```go
func TestMyRepository(t *testing.T) {
    // Skip if short flag is set
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Set up test repositories using the helper
    factory, ctx, cleanup := postgres.SetupTestRepositories(t)
    defer cleanup()

    // Get repositories from the factory
    repo := factory.NewMyRepository()

    // Test repository operations...
}
```

## Best Practices for Hexagonal Architecture

When working with this package, follow these best practices to maintain a clean hexagonal architecture:

1. **Dependency Direction**: Dependencies should always point inward, from adapters to the application core to the domain.
2. **Interface Segregation**: Define small, focused interfaces in the ports package that are implemented by adapters.
3. **Domain Independence**: The domain model should be independent of any external concerns, including the database.
4. **Adapter Isolation**: Adapters should be isolated from each other and should only communicate through the application core.
5. **Testing at Boundaries**: Test adapters at their boundaries to ensure they correctly implement the port interfaces.

By following these practices, you'll maintain a clean separation of concerns and a flexible, maintainable architecture.
