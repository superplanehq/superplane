package canvases

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func SerializeCanvasRunRef(run models.CanvasRun) *pb.CanvasRunRef {
	return &pb.CanvasRunRef{
		Id:       run.ID.String(),
		CanvasId: run.WorkflowID.String(),
		State:    RunStateToProto(run.State),
		Result:   RunResultToProto(run.Result),
	}
}

func SerializeCanvasRunRefs(runs []models.CanvasRun) []*pb.CanvasRunRef {
	refs := make([]*pb.CanvasRunRef, 0, len(runs))
	for _, run := range runs {
		refs = append(refs, SerializeCanvasRunRef(run))
	}
	return refs
}

func parentRunCacheKey(workflowID, runID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", workflowID, runID)
}

func groupChildRunsByParentExecutionID(runs []models.CanvasRun) map[string][]models.CanvasRun {
	grouped := make(map[string][]models.CanvasRun)
	for _, run := range runs {
		if run.ParentExecutionID == nil {
			continue
		}

		key := run.ParentExecutionID.String()
		grouped[key] = append(grouped[key], run)
	}

	return grouped
}

func indexParentRunsByChildID(runs []models.CanvasRun, parents []models.CanvasRun) map[string]models.CanvasRun {
	parentByKey := make(map[string]models.CanvasRun, len(parents))
	for _, parent := range parents {
		parentByKey[parentRunCacheKey(parent.WorkflowID, parent.ID)] = parent
	}

	indexed := make(map[string]models.CanvasRun)
	for _, run := range runs {
		if run.ParentRunID == nil || run.ParentWorkflowID == nil {
			continue
		}

		key := parentRunCacheKey(*run.ParentWorkflowID, *run.ParentRunID)
		parent, ok := parentByKey[key]
		if !ok {
			continue
		}

		indexed[run.ID.String()] = parent
	}

	return indexed
}
