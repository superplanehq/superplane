package workers

import (
	"context"
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

func (c *FactoryNotificationConsumer) Start(ctx context.Context) error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: FactoryNotificationConnectionName,
		Service:        FactoryNotificationServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.FactoryWorkOrderNotificationRoutingKey,
	}

	for {
		if ctx.Err() != nil {
			return nil
		}

		log.Infof("Connecting to RabbitMQ queue for %s events", messages.FactoryWorkOrderNotificationRoutingKey)

		if err := c.Consumer.Start(&options, c.Consume); err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.FactoryWorkOrderNotificationRoutingKey, err)
		} else {
			log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.FactoryWorkOrderNotificationRoutingKey)
		}

		//
		// Wait before reconnecting, but give up as soon as the process is shutting
		// down, so a cancelled consumer does not reconnect and take new messages.
		//
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(reconnectDelay):
		}
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

	actorName := c.actorDisplayName(db, orgID, message)
	executions := loadWorkOrderExecutionsForEmail(db, order.ID)
	return c.sendWorkOrderNotificationEmails(factoryModel, order, message, actorName, executions, recipients), nil
}

type workOrderEmailRecipient struct {
	email            string
	notificationType string
}

func (c *FactoryNotificationConsumer) sendWorkOrderNotificationEmails(
	factoryModel *models.Factory,
	order *models.FactoryWorkOrder,
	message messages.FactoryWorkOrderNotificationMessage,
	actorName string,
	executions []workOrderEmailExecution,
	recipients []workOrderEmailRecipient,
) bool {
	contentByType := map[string]workOrderNotificationContent{}
	sent := false
	for _, recipient := range recipients {
		content, ok := contentByType[recipient.notificationType]
		if !ok {
			content = buildWorkOrderNotificationContent(
				factoryModel,
				order,
				message,
				actorName,
				recipient.notificationType,
			)
			applyWorkOrderEmailCard(&content.Data, order, executions, time.Now())
			content.Data.WorkOrderLink = c.BaseURL + order.URLPath(factoryModel.Key)
			contentByType[recipient.notificationType] = content
		}
		if err := c.EmailService.SendWorkOrderNotificationEmail(recipient.email, content.Subject, content.Data); err != nil {
			log.Errorf("Failed to send work order notification email to %s: %v", recipient.email, err)
			continue
		}
		sent = true
	}

	return sent
}

// resolveRecipients returns the email addresses that should receive this
// notification: candidates by event type, minus the actor, filtered by
// each user's notification settings.
func (c *FactoryNotificationConsumer) resolveRecipients(
	db *gorm.DB,
	orgID, factoryID uuid.UUID,
	order *models.FactoryWorkOrder,
	message messages.FactoryWorkOrderNotificationMessage,
) ([]workOrderEmailRecipient, error) {
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
	allowedTypes := map[string]string{}
	for userID, notificationType := range candidates {
		settings, ok := settingsByUserID[userID]
		if !ok {
			settings = models.DefaultUserNotificationSettings()
		}
		if !settings.Notifies(factoryID, notificationType) {
			continue
		}
		id := userID.String()
		allowedIDs = append(allowedIDs, id)
		allowedTypes[id] = notificationType
	}
	if len(allowedIDs) == 0 {
		return nil, nil
	}

	users, err := models.FindUsersByIDsInOrganization(db, orgID.String(), allowedIDs)
	if err != nil {
		return nil, err
	}

	recipients := make([]workOrderEmailRecipient, 0, len(users))
	for i := range users {
		if users[i].DeletedAt.Valid {
			continue
		}
		email := users[i].GetEmail()
		if email == "" {
			continue
		}
		recipients = append(recipients, workOrderEmailRecipient{
			email:            email,
			notificationType: allowedTypes[users[i].ID.String()],
		})
	}

	return recipients, nil
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
	add := func(userID uuid.UUID, notificationType string) {
		if userID == uuid.Nil {
			return
		}
		if _, exists := candidates[userID]; exists {
			return
		}
		candidates[userID] = notificationType
	}

	switch message.EventType {
	case factory.EventTypeOrderAssigneesUpdated:
		for _, assignedID := range message.AssignedUserIDs {
			if userID, err := uuid.Parse(assignedID); err == nil {
				add(userID, models.NotificationTypeWorkOrderAssigned)
			}
		}
	case factory.EventTypeOrderCommentAdded:
		for _, mentionedID := range message.MentionedUserIDs {
			if userID, err := uuid.Parse(mentionedID); err == nil {
				add(userID, models.NotificationTypeWorkOrderMention)
			}
		}
		for _, assignee := range order.Assignees {
			add(assignee.UserID, models.NotificationTypeWorkOrderCommentOwned)
		}
		if order.CreatedByID != nil {
			add(*order.CreatedByID, models.NotificationTypeWorkOrderCommentCreated)
		}
	case factory.EventTypeOrderStatusUpdated:
		// The initial `"" → draft` transition is work order creation,
		// not a change anyone needs an email about.
		if message.FromState == "" {
			return candidates
		}
		for _, assignee := range order.Assignees {
			add(assignee.UserID, models.NotificationTypeWorkOrderStatusOwned)
		}
		if order.CreatedByID != nil {
			add(*order.CreatedByID, models.NotificationTypeWorkOrderStatusOwned)
		}
	case factory.EventTypeOrderArtifactAdded:
		for _, assignee := range order.Assignees {
			add(assignee.UserID, models.NotificationTypeWorkOrderArtifactOwned)
		}
	case factory.EventTypeOrderStatusNoteUpdated:
		for _, assignee := range order.Assignees {
			add(assignee.UserID, models.NotificationTypeWorkOrderStatusNoteOwned)
		}
		if order.CreatedByID != nil {
			add(*order.CreatedByID, models.NotificationTypeWorkOrderStatusNoteOwned)
		}
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
	notificationType string,
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
		if notificationType == models.NotificationTypeWorkOrderMention {
			content.Subject = fmt.Sprintf("[%s] %s mentioned you", orderKey, actorName)
			content.Data.Summary = fmt.Sprintf("%s mentioned you in a comment on %s.", actorName, orderKey)
		} else {
			content.Subject = fmt.Sprintf("[%s] New comment from %s", orderKey, actorName)
			content.Data.Summary = fmt.Sprintf("%s commented on %s.", actorName, orderKey)
		}
		content.Data.Detail = truncateNotificationDetail(message.CommentBody)
	case factory.EventTypeOrderStatusUpdated:
		verb := statusChangeDescription(message)
		content.Subject = fmt.Sprintf("[%s] Work order %s", orderKey, verb)
		content.Data.Summary = fmt.Sprintf("%s %s %s.", actorName, verb, orderKey)
	case factory.EventTypeOrderArtifactAdded:
		content.Subject = fmt.Sprintf("[%s] New artifact", orderKey)
		content.Data.Summary = fmt.Sprintf("%s added a %s artifact to %s.", actorName, artifactTypeLabel(message.ArtifactType), orderKey)
	case factory.EventTypeOrderStatusNoteUpdated:
		content.Subject = fmt.Sprintf("[%s] %s", orderKey, message.StatusNoteHeadline)
		content.Data.Summary = fmt.Sprintf("%s flagged %s as waiting on you: %s.", actorName, orderKey, message.StatusNoteHeadline)
		content.Data.Detail = truncateNotificationDetail(message.StatusNoteBody)
		content.Data.DetailCtaLabel = message.StatusNoteCtaLabel
		content.Data.DetailCtaURL = message.StatusNoteCtaURL
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
