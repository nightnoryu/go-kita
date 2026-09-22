# slices

`slices` contains small generic helpers for sequential slice transformations.

```go
labels := slices.Map([]int{1, 2, 3}, func(id int) string {
	return fmt.Sprintf("user-%d", id)
})

ids, err := slices.MapErr([]string{"1", "2"}, strconv.Atoi)
```

`Map` always returns a newly allocated slice. `MapErr` stops at the first
error and returns that error with a nil result; it does not wrap errors and
does not run work concurrently. A nil input produces a non-nil empty result.
Neither helper owns resources or accepts a context because the supplied
function determines whether an operation can block.
