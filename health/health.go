// Package health provides small, router-agnostic HTTP liveness and readiness
// handlers.
package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	okBody          = "ok\n"
	unavailableBody = "unavailable\n"
)

// Check verifies that a dependency is usable. It must honor ctx cancellation.
type Check func(context.Context) error

// NamedCheck associates a readiness check with the dependency it verifies.
// Names are supplied to FailureReporter and are never sent to HTTP clients.
type NamedCheck struct {
	Name  string
	Check Check
}

// FailureReporter receives a check failure exactly once. It is intended for
// application-owned structured logging. Health never logs itself, preventing
// duplicate log entries when the application also owns logging policy.
//
// Reporters must be safe for concurrent use and must not panic.
type FailureReporter func(name string, err error)

// LivenessConfig configures a liveness handler. A nil Check represents a
// process-only liveness endpoint and always succeeds. Timeout bounds Check
// when one is configured; zero disables an additional timeout.
type LivenessConfig struct {
	Check     Check
	Timeout   time.Duration
	OnFailure FailureReporter
}

// ReadinessConfig configures a readiness handler. Timeout bounds all checks
// together, while CheckTimeout independently bounds each check. Zero disables
// the respective additional timeout. The request context always bounds checks.
type ReadinessConfig struct {
	Checks       []NamedCheck
	Timeout      time.Duration
	CheckTimeout time.Duration
	OnFailure    FailureReporter
}

// NewLivenessHandler creates a handler without choosing a path or router.
func NewLivenessHandler(cfg LivenessConfig) (http.Handler, error) {
	if cfg.Timeout < 0 {
		return nil, errors.New("health: liveness timeout must not be negative")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.Check == nil {
			writeOK(w)
			return
		}

		ctx, cancel := withTimeout(r.Context(), cfg.Timeout)
		defer cancel()
		if err := runCheckWithContext(ctx, cfg.Check); err != nil {
			reportFailure(cfg.OnFailure, "liveness", err)
			writeUnavailable(w)
			return
		}
		writeOK(w)
	}), nil
}

// NewReadinessHandler creates a handler without choosing a path or router.
// Checks begin concurrently. A request returns unavailable as soon as its
// context or the configured overall timeout expires.
func NewReadinessHandler(cfg ReadinessConfig) (http.Handler, error) {
	if err := validateReadinessConfig(cfg); err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := withTimeout(r.Context(), cfg.Timeout)
		defer cancel()

		results := make(chan error, len(cfg.Checks))
		for _, check := range cfg.Checks {
			go func(check NamedCheck) {
				checkCtx, checkCancel := withTimeout(ctx, cfg.CheckTimeout)
				defer checkCancel()
				err := runCheckWithContext(checkCtx, check.Check)
				// The result must reach the handler before application logging. A
				// blocked reporter must not keep a readiness response alive.
				results <- err
				// Returning after another check fails cancels ctx. Do not turn that
				// internal cancellation into spurious dependency-error logs.
				if err != nil && (!errors.Is(err, context.Canceled) || ctx.Err() == nil) {
					reportFailure(cfg.OnFailure, check.Name, err)
				}
			}(check)
		}

		for range cfg.Checks {
			select {
			case err := <-results:
				if err != nil {
					writeUnavailable(w)
					return
				}
			case <-ctx.Done():
				writeUnavailable(w)
				return
			}
		}
		writeOK(w)
	}), nil
}

func validateReadinessConfig(cfg ReadinessConfig) error {
	if cfg.Timeout < 0 || cfg.CheckTimeout < 0 {
		return errors.New("health: timeouts must not be negative")
	}
	if len(cfg.Checks) == 0 {
		return errors.New("health: readiness requires at least one check")
	}
	names := make(map[string]struct{}, len(cfg.Checks))
	for _, check := range cfg.Checks {
		if check.Name == "" {
			return errors.New("health: check name must not be empty")
		}
		if check.Check == nil {
			return fmt.Errorf("health: check %q is nil", check.Name)
		}
		if _, exists := names[check.Name]; exists {
			return fmt.Errorf("health: duplicate check name %q", check.Name)
		}
		names[check.Name] = struct{}{}
	}
	return nil
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func runCheck(ctx context.Context, check Check) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("health: check panicked: %v", recovered)
		}
	}()
	return check(ctx)
}

// runCheckWithContext returns when either the check completes or its context
// ends. A check that ignores cancellation may continue in the background, but
// cannot keep a health request alive past its deadline.
func runCheckWithContext(ctx context.Context, check Check) error {
	result := make(chan error, 1)
	go func() {
		result <- runCheck(ctx, check)
	}()

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func reportFailure(reporter FailureReporter, name string, err error) {
	if reporter != nil {
		reporter(name, err)
	}
}

func writeOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(okBody))
}

func writeUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(unavailableBody))
}
