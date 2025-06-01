package mocks

import (
	"context"

	"github.com/abitofhelp/family_service_hexarch_graphql/internal/domain"
	"github.com/abitofhelp/family_service_hexarch_graphql/internal/ports"
	"github.com/google/uuid"
)

// MockFamilyService is a mock implementation of the ports.FamilyService interface
type MockFamilyService struct {
	// Function mocks for ParentService methods
	CreateParentFunc  func(ctx context.Context, firstName, lastName, email string, birthDate string) (*domain.Parent, error)
	GetParentByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Parent, error)
	UpdateParentFunc  func(ctx context.Context, id uuid.UUID, firstName, lastName, email string, birthDate string) (*domain.Parent, error)
	DeleteParentFunc  func(ctx context.Context, id uuid.UUID) error
	ListParentsFunc   func(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error)
	CountParentsFunc  func(ctx context.Context, filter ports.FilterOptions) (int64, error)

	// Function mocks for ChildService methods
	CreateChildFunc            func(ctx context.Context, firstName, lastName string, birthDate string, parentID uuid.UUID) (*domain.Child, error)
	GetChildByIDFunc           func(ctx context.Context, id uuid.UUID) (*domain.Child, error)
	UpdateChildFunc            func(ctx context.Context, id uuid.UUID, firstName, lastName string, birthDate string) (*domain.Child, error)
	DeleteChildFunc            func(ctx context.Context, id uuid.UUID) error
	ListChildrenByParentIDFunc func(ctx context.Context, parentID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error)
	ListChildrenFunc           func(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error)
	CountChildrenFunc          func(ctx context.Context, filter ports.FilterOptions) (int64, error)

	// Function mocks for additional FamilyService methods
	AddChildToParentFunc      func(ctx context.Context, parentID, childID uuid.UUID) error
	RemoveChildFromParentFunc func(ctx context.Context, parentID, childID uuid.UUID) error
}

// NewMockFamilyService creates a new mock family service
func NewMockFamilyService() *MockFamilyService {
	return &MockFamilyService{}
}

// CreateParent implements ports.ParentService
func (m *MockFamilyService) CreateParent(ctx context.Context, firstName, lastName, email string, birthDate string) (*domain.Parent, error) {
	if m.CreateParentFunc != nil {
		return m.CreateParentFunc(ctx, firstName, lastName, email, birthDate)
	}
	return nil, nil
}

// GetParentByID implements ports.ParentService
func (m *MockFamilyService) GetParentByID(ctx context.Context, id uuid.UUID) (*domain.Parent, error) {
	if m.GetParentByIDFunc != nil {
		return m.GetParentByIDFunc(ctx, id)
	}
	return nil, nil
}

// UpdateParent implements ports.ParentService
func (m *MockFamilyService) UpdateParent(ctx context.Context, id uuid.UUID, firstName, lastName, email string, birthDate string) (*domain.Parent, error) {
	if m.UpdateParentFunc != nil {
		return m.UpdateParentFunc(ctx, id, firstName, lastName, email, birthDate)
	}
	return nil, nil
}

// DeleteParent implements ports.ParentService
func (m *MockFamilyService) DeleteParent(ctx context.Context, id uuid.UUID) error {
	if m.DeleteParentFunc != nil {
		return m.DeleteParentFunc(ctx, id)
	}
	return nil
}

// ListParents implements ports.ParentService
func (m *MockFamilyService) ListParents(ctx context.Context, options ports.QueryOptions) ([]*domain.Parent, *ports.PagedResult, error) {
	if m.ListParentsFunc != nil {
		return m.ListParentsFunc(ctx, options)
	}
	return nil, nil, nil
}

// CountParents implements ports.ParentService
func (m *MockFamilyService) CountParents(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	if m.CountParentsFunc != nil {
		return m.CountParentsFunc(ctx, filter)
	}
	return 0, nil
}

// CreateChild implements ports.ChildService
func (m *MockFamilyService) CreateChild(ctx context.Context, firstName, lastName string, birthDate string, parentID uuid.UUID) (*domain.Child, error) {
	if m.CreateChildFunc != nil {
		return m.CreateChildFunc(ctx, firstName, lastName, birthDate, parentID)
	}
	return nil, nil
}

// GetChildByID implements ports.ChildService
func (m *MockFamilyService) GetChildByID(ctx context.Context, id uuid.UUID) (*domain.Child, error) {
	if m.GetChildByIDFunc != nil {
		return m.GetChildByIDFunc(ctx, id)
	}
	return nil, nil
}

// UpdateChild implements ports.ChildService
func (m *MockFamilyService) UpdateChild(ctx context.Context, id uuid.UUID, firstName, lastName string, birthDate string) (*domain.Child, error) {
	if m.UpdateChildFunc != nil {
		return m.UpdateChildFunc(ctx, id, firstName, lastName, birthDate)
	}
	return nil, nil
}

// DeleteChild implements ports.ChildService
func (m *MockFamilyService) DeleteChild(ctx context.Context, id uuid.UUID) error {
	if m.DeleteChildFunc != nil {
		return m.DeleteChildFunc(ctx, id)
	}
	return nil
}

// ListChildrenByParentID implements ports.ChildService
func (m *MockFamilyService) ListChildrenByParentID(ctx context.Context, parentID uuid.UUID, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	if m.ListChildrenByParentIDFunc != nil {
		return m.ListChildrenByParentIDFunc(ctx, parentID, options)
	}
	return nil, nil, nil
}

// ListChildren implements ports.ChildService
func (m *MockFamilyService) ListChildren(ctx context.Context, options ports.QueryOptions) ([]*domain.Child, *ports.PagedResult, error) {
	if m.ListChildrenFunc != nil {
		return m.ListChildrenFunc(ctx, options)
	}
	return nil, nil, nil
}

// CountChildren implements ports.ChildService
func (m *MockFamilyService) CountChildren(ctx context.Context, filter ports.FilterOptions) (int64, error) {
	if m.CountChildrenFunc != nil {
		return m.CountChildrenFunc(ctx, filter)
	}
	return 0, nil
}

// AddChildToParent implements ports.FamilyService
func (m *MockFamilyService) AddChildToParent(ctx context.Context, parentID, childID uuid.UUID) error {
	if m.AddChildToParentFunc != nil {
		return m.AddChildToParentFunc(ctx, parentID, childID)
	}
	return nil
}

// RemoveChildFromParent implements ports.FamilyService
func (m *MockFamilyService) RemoveChildFromParent(ctx context.Context, parentID, childID uuid.UUID) error {
	if m.RemoveChildFromParentFunc != nil {
		return m.RemoveChildFromParentFunc(ctx, parentID, childID)
	}
	return nil
}
