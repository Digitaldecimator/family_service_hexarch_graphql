package mocks

import (
	"context"
	"errors"
)

// MockTransactionManager is a mock implementation of the ports.TransactionManager interface
type MockTransactionManager struct {
	// Function mocks for testing specific scenarios
	BeginTxFunc    func(ctx context.Context) (context.Context, error)
	CommitTxFunc   func(ctx context.Context) error
	RollbackTxFunc func(ctx context.Context) error

	// State tracking for assertions
	BeginTxCalled    bool
	CommitTxCalled   bool
	RollbackTxCalled bool

	// Error simulation
	ShouldFailBegin    bool
	ShouldFailCommit   bool
	ShouldFailRollback bool
}

// NewMockTransactionManager creates a new mock transaction manager
func NewMockTransactionManager() *MockTransactionManager {
	return &MockTransactionManager{}
}

// BeginTx begins a new transaction and stores it in the context
func (tm *MockTransactionManager) BeginTx(ctx context.Context) (context.Context, error) {
	if tm.BeginTxFunc != nil {
		return tm.BeginTxFunc(ctx)
	}

	tm.BeginTxCalled = true

	if tm.ShouldFailBegin {
		return ctx, errors.New("simulated begin transaction failure")
	}

	// Store transaction state in context
	ctx = context.WithValue(ctx, txKey("tx_started"), true)

	return ctx, nil
}

// CommitTx commits the transaction stored in the context
func (tm *MockTransactionManager) CommitTx(ctx context.Context) error {
	if tm.CommitTxFunc != nil {
		return tm.CommitTxFunc(ctx)
	}

	tm.CommitTxCalled = true

	// Check if transaction was started
	if _, ok := ctx.Value(txKey("tx_started")).(bool); !ok {
		return errors.New("no transaction found in context")
	}

	if tm.ShouldFailCommit {
		return errors.New("simulated commit transaction failure")
	}

	return nil
}

// RollbackTx rolls back the transaction stored in the context
func (tm *MockTransactionManager) RollbackTx(ctx context.Context) error {
	if tm.RollbackTxFunc != nil {
		return tm.RollbackTxFunc(ctx)
	}

	tm.RollbackTxCalled = true

	// Check if transaction was started
	if _, ok := ctx.Value(txKey("tx_started")).(bool); !ok {
		return errors.New("no transaction found in context")
	}

	if tm.ShouldFailRollback {
		return errors.New("simulated rollback transaction failure")
	}

	return nil
}

// Reset resets the state of the mock transaction manager
func (tm *MockTransactionManager) Reset() {
	tm.BeginTxCalled = false
	tm.CommitTxCalled = false
	tm.RollbackTxCalled = false
	tm.ShouldFailBegin = false
	tm.ShouldFailCommit = false
	tm.ShouldFailRollback = false
}

// txKey is a type for context keys
type txKey string
