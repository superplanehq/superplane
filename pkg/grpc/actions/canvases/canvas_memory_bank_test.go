package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func structpbEntries(t *testing.T, entries ...map[string]any) []*structpb.Value {
	values := make([]*structpb.Value, 0, len(entries))
	for _, entry := range entries {
		value, err := structpb.NewValue(entry)
		require.NoError(t, err)
		values = append(values, value)
	}
	return values
}

func Test__CreateCanvasMemoryBank(t *testing.T) {
	r := support.Setup(t)

	t.Run("empty entries -> error and no bank persisted", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := CreateCanvasMemoryBank(context.Background(), r.Registry, r.Organization.ID.String(), canvas.ID.String(), "empty-bank", nil)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())

		source, err := models.CanvasMemoryNamespaceSource(canvas.ID, "empty-bank")
		require.NoError(t, err)
		assert.Equal(t, "", source, "no rows should be persisted for a rejected empty bank")
	})

	t.Run("non-empty entries -> bank persisted", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		resp, err := CreateCanvasMemoryBank(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"release-cache",
			structpbEntries(t, map[string]any{"key": "value"}),
		)
		require.NoError(t, err)
		require.Len(t, resp.Items, 1)

		source, err := models.CanvasMemoryNamespaceSource(canvas.ID, "release-cache")
		require.NoError(t, err)
		assert.Equal(t, models.CanvasMemorySourceManual, source)
	})
}

func Test__UpdateCanvasMemoryBank(t *testing.T) {
	r := support.Setup(t)

	t.Run("empty entries -> error and existing bank preserved", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := CreateCanvasMemoryBank(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"durable-bank",
			structpbEntries(t, map[string]any{"key": "value"}),
		)
		require.NoError(t, err)

		_, err = UpdateCanvasMemoryBank(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"durable-bank",
			"",
			nil,
		)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())

		records, err := models.ListCanvasMemoriesByNamespace(canvas.ID, "durable-bank")
		require.NoError(t, err)
		assert.Len(t, records, 1, "existing bank data must not be destroyed by a rejected empty update")
	})

	t.Run("non-empty entries -> bank replaced", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := CreateCanvasMemoryBank(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"replace-bank",
			structpbEntries(t, map[string]any{"key": "v1"}),
		)
		require.NoError(t, err)

		resp, err := UpdateCanvasMemoryBank(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"replace-bank",
			"",
			structpbEntries(t, map[string]any{"key": "v2"}, map[string]any{"key": "v3"}),
		)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)
	})
}
