# Transactional

`Executor` creates one unit of work, runs the callback, then calls `Complete`
with the callback error. A successful callback therefore lets the unit of work
commit; a callback error lets it roll back according to the implementation.

If the callback panics, `Executor` calls `Complete` with a synthetic error whose
text begins with `panic:`, then rethrows the original panic value. A completion
error cannot replace that panic.
