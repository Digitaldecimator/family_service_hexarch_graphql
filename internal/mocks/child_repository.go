package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/abitofhelp/family-service2/internal/domain"
	"github.com/abitofhelp/family-service2/internal/ports"
	"github.com/google/uuid"
)

// MockChildRepository is a mock implementation of the ports.ChildRepository interface
type MockChildRepository struct {
	mu       sync.RWMutex
	children map[uuid.UUID]*domain.Child

	// Function mocks for testing specific scenarios
	CreateFunc         func(ctx context.Context, child *domain.Child) error
	GetByIDFunc        func(ctx context.Context, id uuid.UUID) (*domain.Child, error)
	UpdateFunc         func(ctx context.Context, child *domain.Child) error
	DeleteFunc         func(ctx context.Context, id uuid.UUID) error
	ListByParentIDFunc func(ctx context.Context, parentID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error)
	ListFunc           func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error)
	CountFunc          func(ctx context.Context, filter ports.FilterOptions) (int64, error)
}

// NewMockChildRepository creates a new mock child repository
func NewMockChildRepository() *MockChildRepository {
	return &MockChildRepository{
		children: make(map[uuid.UUID]*domain.Child),
	}
}

// Create adds a child to the mock repository
func (r *MockChildRepository) Create(ctx context.Context, child *domain.Child) error {
	if r.CreateFunc != nil {
		return r.CreateFunc(ctx, child)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if child already exists
	if _, exists := r.children[child.ID]; exists {
		return errors.New("child already exists")
	}

	// Create a deep copy to avoid reference issues
	childCopy := *child
	r.children[child.ID] = &childCopy

	return nil
}

// GetByID retrieves a child by ID from the mock repository
func (r *MockChildRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	if r.GetByIDFunc != nil {
		return r.GetByIDFunc(ctx, id)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	child, exists := r.children[id]
	if !exists || child.DeletedAt != nil {
		return nil, errors.New("child not found")
	}

	// Return a copy to avoid reference issues
	childCopy := *child
	return &childCopy, nil
}

// Update updates a child in the mock repository
func (r *MockChildRepository) Update(ctx context.Context, child *domain.Child) error {
	if r.UpdateFunc != nil {
		return r.UpdateFunc(ctx, child)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if child exists
	_, exists := r.children[child.ID]
	if !exists || r.children[child.ID].DeletedAt != nil {
		return errors.New("child not found")
	}

	// Update the child
	childCopy := *child
	r.children[child.ID] = &childCopy

	return nil
}

// Delete marks a child as deleted in the mock repository
func (r *MockChildRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.DeleteFunc != nil {
		return r.DeleteFunc(ctx, id)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if child exists
	child, exists := r.children[id]
	if !exists || child.DeletedAt != nil {
		return errors.New("child not found")
	}

	// Mark child as deleted
	child.MarkAsDeleted()

	return nil
}

// ListByParentID retrieves children for a specific parent with pagination, filtering, and sorting
func (r *MockChildRepository) ListByParentID(ctx context.Context, parentID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	if r.ListByParentIDFunc != nil {
		return r.ListByParentIDFunc(ctx, parentID, options)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter children by parent ID
	var filteredChildren []*domain.Child
	for _, child := range r.children {
		if child.DeletedAt != nil {
			continue
		}

		if child.ParentID != parentID {
			continue
		}

		// Apply filters
		if options.Filter.FirstName != "" && child.FirstName != options.Filter.FirstName {
			continue
		}
		if options.Filter.LastName != "" && child.LastName != options.Filter.LastName {
			continue
		}

		// Add child to filtered list
		childCopy := *child
		filteredChildren = append(filteredChildren, &childCopy)
	}

	// Calculate total count
	totalCount := int64(len(filteredChildren))

	// Apply pagination
	start := options.Pagination.Page * options.Pagination.PageSize
	end := start + options.Pagination.PageSize
	if start >= len(filteredChildren) {
		start = len(filteredChildren)
	}
	if end > len(filteredChildren) {
		end = len(filteredChildren)
	}

	paginatedChildren := filteredChildren[start:end]

	// Create paged result
	pagedResult := &ports.PagedResult{
		TotalCount: totalCount,
		Page:       options.Pagination.Page,
		PageSize:   options.Pagination.PageSize,
		HasNext:    end < len(filteredChildren),
	}

	return paginatedChildren, pagedResult, nil
}

// List retrieves a list of children with pagination, filtering, and sorting
func (r *MockChildRepository) List(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	if r.ListFunc != nil {
		return r.ListFunc(ctx, options)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter children
	var filteredChildren []*domain.Child
	for _, child := range r.children {
		if child.DeletedAt != nil {
			continue
		}

		// Apply filters
		if options.Filter.FirstName != "" && child.FirstName != options.Filter.FirstName {
			continue
		}
		if options.Filter.LastName != "" && child.LastName != options.Filter.LastName {
			continue
		}

		// Add child to filtered list
		childCopy := *child
		filteredChildren = append(filteredChildren, &childCopy)
	}

	// Calculate total count
	totalCount := int64(len(filteredChildren))

	// Apply pagination
	start := options.Pagination.Page * options.Pagination.PageSize
	end := start + options.Pagination.PageSize
	if start >= len(filteredChildren) {
		start = len(filteredChildren)
	}
	if end > len(filteredChildren) {
		end = len(filteredChildren)
	}

	paginatedChildren := filteredChildren[start:end]

	// Create paged result
	pagedResult := &ports.PagedResult{
		TotalCount: totalCount,
		Page:       options.Pagination.Page,
		PageSize:   options.Pagination.PageSize,
		HasNext:    end < len(filteredChildren),
	}

	return paginatedChildren, pagedResult, nil
}

// Count returns the total count of children matching the filter
func (r *MockChildRepository) Count(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	if r.CountFunc != nil {
		return r.CountFunc(ctx, filter)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter children
	var count int64
	for _, child := range r.children {
		if child.DeletedAt != nil {
			continue
		}

		// Apply filters
		if filter.FirstName != "" && child.FirstName != filter.FirstName {
			continue
		}
		if filter.LastName != "" && child.LastName != filter.LastName {
			continue
		}

		count++
	}

	return count, nil
}

// AddTestChild adds a test child to the mock repository
func (r *MockChildRepository) AddTestChild(child *domain.Child) {
	r.mu.Lock()
	defer r.mu.Unlock()

	childCopy := *child
	r.children[child.ID] = &childCopy
}

// Reset clears all children from the mock repository
func (r *MockChildRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.children = make(map[uuid.UUID]*domain.Child)
}
