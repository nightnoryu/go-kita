package jsonlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevelUnmarshalText(t *testing.T) {
	tests := map[string]struct {
		text string
		want Level
		err  bool
	}{
		"debug":   {text: "DEBUG", want: DebugLevel},
		"info":    {text: " info ", want: InfoLevel},
		"warning": {text: "warning", want: WarnLevel},
		"error":   {text: "error", want: ErrorLevel},
		"fatal":   {text: "fatal", want: FatalLevel},
		"invalid": {text: "verbose", err: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			level := InfoLevel
			err := level.UnmarshalText([]byte(tt.text))
			if tt.err {
				require.Error(t, err)
				assert.Equal(t, InfoLevel, level)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, level)
		})
	}
}
