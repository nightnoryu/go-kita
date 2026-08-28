package transactional

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUnitOfWork struct {
	completeErr    error
	completeCalled bool
	completeWith   error
}

func (f *fakeUnitOfWork) Complete(err error) error {
	f.completeCalled = true
	f.completeWith = err
	if err != nil {
		return err
	}
	return f.completeErr
}

type fakeFactory struct {
	uow         *fakeUnitOfWork
	newErr      error
	gotLockName string
}

func (f *fakeFactory) NewLockableTransaction(_ context.Context, lockName string) (*fakeUnitOfWork, error) {
	f.gotLockName = lockName
	if f.newErr != nil {
		return nil, f.newErr
	}
	return f.uow, nil
}

func TestExecuteWithLock_Success(t *testing.T) {
	uow := &fakeUnitOfWork{}
	factory := &fakeFactory{uow: uow}
	executor := NewExecutor[*fakeUnitOfWork](factory)

	err := executor.ExecuteWithLock(context.Background(), "my-lock", func(*fakeUnitOfWork) error {
		return nil
	})

	require.NoError(t, err)
	assert.True(t, uow.completeCalled)
	require.NoError(t, uow.completeWith)
	assert.Equal(t, "my-lock", factory.gotLockName)
}

func TestExecuteWithLock_FnReturnsError(t *testing.T) {
	fnErr := errors.New("fn failed")
	uow := &fakeUnitOfWork{}
	factory := &fakeFactory{uow: uow}
	executor := NewExecutor[*fakeUnitOfWork](factory)

	err := executor.ExecuteWithLock(context.Background(), "", func(*fakeUnitOfWork) error {
		return fnErr
	})

	require.ErrorIs(t, err, fnErr)
	assert.True(t, uow.completeCalled)
	assert.ErrorIs(t, uow.completeWith, fnErr)
}

func TestExecuteWithLock_CompleteOverridesReturnedError(t *testing.T) {
	completeErr := errors.New("commit failed")
	uow := &fakeUnitOfWork{completeErr: completeErr}
	factory := &fakeFactory{uow: uow}
	executor := NewExecutor[*fakeUnitOfWork](factory)

	err := executor.ExecuteWithLock(context.Background(), "", func(*fakeUnitOfWork) error {
		return nil
	})

	require.ErrorIs(t, err, completeErr)
}

func TestExecuteWithLock_FactoryError(t *testing.T) {
	factoryErr := errors.New("factory failed")
	factory := &fakeFactory{newErr: factoryErr}
	executor := NewExecutor[*fakeUnitOfWork](factory)

	fnCalled := false
	err := executor.ExecuteWithLock(context.Background(), "", func(*fakeUnitOfWork) error {
		fnCalled = true
		return nil
	})

	require.ErrorIs(t, err, factoryErr)
	assert.False(t, fnCalled)
}

func TestExecuteWithLock_FnPanics_RollsBackAndRepanics(t *testing.T) {
	uow := &fakeUnitOfWork{}
	factory := &fakeFactory{uow: uow}
	executor := NewExecutor[*fakeUnitOfWork](factory)

	assert.PanicsWithValue(t, "boom", func() {
		_ = executor.ExecuteWithLock(context.Background(), "", func(*fakeUnitOfWork) error {
			panic("boom")
		})
	})

	assert.True(t, uow.completeCalled)
	assert.Error(t, uow.completeWith)
}
