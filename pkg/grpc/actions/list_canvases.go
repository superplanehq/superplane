package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListCanvases(ctx context.Context, req *pb.ListCanvasesRequest) (*pb.ListCanvasesResponse, error) {
	canvases, err := models.ListCanvases()
	if err != nil {
		return nil, err
	}

	response := &pb.ListCanvasesResponse{
		Canvases: serializeCanvases(canvases),
	}

	return response, nil
}

func serializeCanvases(in []models.Canvas) []*pb.Canvas {
	out := []*pb.Canvas{}
	for _, canvas := range in {
		out = append(out, &pb.Canvas{
			Id:        canvas.ID.String(),
			Name:      canvas.Name,
			CreatedBy: canvas.CreatedBy.String(),
			CreatedAt: timestamppb.New(*canvas.CreatedAt),
		})
	}

	return out
}
