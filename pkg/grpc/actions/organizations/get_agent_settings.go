package organizations

import (
	"github.com/superplanehq/superplane/pkg/database"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func GetAgentSettings(orgID string) (*pb.GetAgentSettingsResponse, error) {
	settings, err := findOrCreateOrganizationAgentSettingsInTransaction(database.Conn(), orgID)
	if err != nil {
		return nil, err
	}

	return &pb.GetAgentSettingsResponse{
		AgentSettings: serializeAgentSettings(settings),
	}, nil
}
