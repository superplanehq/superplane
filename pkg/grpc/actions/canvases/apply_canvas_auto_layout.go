package canvases

import (
	"context"

	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func ApplyCanvasAutoLayout(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	autoLayout *pb.CanvasAutoLayout,
) (*pb.ApplyCanvasAutoLayoutResponse, error) {
	if autoLayout == nil {
		return nil, grpcerrors.InvalidArgument(nil, "auto_layout is required")
	}

	canvas, branch, version, userUUID, err := loadBranchForStaging(ctx, organizationID, canvasID, "", versionID)
	if err != nil {
		return nil, err
	}

	_, rows, err := stagingSummaryForBranch(branch.ID, userUUID)
	if err != nil {
		return nil, err
	}

	canvasYAML, err := effectiveSpecYAML(canvas, version, organizationID, rows, CanvasYAMLRepositoryPath)
	if err != nil {
		return nil, err
	}

	pbCanvas, err := canvasFromYAMLText(canvasYAML)
	if err != nil {
		return nil, err
	}

	nodes := actions.ProtoToNodes(pbCanvas.GetSpec().GetNodes())
	edges := actions.ProtoToEdges(pbCanvas.GetSpec().GetEdges())

	laidOutNodes, laidOutEdges, err := layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "failed to apply layout")
	}

	positioned := &pb.CanvasVersion{
		Metadata: &pb.CanvasVersion_Metadata{
			Name:        pbCanvas.GetMetadata().GetName(),
			Description: pbCanvas.GetMetadata().GetDescription(),
		},
		Spec: &pb.Canvas_Spec{
			Nodes: actions.NodesToProto(laidOutNodes),
			Edges: actions.EdgesToProto(laidOutEdges),
		},
	}

	positionedYAML, err := canvasyaml.CanvasResourceYAML(positioned, canvas.ID.String())
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to serialize canvas")
	}

	if _, err := models.UpsertWorkflowStagingPath(
		branch.ID,
		userUUID,
		CanvasYAMLRepositoryPath,
		positionedYAML,
		&userUUID,
	); err != nil {
		return nil, grpcerrors.Internal(err, "failed to stage canvas layout")
	}

	state, _, err := stagingSummaryForBranch(branch.ID, userUUID)
	if err != nil {
		return nil, err
	}

	publishStagingUpdated(canvas.ID, version.ID)

	return &pb.ApplyCanvasAutoLayoutResponse{StagingSummary: state}, nil
}
