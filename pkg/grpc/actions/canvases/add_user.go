package canvases

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUser(ctx context.Context, authService authorization.Authorization, orgID string, canvasID string, userID string) (*pb.AddUserResponse, error) {
	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	isGlobalDomain := canvasID == "*" && orgID != ""
	if isGlobalDomain {
		err = authService.AssignRoleWithOrgContext(user.ID.String(), models.RoleCanvasViewer, canvasID, models.DomainTypeCanvas, orgID)
	} else {
		err = authService.AssignRole(user.ID.String(), models.RoleCanvasViewer, canvasID, models.DomainTypeCanvas)
	}
	if err != nil {
		log.Errorf("Error adding user %s to canvas %s: %v", userID, canvasID, err)
		return nil, status.Error(codes.Internal, "error adding user")
	}
	return &pb.AddUserResponse{}, nil

}
