package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReadinessHandler_HealthyChecksRunConcurrently(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	handler, err := NewReadinessHandler(ReadinessConfig{Checks: []NamedCheck{
		{Name: "postgres", Check: waitingCheck(started, release)},
		{Name: "redis", Check: waitingCheck(started, release)},
	}})
	require.NoError(t, err)

	responseDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))
		responseDone <- recorder
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("checks did not start concurrently")
		}
	}
	close(release)

	select {
	case recorder := <-responseDone:
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, okBody, recorder.Body.String())
	case <-time.After(time.Second):
		t.Fatal("handler did not complete")
	}
}

func TestReadinessHandler_UnhealthyDoesNotExposeError(t *testing.T) {
	dependencyErr := errors.New("postgres password is secret")
	var reportedName string
	var reportedErr error
	handler, err := NewReadinessHandler(ReadinessConfig{
		Checks: []NamedCheck{{Name: "postgres", Check: func(context.Context) error { return dependencyErr }}},
		OnFailure: func(name string, failure error) {
			reportedName = name
			reportedErr = failure
		},
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, unavailableBody, recorder.Body.String())
	require.NotContains(t, recorder.Body.String(), dependencyErr.Error())
	require.Equal(t, "postgres", reportedName)
	require.ErrorIs(t, reportedErr, dependencyErr)
}

func TestReadinessHandler_FailingCheckDoesNotReportCanceledPeers(t *testing.T) {
	reported := make(chan string, 2)
	peerDone := make(chan struct{})
	handler, err := NewReadinessHandler(ReadinessConfig{
		Checks: []NamedCheck{
			{Name: "failed", Check: func(context.Context) error { return errors.New("unhealthy") }},
			{Name: "peer", Check: func(ctx context.Context) error {
				defer close(peerDone)
				<-ctx.Done()
				return ctx.Err()
			}},
		},
		OnFailure: func(name string, _ error) { reported <- name },
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	<-peerDone

	require.Equal(t, "failed", <-reported)
	select {
	case name := <-reported:
		t.Fatalf("unexpected canceled peer report: %s", name)
	default:
	}
}

func TestReadinessHandler_OverallTimeout(t *testing.T) {
	reported := make(chan error, 1)
	handler, err := NewReadinessHandler(ReadinessConfig{
		Timeout: 20 * time.Millisecond,
		Checks: []NamedCheck{{Name: "slow", Check: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}}},
		OnFailure: func(_ string, err error) { reported <- err },
	})
	require.NoError(t, err)

	started := time.Now()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Less(t, time.Since(started), 250*time.Millisecond)
	select {
	case err := <-reported:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(time.Second):
		t.Fatal("timed-out check was not reported")
	}
}

func TestReadinessHandler_PerCheckTimeout(t *testing.T) {
	handler, err := NewReadinessHandler(ReadinessConfig{
		Timeout:      time.Second,
		CheckTimeout: 20 * time.Millisecond,
		Checks: []NamedCheck{{Name: "slow", Check: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}}},
	})
	require.NoError(t, err)

	started := time.Now()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Less(t, time.Since(started), 250*time.Millisecond)
}

func TestReadinessHandler_PerCheckTimeoutDoesNotRequireCheckToHonorContext(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	handler, err := NewReadinessHandler(ReadinessConfig{
		CheckTimeout: 20 * time.Millisecond,
		Checks: []NamedCheck{{Name: "slow", Check: func(context.Context) error {
			<-release
			return nil
		}}},
	})
	require.NoError(t, err)

	started := time.Now()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Less(t, time.Since(started), 250*time.Millisecond)
}

func TestReadinessHandler_FailureDoesNotWaitForReporter(t *testing.T) {
	releaseReporter := make(chan struct{})
	defer close(releaseReporter)
	handler, err := NewReadinessHandler(ReadinessConfig{
		Checks: []NamedCheck{{Name: "postgres", Check: func(context.Context) error {
			return errors.New("unhealthy")
		}}},
		OnFailure: func(string, error) { <-releaseReporter },
	})
	require.NoError(t, err)

	responseDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))
		responseDone <- recorder
	}()

	select {
	case recorder := <-responseDone:
		require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	case <-time.After(time.Second):
		t.Fatal("handler waited for failure reporter")
	}
}

func TestReadinessHandler_CanceledCheck(t *testing.T) {
	handler, err := NewReadinessHandler(ReadinessConfig{Checks: []NamedCheck{{
		Name: "postgres",
		Check: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}}})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody).WithContext(ctx))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, unavailableBody, recorder.Body.String())
}

func TestReadinessHandler_PanickingCheckDoesNotEscapeOrExposePanic(t *testing.T) {
	reported := make(chan error, 1)
	handler, err := NewReadinessHandler(ReadinessConfig{
		Checks: []NamedCheck{{Name: "redis", Check: func(context.Context) error {
			panic("redis password is secret")
		}}},
		OnFailure: func(_ string, err error) { reported <- err },
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, unavailableBody, recorder.Body.String())
	select {
	case err := <-reported:
		require.Contains(t, err.Error(), "panicked")
	case <-time.After(time.Second):
		t.Fatal("panicking check was not reported")
	}
}

func TestLivenessHandler_NilCheckIsHealthy(t *testing.T) {
	handler, err := NewLivenessHandler(LivenessConfig{})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, okBody, recorder.Body.String())
}

func TestLivenessHandler_TimeoutDoesNotRequireCheckToHonorContext(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	handler, err := NewLivenessHandler(LivenessConfig{
		Timeout: 20 * time.Millisecond,
		Check: func(context.Context) error {
			<-release
			return nil
		},
	})
	require.NoError(t, err)

	started := time.Now()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Less(t, time.Since(started), 250*time.Millisecond)
}

func waitingCheck(started chan<- struct{}, release <-chan struct{}) Check {
	return func(context.Context) error {
		started <- struct{}{}
		<-release
		return nil
	}
}
