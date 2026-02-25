package organizations

import (
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateAgentSettings(
	orgID string,
	agentModeEnabled bool,
	requesterUserID string,
) (*pb.UpdateAgentSettingsResponse, error) {
	var settings *models.OrganizationAgentSettings

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		settings, txErr = findOrCreateOrganizationAgentSettingsInTransaction(tx, orgID)
		if txErr != nil {
			return txErr
		}

		updatedBy, txErr := optionalUUID(requesterUserID)
		if txErr != nil {
			return txErr
		}

		now := time.Now()
		settings.AgentModeEnabled = agentModeEnabled
		settings.UpdatedBy = updatedBy
		settings.UpdatedAt = now

		if txErr = models.UpsertOrganizationAgentSettingsInTransaction(tx, settings); txErr != nil {
			return status.Error(codes.Internal, "failed to update agent settings")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.UpdateAgentSettingsResponse{
		AgentSettings: serializeAgentSettings(settings),
	}, nil
}
