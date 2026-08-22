package slices

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Run("transforms each element preserving order", func(t *testing.T) {
		result := Map([]int{1, 2, 3}, func(v int) int { return v * 2 })
		assert.Equal(t, []int{2, 4, 6}, result)
	})

	t.Run("supports changing the result type", func(t *testing.T) {
		result := Map([]int{1, 2, 3}, strconv.Itoa)
		assert.Equal(t, []string{"1", "2", "3"}, result)
	})

	t.Run("empty slice returns empty slice", func(t *testing.T) {
		result := Map([]int{}, func(v int) int { return v })
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("nil slice returns empty non-nil slice", func(t *testing.T) {
		result := Map[int, int](nil, func(v int) int { return v })
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("single element slice", func(t *testing.T) {
		result := Map([]string{"hello"}, func(v string) int { return len(v) })
		assert.Equal(t, []int{5}, result)
	})

	t.Run("does not mutate the source slice", func(t *testing.T) {
		source := []int{1, 2, 3}
		_ = Map(source, func(v int) int { return v * 10 })
		assert.Equal(t, []int{1, 2, 3}, source)
	})

	t.Run("function is applied to every element exactly once, in order", func(t *testing.T) {
		var visited []int
		source := []int{10, 20, 30}

		result := Map(source, func(v int) int {
			visited = append(visited, v)
			return v
		})

		assert.Equal(t, source, visited)
		assert.Equal(t, source, result)
	})
}
