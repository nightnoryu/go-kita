package transactional

import (
	"context"
	"errors"
	"fmt"
)

type UnitOfWork interface {
	Complete(err error) error
}

type Executor[TRepoProvider UnitOfWork] interface {
	Execute(ctx context.Context, fn func(repoProvider TRepoProvider) error) error
	ExecuteWithLock(ctx context.Context, lockName string, fn func(repoProvider TRepoProvider) error) error
}

func NewExecutor[TRepoProvider UnitOfWork](transactionFactory TransactionFactory[TRepoProvider]) Executor[TRepoProvider] {
	return &executor[TRepoProvider]{
		transactionFactory: transactionFactory,
	}
}

type executor[TRepoProvider UnitOfWork] struct {
	transactionFactory TransactionFactory[TRepoProvider]
}

func (e *executor[TRepoProvider]) Execute(ctx context.Context, fn func(repoProvider TRepoProvider) error) error {
	return e.ExecuteWithLock(ctx, "", fn)
}

func (e *executor[TRepoProvider]) ExecuteWithLock(ctx context.Context, lockName string, fn func(repoProvider TRepoProvider) error) (err error) {
	transaction, err := e.transactionFactory.NewLockableTransaction(ctx, lockName)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			err = transaction.Complete(fmt.Errorf("panic: %v", r))
			panic(r)
		}
		completeErr := transaction.Complete(err)
		if err == nil {
			err = completeErr
		} else if completeErr != nil && !errors.Is(completeErr, err) {
			err = errors.Join(err, completeErr)
		}
	}()
	err = fn(transaction)
	return err
}
