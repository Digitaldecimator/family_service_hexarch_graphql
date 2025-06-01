package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// GenericParentRepository implements the ports.Repository interface for Parent entities
type GenericParentRepository struct {
	*BaseRepository[*domain.Parent]
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewGenericParentRepository creates a new generic parent repository
func NewGenericParentRepository(pool *pgxpool.Pool, logger *zap.Logger) *GenericParentRepository {
	repo := &GenericParentRepository{
		pool:   pool,
		logger: logger,
	}

	// Create the base repository with Parent-specific functions
	baseRepo := NewBaseRepository[*domain.Parent](
		pool,
		logger,
		"postgres.parent_repository",
		"parents",
		repo.scanParent,
		repo.buildListQuery,
	)

	repo.BaseRepository = baseRepo
	return repo
}

// scanParent scans a database row into a Parent entity
func (r *GenericParentRepository) scanParent(row pgx.Row) (*domain.Parent, error) {
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
		return nil, err
	}

	if deletedAt.Valid {
		parent.DeletedAt = &deletedAt.Time
	}

	return &parent, nil
}

// buildListQuery builds a query for listing parents with filtering and sorting
func (r *GenericParentRepository) buildListQuery(filter ports.FilterOptions, sort ports.SortOptions) (string, []interface{}) {
	query := `
		SELECT id, first_name, last_name, email, birth_date, created_at, updated_at, deleted_at
		FROM parents
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

	if filter.Email != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("email ILIKE $%d", paramIndex))
		params = append(params, "%"+filter.Email+"%")
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
		case "email":
			sortField = "email"
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

// Create creates a new parent in the database
func (r *GenericParentRepository) Create(ctx context.Context, parent *domain.Parent) error {
	ctx, span := r.tracer.Start(ctx, "GenericParentRepository.Create")
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

// Update updates an existing parent in the database
func (r *GenericParentRepository) Update(ctx context.Context, parent *domain.Parent) error {
	ctx, span := r.tracer.Start(ctx, "GenericParentRepository.Update")
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

// Ensure GenericParentRepository implements ports.Repository
var _ ports.Repository[*domain.Parent] = (*GenericParentRepository)(nil)
