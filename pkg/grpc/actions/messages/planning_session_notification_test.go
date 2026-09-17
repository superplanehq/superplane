package messages

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
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
