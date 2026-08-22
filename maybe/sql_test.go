package maybe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan_NilSetsNone(t *testing.T) {
	m := NewJust(7)

	err := m.Scan(nil)

	require.NoError(t, err)
	assert.True(t, IsNone(m))
	assert.False(t, Valid(m))
}

func TestScan_ValueSetsJust(t *testing.T) {
	var m Maybe[int]

	err := m.Scan(int64(7))

	require.NoError(t, err)
	assert.True(t, Valid(m))
	assert.Equal(t, 7, Just(m))
}

func TestValue_Just(t *testing.T) {
	m := NewJust("hello")

	v, err := m.Value()

	require.NoError(t, err)
	assert.Equal(t, "hello", v)
}

func TestValue_None(t *testing.T) {
	m := NewNone[string]()

	v, err := m.Value()

	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestValue_Absent(t *testing.T) {
	m := NewAbsent[string]()

	v, err := m.Value()

	require.NoError(t, err)
	assert.Nil(t, v)
}
