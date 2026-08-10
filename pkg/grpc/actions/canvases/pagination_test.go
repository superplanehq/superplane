package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__TrimPage(t *testing.T) {
	t.Run("no sentinel row means the page is the last one", func(t *testing.T) {
		rows, hasNext := trimPage([]int{1, 2}, 2)
		assert.Equal(t, []int{1, 2}, rows)
		assert.False(t, hasNext)
	})

	t.Run("a sentinel row means more rows follow and is not returned", func(t *testing.T) {
		rows, hasNext := trimPage([]int{1, 2, 3}, 2)
		assert.Equal(t, []int{1, 2}, rows)
		assert.True(t, hasNext)
	})

	t.Run("partial page", func(t *testing.T) {
		rows, hasNext := trimPage([]int{1}, 2)
		assert.Equal(t, []int{1}, rows)
		assert.False(t, hasNext)
	})

	t.Run("empty page", func(t *testing.T) {
		rows, hasNext := trimPage([]int{}, 2)
		assert.Empty(t, rows)
		assert.False(t, hasNext)
	})
}
