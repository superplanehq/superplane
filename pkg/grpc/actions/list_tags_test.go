package actions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListTags(t *testing.T) {
	r := support.Setup(t)

	// Create another stage
	err := r.Canvas.CreateStage(
		"stage-2",
		r.User.String(),
		[]models.StageCondition{},
		support.RunTemplate(),
		[]models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		},
		support.TagUsageDef(r.Source.Name),
	)

	require.NoError(t, err)
	secondStage, err := r.Canvas.FindStageByName("stage-2")
	require.NoError(t, err)

	event1 := support.CreateStageEvent(t, r.Source, r.Stage)
	event2 := support.CreateStageEventWithData(t, r.Source, r.Stage, []byte(`{"ref":"v2"}`), []byte(`{"ref":"v2"}`))
	event3 := support.CreateStageEvent(t, r.Source, secondStage)

	// create some healthy tags
	require.NoError(t, models.UpdateStageEventTagStateInBulk(
		database.Conn(),
		event1.ID,
		models.TagStateHealthy,
		map[string]string{
			"VERSION": "v1",
			"SHA":     "1234",
		},
	))

	// create some unknown tags
	require.NoError(t, models.UpdateStageEventTagStateInBulk(
		database.Conn(),
		event2.ID,
		models.TagStateUnknown,
		map[string]string{
			"VERSION": "v2",
			"SHA":     "5678",
		},
	))

	// create some unhealthy tags
	require.NoError(t, models.UpdateStageEventTagStateInBulk(
		database.Conn(),
		event3.ID,
		models.TagStateUnhealthy,
		map[string]string{
			"VERSION": "v1",
			"SHA":     "1234",
		},
	))

	t.Run("invalid stage ID", func(t *testing.T) {
		_, err := ListTags(context.Background(), &protos.ListTagsRequest{
			StageId: "not-a-uuid",
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid stage ID")
	})

	t.Run("with name -> list", func(t *testing.T) {
		res, err := ListTags(context.Background(), &protos.ListTagsRequest{
			Name: "VERSION",
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 3)
	})

	t.Run("with name that does not exist -> empty list", func(t *testing.T) {
		res, err := ListTags(context.Background(), &protos.ListTagsRequest{
			Name: "does-not-exist",
		})

		require.NoError(t, err)
		require.Empty(t, res.Tags)
	})

	t.Run("with name and value -> list", func(t *testing.T) {
		// version=v1 is in event1 and event3, so we should get 2 tags
		res, err := ListTags(context.Background(), &protos.ListTagsRequest{
			Name:  "VERSION",
			Value: "v1",
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 2)

		// version=v2 is only in event2
		res, err = ListTags(context.Background(), &protos.ListTagsRequest{
			Name:  "VERSION",
			Value: "v2",
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 1)
	})

	t.Run("with inexistent name and value -> empty list", func(t *testing.T) {
		res, err := ListTags(context.Background(), &protos.ListTagsRequest{
			Name:  "VERSION",
			Value: "v3",
		})

		require.NoError(t, err)
		require.Empty(t, res.Tags)
	})

	t.Run("with states", func(t *testing.T) {
		// Just the healthy ones
		res, err := ListTags(context.Background(), &protos.ListTagsRequest{
			States: []protos.Tag_State{protos.Tag_TAG_STATE_HEALTHY},
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 2)

		// Just the unhealthy ones
		res, err = ListTags(context.Background(), &protos.ListTagsRequest{
			States: []protos.Tag_State{protos.Tag_TAG_STATE_UNHEALTHY},
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 2)

		// healthy and unhealthy ones
		res, err = ListTags(context.Background(), &protos.ListTagsRequest{
			States: []protos.Tag_State{protos.Tag_TAG_STATE_HEALTHY, protos.Tag_TAG_STATE_UNHEALTHY},
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 4)
	})

	t.Run("with stage ID", func(t *testing.T) {
		// first stage has 2 events with 2 tags each -> 4 tags
		res, err := ListTags(context.Background(), &protos.ListTagsRequest{
			StageId: r.Stage.ID.String(),
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 4)

		// second stage has 1 events with 2 tags each -> 2 tags
		res, err = ListTags(context.Background(), &protos.ListTagsRequest{
			StageId: secondStage.ID.String(),
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 2)
	})

	t.Run("with name, value, state and stage ID", func(t *testing.T) {
		// first stage only has one healthy tag version=v1
		res, err := ListTags(context.Background(), &protos.ListTagsRequest{
			StageId: r.Stage.ID.String(),
			Name:    "VERSION",
			Value:   "v1",
			States:  []protos.Tag_State{protos.Tag_TAG_STATE_HEALTHY},
		})

		require.NoError(t, err)
		require.Len(t, res.Tags, 1)
	})
}
