package eventdistributer

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	factoriespb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type RunStateWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

const (
	RunPendingEvent    = "run_pending"
	RunStartedEvent    = "run_started"
	RunCancellingEvent = "run_cancelling"
	RunFinishedEvent   = "run_finished"
)

func HandleCanvasRun(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received run event")

	pbMsg, err := parseCanvasRunMessage(messageBody)
	if err != nil {
		return err
	}

	return handleRunState(pbMsg.CanvasId, pbMsg.Id, wsHub)
}

func HandlePendingCanvasRun(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received pending run event")

	pbMsg, err := parseCanvasRunMessage(messageBody)
	if err != nil {
		return err
	}

	return handlePendingRunState(pbMsg.CanvasId, pbMsg.Id, wsHub)
}

func parseCanvasRunMessage(messageBody []byte) (*pb.CanvasRunMessage, error) {
	pbMsg := &pb.CanvasRunMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal run event: %w", err)
	}

	return pbMsg, nil
}

func runStateToWsEvent(runState string) string {
	switch runState {
	case models.CanvasRunStatePending:
		return RunPendingEvent
	case models.CanvasRunStateStarted:
		return RunStartedEvent
	case models.CanvasRunStateCancelling:
		return RunCancellingEvent
	case models.CanvasRunStateFinished:
		return RunFinishedEvent
	default:
		return ""
	}
}

func handleRunState(workflowID string, runID string, wsHub *ws.Hub) error {
	runUUID, eventName, err := broadcastRunState(workflowID, runID, wsHub)
	if err != nil {
		return err
	}

	broadcastFactoryWorkOrderForRun(wsHub, runUUID, eventName)
	return nil
}

func handlePendingRunState(workflowID string, runID string, wsHub *ws.Hub) error {
	_, _, err := broadcastRunState(workflowID, runID, wsHub)
	return err
}

func broadcastRunState(workflowID string, runID string, wsHub *ws.Hub) (uuid.UUID, string, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to parse workflow id: %w", err)
	}

	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to parse run id: %w", err)
	}

	db := database.Conn()
	run, err := models.FindCanvasRunInTransaction(db, workflowUUID, runUUID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to find run: %w", err)
	}

	eventName := runStateToWsEvent(run.State)
	if eventName == "" {
		return uuid.Nil, "", fmt.Errorf("unknown run state: %s", run.State)
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
		queueItems, err = models.ListNodeQueueItemsForRuns(database.Conn(), workflowUUID, []uuid.UUID{runUUID})
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
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to find run root event: %w", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return uuid.Nil, "", err
	}

	serializedRun, err := canvases.SerializeCanvasRun(db, *run, rootEvent, executions, queueItems, nil, map[string][]models.CanvasRun{})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to serialize run: %w", err)
	}

	serializedRunJSON, err := marshalCanvasRunJSON(serializedRun)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to marshal run: %w", err)
	}

	event, err := json.Marshal(RunStateWebsocketEvent{
		Event:   eventName,
		Payload: json.RawMessage(serializedRunJSON),
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(workflowID, event)
	log.Debugf("Broadcasted %s event to workflow %s", eventName, workflowID)

	return runUUID, eventName, nil
}

func broadcastFactoryWorkOrderForRun(wsHub *ws.Hub, runID uuid.UUID, reason string) {
	execution, err := models.FindWorkOrderExecutionByRunID(database.Conn(), runID)
	if err != nil {
		if !errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
			log.WithError(err).Warnf("Failed to look up factory work order execution for run %s", runID)
		}
		return
	}

	if err := BroadcastFactoryWorkOrderUpdated(wsHub, &factoriespb.FactoryWorkOrderUpdatedMessage{
		FactoryId: execution.FactoryID.String(),
		OrderId:   execution.WorkOrderID.String(),
		Reason:    reason,
	}); err != nil {
		log.WithError(err).Warnf("Failed to broadcast factory work order update for run %s", runID)
	}
}

func marshalCanvasRunJSON(run *pb.CanvasRun) ([]byte, error) {
	return protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(run)
}
