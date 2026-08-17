package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListWorkOrderEvents_CommentReactions(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	userIDStr := r.User.String()
	commentEvent, err := order.RecordCommentAdded(database.DB(t.Context()), "Nice work", factory.WorkOrderCommentAuthor{
		Kind:   factory.CommentAuthorKindUser,
		UserID: &userIDStr,
	}, nil)
	require.NoError(t, err)

	// Every event, comment or not, exposes its stable id.
	resp, err := ListWorkOrderEvents(ctx, r.Organization.ID.String(), &pb.ListWorkOrderEventsRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Events)
	for _, e := range resp.Events {
		assert.NotEmpty(t, e.Id)
	}

	commentEventProto := findEventByID(t, resp.Events, commentEvent.ID.String())
	assert.Equal(t, factory.EventTypeOrderCommentAdded, commentEventProto.Type)
	reactions, ok := commentEventProto.Event.AsMap()["reactions"].([]any)
	require.True(t, ok)
	assert.Empty(t, reactions, "a fresh comment has no reactions, but the field must still be present")

	_, err = AddWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		CommentId: commentEvent.ID.String(),
		Emoji:     "rocket",
	})
	require.NoError(t, err)

	resp, err = ListWorkOrderEvents(ctx, r.Organization.ID.String(), &pb.ListWorkOrderEventsRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
	})
	require.NoError(t, err)

	commentEventProto = findEventByID(t, resp.Events, commentEvent.ID.String())
	reactionsMap := commentEventProto.Event.AsMap()["reactions"].([]any)
	require.Len(t, reactionsMap, 1)
	reaction := reactionsMap[0].(map[string]any)
	assert.Equal(t, "rocket", reaction["emoji"])
	assert.EqualValues(t, 1, reaction["count"])
	assert.Equal(t, true, reaction["reactedByMe"])
}

func findEventByID(t *testing.T, events []*pb.WorkOrderEvent, id string) *pb.WorkOrderEvent {
	t.Helper()
	for _, e := range events {
		if e.Id == id {
			return e
		}
	}
	t.Fatalf("event %s not found", id)
	return nil
}
