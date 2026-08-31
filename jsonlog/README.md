# jsonlog

Structured JSON logging: a [zap](https://github.com/uber-go/zap)-backed implementation of `log.MainLogger`.

## Behavior notes

- `Debug` / `Info` join their variadic args with `fmt.Sprint`.
- `Error(err, args...)` and `FatalError(err, args...)` attach `err` as a structured `error` field; `FatalError` then
  exits the process via `os.Exit(1)`.
- `WithFields` emits `string`, `int`, and `bool` values with typed zap fields and falls back to `zap.Any` for everything
  else. Keys are sorted, so field order in the output is deterministic.
- `Sync` (from `log.MainLogger`) flushes zap's buffer; call it with `defer` in `main`.

## Example

```go
package main

import (
	"errors"

	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
)

func main() {
	logger := jsonlog.NewLogger(&jsonlog.Config{
		Level:   jsonlog.InfoLevel,
		AppName: "test",
	})
	defer func() { _ = logger.Sync() }()

	logger.
		WithFields(log.Fields{"duration": "1s", "method": "GET /test"}).
		Info("call finished")

	logger.Error(errors.New("db timeout"), "query failed")

	// logger.FatalError(errors.New("boot failed"), "cannot start") // logs, then os.Exit(1)
}
```
