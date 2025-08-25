package workers

import (
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/proto"
)

const EventSourceUpdatedServiceName = "superplane" + "." + messages.DeliveryHubCanvasExchange + "." + messages.EventSourceUpdatedRoutingKey + ".worker-consumer"
const EventSourceUpdatedConnectionName = "superplane"

type EventSourceUpdatedConsumer struct {
	Consumer       *tackle.Consumer
	CleanupService *ResourceCleanupService
	RabbitMQURL    string
}

func NewEventSourceUpdatedConsumer(registry *registry.Registry, rabbitMQURL string) *EventSourceUpdatedConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "event_source_updated",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &EventSourceUpdatedConsumer{
		RabbitMQURL:    rabbitMQURL,
		Consumer:       consumer,
		CleanupService: NewResourceCleanupService(registry),
	}
}

func (c *EventSourceUpdatedConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: EventSourceUpdatedConnectionName,
		Service:        EventSourceUpdatedServiceName,
		RemoteExchange: messages.DeliveryHubCanvasExchange,
		RoutingKey:     messages.EventSourceUpdatedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.EventSourceUpdatedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.EventSourceUpdatedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.EventSourceUpdatedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *EventSourceUpdatedConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *EventSourceUpdatedConsumer) Consume(delivery tackle.Delivery) error {
	data := &protos.EventSourceUpdated{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		return err
	}

	if data.OldResourceId == "" || data.OldResourceId == data.NewResourceId {
		log.Info("No resource change detected, skipping cleanup")
		return nil
	}

	oldResourceID, err := uuid.Parse(data.OldResourceId)
	if err != nil {
		log.Errorf("invalid old resource ID %s: %v", data.OldResourceId, err)
		return nil
	}

	eventSourceID, err := uuid.Parse(data.SourceId)
	if err != nil {
		log.Errorf("invalid event source ID %s: %v", data.SourceId, err)
		return nil
	}

	log.Infof("Processing resource change for event source %s: old=%s, new=%s",
		eventSourceID, data.OldResourceId, data.NewResourceId)

	return c.CleanupService.CleanupUnusedResource(oldResourceID, uuid.Nil)
}
