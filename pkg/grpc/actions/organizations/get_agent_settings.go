package organizations

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func GetAgentSettings(orgID string) (*pb.GetAgentSettingsResponse, error) {
	settings, err := findOrCreateOrganizationAgentSettingsInTransaction(database.Conn(), orgID)
	if err != nil {
		return nil, err
	}

	credential, err := models.FindOrganizationAgentCredentialByOrganizationID(orgID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.Internal, "failed to load agent credential")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		credential = nil
	}

	return &pb.GetAgentSettingsResponse{
		AgentSettings: serializeAgentSettings(settings, credential),
	}, nil
}
