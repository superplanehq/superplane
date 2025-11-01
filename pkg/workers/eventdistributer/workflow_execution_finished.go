package eventdistributer

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

func HandleWorkflowExecutionFinished(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_finished event")

	pbMsg := &pb.ExecutionFinished{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal execution_finished message: %w", err)
	}

	return handleExecutionState(pbMsg.WorkflowId, pbMsg.Id, wsHub, "execution_finished")
}
