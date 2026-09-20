package retry

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryerDo_ImmediateSuccess(t *testing.T) {
	r := newRetryer(t, Config{MaxAttempts: 3, Multiplier: 2})
	calls := 0

	err := r.Do(context.Background(), func(context.Context) error {
		calls++
		return nil
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryerDo_EventualSuccess(t *testing.T) {
	var delays []time.Duration
	r := newRetryer(t, Config{MaxAttempts: 3, InitialDelay: time.Second, Multiplier: 2}, WithWait(recordWait(&delays)))
	calls := 0

	err := r.Do(context.Background(), func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}
		return nil
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second}, delays)
}

func TestRetryerDo_ExhaustionReturnsLastError(t *testing.T) {
	r := newRetryer(t, Config{MaxAttempts: 2, Multiplier: 1}, WithWait(noWait))
	first := errors.New("first")
	last := errors.New("last")
	calls := 0

	err := r.Do(context.Background(), func(context.Context) error {
		calls++
		if calls == 1 {
			return first
		}
		return last
	}, nil)

	require.ErrorIs(t, err, last)
	assert.Equal(t, 2, calls)
}

func TestRetryerDo_NonRetryableError(t *testing.T) {
	r := newRetryer(t, Config{MaxAttempts: 3, Multiplier: 1}, WithWait(noWait))
	errNotRetryable := errors.New("not retryable")
	calls := 0

	err := r.Do(context.Background(), func(context.Context) error {
		calls++
		return errNotRetryable
	}, func(err error) bool { return !errors.Is(err, errNotRetryable) })

	require.ErrorIs(t, err, errNotRetryable)
	assert.Equal(t, 1, calls)
}

func TestRetryerDo_CancellationDuringWork(t *testing.T) {
	r := newRetryer(t, Config{MaxAttempts: 2, Multiplier: 1})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := r.Do(ctx, func(ctx context.Context) error {
		cancel()
		<-ctx.Done()
		return errors.New("work failed")
	}, nil)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestRetryerDo_CancellationDuringDelay(t *testing.T) {
	enteredWait := make(chan struct{})
	r := newRetryer(t, Config{MaxAttempts: 2, InitialDelay: time.Hour, Multiplier: 1}, WithWait(func(ctx context.Context, _ time.Duration) error {
		close(enteredWait)
		<-ctx.Done()
		return ctx.Err()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- r.Do(ctx, func(context.Context) error { return errors.New("temporary") }, nil)
	}()

	select {
	case <-enteredWait:
	case <-time.After(time.Second):
		t.Fatal("retry did not begin waiting")
	}
	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("retry did not stop after cancellation")
	}
}

func TestRetryerDo_CapsBackoff(t *testing.T) {
	var delays []time.Duration
	r := newRetryer(t, Config{MaxAttempts: 4, InitialDelay: time.Second, MaxDelay: 3 * time.Second, Multiplier: 2}, WithWait(recordWait(&delays)))

	err := r.Do(context.Background(), func(context.Context) error { return errors.New("temporary") }, nil)

	require.Error(t, err)
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}, delays)
}

func TestRetryerDo_CapsInitialDelay(t *testing.T) {
	var delays []time.Duration
	r := newRetryer(t, Config{MaxAttempts: 2, InitialDelay: 10 * time.Second, MaxDelay: 3 * time.Second, Multiplier: 2}, WithWait(recordWait(&delays)))

	err := r.Do(context.Background(), func(context.Context) error { return errors.New("temporary") }, nil)

	require.Error(t, err)
	assert.Equal(t, []time.Duration{3 * time.Second}, delays)
}

func TestRetryerDo_SaturatesBackoff(t *testing.T) {
	const maxDuration = time.Duration(1<<63 - 1)
	var delays []time.Duration
	r := newRetryer(t, Config{MaxAttempts: 3, InitialDelay: maxDuration, Multiplier: 2}, WithWait(recordWait(&delays)))

	require.Error(t, r.Do(context.Background(), func(context.Context) error { return errors.New("temporary") }, nil))
	assert.Equal(t, []time.Duration{maxDuration, maxDuration}, delays)
}

func TestNew_InvalidConfig(t *testing.T) {
	for _, config := range []Config{
		{MaxAttempts: 0, Multiplier: 1},
		{MaxAttempts: 1, InitialDelay: -time.Second, Multiplier: 1},
		{MaxAttempts: 1, Multiplier: 0.5},
		{MaxAttempts: 1, Multiplier: math.NaN()},
		{MaxAttempts: 1, Multiplier: math.Inf(1)},
	} {
		_, err := New(config)
		assert.Error(t, err)
	}
}

func newRetryer(t *testing.T, config Config, options ...Option) *Retryer {
	t.Helper()
	r, err := New(config, options...)
	require.NoError(t, err)
	return r
}

func recordWait(delays *[]time.Duration) func(context.Context, time.Duration) error {
	return func(_ context.Context, delay time.Duration) error {
		*delays = append(*delays, delay)
		return nil
	}
}

func noWait(context.Context, time.Duration) error {
	return nil
}
