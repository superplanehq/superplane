package stageevents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func Test__BulkListStageEvents(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas with no stage events -> empty results", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		stages := []*protos.StageEventItemRequest{
			{
				StageIdOrName: r.Stage.ID.String(),
			},
		}
		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)
		assert.Empty(t, res.Results[0].Events)
		assert.Equal(t, r.Stage.ID.String(), res.Results[0].StageId)
	})

	t.Run("canvas with stage events - bulk list", func(t *testing.T) {
		stageEvent1 := support.CreateStageEvent(t, r.Source, r.Stage)
		stageEvent2 := support.CreateStageEvent(t, r.Source, r.Stage)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		stages := []*protos.StageEventItemRequest{
			{
				StageIdOrName: r.Stage.ID.String(),
			},
		}

		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)
		require.Len(t, res.Results[0].Events, 2)

		result := res.Results[0]
		assert.Equal(t, r.Stage.ID.String(), result.StageId)

		eventIDs := []string{result.Events[0].Id, result.Events[1].Id}
		assert.Contains(t, eventIDs, stageEvent1.ID.String())
		assert.Contains(t, eventIDs, stageEvent2.ID.String())

		e := result.Events[0]
		assert.NotEmpty(t, e.Id)
		assert.Equal(t, r.Source.ID.String(), e.SourceId)
		assert.Equal(t, protos.Connection_TYPE_EVENT_SOURCE, e.SourceType)
		assert.Equal(t, protos.StageEvent_STATE_PENDING, e.State)
		assert.NotNil(t, e.CreatedAt)
	})

	t.Run("limit per stage is respected", func(t *testing.T) {

		support.CreateStageEvent(t, r.Source, r.Stage)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		stages := []*protos.StageEventItemRequest{
			{
				StageIdOrName: r.Stage.ID.String(),
			},
		}

		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 2)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)

		assert.Len(t, res.Results[0].Events, 2)
	})

	t.Run("stage not found", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		stages := []*protos.StageEventItemRequest{
			{
				StageIdOrName: "non-existent-stage",
			},
		}

		_, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stage not found")
	})

	t.Run("find stage by name", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		stages := []*protos.StageEventItemRequest{
			{
				StageIdOrName: r.Stage.Name,
			},
		}

		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)
		assert.Equal(t, r.Stage.ID.String(), res.Results[0].StageId)
		assert.Len(t, res.Results[0].Events, 3)
	})
}
