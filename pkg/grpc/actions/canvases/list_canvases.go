package canvases

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListCanvases(ctx context.Context, registry *registry.Registry, organizationID string, includeTemplates bool) (*pb.ListCanvasesResponse, error) {
	workflows, err := models.ListWorkflows(organizationID, includeTemplates)
	if err != nil {
		log.Errorf("failed to list canvases for organization %s: %v", organizationID, err)
		return nil, status.Error(codes.Internal, "failed to list canvases")
	}

	protoCanvases := make([]*pb.Canvas, len(workflows))
	for i, workflow := range workflows {
		protoCanvas, err := SerializeCanvas(&workflow, false)
		if err != nil {
			return nil, err
		}

		protoCanvases[i] = protoCanvas
	}

	return &pb.ListCanvasesResponse{
		Canvases: protoCanvases,
	}, nil
}
