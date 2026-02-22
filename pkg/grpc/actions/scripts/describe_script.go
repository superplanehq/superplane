package scripts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/scripts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeScript(ctx context.Context, organizationID string, id string) (*pb.DescribeScriptResponse, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid script id: %v", err)
	}

	script, err := models.FindScript(organizationID, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "script not found")
	}

	return &pb.DescribeScriptResponse{
		Script: SerializeScript(script),
	}, nil
}
