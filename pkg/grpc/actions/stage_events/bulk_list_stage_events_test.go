package stageevents

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10, nil, nil, nil, nil, nil)
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

		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10, nil, nil, nil, nil, nil)
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

		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 2, nil, nil, nil, nil, nil)
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

		_, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10, nil, nil, nil, nil, nil)
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

		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)
		assert.Equal(t, r.Stage.ID.String(), res.Results[0].StageId)
		assert.Len(t, res.Results[0].Events, 3)
	})

	t.Run("filter by before timestamp", func(t *testing.T) {
		// Create a stage event, wait, then create another
		stageEvent1 := support.CreateStageEvent(t, r.Source, r.Stage)

		// Wait a bit to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
		beforeTime := time.Now()
		time.Sleep(10 * time.Millisecond)

		stageEvent2 := support.CreateStageEvent(t, r.Source, r.Stage)

		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		stages := []*protos.StageEventItemRequest{
			{
				StageIdOrName: r.Stage.ID.String(),
			},
		}

		// Test with before filter - should only return events created before the timestamp
		beforeTimestamp := timestamppb.New(beforeTime)
		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10, beforeTimestamp, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)

		// Should contain stageEvent1 but not stageEvent2
		foundEvent1 := false
		foundEvent2 := false
		for _, event := range res.Results[0].Events {
			if event.Id == stageEvent1.ID.String() {
				foundEvent1 = true
			}
			if event.Id == stageEvent2.ID.String() {
				foundEvent2 = true
			}
		}

		assert.True(t, foundEvent1, "Should find stageEvent1 (created before the timestamp)")
		assert.False(t, foundEvent2, "Should not find stageEvent2 (created after the timestamp)")
	})

	t.Run("filter by states", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		stages := []*protos.StageEventItemRequest{
			{
				StageIdOrName: r.Stage.ID.String(),
			},
		}

		// Test filtering by pending state only
		states := []protos.StageEvent_State{protos.StageEvent_STATE_PENDING}
		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10, nil, states, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)

		// All returned events should be in pending state
		for _, event := range res.Results[0].Events {
			assert.Equal(t, protos.StageEvent_STATE_PENDING, event.State)
		}
	})

	t.Run("filter by state reasons", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), authorization.OrganizationContextKey, r.Organization.ID.String())
		stages := []*protos.StageEventItemRequest{
			{
				StageIdOrName: r.Stage.ID.String(),
			},
		}

		// Test filtering by approval state reason
		stateReasons := []protos.StageEvent_StateReason{protos.StageEvent_STATE_REASON_APPROVAL}
		res, err := BulkListStageEvents(ctx, r.Canvas.ID.String(), stages, 10, nil, nil, stateReasons, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)

		// All returned events should have approval state reason (if any)
		for _, event := range res.Results[0].Events {
			assert.Equal(t, protos.StageEvent_STATE_REASON_APPROVAL, event.StateReason)
		}
	})
}
