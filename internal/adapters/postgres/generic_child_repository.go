package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/abitofhelp/family-service2/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// GenericChildRepository implements the ports.Repository interface for Child entities
type GenericChildRepository struct {
	*BaseRepository[*domain.Child]
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewGenericChildRepository creates a new generic child repository
func NewGenericChildRepository(pool *pgxpool.Pool, logger *zap.Logger) *GenericChildRepository {
	repo := &GenericChildRepository{
		pool:   pool,
		logger: logger,
	}

	// Create the base repository with Child-specific functions
	baseRepo := NewBaseRepository[*domain.Child](
		pool,
		logger,
		"postgres.child_repository",
		"children",
		repo.scanChild,
		repo.buildListQuery,
	)

	repo.BaseRepository = baseRepo
	return repo
}

// scanChild scans a database row into a Child entity
func (r *GenericChildRepository) scanChild(row pgx.Row) (*domain.Child, error) {
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
		return nil, err
	}

	if deletedAt.Valid {
		child.DeletedAt = &deletedAt.Time
	}

	return &child, nil
}

// buildListQuery builds a query for listing children with filtering and sorting
func (r *GenericChildRepository) buildListQuery(filter ports.FilterOptions, sort ports.SortOptions) (string, []interface{}) {
	query := `
		SELECT id, first_name, last_name, birth_date, parent_id, created_at, updated_at, deleted_at
		FROM children
		WHERE deleted_at IS NULL
	`

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
		query += " AND " + fmt.Sprintf("(%s)", whereConditions[0])
		for i := 1; i < len(whereConditions); i++ {
			query += " AND " + fmt.Sprintf("(%s)", whereConditions[i])
		}
	}

	// Add sorting
	if sort.Field != "" {
		direction := "ASC"
		if sort.Direction == "desc" {
			direction = "DESC"
		}

		var sortField string
		switch sort.Field {
		case "firstName", "first_name":
			sortField = "first_name"
		case "lastName", "last_name":
			sortField = "last_name"
		case "birthDate", "birth_date":
			sortField = "birth_date"
		case "createdAt", "created_at":
			sortField = "created_at"
		case "updatedAt", "updated_at":
			sortField = "updated_at"
		default:
			sortField = "created_at"
		}

		query += fmt.Sprintf(" ORDER BY %s %s", sortField, direction)
	} else {
		query += " ORDER BY created_at DESC"
	}

	return query, params
}

// Create creates a new child in the database
func (r *GenericChildRepository) Create(ctx context.Context, child *domain.Child) error {
	ctx, span := r.tracer.Start(ctx, "GenericChildRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("child.id", child.ID.String()),
		attribute.String("parent.id", child.ParentID.String()),
	)

	// First check if the parent exists
	parentQuery := `
		SELECT 1 FROM parents WHERE id = $1 AND deleted_at IS NULL
	`
	var exists bool
	err := r.pool.QueryRow(ctx, parentQuery, child.ParentID).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
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

// Update updates an existing child in the database
func (r *GenericChildRepository) Update(ctx context.Context, child *domain.Child) error {
	ctx, span := r.tracer.Start(ctx, "GenericChildRepository.Update")
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
		return fmt.Errorf("child not found for update")
	}

	return nil
}

// ListByParentID retrieves children for a specific parent with pagination, filtering, and sorting
func (r *GenericChildRepository) ListByParentID(ctx context.Context, parentID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	ctx, span := r.tracer.Start(ctx, "GenericChildRepository.ListByParentID")
	defer span.End()

	span.SetAttributes(attribute.String("parent.id", parentID.String()))

	// Modify the base query to filter by parent ID
	query := `
		SELECT id, first_name, last_name, birth_date, parent_id, created_at, updated_at, deleted_at
		FROM children
		WHERE deleted_at IS NULL AND parent_id = $1
	`

	params := []interface{}{parentID}
	paramIndex := 2
	whereConditions := []string{}

	if options.Filter.FirstName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("first_name ILIKE $%d", paramIndex))
		params = append(params, "%"+options.Filter.FirstName+"%")
		paramIndex++
	}

	if options.Filter.LastName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("last_name ILIKE $%d", paramIndex))
		params = append(params, "%"+options.Filter.LastName+"%")
		paramIndex++
	}

	if options.Filter.MinAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("birth_date <= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-options.Filter.MinAge, 0, 0))
		paramIndex++
	}

	if options.Filter.MaxAge > 0 {
		whereConditions = append(whereConditions, fmt.Sprintf("birth_date >= $%d", paramIndex))
		params = append(params, time.Now().AddDate(-options.Filter.MaxAge, 0, 0))
		paramIndex++
	}

	if len(whereConditions) > 0 {
		query += " AND " + fmt.Sprintf("(%s)", whereConditions[0])
		for i := 1; i < len(whereConditions); i++ {
			query += " AND " + fmt.Sprintf("(%s)", whereConditions[i])
		}
	}

	// Add sorting
	if options.Sort.Field != "" {
		direction := "ASC"
		if options.Sort.Direction == "desc" {
			direction = "DESC"
		}

		var sortField string
		switch options.Sort.Field {
		case "firstName", "first_name":
			sortField = "first_name"
		case "lastName", "last_name":
			sortField = "last_name"
		case "birthDate", "birth_date":
			sortField = "birth_date"
		case "createdAt", "created_at":
			sortField = "created_at"
		case "updatedAt", "updated_at":
			sortField = "updated_at"
		default:
			sortField = "created_at"
		}

		query += fmt.Sprintf(" ORDER BY %s %s", sortField, direction)
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Add pagination
	limit := options.Pagination.PageSize
	if limit <= 0 {
		limit = 10 // Default page size
	}

	offset := options.Pagination.Page * limit
	if offset < 0 {
		offset = 0
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	params = append(params, limit, offset)

	rows, err := r.pool.Query(ctx, query, params...)
	if err != nil {
		r.logger.Error("Failed to list children by parent ID", zap.Error(err), zap.String("parent_id", parentID.String()))
		return nil, nil, fmt.Errorf("failed to list children by parent ID: %w", err)
	}
	defer rows.Close()

	children := []*domain.Child{}

	for rows.Next() {
		child, err := r.scanChild(rows)
		if err != nil {
			r.logger.Error("Failed to scan child row", zap.Error(err))
			return nil, nil, fmt.Errorf("failed to scan child row: %w", err)
		}

		children = append(children, child)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating child rows", zap.Error(err))
		return nil, nil, fmt.Errorf("error iterating child rows: %w", err)
	}

	// Count total children for this parent
	countQuery := `
		SELECT COUNT(*)
		FROM children
		WHERE deleted_at IS NULL AND parent_id = $1
	`
	countParams := []interface{}{parentID}
	countParamIndex := 2
	countWhereConditions := []string{}

	if options.Filter.FirstName != "" {
		countWhereConditions = append(countWhereConditions, fmt.Sprintf("first_name ILIKE $%d", countParamIndex))
		countParams = append(countParams, "%"+options.Filter.FirstName+"%")
		countParamIndex++
	}

	if options.Filter.LastName != "" {
		countWhereConditions = append(countWhereConditions, fmt.Sprintf("last_name ILIKE $%d", countParamIndex))
		countParams = append(countParams, "%"+options.Filter.LastName+"%")
		countParamIndex++
	}

	if options.Filter.MinAge > 0 {
		countWhereConditions = append(countWhereConditions, fmt.Sprintf("birth_date <= $%d", countParamIndex))
		countParams = append(countParams, time.Now().AddDate(-options.Filter.MinAge, 0, 0))
		countParamIndex++
	}

	if options.Filter.MaxAge > 0 {
		countWhereConditions = append(countWhereConditions, fmt.Sprintf("birth_date >= $%d", countParamIndex))
		countParams = append(countParams, time.Now().AddDate(-options.Filter.MaxAge, 0, 0))
		countParamIndex++
	}

	if len(countWhereConditions) > 0 {
		countQuery += " AND " + fmt.Sprintf("(%s)", countWhereConditions[0])
		for i := 1; i < len(countWhereConditions); i++ {
			countQuery += " AND " + fmt.Sprintf("(%s)", countWhereConditions[i])
		}
	}

	var totalCount int64
	err = r.pool.QueryRow(ctx, countQuery, countParams...).Scan(&totalCount)
	if err != nil {
		r.logger.Error("Failed to count children by parent ID", zap.Error(err), zap.String("parent_id", parentID.String()))
		return nil, nil, fmt.Errorf("failed to count children by parent ID: %w", err)
	}

	pagedResult := &ports.PagedResult{
		TotalCount: totalCount,
		Page:       options.Pagination.Page,
		PageSize:   limit,
		HasNext:    int64(offset+len(children)) < totalCount,
	}

	return children, pagedResult, nil
}

// Ensure GenericChildRepository implements ports.Repository
var _ ports.Repository[*domain.Child] = (*GenericChildRepository)(nil)
