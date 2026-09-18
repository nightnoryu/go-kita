# health

`health` provides router-agnostic HTTP handlers for liveness and readiness.
Applications choose endpoint paths and register the returned handlers with
their router.

```go
ready, err := health.NewReadinessHandler(health.ReadinessConfig{
	Timeout:      2 * time.Second,
	CheckTimeout: time.Second,
	Checks: []health.NamedCheck{
		{Name: "postgres", Check: db.Ping},
		{Name: "redis", Check: cache.Ping},
	},
	OnFailure: func(name string, err error) {
		logger.WithFields(log.Fields{"dependency": name}).Error(err, "readiness check failed")
	},
})
if err != nil {
	return err
}
router.Handle("/readyz", ready)
router.Handle("/healthz", must(health.NewLivenessHandler(health.LivenessConfig{})))
```

Readiness checks run concurrently. The request context, optional overall
`Timeout`, and optional per-check `CheckTimeout` bound their execution. Checks
must honor their context; a check that ignores it can continue after the HTTP
response has returned, but cannot delay that response beyond its deadline.

Successful handlers return `200 OK` with `ok\n`; failed handlers return `503
Service Unavailable` with `unavailable\n`. Dependency names and errors are
never included in a response.

The package does not log. Use `OnFailure` to log named check errors once using
the application's structured logger. The callback can run concurrently and
must not panic.
