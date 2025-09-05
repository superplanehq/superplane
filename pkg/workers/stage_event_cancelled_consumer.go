package workers

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const StageEventCancelledServiceName = "superplane" + "." + messages.DeliveryHubCanvasExchange + "." + messages.StageEventCancelledRoutingKey + ".worker-consumer"
const StageEventCancelledConnectionName = "superplane"

type StageEventCancelledConsumer struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
}

func NewStageEventCancelledConsumer(rabbitMQURL string) *StageEventCancelledConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "stage_event_cancelled",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &StageEventCancelledConsumer{
		RabbitMQURL: rabbitMQURL,
		Consumer:    consumer,
	}
}

func (c *StageEventCancelledConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: StageEventCancelledConnectionName,
		Service:        StageEventCancelledServiceName,
		RemoteExchange: messages.DeliveryHubCanvasExchange,
		RoutingKey:     messages.StageEventCancelledRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.StageEventCancelledRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.StageEventCancelledRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.StageEventCancelledRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *StageEventCancelledConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *StageEventCancelledConsumer) Consume(delivery tackle.Delivery) error {
	data := &protos.StageEventCancelled{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(data.StageId)
	if err != nil {
		log.Errorf("invalid stage ID %s: %v", data.StageId, err)
		return nil
	}

	stage, err := models.FindUnscopedStage(stageID.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warningf("stage %s not found", stageID)
			return nil
		}

		log.Errorf("Error finding stage %s: %v", stageID, err)
		return err
	}

	logger := logging.ForStage(stage)

	// For cancelled events, we mainly want to log
	// The Cancel method already handled the state update and execution cancellation
	logger.Infof("Stage event %s has been cancelled", data.EventId)

	return nil
}