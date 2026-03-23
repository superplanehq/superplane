package workers

import (
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

func (c *MagicCodeEmailConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: MagicCodeEmailConnectionName,
		Service:        MagicCodeEmailServiceName,
		RemoteExchange: messages.CanvasExchange,
		RoutingKey:     messages.MagicCodeRequestedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.MagicCodeRequestedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.MagicCodeRequestedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.MagicCodeRequestedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *MagicCodeEmailConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *MagicCodeEmailConsumer) Consume(delivery tackle.Delivery) error {
	var data messages.MagicCodeRequestedMessage
	err := json.Unmarshal(delivery.Body(), &data)
	if err != nil {
		log.Errorf("Error unmarshaling magic code requested message: %v", err)
		return err
	}

	if data.Email == "" || data.Code == "" {
		log.Errorf("Invalid magic code requested message: missing fields")
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
	}

	readableCode := formatReadableCode(data.Code)

	err = c.EmailService.SendMagicCodeEmail(data.Email, readableCode, magicLink)
	if err != nil {
		log.Errorf("Failed to send magic code email to %s: %v", data.Email, err)
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
