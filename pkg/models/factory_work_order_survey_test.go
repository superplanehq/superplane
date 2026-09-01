package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestFactoryWorkOrder_CreateSurvey(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "create-survey")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	runID := uuid.New()
	survey, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID:    runID,
		TimeoutSeconds: 3600,
		Questions: []WorkOrderSurveyQuestion{
			{ID: "scope", Prompt: "Where?", Options: []string{"A", "B"}, AllowFreeText: true},
			{ID: "tests", Prompt: "Add a test?"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, survey)
	assert.Equal(t, FactoryWorkOrderSurveyPending, survey.Status)
	assert.Equal(t, order.ID, survey.WorkOrderID)
	assert.Equal(t, runID, survey.CanvasRunID)
	assert.Equal(t, 3600, survey.TimeoutSeconds)
	require.Len(t, survey.Questions, 2)
	assert.Equal(t, "scope", survey.Questions[0].ID)
	assert.WithinDuration(t, time.Now().Add(time.Hour), survey.ExpiresAt, 5*time.Second)

	t.Run("returns the same pending survey for the same run", func(t *testing.T) {
		again, created, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
			CanvasRunID: runID,
			Questions:   []WorkOrderSurveyQuestion{{ID: "other", Prompt: "Ignored"}},
		})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Equal(t, survey.ID, again.ID)
		assert.Equal(t, "scope", again.Questions[0].ID)
	})

	t.Run("rejects a second pending survey from another run", func(t *testing.T) {
		_, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
			CanvasRunID: uuid.New(),
			Questions:   []WorkOrderSurveyQuestion{{ID: "x", Prompt: "Nope"}},
		})
		assert.ErrorIs(t, err, ErrFactoryWorkOrderSurveyConflict)
	})
}

func TestFactoryWorkOrder_CreateSurvey_ValidatesQuestions(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "survey-validate")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	_, _, err = order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions:   nil,
	})
	assert.ErrorIs(t, err, ErrFactoryWorkOrderSurveyInvalid)
}

func TestFactoryWorkOrderSurvey_Answer(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "survey-answer")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	survey, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions: []WorkOrderSurveyQuestion{
			{ID: "scope", Prompt: "Where?"},
			{ID: "tests", Prompt: "Add a test?"},
		},
	})
	require.NoError(t, err)

	require.NoError(t, survey.Answer(db, userID, []WorkOrderSurveyAnswer{
		{ID: "scope", Value: "Existing handler"},
	}))
	assert.Equal(t, FactoryWorkOrderSurveyAnswered, survey.Status)
	require.Len(t, survey.Answers, 2)
	assert.Equal(t, "Existing handler", survey.Answers[0].Value)
	assert.Equal(t, "skipped", survey.Answers[1].Value)
	require.NotNil(t, survey.AnsweredByUserID)
	assert.Equal(t, userID, *survey.AnsweredByUserID)

	pending, err := order.PendingSurvey(db)
	require.ErrorIs(t, err, ErrFactoryWorkOrderSurveyNotFound)
	assert.Nil(t, pending)
}

func TestFactoryWorkOrderSurvey_AnswerRejectsUnknownQuestion(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "survey-unknown")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	survey, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions:   []WorkOrderSurveyQuestion{{ID: "scope", Prompt: "Where?"}},
	})
	require.NoError(t, err)

	err = survey.Answer(db, userID, []WorkOrderSurveyAnswer{{ID: "nope", Value: "x"}})
	assert.ErrorIs(t, err, ErrFactoryWorkOrderSurveyInvalid)
	assert.Equal(t, FactoryWorkOrderSurveyPending, survey.Status)
}

func TestFactoryWorkOrderSurvey_TimeoutAndCancel(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "survey-timeout")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	survey, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions:   []WorkOrderSurveyQuestion{{ID: "scope", Prompt: "Where?"}},
	})
	require.NoError(t, err)

	require.NoError(t, survey.MarkTimedOut(db))
	assert.Equal(t, FactoryWorkOrderSurveyTimedOut, survey.Status)

	err = survey.Answer(db, userID, []WorkOrderSurveyAnswer{{ID: "scope", Value: "A"}})
	assert.ErrorIs(t, err, ErrFactoryWorkOrderSurveyNotPending)

	other, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions:   []WorkOrderSurveyQuestion{{ID: "next", Prompt: "Next?"}},
	})
	require.NoError(t, err)
	require.NoError(t, other.Cancel(db))
	assert.Equal(t, FactoryWorkOrderSurveyCancelled, other.Status)
}

func TestFactoryWorkOrderSurvey_ExpireIfDue(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "expire-survey")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	survey, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID:    uuid.New(),
		TimeoutSeconds: 60,
		Questions:      []WorkOrderSurveyQuestion{{ID: "scope", Prompt: "Where?"}},
	})
	require.NoError(t, err)

	require.NoError(t, survey.ExpireIfDue(db, survey.ExpiresAt.Add(-time.Second)))
	assert.Equal(t, FactoryWorkOrderSurveyPending, survey.Status)

	require.NoError(t, survey.ExpireIfDue(db, survey.ExpiresAt))
	assert.Equal(t, FactoryWorkOrderSurveyTimedOut, survey.Status)
}

func TestFactoryWorkOrder_CloseCancelsPendingSurvey(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "close-survey")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	survey, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions:   []WorkOrderSurveyQuestion{{ID: "scope", Prompt: "Where?"}},
	})
	require.NoError(t, err)

	_, err = order.UpdateStatus(db, FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateClosed,
		Result:  FactoryWorkOrderResultRejected,
	})
	require.NoError(t, err)

	reloaded, err := FindWorkOrderSurvey(db, order.OrganizationID, survey.ID)
	require.NoError(t, err)
	assert.Equal(t, FactoryWorkOrderSurveyCancelled, reloaded.Status)
}

func TestFactoryWorkOrder_CancelPendingSurvey(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "cancel-pending")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	require.NoError(t, order.CancelPendingSurvey(db))

	survey, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID: uuid.New(),
		Questions:   []WorkOrderSurveyQuestion{{ID: "scope", Prompt: "Where?"}},
	})
	require.NoError(t, err)
	require.NoError(t, order.CancelPendingSurvey(db))
	reloaded, err := FindWorkOrderSurvey(db, order.OrganizationID, survey.ID)
	require.NoError(t, err)
	assert.Equal(t, FactoryWorkOrderSurveyCancelled, reloaded.Status)
}

func TestCancelPendingWorkOrderSurveysForRun(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "cancel-run")
	db := database.Conn()
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	runID := uuid.New()
	survey, _, err := order.CreateSurvey(db, FactoryWorkOrderSurveyParams{
		CanvasRunID: runID,
		Questions:   []WorkOrderSurveyQuestion{{ID: "scope", Prompt: "Where?"}},
	})
	require.NoError(t, err)

	require.NoError(t, CancelPendingWorkOrderSurveysForRun(db, runID))
	reloaded, err := FindWorkOrderSurvey(db, order.OrganizationID, survey.ID)
	require.NoError(t, err)
	assert.Equal(t, FactoryWorkOrderSurveyCancelled, reloaded.Status)
}
