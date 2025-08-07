package connectiongroups

import (
	"context"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateConnectionGroup(ctx context.Context, canvasID string, idOrName string, connectionGroup *pb.ConnectionGroup) (*pb.UpdateConnectionGroupResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	err := actions.ValidateUUIDs(idOrName)
	var existingGroup *models.ConnectionGroup
	if err != nil {
		existingGroup, err = models.FindConnectionGroupByName(canvasID, idOrName)
	} else {
		existingGroup, err = models.FindConnectionGroupByID(canvasID, uuid.MustParse(idOrName))
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "connection group not found")
	}

	connections, err := actions.ValidateConnections(canvasID, connectionGroup.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	spec, err := validateSpec(connectionGroup.Spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if connectionGroup.Metadata != nil && connectionGroup.Metadata.Name != "" && connectionGroup.Metadata.Name != existingGroup.Name {
		_, err := models.FindConnectionGroupByName(canvasID, connectionGroup.Metadata.Name)
		if err == nil {
			return nil, status.Error(codes.InvalidArgument, "connection group name already in use")
		}

		existingGroup.Name = connectionGroup.Metadata.Name
	}

	if connectionGroup.Metadata != nil && connectionGroup.Metadata.Description != "" {
		existingGroup.Description = connectionGroup.Metadata.Description
	}

	existingGroup.UpdatedBy = uuid.Must(uuid.Parse(userID))
	err = existingGroup.Update(connections, *spec)
	if err != nil {
		log.Errorf("Error updating connection group %s in canvas %s. Error: %v", idOrName, canvasID, err)
		return nil, err
	}

	group, err := serializeConnectionGroup(*existingGroup, connections)
	if err != nil {
		return nil, err
	}

	response := &pb.UpdateConnectionGroupResponse{
		ConnectionGroup: group,
	}

	return response, nil
}
