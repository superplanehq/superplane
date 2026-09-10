package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/services"
)

// reconnectDelay is how long an email consumer waits before it reconnects to
// RabbitMQ after the connection drops or fails to open.
const reconnectDelay = 5 * time.Second

const MagicCodeEmailServiceName = "superplane" + "." + messages.CanvasExchange + "." + messages.MagicCodeRequestedRoutingKey + ".worker-consumer"
const MagicCodeEmailConnectionName = "superplane"

type MagicCodeEmailConsumer struct {
	Consumer     *tackle.Consumer
	RabbitMQURL  string
	EmailService services.EmailService
	BaseURL      string
}

func NewMagicCodeEmailConsumer(rabbitMQURL string, emailService services.EmailService, baseURL string) *MagicCodeEmailConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "magic_code_email",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &MagicCodeEmailConsumer{
		RabbitMQURL:  rabbitMQURL,
		Consumer:     consumer,
		EmailService: emailService,
		BaseURL:      baseURL,
	}
}

func (c *MagicCodeEmailConsumer) Start(ctx context.Context) error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: MagicCodeEmailConnectionName,
		Service:        MagicCodeEmailServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.MagicCodeRequestedRoutingKey,
	}

	for {
		if ctx.Err() != nil {
			return nil
		}

		log.Infof("Connecting to RabbitMQ queue for %s events", messages.MagicCodeRequestedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)

		//
		// A cancelled context means shutdown closed the connection on purpose,
		// so this is neither a failure nor a reconnect. Keep a real error
		// visible, at a level that does not read as an alert during a deploy.
		//
		if ctx.Err() != nil {
			if err != nil {
				log.Infof("Consumer for %s stopped during shutdown: %v", messages.MagicCodeRequestedRoutingKey, err)
			}

			return nil
		}

		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.MagicCodeRequestedRoutingKey, err)
		} else {
			log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.MagicCodeRequestedRoutingKey)
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

func (c *MagicCodeEmailConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *MagicCodeEmailConsumer) Consume(delivery tackle.Delivery) error {
	start := time.Now()
	outcome := executorOutcomeSuccess
	reason := executorReasonNone
	defer func() {
		recordEmailWorkerProcessing(start, emailTypeMagicCode, outcome, reason)
	}()

	var data messages.MagicCodeRequestedMessage
	err := json.Unmarshal(delivery.Body(), &data)
	if err != nil {
		log.Errorf("Error unmarshaling magic code requested message: %v", err)
		outcome = executorOutcomeFailed
		reason = emailWorkerReasonInvalidMessage
		return err
	}

	if data.Email == "" || data.Code == "" {
		log.Errorf("Invalid magic code requested message: missing fields")
		outcome = executorOutcomeSkipped
		reason = emailWorkerReasonInvalidMessage
		return nil
	}

	var magicLink string
	if data.MagicLinkToken != "" {
		magicLink = fmt.Sprintf("%s/auth/magic-code/verify?token=%s",
			c.BaseURL,
			url.QueryEscape(data.MagicLinkToken),
		)
		if data.RedirectURL != "" {
			magicLink += "&redirect=" + url.QueryEscape(data.RedirectURL)
		}
		if data.SignupIntent {
			magicLink += "&signup=true"
		}
	}

	readableCode := formatReadableCode(data.Code)

	err = c.EmailService.SendMagicCodeEmail(data.Email, readableCode, magicLink)
	if err != nil {
		log.Errorf("Failed to send magic code email to %s: %v", data.Email, err)
		outcome = executorOutcomeFailed
		reason = emailWorkerReasonSendError
		return err
	}

	log.Infof("Successfully sent magic code email to %s", data.Email)
	return nil
}

func formatReadableCode(code string) string {
	if len(code) == 6 {
		return code[:3] + " " + code[3:]
	}
	return code
}
