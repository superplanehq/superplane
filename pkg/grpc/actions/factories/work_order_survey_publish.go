package factories

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func PublishWorkOrderSurveyUpdated(organizationID, factoryID, orderID, surveyID uuid.UUID, email bool) {
	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factory.EventTypeOrderSurveyUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for survey %s", surveyID)
	}

	runneraction.NotifyWorkOrderSurvey(surveyID)

	if !email {
		return
	}

	notification := messages.FactoryWorkOrderNotificationMessage{
		OrganizationID:     organizationID.String(),
		FactoryID:          factoryID.String(),
		OrderID:            orderID.String(),
		EventType:          factory.EventTypeOrderSurveyUpdated,
		StatusNoteHeadline: "The agent needs an answer",
		StatusNoteBody:     "The run waits until you submit. If you do not answer in time, the agent continues without an answer.",
	}
	if err := notification.Publish(); err != nil {
		log.WithError(err).Warnf("Failed to publish work order survey notification for order %s", orderID)
	}
}
