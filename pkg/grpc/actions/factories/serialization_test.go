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

	creator := serializeWorkOrder(nil, order, nil, nil, nil).GetCreatedBy()
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

	creator := serializeWorkOrder(nil, order, nil, nil, automation).GetCreatedBy()
	require.NotNil(t, creator)
	assert.Nil(t, creator.GetUser())
	require.NotNil(t, creator.GetAutomation())
	assert.Equal(t, "node-1", creator.GetAutomation().GetNodeId())
	assert.Equal(t, "Release gate", creator.GetAutomation().GetNodeName())
	assert.Equal(t, appID.String(), creator.GetAutomation().GetAppId())
}

func TestSerializeWorkOrderCreator_NoneReturnsNil(t *testing.T) {
	order := &models.FactoryWorkOrder{ID: uuid.New()}
	assert.Nil(t, serializeWorkOrder(nil, order, nil, nil, nil).GetCreatedBy())
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
