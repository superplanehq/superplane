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

const StageUpdatedServiceName = "superplane" + "." + messages.DeliveryHubCanvasExchange + "." + messages.StageUpdatedRoutingKey + ".worker-consumer"
const StageUpdatedConnectionName = "superplane"

type StageUpdatedConsumer struct {
	Consumer       *tackle.Consumer
	CleanupService *ResourceCleanupService
	RabbitMQURL    string
}

func NewStageUpdatedConsumer(registry *registry.Registry, rabbitMQURL string) *StageUpdatedConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "stage_updated",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &StageUpdatedConsumer{
		RabbitMQURL:    rabbitMQURL,
		Consumer:       consumer,
		CleanupService: NewResourceCleanupService(registry),
	}
}

func (c *StageUpdatedConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: StageUpdatedConnectionName,
		Service:        StageUpdatedServiceName,
		RemoteExchange: messages.DeliveryHubCanvasExchange,
		RoutingKey:     messages.StageUpdatedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.StageUpdatedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.StageUpdatedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.StageUpdatedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *StageUpdatedConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *StageUpdatedConsumer) Consume(delivery tackle.Delivery) error {
	data := &protos.StageUpdated{}
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

	stageID, err := uuid.Parse(data.StageId)
	if err != nil {
		log.Errorf("invalid stage ID %s: %v", data.StageId, err)
		return nil
	}

	log.Infof("Processing resource change for stage %s: old=%s, new=%s",
		stageID, data.OldResourceId, data.NewResourceId)

	return c.CleanupService.CleanupUnusedResource(oldResourceID, stageID)
}
