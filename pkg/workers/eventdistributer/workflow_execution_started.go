package eventdistributer

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

func HandleWorkflowExecutionStarted(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_started event")

	pbMsg := &pb.ExecutionStarted{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal execution_started message: %w", err)
	}

	return handleExecutionState(pbMsg.WorkflowId, pbMsg.Id, wsHub, "execution_started")
}
