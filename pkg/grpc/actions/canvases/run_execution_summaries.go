package canvases

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

// Load compact evidence once per canvas in the page, not once per run. Full
// output payloads remain on the execution detail endpoint.
func attachExecutionSummaries(db *gorm.DB, runs []models.CanvasRun, serialized []*pb.CanvasRun) error {
	byCanvas := map[uuid.UUID][]int{}
	for i, run := range runs {
		byCanvas[run.WorkflowID] = append(byCanvas[run.WorkflowID], i)
	}
	for canvasID, indices := range byCanvas {
		versionIDs := []uuid.UUID{}
		executionIDs := []uuid.UUID{}
		for _, i := range indices {
			versionIDs = append(versionIDs, runs[i].VersionID)
			for _, execution := range serialized[i].Executions {
				id, err := uuid.Parse(execution.Id)
				if err != nil {
					return fmt.Errorf("read execution summary: %w", err)
				}
				executionIDs = append(executionIDs, id)
			}
		}
		if len(executionIDs) == 0 {
			continue
		}
		versions, err := models.FindCanvasVersionsByIDsInTransaction(db, canvasID, versionIDs)
		if err != nil {
			return fmt.Errorf("load run versions: %w", err)
		}
		channels, err := models.ListExecutionOutputChannels(db, canvasID, executionIDs)
		if err != nil {
			return fmt.Errorf("load execution output channels: %w", err)
		}
		channelsByExecution := map[string][]string{}
		for _, output := range channels {
			channelsByExecution[output.ExecutionID.String()] = append(channelsByExecution[output.ExecutionID.String()], output.Channel)
		}
		for _, i := range indices {
			components := map[string]string{}
			if version := versions[runs[i].VersionID]; version != nil {
				for _, node := range version.Nodes {
					components[node.ID] = node.ComponentName()
				}
			}
			for _, execution := range serialized[i].Executions {
				execution.OutputChannels = channelsByExecution[execution.Id]
				execution.ComponentName = components[execution.NodeId]
			}
		}
	}
	return attachSubscriptionSummaries(db, runs, serialized)
}

func attachSubscriptionSummaries(db *gorm.DB, runs []models.CanvasRun, serialized []*pb.CanvasRun) error {
	runIDs := make([]uuid.UUID, 0, len(runs))
	for _, run := range runs {
		runIDs = append(runIDs, run.ID)
	}
	summaries, err := models.ListRunnerResultSummariesForRuns(db, runIDs)
	if err != nil {
		return fmt.Errorf("load subscription run summaries: %w", err)
	}
	for i, run := range runs {
		summary := summaries[run.ID]
		serialized[i].SubscriptionFailure = actions.SerializeSubscriptionFailure(summary.SubscriptionFailure)
		serialized[i].UsageState = summary.UsageState
	}
	return nil
}
