package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/proto"
)

// MergeConfidencePublicationConsumer owns an independent queue so publication
// retries do not block assessment completion or other run consumers.
type MergeConfidencePublicationConsumer struct {
	Consumer     *tackle.Consumer
	RabbitMQURL  string
	Dependencies factories.IntakeDependencies
	BaseURL      string
}

func NewMergeConfidencePublicationConsumer(rabbitMQURL string, encryptor crypto.Encryptor, componentRegistry *registry.Registry, baseURL string) *MergeConfidencePublicationConsumer {
	consumer := tackle.NewConsumer()
	consumer.SetLogger(logging.NewTackleLogger(log.WithField("consumer", "merge_confidence_publication")))
	return &MergeConfidencePublicationConsumer{Consumer: consumer, RabbitMQURL: rabbitMQURL, Dependencies: factories.IntakeDependencies{Encryptor: encryptor, Registry: componentRegistry}, BaseURL: baseURL}
}

func (c *MergeConfidencePublicationConsumer) Start() error {
	options := tackle.Options{URL: c.RabbitMQURL, ConnectionName: "superplane", Service: "superplane." + messages.CanvasExchange + ".merge-confidence-publication", RemoteExchange: messages.CanvasExchange, RoutingKey: messages.CanvasRunRoutingKey}
	for {
		if err := c.Consumer.Start(&options, c.Consume); err != nil {
			log.WithError(err).Error("Error consuming merge confidence run messages")
		}
		time.Sleep(5 * time.Second)
	}
}

func (c *MergeConfidencePublicationConsumer) Stop() { c.Consumer.Stop() }

func (c *MergeConfidencePublicationConsumer) Consume(delivery tackle.Delivery) error {
	var message pb.CanvasRunMessage
	if err := proto.Unmarshal(delivery.Body(), &message); err != nil {
		log.WithError(err).Warn("Skipping invalid merge confidence run message")
		return nil
	}
	runID, err := uuid.Parse(message.Id)
	if err != nil {
		log.WithError(err).Warn("Skipping merge confidence message with invalid run ID")
		return nil
	}
	canvasID, err := uuid.Parse(message.CanvasId)
	if err != nil {
		log.WithError(err).Warn("Skipping merge confidence message with invalid canvas ID")
		return nil
	}
	ctx := context.Background()
	return factories.PublishMergeConfidence(ctx, database.DB(ctx), c.Dependencies, canvasID, runID, c.BaseURL)
}
