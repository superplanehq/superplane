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

const StageUpdatedServiceName = "superplane" + "." + messages.DeliveryHubCanvasExchange + "." + messages.StageUpdatedRoutingKey + ".worker-consumer"
const StageUpdatedConnectionName = "superplane"

type StageUpdatedConsumer struct {
	Consumer    *tackle.Consumer
	Registry    *registry.Registry
	RabbitMQURL string
}

func NewStageUpdatedConsumer(registry *registry.Registry, rabbitMQURL string) *StageUpdatedConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "stage_updated",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &StageUpdatedConsumer{
		RabbitMQURL: rabbitMQURL,
		Consumer:    consumer,
		Registry:    registry,
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

	externalEventSourceCount, err := models.CountExternalEventSourcesUsingResource(oldResourceID)
	if err != nil {
		logger.Errorf("Error counting external event sources using resource: %v", err)
		return err
	}

	otherStagesCount, err := models.CountOtherStagesUsingResource(oldResourceID, stageID)
	if err != nil {
		logger.Errorf("Error counting other stages using resource: %v", err)
		return err
	}

	totalUsages := externalEventSourceCount + otherStagesCount
	if totalUsages > 0 {
		logger.Infof("Resource is used by %d external event sources and %d other stages, skipping cleanup",
			externalEventSourceCount, otherStagesCount)
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
		logger.Infof("Cleaning up resource %s", childResource.Id())
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
