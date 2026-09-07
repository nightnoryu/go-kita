package runtime

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// ListenOSKillSignals cancels the context upon receiving SIGTERM or SIGINT
func ListenOSKillSignals(ctx context.Context) context.Context {
	var cancelFunc context.CancelFunc
	ctx, cancelFunc = context.WithCancel(ctx)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
		select {
		case <-ch:
			cancelFunc()
		case <-ctx.Done():
			signal.Reset()
			return
		}
	}()
	return ctx
}
