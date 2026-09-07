# Runtime

Service runtime helpers.

## Example

```go
package main

import (
	"context"
	"log"

	"github.com/nightnoryu/go-kita/runtime"
)

func main() {
	ctx := context.Background()
	if err := runApp(ctx); err != nil {
		log.Fatal(err)
	}
}

func runApp(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	
	// Pass this context further and shutdown gracefully when it's canceled
	ctx = runtime.ListenOSKillSignals(ctx)

	// The rest of the application...

	return nil
}
```
