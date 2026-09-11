package workers

import (
	"encoding/json"
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/services"
)

const SupportFeedbackServiceName = "superplane" + "." + messages.CanvasExchange + "." + messages.SupportFeedbackRequestedRoutingKey + ".worker-consumer"
const SupportFeedbackConnectionName = "superplane"

type SupportFeedbackConsumer struct {
	Consumer     *tackle.Consumer
	RabbitMQURL  string
	EmailService services.EmailService
	Discord      *services.DiscordWebhookClient
}

func NewSupportFeedbackConsumer(
	rabbitMQURL string,
	emailService services.EmailService,
	discord *services.DiscordWebhookClient,
) *SupportFeedbackConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "support_feedback",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &SupportFeedbackConsumer{
		RabbitMQURL:  rabbitMQURL,
		Consumer:     consumer,
		EmailService: emailService,
		Discord:      discord,
	}
}

func (c *SupportFeedbackConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: SupportFeedbackConnectionName,
		Service:        SupportFeedbackServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.SupportFeedbackRequestedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.SupportFeedbackRequestedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.SupportFeedbackRequestedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.SupportFeedbackRequestedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *SupportFeedbackConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *SupportFeedbackConsumer) Consume(delivery tackle.Delivery) error {
	start := time.Now()
	outcome := executorOutcomeSuccess
	reason := executorReasonNone
	defer func() {
		recordEmailWorkerProcessing(start, emailTypeSupportFeedback, outcome, reason)
	}()

	var data messages.SupportFeedbackRequestedMessage
	err := json.Unmarshal(delivery.Body(), &data)
	if err != nil {
		log.Errorf("Error unmarshaling support feedback message: %v", err)
		outcome = executorOutcomeFailed
		reason = emailWorkerReasonInvalidMessage
		return err
	}

	feedback, err := supportFeedbackFromMessage(data)
	if err != nil {
		log.Errorf("Invalid support feedback message: %v", err)
		outcome = executorOutcomeSkipped
		reason = emailWorkerReasonInvalidMessage
		return nil
	}

	emailEnabled := c.EmailService != nil
	discordEnabled := c.Discord.Enabled()
	if !emailEnabled && !discordEnabled {
		log.Warn("Skipping support feedback: email and Discord are not configured")
		outcome = executorOutcomeSkipped
		reason = emailWorkerReasonInvalidMessage
		return nil
	}

	if emailEnabled {
		if err := c.EmailService.SendSupportFeedbackEmail(services.SupportFeedbackToEmail(), feedback); err != nil {
			log.Errorf("Failed to send support feedback email: %v", err)
			outcome = executorOutcomeFailed
			reason = emailWorkerReasonSendError
			return err
		}
	}

	if discordEnabled {
		if err := c.Discord.SendSupportFeedback(feedback); err != nil {
			log.Errorf("Failed to send support feedback to Discord: %v", err)
			outcome = executorOutcomeFailed
			reason = emailWorkerReasonSendError
			return err
		}
	}

	log.Infof("Delivered support feedback from %s", feedback.UserEmail)
	return nil
}

func supportFeedbackFromMessage(data messages.SupportFeedbackRequestedMessage) (services.SupportFeedback, error) {
	category, err := services.NormalizeFeedbackCategory(data.Category)
	if err != nil {
		return services.SupportFeedback{}, err
	}
	details, err := services.NormalizeFeedbackDetails(data.Details)
	if err != nil {
		return services.SupportFeedback{}, err
	}

	feedback := services.SupportFeedback{
		Category:         category,
		Details:          details,
		UserName:         data.UserName,
		UserEmail:        data.UserEmail,
		OrganizationID:   data.OrganizationID,
		OrganizationName: data.OrganizationName,
		PagePath:         services.NormalizeFeedbackPagePath(data.PagePath),
	}
	if data.Attachment == nil {
		return feedback, nil
	}

	attachment, err := services.NormalizeFeedbackAttachment(
		data.Attachment.Filename,
		data.Attachment.ContentType,
		data.Attachment.Content,
	)
	if err != nil {
		return services.SupportFeedback{}, err
	}
	feedback.Attachment = attachment
	return feedback, nil
}
