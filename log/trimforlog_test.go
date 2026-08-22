package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimForLogs(t *testing.T) {
	t.Run("passthrough when marshaling fails", func(t *testing.T) {
		v := make(chan int)
		result := TrimForLogs(v, DefaultTrimForLogsOpts)
		assert.Equal(t, v, result)
	})

	t.Run("nil value marshals to nil", func(t *testing.T) {
		result := TrimForLogs(nil, DefaultTrimForLogsOpts)
		assert.Nil(t, result)
	})

	t.Run("non-string non-slice non-map values pass through unchanged", func(t *testing.T) {
		result := TrimForLogs(42, TrimForLogOptions{})
		assert.Equal(t, float64(42), result) //nolint:testifylint // exact deterministic value, not a computed float
	})

	t.Run("string within limit is unchanged", func(t *testing.T) {
		opts := TrimForLogOptions{MaxStringLength: 10}
		result := TrimForLogs("short", opts)
		assert.Equal(t, "short", result)
	})

	t.Run("string over limit is truncated", func(t *testing.T) {
		opts := TrimForLogOptions{MaxStringLength: 5}
		result := TrimForLogs("this is a long string", opts)
		assert.Equal(t, "this ...(truncated)", result)
	})

	t.Run("zero max string length disables truncation", func(t *testing.T) {
		opts := TrimForLogOptions{MaxStringLength: 0}
		result := TrimForLogs("arbitrarily long string value", opts)
		assert.Equal(t, "arbitrarily long string value", result)
	})

	t.Run("slice within limit is unchanged", func(t *testing.T) {
		opts := TrimForLogOptions{MaxSliceLength: 5}
		result := TrimForLogs([]int{1, 2, 3}, opts)
		assert.Equal(t, []any{float64(1), float64(2), float64(3)}, result)
	})

	t.Run("slice over limit is truncated with omitted count marker", func(t *testing.T) {
		opts := TrimForLogOptions{MaxSliceLength: 2}
		result := TrimForLogs([]int{1, 2, 3, 4, 5}, opts)
		assert.Equal(t, []any{float64(1), float64(2), "... (3 more)"}, result)
	})

	t.Run("zero max slice length disables truncation", func(t *testing.T) {
		opts := TrimForLogOptions{MaxSliceLength: 0}
		result := TrimForLogs([]int{1, 2, 3}, opts)
		assert.Equal(t, []any{float64(1), float64(2), float64(3)}, result)
	})

	t.Run("sensitive fields are replaced case-insensitively", func(t *testing.T) {
		opts := TrimForLogOptions{
			SensitiveFields:      []string{"Password", "token"},
			SensitivePlaceholder: "[REDACTED]",
		}
		input := map[string]any{
			"username": "alice",
			"PASSWORD": "hunter2",
			"Token":    "abc123",
		}
		result := TrimForLogs(input, opts)
		assert.Equal(t, map[string]any{
			"username": "alice",
			"PASSWORD": "[REDACTED]",
			"Token":    "[REDACTED]",
		}, result)
	})

	t.Run("sensitive fields are redacted at nested levels", func(t *testing.T) {
		opts := TrimForLogOptions{
			SensitiveFields:      []string{"secret"},
			SensitivePlaceholder: "[REDACTED]",
		}
		input := map[string]any{
			"outer": map[string]any{
				"secret": "value",
				"nested": []any{
					map[string]any{"secret": "another"},
				},
			},
		}
		result := TrimForLogs(input, opts)
		assert.Equal(t, map[string]any{
			"outer": map[string]any{
				"secret": "[REDACTED]",
				"nested": []any{
					map[string]any{"secret": "[REDACTED]"},
				},
			},
		}, result)
	})

	t.Run("nested strings and slices are trimmed recursively", func(t *testing.T) {
		opts := TrimForLogOptions{MaxStringLength: 3, MaxSliceLength: 1}
		input := map[string]any{
			"list": []string{"abcdef", "ghijkl"},
		}
		result := TrimForLogs(input, opts)
		assert.Equal(t, map[string]any{
			"list": []any{"abc...(truncated)", "... (1 more)"},
		}, result)
	})
}

func TestDefaultTrimForLogsOpts(t *testing.T) {
	assert.Equal(t, 100, DefaultTrimForLogsOpts.MaxStringLength)
	assert.Equal(t, 10, DefaultTrimForLogsOpts.MaxSliceLength)
	assert.Empty(t, DefaultTrimForLogsOpts.SensitiveFields)
	assert.Equal(t, "(SENSITIVE)", DefaultTrimForLogsOpts.SensitivePlaceholder)
}
