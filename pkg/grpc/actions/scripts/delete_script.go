package scripts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/scripts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteScript(ctx context.Context, organizationID string, id string) (*pb.DeleteScriptResponse, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid script id: %v", err)
	}

	script, err := models.FindScript(organizationID, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "script not found")
	}

	if err := database.Conn().Delete(script).Error; err != nil {
		return nil, err
	}

	return &pb.DeleteScriptResponse{}, nil
}
