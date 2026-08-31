package env

import (
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/go-faster/errors"
)

// ParseEnv populates struct from environment variables
func ParseEnv[T any](appID string) (*T, error) {
	c := new(T)
	if err := env.ParseWithOptions(c, env.Options{Prefix: strings.ToUpper(appID) + "_"}); err != nil {
		return nil, errors.Wrap(err, "failed to parse env")
	}
	return c, nil
}
