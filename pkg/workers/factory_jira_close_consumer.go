package workers

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	FactoryJiraCloseServiceName    = "superplane" + "." + messages.CanvasExchange + "." + messages.FactoryWorkOrderNotificationRoutingKey + ".jira-close-consumer"
	FactoryJiraCloseConnectionName = "superplane"
)

// FactoryJiraCloseConsumer moves the originating Jira issue after a Jira-intake
// task closes as completed. It shares the work-order notification routing key
// with FactoryNotificationConsumer and uses its own queue so Jira failures
// never block email delivery or SuperPlane Complete.
type FactoryJiraCloseConsumer struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
	Encryptor   crypto.Encryptor
	Registry    *registry.Registry
	BaseURL     string
	httpContext core.HTTPContext
}

func NewFactoryJiraCloseConsumer(
	rabbitMQURL string,
	encryptor crypto.Encryptor,
	componentRegistry *registry.Registry,
	baseURL string,
) *FactoryJiraCloseConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "factory_jira_close",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &FactoryJiraCloseConsumer{
		RabbitMQURL: rabbitMQURL,
		Consumer:    consumer,
		Encryptor:   encryptor,
		Registry:    componentRegistry,
		BaseURL:     baseURL,
	}
}

func (c *FactoryJiraCloseConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: FactoryJiraCloseConnectionName,
		Service:        FactoryJiraCloseServiceName,
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

func (c *FactoryJiraCloseConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *FactoryJiraCloseConsumer) Consume(delivery tackle.Delivery) error {
	var message messages.FactoryWorkOrderNotificationMessage
	if err := json.Unmarshal(delivery.Body(), &message); err != nil {
		log.Errorf("Error unmarshaling work order notification message: %v", err)
		return err
	}

	return c.process(database.Conn(), message)
}

func (c *FactoryJiraCloseConsumer) process(
	db *gorm.DB,
	message messages.FactoryWorkOrderNotificationMessage,
) error {
	if !isCompletedWorkOrderMessage(message) {
		return nil
	}

	orderID, err := uuid.Parse(message.OrderID)
	if err != nil {
		log.Warnf("Skipping Jira close with invalid order id %q", message.OrderID)
		return nil
	}

	order, err := models.FindUnscopedWorkOrder(db, orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !matchesWorkOrderMessage(order, message) {
		return nil
	}
	if !isCompletedWorkOrder(order) {
		return nil
	}

	factoryModel, err := models.FindFactory(db, order.OrganizationID, order.FactoryID)
	if errors.Is(err, models.ErrFactoryNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return factories.CloseJiraOrigin(db, factories.JiraCloseContext{
		HTTP:      c.http(),
		Encryptor: c.Encryptor,
		Registry:  c.Registry,
		BaseURL:   c.BaseURL,
	}, factoryModel, order)
}

func (c *FactoryJiraCloseConsumer) http() core.HTTPContext {
	if c.httpContext != nil {
		return c.httpContext
	}
	return c.Registry.HTTPContext()
}
