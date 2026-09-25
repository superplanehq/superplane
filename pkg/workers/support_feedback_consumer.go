package workers

import (
	"encoding/json"
	"errors"
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
	publish      func(messages.SupportFeedbackRequestedMessage) error
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
		publish: func(message messages.SupportFeedbackRequestedMessage) error {
			return message.Publish()
		},
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

	sendEmail := c.EmailService != nil && !data.EmailDelivered
	sendDiscord := c.Discord.Enabled() && !data.DiscordDelivered
	if !sendEmail && !sendDiscord && !data.EmailDelivered && !data.DiscordDelivered {
		log.Warn("Skipping support feedback: email and Discord are not configured")
		outcome = executorOutcomeSkipped
		reason = emailWorkerReasonInvalidMessage
		return nil
	}

	var emailErr error
	var discordErr error
	if sendEmail {
		emailErr = c.EmailService.SendSupportFeedbackEmail(services.SupportFeedbackToEmail(), feedback)
		if emailErr != nil {
			log.Errorf("Failed to send support feedback email: %v", emailErr)
		}
	}
	if sendDiscord {
		discordErr = c.Discord.SendSupportFeedback(feedback)
		if discordErr != nil {
			log.Errorf("Failed to send support feedback to Discord: %v", discordErr)
		}
	}

	if err := c.finishSupportFeedbackDelivery(data, sendEmail, emailErr, sendDiscord, discordErr); err != nil {
		outcome = executorOutcomeFailed
		reason = emailWorkerReasonSendError
		return err
	}

	return nil
}

// finishSupportFeedbackDelivery retries only the channel that failed.
// A full queue retry would send the successful channel again.
func (c *SupportFeedbackConsumer) finishSupportFeedbackDelivery(
	data messages.SupportFeedbackRequestedMessage,
	sendEmail bool,
	emailErr error,
	sendDiscord bool,
	discordErr error,
) error {
	emailFailed := sendEmail && emailErr != nil
	discordFailed := sendDiscord && discordErr != nil
	if !emailFailed && !discordFailed {
		log.Infof("Delivered support feedback from %s", data.UserEmail)
		return nil
	}

	emailSucceeded := sendEmail && emailErr == nil
	discordSucceeded := sendDiscord && discordErr == nil
	if !emailSucceeded && !discordSucceeded {
		return errors.Join(emailErr, discordErr)
	}

	next := data
	if emailSucceeded {
		next.EmailDelivered = true
	}
	if discordSucceeded {
		next.DiscordDelivered = true
	}
	if err := c.publish(next); err != nil {
		log.Errorf("Failed to republish partial support feedback: %v", err)
		return errors.Join(emailErr, discordErr, err)
	}

	log.Warnf("Republished support feedback after a partial delivery failure")
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
