package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Host    string `env:"HOST"`
	Port    int    `env:"PORT"`
	Debug   bool   `env:"DEBUG"`
	Timeout string `env:"TIMEOUT" envDefault:"30s"`
}

func TestParseEnv(t *testing.T) {
	t.Run("parses values into the struct using the app-id prefix", func(t *testing.T) {
		t.Setenv("MYAPP_HOST", "localhost")
		t.Setenv("MYAPP_PORT", "5432")
		t.Setenv("MYAPP_DEBUG", "true")

		cfg, err := ParseEnv[testConfig]("myapp")
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 5432, cfg.Port)
		assert.True(t, cfg.Debug)
	})

	t.Run("uppercases the app id when building the prefix", func(t *testing.T) {
		t.Setenv("MYAPP_HOST", "from-upper")

		cfg, err := ParseEnv[testConfig]("myapp")
		require.NoError(t, err)
		assert.Equal(t, "from-upper", cfg.Host)
	})

	t.Run("applies envDefault when the variable is unset", func(t *testing.T) {
		cfg, err := ParseEnv[testConfig]("myapp")
		require.NoError(t, err)
		assert.Equal(t, "30s", cfg.Timeout)
	})

	t.Run("env value overrides envDefault", func(t *testing.T) {
		t.Setenv("MYAPP_TIMEOUT", "5s")

		cfg, err := ParseEnv[testConfig]("myapp")
		require.NoError(t, err)
		assert.Equal(t, "5s", cfg.Timeout)
	})

	t.Run("ignores variables without the matching prefix", func(t *testing.T) {
		t.Setenv("HOST", "no-prefix")
		t.Setenv("OTHERAPP_HOST", "other-prefix")

		cfg, err := ParseEnv[testConfig]("myapp")
		require.NoError(t, err)
		assert.Empty(t, cfg.Host)
	})

	t.Run("returns a wrapped error on invalid value", func(t *testing.T) {
		t.Setenv("MYAPP_PORT", "not-a-number")

		cfg, err := ParseEnv[testConfig]("myapp")
		assert.Nil(t, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse env")
	})

	t.Run("returns an error when a required field is missing", func(t *testing.T) {
		type requiredConfig struct {
			Token string `env:"TOKEN,required"`
		}

		cfg, err := ParseEnv[requiredConfig]("myapp")
		assert.Nil(t, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse env")
	})

	t.Run("returns a zero-value struct when nothing is set", func(t *testing.T) {
		type plainConfig struct {
			Name string `env:"NAME"`
		}

		cfg, err := ParseEnv[plainConfig]("myapp")
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, plainConfig{}, *cfg)
	})
}
