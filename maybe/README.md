# maybe

`Maybe[T]` represents a value in one of three states, which is useful for
partial updates and nullable database columns:

- `Absent` is the zero value and means no value was supplied.
- `None` means an explicit null value was supplied.
- `Just` holds a value, including a Go zero value such as `""` or `0`.

```go
name := maybe.NewJust("Ada")
clearNickname := maybe.NewNone[string]()
unchangedEmail := maybe.NewAbsent[string]()

if value, ok := maybe.JustValid(name); ok {
	fmt.Println(value) // Ada
}
if maybe.IsNone(clearNickname) {
	// Set the nickname column to NULL.
}
if maybe.IsAbsent(unchangedEmail) {
	// Do not update the email column.
}
```

`Just` panics when called for `Absent` or `None`; use `JustValid` when the
state is not already known. `Maybe` is a value type and has no shared mutable
state, so values may be copied and used concurrently. It holds no resources.

`Maybe[T]` implements `sql.Scanner` and `driver.Valuer`. Scanning SQL `NULL`
sets `None`; scanning a non-null value sets `Just`. Both `Absent` and `None`
produce SQL `NULL` when used as a query argument, so applications that need to
distinguish them in an update must choose the SQL statement from `IsSet` or
`IsAbsent` before binding values.
