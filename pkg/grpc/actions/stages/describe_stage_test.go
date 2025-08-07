package stages

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeStage(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
		Stage:       true,
		Approvals:   1,
	})

	t.Run("no name and no ID -> error", func(t *testing.T) {
		_, err := DescribeStage(context.Background(), r.Canvas.ID.String(), "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "must specify either the ID or name of the stage", s.Message())
	})

	t.Run("stage does not exist -> error", func(t *testing.T) {
		_, err := DescribeStage(context.Background(), r.Canvas.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "stage not found", s.Message())
	})

	t.Run("with name", func(t *testing.T) {
		response, err := DescribeStage(context.Background(), r.Canvas.ID.String(), r.Stage.Name)
		require.NoError(t, err)
		require.Equal(t, r.Stage.Name, response.Stage.Metadata.Name)
		require.Equal(t, r.Stage.ID.String(), response.Stage.Metadata.Id)
		require.Equal(t, r.Canvas.ID.String(), response.Stage.Metadata.CanvasId)
		require.NotNil(t, response.Stage.Metadata.CreatedAt)
		require.NotNil(t, response.Stage.Spec.Executor)
		require.Len(t, response.Stage.Spec.Connections, 1)
		require.Len(t, response.Stage.Spec.Conditions, 1)
		require.Len(t, response.Stage.Spec.Inputs, 1)
		require.Len(t, response.Stage.Spec.InputMappings, 1)
		assert.Equal(t, protos.Condition_CONDITION_TYPE_APPROVAL, response.Stage.Spec.Conditions[0].Type)
		assert.Equal(t, uint32(1), response.Stage.Spec.Conditions[0].Approval.Count)
	})

	t.Run("with ID", func(t *testing.T) {
		response, err := DescribeStage(context.Background(), r.Canvas.ID.String(), r.Stage.ID.String())
		require.NoError(t, err)
		require.Equal(t, r.Stage.Name, response.Stage.Metadata.Name)
		require.Equal(t, r.Stage.ID.String(), response.Stage.Metadata.Id)
		require.Equal(t, r.Canvas.ID.String(), response.Stage.Metadata.CanvasId)
		require.NotNil(t, response.Stage.Metadata.CreatedAt)
		require.Len(t, response.Stage.Spec.Conditions, 1)
		require.NotNil(t, response.Stage.Spec.Executor)
		require.Len(t, response.Stage.Spec.Connections, 1)
		require.Len(t, response.Stage.Spec.Inputs, 1)
		require.Len(t, response.Stage.Spec.InputMappings, 1)
		assert.Equal(t, protos.Condition_CONDITION_TYPE_APPROVAL, response.Stage.Spec.Conditions[0].Type)
		assert.Equal(t, uint32(1), response.Stage.Spec.Conditions[0].Approval.Count)
	})
}
