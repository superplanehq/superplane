package actions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/delivery"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListStages(t *testing.T) {
	r := support.Setup(t)

	t.Run("no org ID -> error", func(t *testing.T) {
		_, err := ListStages(context.Background(), &protos.ListStagesRequest{
			CanvasId: r.Canvas.ID.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid UUID")
	})

	t.Run("no canvas ID -> error", func(t *testing.T) {
		_, err := ListStages(context.Background(), &protos.ListStagesRequest{
			OrganizationId: r.Org.String(),
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid UUID")
	})

	t.Run("no stages -> empty list", func(t *testing.T) {
		canvas, err := models.CreateCanvas(r.Org, r.User, "empty-canvas")
		require.NoError(t, err)

		res, err := ListStages(context.Background(), &protos.ListStagesRequest{
			CanvasId:       canvas.ID.String(),
			OrganizationId: r.Org.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Stages)
	})

	t.Run("with stage -> list", func(t *testing.T) {
		res, err := ListStages(context.Background(), &protos.ListStagesRequest{
			OrganizationId: r.Org.String(),
			CanvasId:       r.Canvas.ID.String(),
		})

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Stages, 1)
		assert.Equal(t, r.Stage.ID.String(), res.Stages[0].Id)
		assert.Equal(t, r.Canvas.ID.String(), res.Stages[0].CanvasId)
		assert.Equal(t, r.Org.String(), res.Stages[0].OrganizationId)
		assert.NotEmpty(t, res.Stages[0].CreatedAt)
		assert.NotEmpty(t, res.Stages[0].RunTemplate)
		require.Len(t, res.Stages[0].Conditions, 1)
		assert.Equal(t, protos.Condition_CONDITION_TYPE_APPROVAL, res.Stages[0].Conditions[0].Type)
		assert.Equal(t, uint32(1), res.Stages[0].Conditions[0].Approval.Count)
		assert.Empty(t, res.Stages[0].Connections)
	})
}
