package workers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const EventSourceUpdatedServiceName = "superplane" + "." + messages.DeliveryHubCanvasExchange + "." + messages.EventSourceUpdatedRoutingKey + ".worker-consumer"
const EventSourceUpdatedConnectionName = "superplane"

type EventSourceUpdatedConsumer struct {
	Consumer    *tackle.Consumer
	Registry    *registry.Registry
	RabbitMQURL string
}

func NewEventSourceUpdatedConsumer(registry *registry.Registry, rabbitMQURL string) *EventSourceUpdatedConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "event_source_updated",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &EventSourceUpdatedConsumer{
		RabbitMQURL: rabbitMQURL,
		Consumer:    consumer,
		Registry:    registry,
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

	oldResource, err := models.FindResourceByID(oldResourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warningf("Old resource %s not found, skipping cleanup", oldResourceID)
			return nil
		}
		log.Errorf("Error finding old resource %s: %v", oldResourceID, err)
		return err
	}

	logger := log.WithField("old_resource_id", oldResourceID)

	otherEventSourceCount, err := models.CountOtherEventSourcesUsingResource(oldResourceID, eventSourceID)
	if err != nil {
		logger.Errorf("Error counting other event sources using resource: %v", err)
		return err
	}

	stageCount, err := models.CountStagesUsingResource(oldResourceID)
	if err != nil {
		logger.Errorf("Error counting stages using resource: %v", err)
		return err
	}

	totalUsages := otherEventSourceCount + stageCount
	if totalUsages > 0 {
		logger.Infof("Resource is used by %d other event sources and %d stages, skipping cleanup",
			otherEventSourceCount, stageCount)
		return nil
	}

	integration, err := models.FindIntegrationByID(oldResource.IntegrationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("Integration not found, skipping cleanup")
			return nil
		}
		logger.Errorf("Error finding integration: %v", err)
		return err
	}

	resourceManager, err := c.Registry.NewResourceManager(context.Background(), integration)
	if err != nil {
		logger.Errorf("Error creating resource manager: %v", err)
		return err
	}

	childResources, err := oldResource.FindChildren()
	if err != nil {
		logger.Errorf("Error finding child resources: %v", err)
		return err
	}

	for _, childResource := range childResources {
		logger.Infof("Cleaning up webhook resource %s", childResource.Id())
		err = resourceManager.CleanupWebhook(oldResource, &childResource)
		if err != nil {
			logger.Errorf("Error cleaning up webhook: %v", err)
		} else {
			logger.Infof("Successfully cleaned up webhook %s", childResource.Id())
		}
	}

	// Delete the old resource and its children from database since they're no longer used
	err = models.DeleteResourceWithChildren(oldResourceID)
	if err != nil {
		logger.Errorf("Error deleting old resource and children from database: %v", err)
		return err
	}

	logger.Infof("Successfully cleaned up and deleted old resource %s and %d child resources", oldResourceID, len(childResources))

	return nil
}
