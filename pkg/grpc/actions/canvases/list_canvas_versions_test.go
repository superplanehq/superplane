package canvases

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCanvasVersionLimit(t *testing.T) {
	require.Equal(t, uint32(DefaultLimit), getCanvasVersionLimit(0))
	require.Equal(t, uint32(20), getCanvasVersionLimit(20))
	require.Equal(t, uint32(MaxCanvasVersionLimit), getCanvasVersionLimit(MaxCanvasVersionLimit+1))
}
