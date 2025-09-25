package builders

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
)

type MessageStageNotifier struct{}

func NewMessageStageNotifier() models.StageNotifier {
	return &MessageStageNotifier{}
}

func (n *MessageStageNotifier) NotifyStageUpdated(stage *models.Stage) {
	go func() {
		time.Sleep(500 * time.Millisecond)
		message := messages.NewStageUpdatedMessage(stage, nil, nil)
		if err := message.Publish(); err != nil {
			log.Errorf("Failed to publish stage updated message: %v", err)
		}
	}()
}
