package workers

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/services"
)

const OrganizationMemberJoinedEmailServiceName = "superplane." + messages.CanvasExchange + "." + messages.OrganizationMemberJoinedRoutingKey + ".worker-consumer"

type OrganizationMemberJoinedEmailConsumer struct {
	Consumer     *tackle.Consumer
	RabbitMQURL  string
	EmailService services.EmailService
	BaseURL      string
}

func NewOrganizationMemberJoinedEmailConsumer(rabbitMQURL string, emailService services.EmailService, baseURL string) *OrganizationMemberJoinedEmailConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "organization_member_joined_email",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)
	return &OrganizationMemberJoinedEmailConsumer{
		Consumer:     consumer,
		RabbitMQURL:  rabbitMQURL,
		EmailService: emailService,
		BaseURL:      baseURL,
	}
}

func (c *OrganizationMemberJoinedEmailConsumer) Start() {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: MagicCodeEmailConnectionName,
		Service:        OrganizationMemberJoinedEmailServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.OrganizationMemberJoinedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.OrganizationMemberJoinedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.WithError(err).Errorf("Error consuming messages from %s", messages.OrganizationMemberJoinedRoutingKey)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.OrganizationMemberJoinedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *OrganizationMemberJoinedEmailConsumer) Stop() { c.Consumer.Stop() }

func (c *OrganizationMemberJoinedEmailConsumer) Consume(delivery tackle.Delivery) error {
	start := time.Now()
	outcome, reason := executorOutcomeSuccess, executorReasonNone
	defer func() { recordEmailWorkerProcessing(start, emailTypeOrganizationMemberJoined, outcome, reason) }()

	var data messages.OrganizationMemberJoinedMessage
	if err := json.Unmarshal(delivery.Body(), &data); err != nil {
		log.WithError(err).Warn("Discarding malformed organization member joined email message")
		outcome, reason = executorOutcomeSkipped, emailWorkerReasonInvalidMessage
		return nil
	}
	data.ToEmail, data.OrganizationID, data.OrganizationName, data.MemberEmail = strings.TrimSpace(data.ToEmail), strings.TrimSpace(data.OrganizationID), strings.TrimSpace(data.OrganizationName), strings.TrimSpace(data.MemberEmail)
	if data.ToEmail == "" || data.OrganizationID == "" || data.OrganizationName == "" || data.MemberEmail == "" {
		log.WithField("organization_id", data.OrganizationID).Warn("Discarding incomplete organization member joined email message")
		outcome, reason = executorOutcomeSkipped, emailWorkerReasonInvalidMessage
		return nil
	}
	settingsURL, err := url.JoinPath(c.BaseURL, data.OrganizationID, "settings", "members")
	if err != nil {
		log.WithError(err).WithField("organization_id", data.OrganizationID).Error("Failed to build organization member settings URL")
		outcome, reason = executorOutcomeFailed, emailWorkerReasonSendError
		return err
	}
	if err := c.EmailService.SendOrganizationMemberJoinedEmail(data.ToEmail, strings.TrimSpace(data.MemberName), data.MemberEmail, data.OrganizationName, settingsURL); err != nil {
		log.WithError(err).WithField("organization_id", data.OrganizationID).Error("Failed to send organization member joined email")
		outcome, reason = executorOutcomeFailed, emailWorkerReasonSendError
		return err
	}

	log.WithField("organization_id", data.OrganizationID).Info("Sent organization member joined email")
	return nil
}
