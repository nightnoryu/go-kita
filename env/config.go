// Package env parses application configuration from environment variables.
package env

import (
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/go-faster/errors"
)

// ParseEnv populates struct from environment variables
func ParseEnv[T any](appID string) (*T, error) {
	c := new(T)
	prefix := strings.ToUpper(strings.ReplaceAll(appID, "-", "_")) + "_"
	if err := env.ParseWithOptions(c, env.Options{Prefix: prefix}); err != nil {
		return nil, errors.Wrap(err, "failed to parse env")
	}
	return c, nil
}
