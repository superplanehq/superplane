package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
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

func Test__CreateCanvasMemoryNamespace(t *testing.T) {
	r := support.Setup(t)

	t.Run("empty entries -> error and no namespace persisted", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := CreateCanvasMemoryNamespace(context.Background(), r.Registry, r.Organization.ID.String(), canvas.ID.String(), "empty-namespace", nil)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)

		source, err := models.CanvasMemoryNamespaceSource(canvas.ID, "empty-namespace")
		require.NoError(t, err)
		assert.Equal(t, "", source, "no rows should be persisted for a rejected empty namespace")
	})

	t.Run("non-empty entries -> namespace persisted", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		resp, err := CreateCanvasMemoryNamespace(
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

func Test__UpdateCanvasMemoryNamespace(t *testing.T) {
	r := support.Setup(t)

	t.Run("empty entries -> error and existing namespace preserved", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := CreateCanvasMemoryNamespace(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"durable-namespace",
			structpbEntries(t, map[string]any{"key": "value"}),
		)
		require.NoError(t, err)

		_, err = UpdateCanvasMemoryNamespace(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"durable-namespace",
			"",
			nil,
		)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)

		records, err := models.ListCanvasMemoriesByNamespace(canvas.ID, "durable-namespace")
		require.NoError(t, err)
		assert.Len(t, records, 1, "existing namespace data must not be destroyed by a rejected empty update")
	})

	t.Run("non-empty entries -> namespace replaced", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := CreateCanvasMemoryNamespace(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"replace-namespace",
			structpbEntries(t, map[string]any{"key": "v1"}),
		)
		require.NoError(t, err)

		resp, err := UpdateCanvasMemoryNamespace(
			context.Background(),
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"replace-namespace",
			"",
			structpbEntries(t, map[string]any{"key": "v2"}, map[string]any{"key": "v3"}),
		)
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)
	})
}
