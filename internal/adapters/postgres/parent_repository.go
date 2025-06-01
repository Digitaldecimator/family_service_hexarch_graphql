package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// ParentRepository implements the ports.ParentRepository interface for PostgreSQL
type ParentRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	tracer trace.Tracer
}

// NewParentRepository creates a new PostgreSQL parent repository
func NewParentRepository(pool *pgxpool.Pool, logger *zap.Logger) *ParentRepository {
	return &ParentRepository{
		pool:   pool,
		logger: logger,
		tracer: otel.Tracer("postgres.parent_repository"),
	}
}

// Create creates a new parent in the database
func (r *ParentRepository) Create(ctx context.Context, parent *domain.Parent) error {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.Create")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parent.ID.String()))

	query := `
		INSERT INTO parents (id, first_name, last_name, email, birth_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		parent.ID,
		parent.FirstName,
		parent.LastName,
		parent.Email,
		parent.BirthDate,
		parent.CreatedAt,
		parent.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create parent", zap.Error(err), zap.String("parent_id", parent.ID.String()))
		return fmt.Errorf("failed to create parent: %w", err)
	}

	return nil
}

// GetByID retrieves a parent by ID from the database
func (r *ParentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", id.String()))

	query := `
		SELECT p.id, p.first_name, p.last_name, p.email, p.birth_date, p.created_at, p.updated_at, p.deleted_at
		FROM parents p
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`

	row := r.pool.QueryRow(ctx, query, id)

	var parent domain.Parent
	var deletedAt sql.NullTime

	err := row.Scan(
		&parent.ID,
		&parent.FirstName,
		&parent.LastName,
		&parent.Email,
		&parent.BirthDate,
		&parent.CreatedAt,
		&parent.UpdatedAt,
		&deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug("Parent not found", zap.String("parent_id", id.String()))
			return nil, fmt.Errorf("parent not found: %w", err)
		}
		r.logger.Error("Failed to get parent", zap.Error(err), zap.String("parent_id", id.String()))
		return nil, fmt.Errorf("failed to get parent: %w", err)
	}

	if deletedAt.Valid {
		parent.DeletedAt = &deletedAt.Time
	}

	// Get children for this parent
	childrenQuery := `
		SELECT c.id, c.first_name, c.last_name, c.birth_date, c.parent_id, c.created_at, c.updated_at, c.deleted_at
		FROM children c
		WHERE c.parent_id = $1 AND c.deleted_at IS NULL
	`

	rows, err := r.pool.Query(ctx, childrenQuery, id)
	if err != nil {
		r.logger.Error("Failed to get children for parent", zap.Error(err), zap.String("parent_id", id.String()))
		return nil, fmt.Errorf("failed to get children for parent: %w", err)
	}
	defer rows.Close()

	parent.Children = []domain.Child{}

	for rows.Next() {
		var child domain.Child
		var childDeletedAt sql.NullTime

		err := rows.Scan(
			&child.ID,
			&child.FirstName,
			&child.LastName,
			&child.BirthDate,
			&child.ParentID,
			&child.CreatedAt,
			&child.UpdatedAt,
			&childDeletedAt,
		)

		if err != nil {
			r.logger.Error("Failed to scan child row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan child row: %w", err)
		}

		if childDeletedAt.Valid {
			child.DeletedAt = &childDeletedAt.Time
		}

		parent.Children = append(parent.Children, child)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating child rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating child rows: %w", err)
	}

	return &parent, nil
}

// Update updates an existing parent in the database
func (r *ParentRepository) Update(ctx context.Context, parent *domain.Parent) error {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.Update")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parent.ID.String()))

	query := `
		UPDATE parents
		SET first_name = $1, last_name = $2, email = $3, birth_date = $4, updated_at = $5
		WHERE id = $6 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query,
		parent.FirstName,
		parent.LastName,
		parent.Email,
		parent.BirthDate,
		time.Now().UTC(),
		parent.ID,
	)

	if err != nil {
		r.logger.Error("Failed to update parent", zap.Error(err), zap.String("parent_id", parent.ID.String()))
		return fmt.Errorf("failed to update parent: %w", err)
	}

	if result.RowsAffected() == 0 {
		r.logger.Debug("Parent not found for update", zap.String("parent_id", parent.ID.String()))
		return fmt.Errorf("parent not found for update")
	}

	return nil
}

// Delete marks a parent as deleted in the database
func (r *ParentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.Delete")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", id.String()))

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()

	// Mark parent as deleted
	parentQuery := `
		UPDATE parents
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := tx.Exec(ctx, parentQuery, now, id)
	if err != nil {
		r.logger.Error("Failed to delete parent", zap.Error(err), zap.String("parent_id", id.String()))
		return fmt.Errorf("failed to delete parent: %w", err)
	}

	if result.RowsAffected() == 0 {
		r.logger.Debug("Parent not found for deletion", zap.String("parent_id", id.String()))
		return fmt.Errorf("parent not found for deletion")
	}

	// Mark all children as deleted
	childrenQuery := `
		UPDATE children
		SET deleted_at = $1, updated_at = $1
		WHERE parent_id = $2 AND deleted_at IS NULL
	`

	_, err = tx.Exec(ctx, childrenQuery, now, id)
	if err != nil {
		r.logger.Error("Failed to delete children", zap.Error(err), zap.String("parent_id", id.String()))
		return fmt.Errorf("failed to delete children: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// buildListQuery builds a query for listing parents with filtering, pagination, and sorting
func (r *ParentRepository) buildListQuery(filter ports.FilterOptions, sort ports.SortOptions) (string, []interface{}) {
	query := `
		SELECT p.id, p.first_name, p.last_name, p.email, p.birth_date, p.created_at, p.updated_at, p.deleted_at
		FROM parents p
		WHERE p.deleted_at IS NULL
	`

	params := []interface{}{}
	paramIndex := 1
	whereConditions := []string{}

	if filter.FirstName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("p.first_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.FirstName+"%")
		paramIndex++
	}

	if filter.LastName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("p.last_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.LastName+"%")
		paramIndex++
	}

	if filter.Email != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("p.email ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.Email+"%")
		paramIndex++
	}

	if filter.MinAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("p.birth_date <= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MinAge, 0, 0))
		paramIndex++
	}

	if filter.MaxAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("p.birth_date >= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MaxAge, 0, 0))
		paramIndex++
	}

	if len(whereConditions) > 0 {
		query += " AND " + strings.Join(whereConditions, " AND ")
	}

	// Add sorting
	if sort.Field != "" {
		direction := "ASC"
		if strings.ToLower(sort.Direction) == "desc" {
			direction = "DESC"
		}

		var sortField string
		switch strings.ToLower(sort.Field) {
		case "firstname", "first_name":
			sortField = "p.first_name"
		case "lastname", "last_name":
			sortField = "p.last_name"
		case "email":
			sortField = "p.email"
		case "birthdate", "birth_date":
			sortField = "p.birth_date"
		case "createdat", "created_at":
			sortField = "p.created_at"
		case "updatedat", "updated_at":
			sortField = "p.updated_at"
		default:
			sortField = "p.created_at"
		}

		query += fmt.Sprintf(" ORDER BY %s %s", sortField, direction)
	} else {
		query += " ORDER BY p.created_at DESC"
	}

	return query, params
}

// List retrieves a list of parents with pagination, filtering, and sorting
func (r *ParentRepository) List(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.List")
	defer span.End()

	baseQuery, params := r.buildListQuery(options.Filter, options.Sort)

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

	// Create channels for concurrent operations
	type queryResult struct {
		parents []*domain.Parent
		err     error
	}
	type countResult struct {
		count int64
		err   error
	}
	queryCh := make(chan queryResult, 1)
	countCh := make(chan countResult, 1)

	// Execute query operation concurrently
	go func() {
		rows, err := r.pool.Query(ctx, query, params...)
		if err != nil {
			queryCh <- queryResult{nil, fmt.Errorf("failed to list parents: %w", err)}
			return
		}
		defer rows.Close()

		parents := []*domain.Parent{}

		for rows.Next() {
			var parent domain.Parent
			var deletedAt sql.NullTime

			err := rows.Scan(
				&parent.ID,
				&parent.FirstName,
				&parent.LastName,
				&parent.Email,
				&parent.BirthDate,
				&parent.CreatedAt,
				&parent.UpdatedAt,
				&deletedAt,
			)

			if err != nil {
				queryCh <- queryResult{nil, fmt.Errorf("failed to scan parent row: %w", err)}
				return
			}

			if deletedAt.Valid {
				parent.DeletedAt = &deletedAt.Time
			}

			parents = append(parents, &parent)
		}

		if err := rows.Err(); err != nil {
			queryCh <- queryResult{nil, fmt.Errorf("error iterating parent rows: %w", err)}
			return
		}

		queryCh <- queryResult{parents, nil}
	}()

	// Execute Count operation concurrently
	go func() {
		count, err := r.Count(ctx, options.Filter)
		if err != nil {
			countCh <- countResult{0, fmt.Errorf("failed to get total count: %w", err)}
			return
		}
		countCh <- countResult{count, nil}
	}()

	// Wait for both operations to complete
	queryRes := <-queryCh
	countRes := <-countCh

	// Check for errors
	if queryRes.err != nil {
		r.logger.Error("Failed to list parents", zap.Error(queryRes.err))
		return nil, nil, queryRes.err
	}
	if countRes.err != nil {
		r.logger.Error("Failed to get total count", zap.Error(countRes.err))
		return nil, nil, countRes.err
	}

	pagedResult := &ports.PagedResult{
		TotalCount: countRes.count,
		Page:       options.Pagination.Page,
		PageSize:   limit,
		HasNext:    int64(offset+len(queryRes.parents)) < countRes.count,
	}

	return queryRes.parents, pagedResult, nil
}

// Count returns the total count of parents matching the filter
func (r *ParentRepository) Count(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "ParentRepository.Count")
	defer span.End()

	query := `
		SELECT COUNT(*)
		FROM parents p
		WHERE p.deleted_at IS NULL
	`

	params := []interface{}{}
	paramIndex := 1
	whereConditions := []string{}

	if filter.FirstName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("p.first_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.FirstName+"%")
		paramIndex++
	}

	if filter.LastName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("p.last_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.LastName+"%")
		paramIndex++
	}

	if filter.Email != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("p.email ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.Email+"%")
		paramIndex++
	}

	if filter.MinAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("p.birth_date <= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MinAge, 0, 0))
		paramIndex++
	}

	if filter.MaxAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("p.birth_date >= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MaxAge, 0, 0))
		paramIndex++
	}

	if len(whereConditions) > 0 {
		query += " AND " + strings.Join(whereConditions, " AND ")
	}

	var count int64
	err := r.pool.QueryRow(ctx, query, params...).Scan(&count)
	if err != nil {
		r.logger.Error("Failed to count parents", zap.Error(err))
		return 0, fmt.Errorf("failed to count parents: %w", err)
	}

	return count, nil
}
