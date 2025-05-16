package workers

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/retry"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type StageEventApprovedConsumer struct {
	Consumer    *tackle.Consumer
	RabbitMQURL string
}

func NewStageEventApprovedConsumer(rabbitMQURL string) *StageEventApprovedConsumer {
	return &StageEventApprovedConsumer{
		RabbitMQURL: rabbitMQURL,
		Consumer:    tackle.NewConsumer(),
	}
}

func (c *StageEventApprovedConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: "superplane",
		Service:        "superplane-worker",
		RemoteExchange: "superplane.canvas-exchange",
		RoutingKey:     "stage-event-approved",
	}

	err := retry.WithConstantWait("RabbitMQ connection", 5, time.Second, func() error {
		return c.Consumer.Start(&options, c.Consume)
	})

	if err != nil {
		return fmt.Errorf("error starting consumer: %v", err)
	}

	return nil
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

	stage, err := models.FindStageByID(stageID)
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
