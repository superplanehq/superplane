package messages

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

var publishWorkOrderNotification = func(message FactoryWorkOrderNotificationMessage) error {
	return message.Publish()
}

func SetWorkOrderNotificationPublisherForTest(fn func(FactoryWorkOrderNotificationMessage) error) func() {
	previous := publishWorkOrderNotification
	publishWorkOrderNotification = fn
	return func() { publishWorkOrderNotification = previous }
}

func PublishPlanningAgentQuestion(session *models.FactoryPlanningSession) {
	message, ok := planningAgentQuestionMessage(session)
	if !ok {
		return
	}
	if err := publishWorkOrderNotification(message); err != nil {
		log.WithError(err).Warnf("Failed to publish agent question notification for session %s", session.ID)
	}
}

func PublishPlanningPlanReady(tx *gorm.DB, session *models.FactoryPlanningSession) {
	publishPlanningPlanReady(tx, session)
}

func HasPlanningReadyPlan(tx *gorm.DB, session *models.FactoryPlanningSession) bool {
	_, ok, err := planningPlanReadyMessage(tx, session)
	return err == nil && ok
}

func publishPlanningPlanReady(tx *gorm.DB, session *models.FactoryPlanningSession) {
	message, ok, err := planningPlanReadyMessage(tx, session)
	if err != nil {
		log.WithError(err).Warnf("Failed to build plan-ready notification for session %s", session.ID)
		return
	}
	if !ok {
		return
	}
	if err := publishWorkOrderNotification(message); err != nil {
		log.WithError(err).Warnf("Failed to publish plan-ready notification for session %s", session.ID)
	}
}

func planningAgentQuestionMessage(session *models.FactoryPlanningSession) (FactoryWorkOrderNotificationMessage, bool) {
	orderID, ok := planningSessionTaskID(session)
	if !ok {
		return FactoryWorkOrderNotificationMessage{}, false
	}
	return planningSessionNotificationMessage(session, orderID, factory.EventTypeOrderAgentQuestion, planningAgentQuestionDetail(session)), true
}

func planningPlanReadyMessage(tx *gorm.DB, session *models.FactoryPlanningSession) (FactoryWorkOrderNotificationMessage, bool, error) {
	orderID, ok := planningSessionTaskID(session)
	if !ok {
		return FactoryWorkOrderNotificationMessage{}, false, nil
	}
	factoryModel, err := models.FindFactory(tx, session.OrganizationID, session.FactoryID)
	if err != nil {
		return FactoryWorkOrderNotificationMessage{}, false, err
	}
	order, err := factoryModel.FindWorkOrder(tx, orderID)
	if err != nil {
		return FactoryWorkOrderNotificationMessage{}, false, err
	}
	if _, err := order.FindArtifactByKey(tx, planningSpecArtifactKey(orderID)); err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderArtifactNotFound) {
			return FactoryWorkOrderNotificationMessage{}, false, nil
		}
		return FactoryWorkOrderNotificationMessage{}, false, err
	}
	return planningSessionNotificationMessage(session, orderID, factory.EventTypeOrderPlanReady, ""), true, nil
}

func planningSessionNotificationMessage(
	session *models.FactoryPlanningSession,
	orderID uuid.UUID,
	eventType string,
	questionPrompt string,
) FactoryWorkOrderNotificationMessage {
	message := FactoryWorkOrderNotificationMessage{
		OrganizationID: session.OrganizationID.String(),
		FactoryID:      session.FactoryID.String(),
		OrderID:        orderID.String(),
		EventType:      eventType,
		QuestionPrompt: questionPrompt,
	}
	if session.CreatedByUserID != nil {
		message.SessionStarterUserID = session.CreatedByUserID.String()
	}
	return message
}

func planningSessionTaskID(session *models.FactoryPlanningSession) (uuid.UUID, bool) {
	if session.DraftWorkOrderID != nil && *session.DraftWorkOrderID != uuid.Nil {
		return *session.DraftWorkOrderID, true
	}
	if session.WaitWorkOrderID != nil && *session.WaitWorkOrderID != uuid.Nil {
		return *session.WaitWorkOrderID, true
	}
	return uuid.Nil, false
}

func planningAgentQuestionDetail(session *models.FactoryPlanningSession) string {
	survey := session.CurrentSurvey()
	if len(survey.Questions) > 0 {
		return strings.TrimSpace(survey.Questions[0].Prompt)
	}
	return strings.TrimSpace(session.WaitText)
}

func planningSpecArtifactKey(orderID uuid.UUID) string {
	return models.PlanningSpecArtifactKey + ":" + orderID.String()
}
