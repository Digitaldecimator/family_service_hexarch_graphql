// Package ports defines the interfaces that connect the application's core business logic
// to external adapters. It follows the ports and adapters (hexagonal architecture) pattern.
package ports

import "time"

// DatabaseConfig defines the interface for database configuration settings.
// This interface abstracts the database configuration from the infrastructure layer,
// allowing adapters to depend only on this interface rather than concrete implementations.
type DatabaseConfig interface {
	// GetConnectionTimeout returns the timeout for establishing a database connection.
	GetConnectionTimeout() time.Duration
	
	// GetPingTimeout returns the timeout for pinging the database to verify connection.
	GetPingTimeout() time.Duration
	
	// GetDisconnectTimeout returns the timeout for disconnecting from the database.
	GetDisconnectTimeout() time.Duration
	
	// GetIndexTimeout returns the timeout for creating database indexes.
	GetIndexTimeout() time.Duration
}

// MongoDBConfig defines MongoDB-specific configuration settings.
type MongoDBConfig interface {
	DatabaseConfig
	
	// GetURI returns the MongoDB connection URI.
	GetURI() string
}

// PostgresConfig defines PostgreSQL-specific configuration settings.
type PostgresConfig interface {
	DatabaseConfig
	
	// GetDSN returns the PostgreSQL connection data source name.
	GetDSN() string
}