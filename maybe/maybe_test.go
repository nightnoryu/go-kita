package maybe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewJust(t *testing.T) {
	m := NewJust(42)

	assert.True(t, Valid(m))
	assert.False(t, IsNone(m))
	assert.False(t, IsAbsent(m))
	assert.True(t, IsSet(m))
	assert.Equal(t, 42, Just(m))
}

func TestNewNone(t *testing.T) {
	m := NewNone[int]()

	assert.False(t, Valid(m))
	assert.True(t, IsNone(m))
	assert.False(t, IsAbsent(m))
	assert.True(t, IsSet(m))
}

func TestNewAbsent(t *testing.T) {
	m := NewAbsent[int]()

	assert.False(t, Valid(m))
	assert.False(t, IsNone(m))
	assert.True(t, IsAbsent(m))
	assert.False(t, IsSet(m))
}

func TestZeroValueIsAbsent(t *testing.T) {
	var m Maybe[int]

	assert.False(t, Valid(m))
	assert.False(t, IsNone(m))
	assert.True(t, IsAbsent(m))
	assert.False(t, IsSet(m))
	assert.Equal(t, NewAbsent[int](), m)
}

func TestJust_PanicsOnNonJust(t *testing.T) {
	assert.Panics(t, func() {
		Just(NewNone[int]())
	})
	assert.Panics(t, func() {
		Just(NewAbsent[int]())
	})
}

func TestJustValid(t *testing.T) {
	tests := []struct {
		name   string
		maybe  Maybe[string]
		wantV  string
		wantOk bool
	}{
		{name: "just", maybe: NewJust("value"), wantV: "value", wantOk: true},
		{name: "none", maybe: NewNone[string](), wantV: "", wantOk: false},
		{name: "absent", maybe: NewAbsent[string](), wantV: "", wantOk: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, ok := JustValid(tt.maybe)

			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantV, v)
		})
	}
}
