package canvases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeCanvas(
	canvas *models.Canvas,
	liveVersion *models.CanvasVersion,
	user *models.User,
	status *pb.Canvas_Status,
) (*pb.Canvas, error) {
	var createdBy *pb.UserRef
	if user != nil {
		createdBy = &pb.UserRef{Id: user.ID.String(), Name: user.Name}
	}

	canvasFolderID := ""
	if canvas.CanvasFolderID != nil {
		canvasFolderID = canvas.CanvasFolderID.String()
	}

	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Id:             canvas.ID.String(),
			OrganizationId: canvas.OrganizationID.String(),
			Name:           canvas.Name,
			Description:    canvas.Description,
			CreatedAt:      timestamppb.New(*canvas.CreatedAt),
			UpdatedAt:      timestamppb.New(*canvas.UpdatedAt),
			CreatedBy:      createdBy,
			FolderId:       canvasFolderID,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: actions.NodesToProto(liveVersion.Nodes),
			Edges: actions.EdgesToProto(liveVersion.Edges),
		},
		Status: status,
	}, nil
}

func serializeCanvas(
	ctx context.Context,
	canvas *models.Canvas,
	liveVersion *models.CanvasVersion,
	user *models.User,
	status *pb.Canvas_Status,
) (proto *pb.Canvas, err error) {
	ctx, done := telemetry.Span(ctx, "canvases.serialize")
	defer done(&err)

	return SerializeCanvas(canvas, liveVersion, user, status)
}

func SerializeCanvasRunRef(run models.CanvasRun) *pb.CanvasRunRef {
	return &pb.CanvasRunRef{
		Id:       run.ID.String(),
		CanvasId: run.WorkflowID.String(),
		State:    RunStateToProto(run.State),
		Result:   RunResultToProto(run.Result),
		Errors:   run.ErrorMessages(),
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
