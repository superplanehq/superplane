package stages

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListStages(t *testing.T) {
	r := support.Setup(t)

	t.Run("no stages -> empty list", func(t *testing.T) {
		org, err := models.CreateOrganization(uuid.New(), "test", "test", "")
		require.NoError(t, err)

		canvas, err := models.CreateCanvas(r.User, org.ID, "empty-canvas", "empty canvas")
		require.NoError(t, err)

		res, err := ListStages(context.Background(), canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Stages)
	})

	t.Run("with stage -> list", func(t *testing.T) {
		res, err := ListStages(context.Background(), r.Canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Stages, 1)
		assert.Equal(t, r.Stage.ID.String(), res.Stages[0].Metadata.Id)
		assert.Equal(t, r.Canvas.ID.String(), res.Stages[0].Metadata.CanvasId)
		assert.NotEmpty(t, res.Stages[0].Metadata.CreatedAt)
		assert.NotEmpty(t, res.Stages[0].Spec.Executor)
		require.Len(t, res.Stages[0].Spec.Conditions, 1)
		assert.Equal(t, protos.Condition_CONDITION_TYPE_APPROVAL, res.Stages[0].Spec.Conditions[0].Type)
		assert.Equal(t, uint32(1), res.Stages[0].Spec.Conditions[0].Approval.Count)
		assert.Len(t, res.Stages[0].Spec.Connections, 1)
	})
}
