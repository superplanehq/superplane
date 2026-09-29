package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

func TestListAnalysisPlanningSessionsForWorkOrders(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "plan-summary")
	db := database.DB(t.Context())

	workingOrder, err := factoryModel.CreateWorkOrder(db, "Working", "", &userID, nil, nil)
	require.NoError(t, err)
	questionOrder, err := factoryModel.CreateWorkOrder(db, "Question", "", &userID, nil, nil)
	require.NoError(t, err)
	plainOrder, err := factoryModel.CreateWorkOrder(db, "Plain", "", &userID, nil, nil)
	require.NoError(t, err)

	require.NoError(t, db.Create(analysisBoardSession(factoryModel, workingOrder.ID, PlanningSessionStateRunning, PlanningWaitIdle, nil)).Error)
	surveyID := uuid.New()
	question := analysisBoardSession(factoryModel, questionOrder.ID, PlanningSessionStateRunning, PlanningWaitPending, &surveyID)
	question.Survey = datatypes.NewJSONType(PlanningSessionSurvey{
		Questions: []PlanningSessionSurveyQuestion{{Prompt: "Which API?", Options: []string{"REST"}}},
	})
	require.NoError(t, db.Create(question).Error)

	sessions, err := ListAnalysisPlanningSessionsForWorkOrders(db, []uuid.UUID{workingOrder.ID, questionOrder.ID, plainOrder.ID})
	require.NoError(t, err)

	working := sessions[workingOrder.ID]
	require.NotNil(t, working)
	assert.Equal(t, PlanningSessionStateRunning, working.State)
	assert.Equal(t, PlanningWaitIdle, working.WaitState)
	assert.Empty(t, working.CurrentSurvey().Questions)

	asked := sessions[questionOrder.ID]
	require.NotNil(t, asked)
	assert.Equal(t, PlanningWaitPending, asked.WaitState)
	require.Len(t, asked.CurrentSurvey().Questions, 1)
	assert.Equal(t, "Which API?", asked.CurrentSurvey().Questions[0].Prompt)

	_, listed := sessions[plainOrder.ID]
	assert.False(t, listed)

	empty, err := ListAnalysisPlanningSessionsForWorkOrders(db, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func analysisBoardSession(
	factoryModel *Factory,
	workOrderID uuid.UUID,
	state string,
	waitState string,
	surveyID *uuid.UUID,
) *FactoryPlanningSession {
	now := time.Now()
	return &FactoryPlanningSession{
		ID:               uuid.New(),
		OrganizationID:   factoryModel.OrganizationID,
		FactoryID:        factoryModel.ID,
		Repository:       "acme/payments",
		Kind:             PlanningSessionKindWorkOrderAnalysis,
		State:            state,
		DraftWorkOrderID: &workOrderID,
		WaitState:        waitState,
		SurveyID:         surveyID,
		HeartbeatAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
