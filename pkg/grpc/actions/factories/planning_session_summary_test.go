package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func TestSerializePlanningSessionSummary_KeepsBoardFields(t *testing.T) {
	surveyID := uuid.New()
	session := &models.FactoryPlanningSession{
		ID:        uuid.New(),
		State:     models.PlanningSessionStateRunning,
		WaitState: models.PlanningWaitPending,
		SurveyID:  &surveyID,
		Survey: datatypes.NewJSONType(models.PlanningSessionSurvey{
			Questions: []models.PlanningSessionSurveyQuestion{{Prompt: "Which API?", Options: []string{"REST"}}},
		}),
	}

	summary := serializePlanningSessionSummary(session, "exec-1")
	require.NotNil(t, summary)
	assert.Equal(t, session.ID.String(), summary.Id)
	assert.Equal(t, models.PlanningSessionStateRunning, summary.State)
	assert.Equal(t, models.PlanningWaitPending, summary.WaitState)
	assert.Equal(t, "exec-1", summary.ExecutionId)
	require.NotNil(t, summary.Survey)
	require.Len(t, summary.Survey.Questions, 1)
	assert.Equal(t, "Which API?", summary.Survey.Questions[0].Prompt)
	assert.Equal(t, []string{"REST"}, summary.Survey.Questions[0].Options)
}

func TestSerializePlanningSessionSummary_OmitsEmptySurvey(t *testing.T) {
	session := &models.FactoryPlanningSession{
		ID:    uuid.New(),
		State: models.PlanningSessionStateRunning,
	}

	summary := serializePlanningSessionSummary(session, "")
	require.NotNil(t, summary)
	assert.Nil(t, summary.Survey)
	assert.Empty(t, summary.ExecutionId)
}
