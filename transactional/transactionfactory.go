package transactional

import "context"

type TransactionFactory[TRepoProvider UnitOfWork] interface {
	NewLockableTransaction(ctx context.Context, lockName string) (TRepoProvider, error)
}
