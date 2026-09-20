# jsonlog

Structured JSON logging: a [zap](https://github.com/uber-go/zap)-backed implementation of `log.MainLogger`.

## Behavior notes

- Time is printed in RFC3339 format: `{"time": "2026-09-16T08:26:12.3456789+03:00"}` - consider this when implementing logs parsing
- `Level` accepts `debug`, `info`, `warn` (or `warning`), `error`, and `fatal`, case-insensitively. It implements
  `encoding.TextUnmarshaler`, so it can be used directly in environment-backed config structs.
- `Debug` / `Info` join their variadic args with `fmt.Sprint`.
- `Warn` emits warning records.
- `Error(err, args...)` and `FatalError(err, args...)` attach `err` as a structured `error` field; `FatalError` then
  exits the process via `os.Exit(1)`.
- `WithFields` emits `string`, `int`, and `bool` values with typed zap fields and falls back to `zap.Any` for everything
  else. Keys are sorted, so field order in the output is deterministic.
- `Sync` (from `log.MainLogger`) flushes zap's buffer; call it with `defer` in `main`.

## Example

```go
package main

import (
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
)

func main() {
	logger, err := jsonlog.NewLogger(&jsonlog.Config{
		Level:   jsonlog.InfoLevel,
		AppName: "test",
	})
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	logger.
		WithFields(log.Fields{"duration": "1s", "method": "GET /test"}).
		Info("call finished")

	logger.Error(errors.New("db timeout"), "query failed")

	// logger.FatalError(err, "cannot start") // logs, then os.Exit(1)
}
```
