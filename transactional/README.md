# Transactional

`Executor` creates one unit of work, runs the callback, then calls `Complete`
with the callback error. A successful callback therefore lets the unit of work
commit; a callback error lets it roll back according to the implementation.

```go
err := executor.ExecuteWithLock(ctx, "band:42", func(repos app.RepoProvider) error {
	return repos.Bands().Rename(ctx, bandID, "The Examples")
})
```

The executor passes `ctx` only to `NewLockableTransaction`; the factory and
unit of work define how it bounds further operations. The executor owns the
single `Complete` call. Callbacks must not complete the unit of work or retain
its repositories after returning. Whether a unit of work is safe for concurrent
use is defined by its implementation, not by `Executor`.

If the callback and completion both return errors, `Execute` joins them. If the
callback succeeds, a completion error is returned directly. Factories should
return a unit of work only when they can later complete it safely.

If the callback panics, `Executor` calls `Complete` with a synthetic error whose
text begins with `panic:`, then rethrows the original panic value. A completion
error cannot replace that panic.
