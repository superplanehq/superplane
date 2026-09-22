package messages

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
)

func TestPlanningAgentQuestionMessage_SkipsSessionWithoutTask(t *testing.T) {
	session := &models.FactoryPlanningSession{
		OrganizationID: uuid.New(),
		FactoryID:      uuid.New(),
	}

	_, ok := planningAgentQuestionMessage(session)
	assert.False(t, ok)
}

func TestPublishPlanningAgentQuestion_SkipsSessionWithoutTask(t *testing.T) {
	published := []FactoryWorkOrderNotificationMessage{}
	restore := SetWorkOrderNotificationPublisherForTest(func(message FactoryWorkOrderNotificationMessage) error {
		published = append(published, message)
		return nil
	})
	defer restore()

	PublishPlanningAgentQuestion(&models.FactoryPlanningSession{})
	assert.Empty(t, published)
}

func TestPublishPlanningPlanReady_SkipsSessionWithoutTask(t *testing.T) {
	published := []FactoryWorkOrderNotificationMessage{}
	restore := SetWorkOrderNotificationPublisherForTest(func(message FactoryWorkOrderNotificationMessage) error {
		published = append(published, message)
		return nil
	})
	defer restore()

	PublishPlanningPlanReady(nil, &models.FactoryPlanningSession{})
	assert.Empty(t, published)
	assert.False(t, HasPlanningReadyPlan(nil, &models.FactoryPlanningSession{}))
}

func TestPublishPlanningBoardStatus_UsesAgentQuestionOnlyWhenWaiting(t *testing.T) {
	var reasons []string
	restore := SetPlanningBoardPublisherForTest(func(_, _, reason string) error {
		reasons = append(reasons, reason)
		return nil
	})
	defer restore()

	session := analysisSessionWithSurvey(models.PlanningWaitIdle)
	PublishPlanningBoardStatus(session)
	assert.Equal(t, []string{factoryevents.EventTypeOrderUpdated}, reasons)

	session.WaitState = models.PlanningWaitPending
	PublishPlanningBoardStatus(session)
	assert.Equal(t, []string{
		factoryevents.EventTypeOrderUpdated,
		factoryevents.EventTypeOrderAgentQuestion,
	}, reasons)
}

func analysisSessionWithSurvey(waitState string) *models.FactoryPlanningSession {
	orderID := uuid.New()
	surveyID := uuid.New()
	return &models.FactoryPlanningSession{
		ID:               uuid.New(),
		FactoryID:        uuid.New(),
		Kind:             models.PlanningSessionKindWorkOrderAnalysis,
		DraftWorkOrderID: &orderID,
		WaitState:        waitState,
		SurveyID:         &surveyID,
		Survey: datatypes.NewJSONType(models.PlanningSessionSurvey{
			Questions: []models.PlanningSessionSurveyQuestion{
				{Prompt: "Which API?", Options: []string{"REST"}},
			},
		}),
	}
}
