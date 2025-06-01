package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/abitofhelp/family-service2/internal/ports"
	"github.com/google/uuid"
)

// MockParentRepository is a mock implementation of the ports.ParentRepository interface
type MockParentRepository struct {
	mu      sync.RWMutex
	parents map[uuid.UUID]*domain.Parent

	// Function mocks for testing specific scenarios
	CreateFunc  func(ctx context.Context, parent *domain.Parent) error
	GetByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Parent, error)
	UpdateFunc  func(ctx context.Context, parent *domain.Parent) error
	DeleteFunc  func(ctx context.Context, id uuid.UUID) error
	ListFunc    func(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error)
	CountFunc   func(ctx context.Context, filter ports.FilterOptions) (int64, error)
}

// NewMockParentRepository creates a new mock parent repository
func NewMockParentRepository() *MockParentRepository {
	return &MockParentRepository{
		parents: make(map[uuid.UUID]*domain.Parent),
	}
}

// Create adds a parent to the mock repository
func (r *MockParentRepository) Create(ctx context.Context, parent *domain.Parent) error {
	if r.CreateFunc != nil {
		return r.CreateFunc(ctx, parent)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if parent already exists
	if _, exists := r.parents[parent.ID]; exists {
		return errors.New("parent already exists")
	}

	// Create a deep copy to avoid reference issues
	parentCopy := *parent
	r.parents[parent.ID] = &parentCopy

	return nil
}

// GetByID retrieves a parent by ID from the mock repository
func (r *MockParentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	if r.GetByIDFunc != nil {
		return r.GetByIDFunc(ctx, id)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	parent, exists := r.parents[id]
	if !exists || parent.DeletedAt != nil {
		return nil, errors.New("parent not found")
	}

	// Return a copy to avoid reference issues
	parentCopy := *parent
	return &parentCopy, nil
}

// Update updates a parent in the mock repository
func (r *MockParentRepository) Update(ctx context.Context, parent *domain.Parent) error {
	if r.UpdateFunc != nil {
		return r.UpdateFunc(ctx, parent)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if parent exists
	_, exists := r.parents[parent.ID]
	if !exists || r.parents[parent.ID].DeletedAt != nil {
		return errors.New("parent not found")
	}

	// Update the parent
	parentCopy := *parent
	r.parents[parent.ID] = &parentCopy

	return nil
}

// Delete marks a parent as deleted in the mock repository
func (r *MockParentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.DeleteFunc != nil {
		return r.DeleteFunc(ctx, id)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if parent exists
	parent, exists := r.parents[id]
	if !exists || parent.DeletedAt != nil {
		return errors.New("parent not found")
	}

	// Mark parent as deleted
	parent.MarkAsDeleted()

	return nil
}

// List retrieves a list of parents with pagination, filtering, and sorting
func (r *MockParentRepository) List(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
	if r.ListFunc != nil {
		return r.ListFunc(ctx, options)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter parents
	var filteredParents []*domain.Parent
	for _, parent := range r.parents {
		if parent.DeletedAt != nil {
			continue
		}

		// Apply filters
		if options.Filter.FirstName != "" && parent.FirstName != options.Filter.FirstName {
			continue
		}
		if options.Filter.LastName != "" && parent.LastName != options.Filter.LastName {
			continue
		}
		if options.Filter.Email != "" && parent.Email != options.Filter.Email {
			continue
		}

		// Add parent to filtered list
		parentCopy := *parent
		filteredParents = append(filteredParents, &parentCopy)
	}

	// Calculate total count
	totalCount := int64(len(filteredParents))

	// Apply pagination
	start := options.Pagination.Page * options.Pagination.PageSize
	end := start + options.Pagination.PageSize
	if start >= len(filteredParents) {
		start = len(filteredParents)
	}
	if end > len(filteredParents) {
		end = len(filteredParents)
	}

	paginatedParents := filteredParents[start:end]

	// Create paged result
	pagedResult := &ports.PagedResult{
		TotalCount: totalCount,
		Page:       options.Pagination.Page,
		PageSize:   options.Pagination.PageSize,
		HasNext:    end < len(filteredParents),
	}

	return paginatedParents, pagedResult, nil
}

// Count returns the total count of parents matching the filter
func (r *MockParentRepository) Count(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	if r.CountFunc != nil {
		return r.CountFunc(ctx, filter)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter parents
	var count int64
	for _, parent := range r.parents {
		if parent.DeletedAt != nil {
			continue
		}

		// Apply filters
		if filter.FirstName != "" && parent.FirstName != filter.FirstName {
			continue
		}
		if filter.LastName != "" && parent.LastName != filter.LastName {
			continue
		}
		if filter.Email != "" && parent.Email != filter.Email {
			continue
		}

		count++
	}

	return count, nil
}

// AddTestParent adds a test parent to the mock repository
func (r *MockParentRepository) AddTestParent(parent *domain.Parent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	parentCopy := *parent
	r.parents[parent.ID] = &parentCopy
}

// Reset clears all parents from the mock repository
func (r *MockParentRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.parents = make(map[uuid.UUID]*domain.Parent)
}
