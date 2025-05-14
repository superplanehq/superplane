package actions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/protos/delivery"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateTagState(t *testing.T) {
	setup := func() []*models.StageEvent {
		r := support.Setup(t)
		event1 := support.CreateStageEvent(t, r.Source, r.Stage)
		event2 := support.CreateStageEventWithData(t, r.Source, r.Stage, []byte(`{"ref":"v2"}`), []byte(`{"ref":"v2"}`))

		// create tags with different tag names
		require.NoError(t, models.UpdateStageEventTagStateInBulk(
			database.Conn(),
			event1.ID,
			models.TagStateUnknown,
			map[string]string{
				"SHA": "1234",
			},
		))

		return []*models.StageEvent{event1, event2}
	}

	t.Run("missing tag name", func(t *testing.T) {
		setup()
		_, err := UpdateTagState(context.Background(), &delivery.UpdateTagStateRequest{
			Tag: &delivery.Tag{},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "missing tag name or value")
	})

	t.Run("missing tag value", func(t *testing.T) {
		setup()
		_, err := UpdateTagState(context.Background(), &delivery.UpdateTagStateRequest{
			Tag: &delivery.Tag{Name: "VERSION"},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "missing tag name or value")
	})

	t.Run("tag is marked as healthy - waiting(unhealthy) events are moved to pending", func(t *testing.T) {
		events := setup()

		// first event will be waiting(unhealthy)
		require.NoError(t,
			events[0].UpdateState(models.StageEventStateWaiting, models.StageEventStateReasonUnhealthy),
		)

		res, err := UpdateTagState(context.Background(), &delivery.UpdateTagStateRequest{
			Tag: &delivery.Tag{
				Name:  "VERSION",
				Value: "v1",
				State: delivery.Tag_TAG_STATE_HEALTHY,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res)

		//
		// Verify tags with different name were not updated
		//
		tags, err := models.ListStageTags("SHA", "", []string{}, "", "")
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "SHA", tags[0].TagName)
		assert.Equal(t, "1234", tags[0].TagValue)
		assert.Equal(t, models.TagStateUnknown, tags[0].TagState)

		//
		// Verify tags with same name but different values were not updated
		//
		tags, err = models.ListStageTags("VERSION", "v2", []string{}, "", "")
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "VERSION", tags[0].TagName)
		assert.Equal(t, "v2", tags[0].TagValue)
		assert.Equal(t, models.TagStateUnknown, tags[0].TagState)

		//
		// Verify tags with same name and value were updated
		//
		tags, err = models.ListStageTags("VERSION", "v1", []string{}, "", "")
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "VERSION", tags[0].TagName)
		assert.Equal(t, "v1", tags[0].TagValue)
		assert.Equal(t, models.TagStateHealthy, tags[0].TagState)

		//
		// Verify stage events in waiting(unhealthy) were moved back to pending
		//
		e, err := models.FindStageEventByID(events[0].ID.String(), events[0].StageID.String())
		require.NoError(t, err)
		assert.Equal(t, models.StageEventStatePending, e.State)
		assert.Empty(t, e.StateReason)
	})

	t.Run("tag is marked as unhealthy", func(t *testing.T) {
		setup()
		res, err := UpdateTagState(context.Background(), &delivery.UpdateTagStateRequest{
			Tag: &delivery.Tag{
				Name:  "VERSION",
				Value: "v1",
				State: delivery.Tag_TAG_STATE_UNHEALTHY,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, res)

		//
		// Verify tags with different name were not updated
		//
		tags, err := models.ListStageTags("SHA", "", []string{}, "", "")
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "SHA", tags[0].TagName)
		assert.Equal(t, "1234", tags[0].TagValue)
		assert.Equal(t, models.TagStateUnknown, tags[0].TagState)

		//
		// Verify tags with same name but different values were not updated
		//
		tags, err = models.ListStageTags("VERSION", "v2", []string{}, "", "")
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "VERSION", tags[0].TagName)
		assert.Equal(t, "v2", tags[0].TagValue)
		assert.Equal(t, models.TagStateUnknown, tags[0].TagState)

		//
		// Verify tags with same name and value were updated
		//
		tags, err = models.ListStageTags("VERSION", "v1", []string{}, "", "")
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "VERSION", tags[0].TagName)
		assert.Equal(t, "v1", tags[0].TagValue)
		assert.Equal(t, models.TagStateUnhealthy, tags[0].TagState)
	})
}
