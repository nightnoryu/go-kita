package jsonlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nightnoryu/go-kita/log"
)

func TestNewLogger(t *testing.T) {
	var l log.MainLogger
	output := captureStderr(t, func() {
		l = newLogger(t, InfoLevel)
		l.Info("hello")
	})

	require.NotNil(t, l)
	entries := decodeLines(t, output)
	require.Len(t, entries, 1)
	assert.Equal(t, "info", entries[0]["level"])
	assert.Equal(t, "hello", entries[0]["msg"])
	assert.Equal(t, "app", entries[0]["app_name"])
}

func TestNewLogger_EncodesTimeAsRFC3339(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		output := captureStderr(t, func() {
			l := newLogger(t, InfoLevel)
			l.Info("hello")
		})

		entries := decodeLines(t, output)
		require.Len(t, entries, 1)
		timestamp, ok := entries[0][timeKey].(string)
		require.True(t, ok, "time must be encoded as an RFC3339 string")

		got, err := time.Parse(time.RFC3339Nano, timestamp)
		require.NoError(t, err)
		assert.True(t, got.Equal(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)))
	})
}

func TestLogger_Debug(t *testing.T) {
	t.Run("emitted when level allows debug", func(t *testing.T) {
		output := captureStderr(t, func() {
			l := newLogger(t, DebugLevel)
			l.Debug("a", "b", 1)
		})

		entries := decodeLines(t, output)
		require.Len(t, entries, 1)
		assert.Equal(t, "debug", entries[0]["level"])
		assert.Equal(t, "ab1", entries[0]["msg"])
	})

	t.Run("suppressed when level is above debug", func(t *testing.T) {
		output := captureStderr(t, func() {
			l := newLogger(t, InfoLevel)
			l.Debug("should not appear")
		})

		assert.Empty(t, strings.TrimSpace(output))
	})
}

func TestLogger_Info(t *testing.T) {
	output := captureStderr(t, func() {
		l := newLogger(t, InfoLevel)
		l.Info("call ", "finished")
	})

	entries := decodeLines(t, output)
	require.Len(t, entries, 1)
	assert.Equal(t, "info", entries[0]["level"])
	assert.Equal(t, "call finished", entries[0]["msg"])
}

func TestLogger_Error(t *testing.T) {
	output := captureStderr(t, func() {
		l := newLogger(t, InfoLevel)
		l.Error(errors.New("boom"), "op failed")
	})

	entries := decodeLines(t, output)
	require.Len(t, entries, 1)
	assert.Equal(t, "error", entries[0]["level"])
	assert.Equal(t, "op failed", entries[0]["msg"])
	assert.Equal(t, "boom", entries[0]["error"])
}

func TestLogger_WithFields(t *testing.T) {
	output := captureStderr(t, func() {
		l := newLogger(t, InfoLevel)
		withFields := l.WithFields(log.Fields{
			"str_field":  "value",
			"int_field":  42,
			"bool_field": true,
			"any_field":  []string{"x", "y"},
		})
		withFields.Info("with fields")
	})

	entries := decodeLines(t, output)
	require.Len(t, entries, 1)
	entry := entries[0]
	assert.Equal(t, "value", entry["str_field"])
	assert.InEpsilon(t, 42, entry["int_field"], 0)
	assert.Equal(t, true, entry["bool_field"])
	assert.Equal(t, []any{"x", "y"}, entry["any_field"])
}

func TestLogger_WithFields_ReturnsNewLogger(t *testing.T) {
	l := newLogger(t, InfoLevel)
	withFields := l.WithFields(log.Fields{"key": "value"})

	assert.NotSame(t, l, withFields)
}

func TestNewLogger_InvalidConfig(t *testing.T) {
	_, err := NewLogger(nil)
	require.Error(t, err)

	_, err = NewLogger(&Config{Level: Level(99)})
	require.Error(t, err)
}

func TestLogger_Warn(t *testing.T) {
	output := captureStderr(t, func() {
		newLogger(t, InfoLevel).Warn("slow request")
	})

	entries := decodeLines(t, output)
	require.Len(t, entries, 1)
	assert.Equal(t, "warn", entries[0]["level"])
	assert.Equal(t, "slow request", entries[0]["msg"])
}

func newLogger(t *testing.T, level Level) log.MainLogger {
	t.Helper()
	l, err := NewLogger(&Config{Level: level, AppName: "app"})
	require.NoError(t, err)
	return l
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	fn()

	require.NoError(t, w.Close())
	os.Stderr = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	return buf.String()
}

func decodeLines(t *testing.T, output string) []map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	result := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m))
		result = append(result, m)
	}
	return result
}
