package mocks

import (
	"context"

	"github.com/abitofhelp/family-service2/internal/ports"
)

// MockRepositoryFactory is a mock implementation of the ports.RepositoryFactory interface
type MockRepositoryFactory struct {
	parentRepo *MockParentRepository
	childRepo  *MockChildRepository
	txManager  *MockTransactionManager
}

// NewMockRepositoryFactory creates a new mock repository factory
func NewMockRepositoryFactory() *MockRepositoryFactory {
	return &MockRepositoryFactory{
		parentRepo: NewMockParentRepository(),
		childRepo:  NewMockChildRepository(),
		txManager:  NewMockTransactionManager(),
	}
}

// NewParentRepository returns a parent repository
func (f *MockRepositoryFactory) NewParentRepository() ports.ParentRepository {
	return f.parentRepo
}

// NewChildRepository returns a child repository
func (f *MockRepositoryFactory) NewChildRepository() ports.ChildRepository {
	return f.childRepo
}

// GetTransactionManager returns the transaction manager
func (f *MockRepositoryFactory) GetTransactionManager() ports.TransactionManager {
	return f.txManager
}

// GetMockParentRepository returns the mock parent repository for test assertions
func (f *MockRepositoryFactory) GetMockParentRepository() *MockParentRepository {
	return f.parentRepo
}

// GetMockChildRepository returns the mock child repository for test assertions
func (f *MockRepositoryFactory) GetMockChildRepository() *MockChildRepository {
	return f.childRepo
}

// GetMockTransactionManager returns the mock transaction manager for test assertions
func (f *MockRepositoryFactory) GetMockTransactionManager() *MockTransactionManager {
	return f.txManager
}

// Reset resets all mock repositories and the transaction manager
func (f *MockRepositoryFactory) Reset() {
	f.parentRepo.Reset()
	f.childRepo.Reset()
	f.txManager.Reset()
}

// Close is a no-op for the mock repository factory
func (f *MockRepositoryFactory) Close(ctx context.Context) error {
	return nil
}

// InitSchema is a no-op for the mock repository factory
func (f *MockRepositoryFactory) InitSchema(ctx context.Context) error {
	return nil
}
