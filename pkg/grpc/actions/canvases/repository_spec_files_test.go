package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func TestResolveCommitCanvasAutoLayout(t *testing.T) {
	t.Run("not specified -> nil", func(t *testing.T) {
		assert.Nil(t, resolveCommitCanvasAutoLayout(false, nil))
	})

	t.Run("explicit disable -> nil", func(t *testing.T) {
		assert.Nil(t, resolveCommitCanvasAutoLayout(true, &pb.CanvasAutoLayout{}))
	})

	t.Run("explicit layout -> preserved", func(t *testing.T) {
		layout := &pb.CanvasAutoLayout{
			Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
			Scope:     pb.CanvasAutoLayout_SCOPE_FULL_CANVAS,
		}
		assert.Equal(t, layout, resolveCommitCanvasAutoLayout(true, layout))
	})
}
