package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func TestSerializeWorkOrderCreator_UserBranch(t *testing.T) {
	userID := uuid.New()
	order := &models.FactoryWorkOrder{
		ID:          uuid.New(),
		CreatedByID: &userID,
		CreatedBy:   &models.User{Name: "Alice"},
	}

	creator := serializeWorkOrder(nil, order, nil, nil, nil, uuid.Nil).GetCreatedBy()
	require.NotNil(t, creator)
	assert.Nil(t, creator.GetAutomation())
	require.NotNil(t, creator.GetUser())
	assert.Equal(t, userID.String(), creator.GetUser().GetId())
	assert.Equal(t, "Alice", creator.GetUser().GetName())
}

func TestSerializeWorkOrderCreator_AutomationBranchWinsOverUser(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()
	order := &models.FactoryWorkOrder{
		ID:          uuid.New(),
		CreatedByID: &userID,
		CreatedBy:   &models.User{Name: "Alice"},
	}
	automation := &factory.AutomationRef{
		NodeID:   "node-1",
		NodeName: "Release gate",
		AppID:    appID,
		AppName:  "Release automation",
	}

	creator := serializeWorkOrder(nil, order, nil, automation, nil, uuid.Nil).GetCreatedBy()
	require.NotNil(t, creator)
	assert.Nil(t, creator.GetUser())
	require.NotNil(t, creator.GetAutomation())
	assert.Equal(t, "node-1", creator.GetAutomation().GetNodeId())
	assert.Equal(t, "Release gate", creator.GetAutomation().GetNodeName())
	assert.Equal(t, appID.String(), creator.GetAutomation().GetAppId())
}

func TestSerializeWorkOrderCreator_NoneReturnsNil(t *testing.T) {
	order := &models.FactoryWorkOrder{ID: uuid.New()}
	assert.Nil(t, serializeWorkOrder(nil, order, nil, nil, nil, uuid.Nil).GetCreatedBy())
}

func TestSerializeExecutionSteps_SnapshotsLineSteps(t *testing.T) {
	steps := []models.FactoryLineStep{
		{Name: "plan"},
		{Name: "implement"},
		{Name: "verify"},
	}

	out := serializeExecutionSteps(steps)
	require.Len(t, out, 3)
	assert.Equal(t, "plan", out[0].GetName())
	assert.EqualValues(t, 0, out[0].GetStepIndex())
	assert.Equal(t, "implement", out[1].GetName())
	assert.EqualValues(t, 1, out[1].GetStepIndex())
	assert.Equal(t, "verify", out[2].GetName())
	assert.EqualValues(t, 2, out[2].GetStepIndex())
}

func TestSerializeExecutionSteps_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, serializeExecutionSteps(nil))
	assert.Nil(t, serializeExecutionSteps([]models.FactoryLineStep{}))
}

func TestSerializeWorkOrderReactions_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, serializeWorkOrderReactions(nil, uuid.New()))
	assert.Nil(t, serializeWorkOrderReactions([]models.FactoryWorkOrderReaction{}, uuid.New()))
}

func TestSerializeWorkOrderReactions_GroupsAndCounts(t *testing.T) {
	me := uuid.New()
	other := uuid.New()

	reactions := []models.FactoryWorkOrderReaction{
		{UserID: me, Content: models.ReactionThumbsUp},
		{UserID: other, Content: models.ReactionThumbsUp},
		{UserID: other, Content: models.ReactionHeart},
	}

	result := serializeWorkOrderReactions(reactions, me)
	require.Len(t, result, 2)

	thumbsUp := result[0]
	assert.Equal(t, models.ReactionThumbsUp, thumbsUp.GetContent())
	assert.EqualValues(t, 2, thumbsUp.GetCount())
	assert.True(t, thumbsUp.GetReactedByMe(), "the calling user reacted with +1")

	heart := result[1]
	assert.Equal(t, models.ReactionHeart, heart.GetContent())
	assert.EqualValues(t, 1, heart.GetCount())
	assert.False(t, heart.GetReactedByMe(), "only the other user reacted with heart")
}

func TestSerializeWorkOrderReactions_NoCurrentUser(t *testing.T) {
	reactions := []models.FactoryWorkOrderReaction{
		{UserID: uuid.New(), Content: models.ReactionRocket},
	}

	result := serializeWorkOrderReactions(reactions, uuid.Nil)
	require.Len(t, result, 1)
	assert.False(t, result[0].GetReactedByMe())
}
