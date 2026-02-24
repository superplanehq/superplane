package organizations

import (
	"errors"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteAgentOpenAIKey(
	orgID string,
	requesterUserID string,
) (*pb.DeleteAgentOpenAIKeyResponse, error) {
	var settings *models.OrganizationAgentSettings
	var credential *models.OrganizationAgentCredential

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

		if txErr = models.DeleteOrganizationAgentCredentialByOrganizationIDInTransaction(tx, orgID); txErr != nil {
			return status.Error(codes.Internal, "failed to delete OpenAI API key")
		}

		now := time.Now()
		settings.OpenAIKeyLast4 = nil
		settings.OpenAIKeyStatus = models.OrganizationAgentOpenAIKeyStatusNotConfigured
		settings.OpenAIKeyValidatedAt = nil
		settings.OpenAIKeyValidationError = nil
		settings.UpdatedBy = updatedBy
		settings.UpdatedAt = now

		if txErr = models.UpsertOrganizationAgentSettingsInTransaction(tx, settings); txErr != nil {
			return status.Error(codes.Internal, "failed to update agent settings")
		}

		credential, txErr = models.FindOrganizationAgentCredentialByOrganizationIDInTransaction(tx, orgID)
		if txErr != nil && !errors.Is(txErr, gorm.ErrRecordNotFound) {
			return status.Error(codes.Internal, "failed to load agent credential")
		}
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			credential = nil
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.DeleteAgentOpenAIKeyResponse{
		AgentSettings: serializeAgentSettings(settings, credential),
	}, nil
}
