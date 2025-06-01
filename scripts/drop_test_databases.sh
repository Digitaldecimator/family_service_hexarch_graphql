#!/bin/bash

# Script to drop test databases for MongoDB and PostgreSQL

echo "Dropping test databases..."

# MongoDB
echo "Dropping MongoDB test database..."
mongo_password=${MONGODB_ROOT_PASSWORD:-${MONGO_INITDB_ROOT_PASSWORD}}
mongo_uri=${TEST_MONGODB_URI:-"mongodb://root:${mongo_password}@localhost:27017/?authSource=admin"}
mongo_db=${TEST_MONGODB_DATABASE:-"family_service_test"}

# Use mongosh to drop the database
mongosh "$mongo_uri/$mongo_db" --eval "db.dropDatabase()"

# PostgreSQL
echo "Dropping PostgreSQL test database..."
pg_dsn=${TEST_POSTGRES_DSN:-"postgres://postgres:${POSTGRESQL_POSTGRES_PASSWORD}@localhost:5432/family_service_test"}

# Extract database name from DSN
pg_db="family_service_test"

# Use psql to drop the database
psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS $pg_db;"
psql -h localhost -U postgres -c "CREATE DATABASE $pg_db;"

echo "Test databases dropped successfully."
