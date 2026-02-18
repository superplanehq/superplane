package workers

import (
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/pkg/utils"
	"google.golang.org/protobuf/proto"
)

const NotificationEmailServiceName = "superplane" + "." + messages.WorkflowExchange + "." + messages.NotificationEmailRequestedRoutingKey + ".worker-consumer"
const NotificationEmailConnectionName = "superplane"

type NotificationEmailConsumer struct {
	Consumer     *tackle.Consumer
	RabbitMQURL  string
	EmailService services.EmailService
	AuthService  authorization.Authorization
}

func NewNotificationEmailConsumer(
	rabbitMQURL string,
	emailService services.EmailService,
	authService authorization.Authorization,
) *NotificationEmailConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "notification_email",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &NotificationEmailConsumer{
		RabbitMQURL:  rabbitMQURL,
		Consumer:     consumer,
		EmailService: emailService,
		AuthService:  authService,
	}
}

func (c *NotificationEmailConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: NotificationEmailConnectionName,
		Service:        NotificationEmailServiceName,
		RemoteExchange: messages.WorkflowExchange,
		RoutingKey:     messages.NotificationEmailRequestedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.NotificationEmailRequestedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.NotificationEmailRequestedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.NotificationEmailRequestedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *NotificationEmailConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *NotificationEmailConsumer) Consume(delivery tackle.Delivery) error {
	data := &protos.NotificationEmailRequested{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		log.Errorf("Error unmarshaling notification email message: %v", err)
		return err
	}

	orgID, err := uuid.Parse(data.OrganizationId)
	if err != nil {
		log.Errorf("Invalid organization ID %s: %v", data.OrganizationId, err)
		return nil
	}

	recipients, err := c.resolveRecipients(orgID, data)
	if err != nil {
		log.Errorf("Error resolving notification recipients: %v", err)
		return err
	}

	if len(recipients) == 0 {
		log.Warnf("No recipients found for notification in org %s", orgID)
		return nil
	}

	err = c.EmailService.SendNotificationEmail(recipients, data.Title, data.Body, data.Url, data.UrlLabel)
	if err != nil {
		log.Errorf("Failed to send notification email for org %s: %v", orgID, err)
		return err
	}

	log.Infof("Successfully sent notification email for org %s to %d recipients", orgID, len(recipients))
	return nil
}

func (c *NotificationEmailConsumer) resolveRecipients(orgID uuid.UUID, data *protos.NotificationEmailRequested) ([]string, error) {
	unique := map[string]struct{}{}

	for _, email := range data.Emails {
		normalized := utils.NormalizeEmail(email)
		if normalized == "" {
			continue
		}
		unique[normalized] = struct{}{}
	}

	if c.AuthService == nil {
		if len(data.Groups) > 0 || len(data.Roles) > 0 {
			log.Warn("Notification email consumer cannot resolve group/role recipients without auth service")
		}
		return mapKeys(unique), nil
	}

	for _, group := range data.Groups {
		userIDs, err := c.AuthService.GetGroupUsers(orgID.String(), models.DomainTypeOrganization, group)
		if err != nil {
			log.Warnf("Error finding users in group %s: %v", group, err)
			continue
		}

		addUsersToRecipientSet(orgID, userIDs, unique)
	}

	for _, role := range data.Roles {
		userIDs, err := c.AuthService.GetOrgUsersForRole(role, orgID.String())
		if err != nil {
			log.Warnf("Error finding users for role %s: %v", role, err)
			continue
		}

		addUsersToRecipientSet(orgID, userIDs, unique)
	}

	return mapKeys(unique), nil
}

func addUsersToRecipientSet(orgID uuid.UUID, userIDs []string, recipients map[string]struct{}) {
	users, err := models.ListActiveUsersByID(orgID.String(), userIDs)
	if err != nil {
		log.Errorf("Error finding users for notification: %v", err)
		return
	}

	for _, user := range users {
		normalized := utils.NormalizeEmail(user.GetEmail())
		if normalized == "" {
			continue
		}

		recipients[normalized] = struct{}{}
	}
}

func mapKeys(input map[string]struct{}) []string {
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	return result
}
