package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ParseLineColumnColors__AcceptsEmeraldAndBlue(t *testing.T) {
	colors, err := parseLineColumnColors(map[string]string{
		"backlog": "emerald",
		"done":    "blue",
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"backlog": "emerald", "done": "blue"}, colors)
}

func Test__ParseLineColumnColors__AcceptsOrangeAndRose(t *testing.T) {
	colors, err := parseLineColumnColors(map[string]string{
		"backlog": "orange",
		"done":    "rose",
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"backlog": "orange", "done": "rose"}, colors)
}

func Test__ParseLineColumnColors__RejectsUnknownColor(t *testing.T) {
	_, err := parseLineColumnColors(map[string]string{"backlog": "not-a-color"})
	assert.ErrorIs(t, err, errInvalidArgument)
}
