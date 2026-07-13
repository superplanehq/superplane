package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type RunStateWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

const (
	RunStartedEvent  = "run_started"
	RunFinishedEvent = "run_finished"
)

func HandleCanvasRun(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received run event")

	pbMsg := &pb.CanvasRunMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal run event: %w", err)
	}

	return handleRunState(pbMsg.CanvasId, pbMsg.Id, wsHub)
}

func runStateToWsEvent(runState string) string {
	switch runState {
	case models.CanvasRunStateStarted:
		return RunStartedEvent
	case models.CanvasRunStateFinished:
		return RunFinishedEvent
	default:
		return ""
	}
}

func handleRunState(workflowID string, runID string, wsHub *ws.Hub) error {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return fmt.Errorf("failed to parse workflow id: %w", err)
	}

	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return fmt.Errorf("failed to parse run id: %w", err)
	}

	db := database.Conn()
	run, err := models.FindCanvasRunInTransaction(db, workflowUUID, runUUID)
	if err != nil {
		return fmt.Errorf("failed to find run: %w", err)
	}

	eventName := runStateToWsEvent(run.State)
	if eventName == "" {
		return fmt.Errorf("unknown run state: %s", run.State)
	}

	var executions []models.CanvasNodeExecution
	var queueItems []models.CanvasNodeQueueItem
	var rootEvent models.CanvasEvent

	var g errgroup.Group
	g.Go(func() error {
		var err error
		executions, err = models.ListExecutionsForRunsInTransaction(database.Conn(), workflowUUID, []uuid.UUID{runUUID})
		if err != nil {
			return fmt.Errorf("failed to find run executions: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		var err error
		queueItems, err = models.ListNodeQueueItemsForRunsInTransaction(database.Conn(), workflowUUID, []uuid.UUID{runUUID})
		if err != nil {
			return fmt.Errorf("failed to find run queue items: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		err := database.Conn().
			Where("workflow_id = ?", workflowUUID).
			Where("run_id = ?", runUUID).
			Where("execution_id IS NULL").
			First(&rootEvent).
			Error
		if err != nil {
			return fmt.Errorf("failed to find run root event: %w", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	serializedRun, err := canvases.SerializeCanvasRun(db, *run, rootEvent, executions, queueItems)
	if err != nil {
		return fmt.Errorf("failed to serialize run: %w", err)
	}

	serializedRunJSON, err := marshalCanvasRunJSON(serializedRun)
	if err != nil {
		return fmt.Errorf("failed to marshal run: %w", err)
	}

	event, err := json.Marshal(RunStateWebsocketEvent{
		Event:   eventName,
		Payload: json.RawMessage(serializedRunJSON),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(workflowID, event)
	log.Debugf("Broadcasted %s event to workflow %s", eventName, workflowID)

	return nil
}

func marshalCanvasRunJSON(run *pb.CanvasRun) ([]byte, error) {
	return protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(run)
}
