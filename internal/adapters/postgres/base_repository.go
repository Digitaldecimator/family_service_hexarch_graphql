package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// BaseRepository is a generic base repository for PostgreSQL
type BaseRepository[T domain.Entity] struct {
	pool         *pgxpool.Pool
	logger       *zap.Logger
	tracer       trace.Tracer
	tableName    string
	entityType   reflect.Type
	scanFunc     func(row pgx.Row) (T, error)
	buildListSQL func(filter ports.FilterOptions, sort ports.SortOptions) (string, []interface{})
}

// NewBaseRepository creates a new base repository
func NewBaseRepository[T domain.Entity](
	pool *pgxpool.Pool,
	logger *zap.Logger,
	tracerName string,
	tableName string,
	scanFunc func(row pgx.Row) (T, error),
	buildListSQL func(filter ports.FilterOptions, sort ports.SortOptions) (string, []interface{}),
) *BaseRepository[T] {
	// Get the entity type using reflection
	var entity T
	entityType := reflect.TypeOf(entity).Elem()

	return &BaseRepository[T]{
		pool:         pool,
		logger:       logger,
		tracer:       otel.Tracer(tracerName),
		tableName:    tableName,
		entityType:   entityType,
		scanFunc:     scanFunc,
		buildListSQL: buildListSQL,
	}
}

// GetByID retrieves an entity by ID from the database
func (r *BaseRepository[T]) GetByID(ctx context.Context, id uuid.UUID) (T, error) {
	var zero T

	ctx, span := r.tracer.Start(ctx, fmt.Sprintf("%s.GetByID", r.entityType.Name()))
	defer span.End()

	span.SetAttributes(attribute.String("entity.id", id.String()))

	query := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE id = $1 AND deleted_at IS NULL
	`, r.tableName)

	row := r.pool.QueryRow(ctx, query, id)
	entity, err := r.scanFunc(row)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug(fmt.Sprintf("%s not found", r.entityType.Name()), zap.String("id", id.String()))
			return zero, fmt.Errorf("%s not found: %w", strings.ToLower(r.entityType.Name()), err)
		}
		r.logger.Error(fmt.Sprintf("Failed to get %s", r.entityType.Name()), zap.Error(err), zap.String("id", id.String()))
		return zero, fmt.Errorf("failed to get %s: %w", strings.ToLower(r.entityType.Name()), err)
	}

	return entity, nil
}

// Delete marks an entity as deleted in the database
func (r *BaseRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := r.tracer.Start(ctx, fmt.Sprintf("%s.Delete", r.entityType.Name()))
	defer span.End()

	span.SetAttributes(attribute.String("entity.id", id.String()))

	now := time.Now().UTC()

	query := fmt.Sprintf(`
		UPDATE %s
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`, r.tableName)

	result, err := r.pool.Exec(ctx, query, now, id)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to delete %s", r.entityType.Name()), zap.Error(err), zap.String("id", id.String()))
		return fmt.Errorf("failed to delete %s: %w", strings.ToLower(r.entityType.Name()), err)
	}

	if result.RowsAffected() == 0 {
		r.logger.Debug(fmt.Sprintf("%s not found for deletion", r.entityType.Name()), zap.String("id", id.String()))
		return fmt.Errorf("%s not found for deletion", strings.ToLower(r.entityType.Name()))
	}

	return nil
}

// List retrieves a list of entities with pagination, filtering, and sorting
func (r *BaseRepository[T]) List(ctx context.Context, options ports.QueryOptions) ([]T, *ports.PagedResult, error) {
	ctx, span := r.tracer.Start(ctx, fmt.Sprintf("%s.List", r.entityType.Name()))
	defer span.End()

	baseQuery, params := r.buildListSQL(options.Filter, options.Sort)

	// Add pagination
	limit := options.Pagination.PageSize
	if limit <= 0 {
		limit = 10 // Default page size
	}

	offset := options.Pagination.Page * limit
	if offset < 0 {
		offset = 0
	}

	query := baseQuery + fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(params)+1, len(params)+2)
	params = append(params, limit, offset)

	rows, err := r.pool.Query(ctx, query, params...)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to list %ss", r.entityType.Name()), zap.Error(err))
		return nil, nil, fmt.Errorf("failed to list %ss: %w", strings.ToLower(r.entityType.Name()), err)
	}
	defer rows.Close()

	entities := make([]T, 0)

	for rows.Next() {
		entity, err := r.scanFunc(rows)
		if err != nil {
			r.logger.Error(fmt.Sprintf("Failed to scan %s row", r.entityType.Name()), zap.Error(err))
			return nil, nil, fmt.Errorf("failed to scan %s row: %w", strings.ToLower(r.entityType.Name()), err)
		}

		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error(fmt.Sprintf("Error iterating %s rows", r.entityType.Name()), zap.Error(err))
		return nil, nil, fmt.Errorf("error iterating %s rows: %w", strings.ToLower(r.entityType.Name()), err)
	}

	// Get total count
	totalCount, err := r.Count(ctx, options.Filter)
	if err != nil {
		r.logger.Error("Failed to get total count", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	pagedResult := &ports.PagedResult{
		TotalCount: totalCount,
		Page:       options.Pagination.Page,
		PageSize:   limit,
		HasNext:    int64(offset+len(entities)) < totalCount,
	}

	return entities, pagedResult, nil
}

// Count returns the total count of entities matching the filter
func (r *BaseRepository[T]) Count(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	ctx, span := r.tracer.Start(ctx, fmt.Sprintf("%s.Count", r.entityType.Name()))
	defer span.End()

	// This is a simplified count query that doesn't use buildListSQL
	// In a real implementation, you might want to extract the WHERE conditions from buildListSQL
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s
		WHERE deleted_at IS NULL
	`, r.tableName)

	params := []interface{}{}
	paramIndex := 1
	whereConditions := []string{}

	if filter.FirstName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("first_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.FirstName+"%")
		paramIndex++
	}

	if filter.LastName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("last_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.LastName+"%")
		paramIndex++
	}

	if filter.MinAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("birth_date <= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MinAge, 0, 0))
		paramIndex++
	}

	if filter.MaxAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("birth_date >= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MaxAge, 0, 0))
		paramIndex++
	}

	if len(whereConditions) > 0 {
		query += " AND " + strings.Join(whereConditions, " AND ")
	}

	var count int64
	err := r.pool.QueryRow(ctx, query, params...).Scan(&count)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to count %ss", r.entityType.Name()), zap.Error(err))
		return 0, fmt.Errorf("failed to count %ss: %w", strings.ToLower(r.entityType.Name()), err)
	}

	return count, nil
}
