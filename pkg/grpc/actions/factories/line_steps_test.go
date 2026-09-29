package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__parseLineColumnColors(t *testing.T) {
	t.Run("accepts rose and indigo", func(t *testing.T) {
		colors, err := parseLineColumnColors(map[string]string{
			"backlog": "rose",
			"plan":    "indigo",
		})

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"backlog": "rose",
			"plan":    "indigo",
		}, colors)
	})

	t.Run("rejects an unknown color id", func(t *testing.T) {
		colors, err := parseLineColumnColors(map[string]string{
			"backlog": "red",
		})

		require.Error(t, err)
		assert.Nil(t, colors)
		assert.ErrorContains(t, err, `column color "red" is not supported`)
	})
}
