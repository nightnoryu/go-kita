# retry

`retry` schedules retries for context-aware operations. The initial operation
call is attempt 1, so `MaxAttempts: 3` makes at most three calls.

```go
r, err := retry.New(retry.Config{
	MaxAttempts:  3,
	InitialDelay: time.Second,
	MaxDelay:     10 * time.Second,
	Multiplier:   2,
})
if err != nil {
	return err
}

err = r.Do(ctx, connect, isTemporary)
```

`Do` returns the last operation error after exhausting attempts, or `ctx.Err()`
when cancellation stops an operation or a delay. The retryer does not own the
context or any resource used by the operation. A nil predicate retries all
operation errors.
