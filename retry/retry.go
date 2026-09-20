package retry

import (
	"context"
	stderrors "errors"
	"fmt"
	"math"
	"time"
)

// Config controls retry scheduling. MaxAttempts includes the initial operation
// call. A zero MaxDelay means delays are not capped.
type Config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// Operation is invoked once per attempt with the context passed to Do.
type Operation func(context.Context) error

// Predicate reports whether an operation error should be retried. A nil
// predicate retries every operation error.
type Predicate func(error) bool

type Retryer struct {
	config Config
	wait   func(context.Context, time.Duration) error
}

type Option func(*Retryer)

// WithWait replaces waiting between attempts. It is useful for deterministic
// tests; implementations must return ctx.Err when the context is canceled.
func WithWait(wait func(context.Context, time.Duration) error) Option {
	return func(r *Retryer) {
		if wait != nil {
			r.wait = wait
		}
	}
}

func New(config Config, options ...Option) (*Retryer, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	r := &Retryer{
		config: config,
		wait:   wait,
	}
	for _, option := range options {
		if option != nil {
			option(r)
		}
	}
	return r, nil
}

// Do calls operation until it succeeds, its error is not retryable, attempts
// are exhausted, or ctx is canceled. On exhaustion it returns the final
// operation error. If cancellation stops work or waiting, it returns ctx.Err.
func (r *Retryer) Do(ctx context.Context, operation Operation, predicate Predicate) error {
	if ctx == nil {
		return stderrors.New("retry: nil context")
	}
	if operation == nil {
		return stderrors.New("retry: nil operation")
	}

	delay := r.capDelay(r.config.InitialDelay)
	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := operation(ctx)
		if err == nil {
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if attempt == r.config.MaxAttempts || (predicate != nil && !predicate(err)) {
			return err
		}

		if err := r.wait(ctx, delay); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return err
		}
		delay = r.nextDelay(delay)
	}

	panic("retry: unreachable")
}

func validateConfig(config Config) error {
	if config.MaxAttempts <= 0 {
		return fmt.Errorf("retry: max attempts must be positive")
	}
	if config.InitialDelay < 0 || config.MaxDelay < 0 {
		return fmt.Errorf("retry: delays must not be negative")
	}
	if math.IsNaN(config.Multiplier) || math.IsInf(config.Multiplier, 0) || config.Multiplier < 1 {
		return fmt.Errorf("retry: multiplier must be finite and at least one")
	}
	return nil
}

func (r *Retryer) nextDelay(delay time.Duration) time.Duration {
	if delay == 0 {
		return 0
	}
	if r.config.Multiplier == 1 {
		return delay
	}
	next := float64(delay) * r.config.Multiplier
	if next >= float64(r.maximumDelay()) {
		return r.maximumDelay()
	}
	return time.Duration(next)
}

func (r *Retryer) capDelay(delay time.Duration) time.Duration {
	if r.config.MaxDelay > 0 && delay > r.config.MaxDelay {
		return r.config.MaxDelay
	}
	return delay
}

func (r *Retryer) maximumDelay() time.Duration {
	if r.config.MaxDelay > 0 {
		return r.config.MaxDelay
	}
	return time.Duration(1<<63 - 1)
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
