package connectiongroups

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateConnectionGroup(ctx context.Context, canvasID, idOrName string, group *pb.ConnectionGroup) (*pb.UpdateConnectionGroupResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	err := actions.ValidateUUIDs(idOrName)
	var connectionGroup *models.ConnectionGroup
	if err != nil {
		connectionGroup, err = models.FindConnectionGroupByName(canvasID, idOrName)
	} else {
		connectionGroup, err = models.FindConnectionGroupByID(canvasID, idOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "connection group not found")
	}

	connections, err := actions.ValidateConnections(canvasID, group.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	spec, err := validateSpec(group.Spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if group.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "missing metadata")
	}

	if group.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "empty group name")
	}

	connectionGroup.Name = group.Metadata.Name
	if group.Metadata.Description != "" {
		connectionGroup.Description = group.Metadata.Description
	}

	connectionGroup.UpdatedBy = uuid.MustParse(userID)
	err = connectionGroup.Update(connections, *spec)
	if err != nil {
		log.Errorf("Error updating connection group in canvas %s. Group: %v. Error: %v", canvasID, group, err)
		return nil, err
	}

	connectionGroup, _ = models.FindConnectionGroupByID(canvasID, connectionGroup.ID.String())
	pbGroup, err := serializeConnectionGroup(*connectionGroup, connections)
	if err != nil {
		return nil, err
	}

	response := &pb.UpdateConnectionGroupResponse{
		ConnectionGroup: pbGroup,
	}

	return response, nil
}
