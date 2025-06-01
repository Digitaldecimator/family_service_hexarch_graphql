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

// ChildRepository implements the ports.ChildRepository interface for PostgreSQL
type ChildRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	tracer trace.Tracer
}

// NewChildRepository creates a new PostgreSQL child repository
func NewChildRepository(pool *pgxpool.Pool, logger *zap.Logger) *ChildRepository {
	return &ChildRepository{
		pool:   pool,
		logger: logger,
		tracer: otel.Tracer("postgres.child_repository"),
	}
}

// Create creates a new child in the database
func (r *ChildRepository) Create(ctx context.Context, child *domain.Child) error {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("child.id", child.ID.String()),
		attribute.String("parent.id", child.ParentID.String()),
	)

	// First check if the parent exists
	parentQuery := `
		SELECT 1 FROM parents WHERE id = $1 AND deleted_at IS NULL
	`
	var exists int
	err := r.pool.QueryRow(ctx, parentQuery, child.ParentID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug("Parent not found for child creation", zap.String("parent_id", child.ParentID.String()))
			return fmt.Errorf("parent not found for child creation")
		}
		r.logger.Error("Failed to check parent existence", zap.Error(err), zap.String("parent_id", child.ParentID.String()))
		return fmt.Errorf("failed to check parent existence: %w", err)
	}

	query := `
		INSERT INTO children (id, first_name, last_name, birth_date, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.pool.Exec(ctx, query,
		child.ID,
		child.FirstName,
		child.LastName,
		child.BirthDate,
		child.ParentID,
		child.CreatedAt,
		child.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create child", zap.Error(err), zap.String("child_id", child.ID.String()))
		return fmt.Errorf("failed to create child: %w", err)
	}

	return nil
}

// GetByID retrieves a child by ID from the database
func (r *ChildRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", id.String()))

	query := `
		SELECT c.id, c.first_name, c.last_name, c.birth_date, c.parent_id, c.created_at, c.updated_at, c.deleted_at
		FROM children c
		WHERE c.id = $1 AND c.deleted_at IS NULL
	`

	row := r.pool.QueryRow(ctx, query, id)

	var child domain.Child
	var deletedAt sql.NullTime

	err := row.Scan(
		&child.ID,
		&child.FirstName,
		&child.LastName,
		&child.BirthDate,
		&child.ParentID,
		&child.CreatedAt,
		&child.UpdatedAt,
		&deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug("Child not found", zap.String("child_id", id.String()))
			return nil, fmt.Errorf("child not found: %w", err)
		}
		r.logger.Error("Failed to get child", zap.Error(err), zap.String("child_id", id.String()))
		return nil, fmt.Errorf("failed to get child: %w", err)
	}

	if deletedAt.Valid {
		child.DeletedAt = &deletedAt.Time
	}

	return &child, nil
}

// Update updates an existing child in the database
func (r *ChildRepository) Update(ctx context.Context, child *domain.Child) error {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.Update")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", child.ID.String()))

	query := `
		UPDATE children
		SET first_name = $1, last_name = $2, birth_date = $3, updated_at = $4
		WHERE id = $5 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query,
		child.FirstName,
		child.LastName,
		child.BirthDate,
		time.Now().UTC(),
		child.ID,
	)

	if err != nil {
		r.logger.Error("Failed to update child", zap.Error(err), zap.String("child_id", child.ID.String()))
		return fmt.Errorf("failed to update child: %w", err)
	}

	if result.RowsAffected() == 0 {
		r.logger.Debug("Child not found for update", zap.String("child_id", child.ID.String()))
		return fmt.Errorf("child not found")
	}

	return nil
}

// Delete marks a child as deleted in the database
func (r *ChildRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.Delete")
	defer span.End()

	span.SetAttributes(attribute.String("child.id", id.String()))

	now := time.Now().UTC()

	query := `
		UPDATE children
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, now, id)
	if err != nil {
		r.logger.Error("Failed to delete child", zap.Error(err), zap.String("child_id", id.String()))
		return fmt.Errorf("failed to delete child: %w", err)
	}

	if result.RowsAffected() == 0 {
		r.logger.Debug("Child not found for deletion", zap.String("child_id", id.String()))
		return fmt.Errorf("child not found")
	}

	return nil
}

// buildListQuery builds a query for listing children with filtering, pagination, and sorting
func (r *ChildRepository) buildListQuery(filter ports.FilterOptions, sort ports.SortOptions, parentID *uuid.UUID) (string, []interface{}) {
	query := `
		SELECT c.id, c.first_name, c.last_name, c.birth_date, c.parent_id, c.created_at, c.updated_at, c.deleted_at
		FROM children c
		WHERE c.deleted_at IS NULL
	`

	params := []interface{}{}
	paramIndex := 1
	whereConditions := []string{}

	if parentID != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("c.parent_id = $%d", paramIndex))
		params = append(params, *parentID)
		paramIndex++
	}

	if filter.FirstName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("c.first_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.FirstName+"%")
		paramIndex++
	}

	if filter.LastName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("c.last_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.LastName+"%")
		paramIndex++
	}

	if filter.MinAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("c.birth_date <= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MinAge, 0, 0))
		paramIndex++
	}

	if filter.MaxAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("c.birth_date >= $%d", paramIndex))
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
			sortField = "c.first_name"
		case "lastname", "last_name":
			sortField = "c.last_name"
		case "birthdate", "birth_date":
			sortField = "c.birth_date"
		case "createdat", "created_at":
			sortField = "c.created_at"
		case "updatedat", "updated_at":
			sortField = "c.updated_at"
		default:
			sortField = "c.created_at"
		}

		query += fmt.Sprintf(" ORDER BY %s %s", sortField, direction)
	} else {
		query += " ORDER BY c.created_at DESC"
	}

	return query, params
}

// ListByParentID retrieves children for a specific parent with pagination, filtering, and sorting
func (r *ChildRepository) ListByParentID(ctx context.Context, parentID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.ListByParentID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parentID.String()))

	baseQuery, params := r.buildListQuery(options.Filter, options.Sort, &parentID)

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
		children []*domain.Child
		err      error
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
			queryCh <- queryResult{nil, fmt.Errorf("failed to list children by parent ID: %w", err)}
			return
		}
		defer rows.Close()

		children := []*domain.Child{}

		for rows.Next() {
			var child domain.Child
			var deletedAt sql.NullTime

			err := rows.Scan(
				&child.ID,
				&child.FirstName,
				&child.LastName,
				&child.BirthDate,
				&child.ParentID,
				&child.CreatedAt,
				&child.UpdatedAt,
				&deletedAt,
			)

			if err != nil {
				queryCh <- queryResult{nil, fmt.Errorf("failed to scan child row: %w", err)}
				return
			}

			if deletedAt.Valid {
				child.DeletedAt = &deletedAt.Time
			}

			children = append(children, &child)
		}

		if err := rows.Err(); err != nil {
			queryCh <- queryResult{nil, fmt.Errorf("error iterating child rows: %w", err)}
			return
		}

		queryCh <- queryResult{children, nil}
	}()

	// Execute Count operation concurrently
	go func() {
		countFilter := options.Filter
		count, err := r.countByParentID(ctx, parentID, countFilter)
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
		r.logger.Error("Failed to list children by parent ID", zap.Error(queryRes.err), zap.String("parent_id", parentID.String()))
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
		HasNext:    int64(offset+len(queryRes.children)) < countRes.count,
	}

	return queryRes.children, pagedResult, nil
}

// List retrieves a list of children with pagination, filtering, and sorting
func (r *ChildRepository) List(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.List")
	defer span.End()

	baseQuery, params := r.buildListQuery(options.Filter, options.Sort, nil)

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
		children []*domain.Child
		err      error
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
			queryCh <- queryResult{nil, fmt.Errorf("failed to list children: %w", err)}
			return
		}
		defer rows.Close()

		children := []*domain.Child{}

		for rows.Next() {
			var child domain.Child
			var deletedAt sql.NullTime

			err := rows.Scan(
				&child.ID,
				&child.FirstName,
				&child.LastName,
				&child.BirthDate,
				&child.ParentID,
				&child.CreatedAt,
				&child.UpdatedAt,
				&deletedAt,
			)

			if err != nil {
				queryCh <- queryResult{nil, fmt.Errorf("failed to scan child row: %w", err)}
				return
			}

			if deletedAt.Valid {
				child.DeletedAt = &deletedAt.Time
			}

			children = append(children, &child)
		}

		if err := rows.Err(); err != nil {
			queryCh <- queryResult{nil, fmt.Errorf("error iterating child rows: %w", err)}
			return
		}

		queryCh <- queryResult{children, nil}
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
		r.logger.Error("Failed to list children", zap.Error(queryRes.err))
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
		HasNext:    int64(offset+len(queryRes.children)) < countRes.count,
	}

	return queryRes.children, pagedResult, nil
}

// countByParentID returns the total count of children for a specific parent matching the filter
func (r *ChildRepository) countByParentID(ctx context.Context, parentID uuid.UUID, filter ports.FilterOptions) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.countByParentID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parentID.String()))

	query := `
		SELECT COUNT(*)
		FROM children c
		WHERE c.deleted_at IS NULL AND c.parent_id = $1
	`

	params := []interface{}{parentID}
	paramIndex := 2
	whereConditions := []string{}

	if filter.FirstName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("c.first_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.FirstName+"%")
		paramIndex++
	}

	if filter.LastName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("c.last_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.LastName+"%")
		paramIndex++
	}

	if filter.MinAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("c.birth_date <= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MinAge, 0, 0))
		paramIndex++
	}

	if filter.MaxAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("c.birth_date >= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MaxAge, 0, 0))
		paramIndex++
	}

	if len(whereConditions) > 0 {
		query += " AND " + strings.Join(whereConditions, " AND ")
	}

	var count int64
	err := r.pool.QueryRow(ctx, query, params...).Scan(&count)
	if err != nil {
		r.logger.Error("Failed to count children by parent ID", zap.Error(err), zap.String("parent_id", parentID.String()))
		return 0, fmt.Errorf("failed to count children by parent ID: %w", err)
	}

	return count, nil
}

// Count returns the total count of children matching the filter
func (r *ChildRepository) Count(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "ChildRepository.Count")
	defer span.End()

	query := `
		SELECT COUNT(*)
		FROM children c
		WHERE c.deleted_at IS NULL
	`

	params := []interface{}{}
	paramIndex := 1
	whereConditions := []string{}

	if filter.FirstName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("c.first_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.FirstName+"%")
		paramIndex++
	}

	if filter.LastName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("c.last_name ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.LastName+"%")
		paramIndex++
	}

	if filter.MinAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("c.birth_date <= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MinAge, 0, 0))
		paramIndex++
	}

	if filter.MaxAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("c.birth_date >= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-filter.MaxAge, 0, 0))
		paramIndex++
	}

	if len(whereConditions) > 0 {
		query += " AND " + strings.Join(whereConditions, " AND ")
	}

	var count int64
	err := r.pool.QueryRow(ctx, query, params...).Scan(&count)
	if err != nil {
		r.logger.Error("Failed to count children", zap.Error(err))
		return 0, fmt.Errorf("failed to count children: %w", err)
	}

	return count, nil
}
