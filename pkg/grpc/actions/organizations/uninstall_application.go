package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UninstallApplication(ctx context.Context, orgID string, ID string) (*pb.UninstallApplicationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization ID: %v", err)
	}

	installationID, err := uuid.Parse(ID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid installation ID: %v", err)
	}

	appInstallation, err := models.FindAppInstallation(org, installationID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "application installation not found: %v", err)
	}

	// Delete the application installation
	err = database.Conn().Delete(appInstallation).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete application installation: %v", err)
	}

	return &pb.UninstallApplicationResponse{}, nil
}
