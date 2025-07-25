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

const StageEventApprovedServiceName = "superplane" + "." + messages.DeliveryHubCanvasExchange + "." + messages.StageEventApprovedRoutingKey + ".worker-consumer"
const StageEventApprovedConnectionName = "superplane"

type StageEventApprovedConsumer struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
}

func NewStageEventApprovedConsumer(rabbitMQURL string) *StageEventApprovedConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "stage_event_approved",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &StageEventApprovedConsumer{
		RabbitMQURL: rabbitMQURL,
		Consumer:    consumer,
	}
}

func (c *StageEventApprovedConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: StageEventApprovedConnectionName,
		Service:        StageEventApprovedServiceName,
		RemoteExchange: messages.DeliveryHubCanvasExchange,
		RoutingKey:     messages.StageEventApprovedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.StageEventApprovedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.StageEventApprovedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.StageEventApprovedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *StageEventApprovedConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *StageEventApprovedConsumer) Consume(delivery tackle.Delivery) error {
	data := &protos.StageEventApproved{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		return err
	}

	stageID, err := uuid.Parse(data.StageId)
	if err != nil {
		log.Errorf("invalid stage ID %s: %v", data.StageId, err)
		return nil
	}

	stage, err := models.FindStageByID(stageID.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warningf("stage %s not found", stageID)
			return nil
		}

		log.Errorf("Error finding stage %s: %v", stageID, err)
		return err
	}

	logger := logging.ForStage(stage)
	if !stage.HasApprovalCondition() {
		log.Infof("Stage %s does not have approval condition - skipping", stageID)
		return nil
	}

	event, err := models.FindStageEventByID(data.EventId, data.StageId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Errorf("Stage event %s not found for stage %s", data.EventId, stageID)
			return err
		}

		logger.Errorf("Error finding stage event %s: %v", data.EventId, err)
		return err
	}

	approvals, err := event.FindApprovals()
	if err != nil {
		logger.Errorf("Error finding approvals for stage event %s: %v", data.EventId, err)
		return err
	}

	//
	// If the number of approvals is still below what we need, we don't do anything.
	//
	approvalsRequired := stage.ApprovalsRequired()
	if len(approvals) < approvalsRequired {
		logger.Infof(
			"Approvals are still below the required amount for event %s - %d/%d",
			data.EventId,
			len(approvals),
			approvalsRequired,
		)
		return nil
	}

	//
	// Otherwise, we move the event back to the pending state.
	//
	logger.Infof(
		"Approvals reached the required amount for %s - %d/%d - moving to pending state",
		data.EventId,
		len(approvals),
		approvalsRequired,
	)

	return event.UpdateState(models.StageEventStatePending, "")
}
