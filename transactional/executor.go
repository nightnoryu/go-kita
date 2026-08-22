package transactional

import "context"

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
	unitOfWork, err := e.transactionFactory.NewLockableTransaction(ctx, lockName)
	if err != nil {
		return err
	}
	defer func() {
		err = unitOfWork.Complete(err)
	}()
	err = fn(unitOfWork)
	return err
}
