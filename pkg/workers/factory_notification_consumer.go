package workers

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/services"
	"gorm.io/gorm"
)

const FactoryNotificationServiceName = "superplane" + "." + messages.CanvasExchange + "." + messages.FactoryWorkOrderNotificationRoutingKey + ".worker-consumer"
const FactoryNotificationConnectionName = "superplane"

const workOrderNotificationDetailMaxRunes = 280

// FactoryNotificationConsumer turns work order activity messages into
// notification emails, honoring each recipient's notification settings.
type FactoryNotificationConsumer struct {
	Consumer     *tackle.Consumer
	RabbitMQURL  string
	EmailService services.EmailService
	BaseURL      string
}

func NewFactoryNotificationConsumer(rabbitMQURL string, emailService services.EmailService, baseURL string) *FactoryNotificationConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "factory_notification",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &FactoryNotificationConsumer{
		RabbitMQURL:  rabbitMQURL,
		Consumer:     consumer,
		EmailService: emailService,
		BaseURL:      baseURL,
	}
}

func (c *FactoryNotificationConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: FactoryNotificationConnectionName,
		Service:        FactoryNotificationServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.FactoryWorkOrderNotificationRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.FactoryWorkOrderNotificationRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.FactoryWorkOrderNotificationRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.FactoryWorkOrderNotificationRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *FactoryNotificationConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *FactoryNotificationConsumer) Consume(delivery tackle.Delivery) error {
	start := time.Now()
	outcome := executorOutcomeSuccess
	reason := executorReasonNone
	defer func() {
		recordEmailWorkerProcessing(start, emailTypeWorkOrderNotification, outcome, reason)
	}()

	var message messages.FactoryWorkOrderNotificationMessage
	if err := json.Unmarshal(delivery.Body(), &message); err != nil {
		log.Errorf("Error unmarshaling work order notification message: %v", err)
		outcome = executorOutcomeFailed
		reason = emailWorkerReasonInvalidMessage
		return err
	}

	sent, err := c.process(database.Conn(), message)
	if err != nil {
		outcome = executorOutcomeFailed
		reason = emailWorkerReasonSendError
		return err
	}
	if !sent {
		outcome = executorOutcomeSkipped
	}

	return nil
}

// process resolves recipients and sends the emails. The bool result
// reports whether at least one email was sent. Send failures for
// individual recipients are logged and do not abort the batch — retrying
// the whole message would duplicate the emails that already went out.
func (c *FactoryNotificationConsumer) process(db *gorm.DB, message messages.FactoryWorkOrderNotificationMessage) (bool, error) {
	orgID, err := uuid.Parse(message.OrganizationID)
	if err != nil {
		log.Warnf("Skipping work order notification with invalid organization id %q", message.OrganizationID)
		return false, nil
	}

	factoryID, err := uuid.Parse(message.FactoryID)
	if err != nil {
		log.Warnf("Skipping work order notification with invalid factory id %q", message.FactoryID)
		return false, nil
	}

	orderID, err := uuid.Parse(message.OrderID)
	if err != nil {
		log.Warnf("Skipping work order notification with invalid order id %q", message.OrderID)
		return false, nil
	}

	factoryModel, err := models.FindFactory(db, orgID, factoryID)
	if errors.Is(err, models.ErrFactoryNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	order, err := factoryModel.FindWorkOrder(db, orderID)
	if errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	recipients, err := c.resolveRecipients(db, orgID, factoryID, order, message)
	if err != nil {
		return false, err
	}
	if len(recipients) == 0 {
		return false, nil
	}

	content := buildWorkOrderNotificationContent(factoryModel, order, message, c.actorDisplayName(db, orgID, message))
	applyWorkOrderEmailCard(&content.Data, order, loadWorkOrderExecutionsForEmail(db, order.ID), time.Now())
	content.Data.WorkOrderLink = fmt.Sprintf(
		"%s/%s/workspaces/%s/work-order/%d",
		c.BaseURL, orgID, factoryModel.Key, order.Number,
	)

	sent := false
	for _, recipient := range recipients {
		if err := c.EmailService.SendWorkOrderNotificationEmail(recipient, content.Subject, content.Data); err != nil {
			log.Errorf("Failed to send work order notification email to %s: %v", recipient, err)
			continue
		}
		sent = true
	}

	return sent, nil
}

// resolveRecipients returns the email addresses that should receive this
// notification: candidates by event type, minus the actor, filtered by
// each user's notification settings.
func (c *FactoryNotificationConsumer) resolveRecipients(
	db *gorm.DB,
	orgID, factoryID uuid.UUID,
	order *models.FactoryWorkOrder,
	message messages.FactoryWorkOrderNotificationMessage,
) ([]string, error) {
	candidates := workOrderNotificationCandidates(order, message)
	delete(candidates, uuid.Nil)
	if actorID, err := uuid.Parse(message.ActorUserID); err == nil {
		delete(candidates, actorID)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	userIDs := make([]uuid.UUID, 0, len(candidates))
	for userID := range candidates {
		userIDs = append(userIDs, userID)
	}

	settingsByUserID, err := models.FindUserNotificationSettingsForUsers(db, orgID, userIDs)
	if err != nil {
		return nil, err
	}

	allowedIDs := make([]string, 0, len(candidates))
	for userID, notificationType := range candidates {
		settings, ok := settingsByUserID[userID]
		if !ok {
			settings = models.DefaultUserNotificationSettings()
		}
		if !settings.NotifiesType(notificationType) {
			continue
		}
		if !settings.AppliesToFactory(factoryID) {
			continue
		}
		allowedIDs = append(allowedIDs, userID.String())
	}
	if len(allowedIDs) == 0 {
		return nil, nil
	}

	users, err := models.FindUsersByIDsInOrganization(db, orgID.String(), allowedIDs)
	if err != nil {
		return nil, err
	}

	emails := make([]string, 0, len(users))
	for i := range users {
		if email := users[i].GetEmail(); email != "" {
			emails = append(emails, email)
		}
	}

	return emails, nil
}

// workOrderNotificationCandidates maps candidate recipients to the
// notification type that covers them for this event. When a user matches
// several types (owner and creator), the owner-facing type wins so each
// user gets at most one email per event.
func workOrderNotificationCandidates(
	order *models.FactoryWorkOrder,
	message messages.FactoryWorkOrderNotificationMessage,
) map[uuid.UUID]string {
	candidates := map[uuid.UUID]string{}

	addOwners := func(notificationType string) {
		for _, assignee := range order.Assignees {
			candidates[assignee.UserID] = notificationType
		}
	}
	addCreator := func(notificationType string) {
		if order.CreatedByID == nil {
			return
		}
		if _, alreadyCovered := candidates[*order.CreatedByID]; alreadyCovered {
			return
		}
		candidates[*order.CreatedByID] = notificationType
	}

	switch message.EventType {
	case factory.EventTypeOrderAssigneesUpdated:
		for _, assignedID := range message.AssignedUserIDs {
			if userID, err := uuid.Parse(assignedID); err == nil {
				candidates[userID] = models.NotificationTypeWorkOrderAssigned
			}
		}
	case factory.EventTypeOrderCommentAdded:
		addOwners(models.NotificationTypeWorkOrderCommentOwned)
		addCreator(models.NotificationTypeWorkOrderCommentCreated)
	case factory.EventTypeOrderStatusUpdated:
		// The initial `"" → draft` transition is work order creation,
		// not a change anyone needs an email about.
		if message.FromState == "" {
			return candidates
		}
		addOwners(models.NotificationTypeWorkOrderStatusOwned)
		addCreator(models.NotificationTypeWorkOrderStatusOwned)
	case factory.EventTypeOrderArtifactAdded:
		addOwners(models.NotificationTypeWorkOrderArtifactOwned)
	}

	return candidates
}

func (c *FactoryNotificationConsumer) actorDisplayName(
	db *gorm.DB,
	orgID uuid.UUID,
	message messages.FactoryWorkOrderNotificationMessage,
) string {
	if message.ActorUserID != "" {
		users, err := models.FindUsersByIDsInOrganization(db, orgID.String(), []string{message.ActorUserID})
		if err == nil && len(users) > 0 && users[0].Name != "" {
			return users[0].Name
		}
		return "A team member"
	}

	if message.ActorName != "" {
		return message.ActorName
	}
	return "An automation"
}

type workOrderNotificationContent struct {
	Subject string
	Data    services.WorkOrderNotificationTemplateData
}

func buildWorkOrderNotificationContent(
	factoryModel *models.Factory,
	order *models.FactoryWorkOrder,
	message messages.FactoryWorkOrderNotificationMessage,
	actorName string,
) workOrderNotificationContent {
	orderKey := factoryModel.WorkOrderKey(order.Number)

	content := workOrderNotificationContent{
		Data: services.WorkOrderNotificationTemplateData{
			WorkOrderKey:   orderKey,
			WorkOrderTitle: order.Title,
		},
	}

	switch message.EventType {
	case factory.EventTypeOrderAssigneesUpdated:
		content.Subject = fmt.Sprintf("[%s] You are now an owner", orderKey)
		content.Data.Summary = fmt.Sprintf("%s made you an owner of %s.", actorName, orderKey)
	case factory.EventTypeOrderCommentAdded:
		content.Subject = fmt.Sprintf("[%s] New comment from %s", orderKey, actorName)
		content.Data.Summary = fmt.Sprintf("%s commented on %s.", actorName, orderKey)
		content.Data.Detail = truncateNotificationDetail(message.CommentBody)
	case factory.EventTypeOrderStatusUpdated:
		verb := statusChangeDescription(message)
		content.Subject = fmt.Sprintf("[%s] Work order %s", orderKey, verb)
		content.Data.Summary = fmt.Sprintf("%s %s %s.", actorName, verb, orderKey)
	case factory.EventTypeOrderArtifactAdded:
		content.Subject = fmt.Sprintf("[%s] New artifact", orderKey)
		content.Data.Summary = fmt.Sprintf("%s added a %s artifact to %s.", actorName, artifactTypeLabel(message.ArtifactType), orderKey)
	default:
		content.Subject = fmt.Sprintf("[%s] Work order update", orderKey)
		content.Data.Summary = fmt.Sprintf("%s updated %s.", actorName, orderKey)
	}

	return content
}

// statusChangeDescription phrases a state transition so it reads naturally
// after the actor name ("Ana <description> SP-42").
func statusChangeDescription(message messages.FactoryWorkOrderNotificationMessage) string {
	switch message.ToState {
	case models.FactoryWorkOrderStateOpen:
		if message.FromState == models.FactoryWorkOrderStateClosed {
			return "reopened"
		}
		return "opened"
	case models.FactoryWorkOrderStateDraft:
		return "moved back to draft"
	case models.FactoryWorkOrderStateClosed:
		if message.Result != "" {
			return fmt.Sprintf("closed as %s", message.Result)
		}
		return "closed"
	default:
		return "updated"
	}
}

func artifactTypeLabel(artifactType string) string {
	switch artifactType {
	case factory.ArtifactTypePR:
		return "pull request"
	case factory.ArtifactTypeMarkdown:
		return "markdown"
	case factory.ArtifactTypeBranch:
		return "branch"
	default:
		return "new"
	}
}

func truncateNotificationDetail(detail string) string {
	if utf8.RuneCountInString(detail) <= workOrderNotificationDetailMaxRunes {
		return detail
	}

	runes := []rune(detail)
	return string(runes[:workOrderNotificationDetailMaxRunes]) + "…"
}
