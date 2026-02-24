package organizations

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const agentOpenAIKeyCredentialName = "agent_mode_openai_api_key"

func findOrCreateOrganizationAgentSettingsInTransaction(
	tx *gorm.DB,
	organizationID string,
) (*models.OrganizationAgentSettings, error) {
	settings, err := models.FindOrganizationAgentSettingsByOrganizationIDInTransaction(tx, organizationID)
	if err == nil {
		return settings, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, status.Error(codes.Internal, "failed to load agent settings")
	}

	organizationUUID, parseErr := uuid.Parse(organizationID)
	if parseErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization id")
	}

	now := time.Now()
	settings = &models.OrganizationAgentSettings{
		OrganizationID:   organizationUUID,
		AgentModeEnabled: false,
		OpenAIKeyStatus:  models.OrganizationAgentOpenAIKeyStatusNotConfigured,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if upsertErr := models.UpsertOrganizationAgentSettingsInTransaction(tx, settings); upsertErr != nil {
		return nil, status.Error(codes.Internal, "failed to create default agent settings")
	}

	return settings, nil
}

func isAgentModeEffective(settings *models.OrganizationAgentSettings, credential *models.OrganizationAgentCredential) bool {
	if settings == nil || credential == nil {
		return false
	}
	if !settings.AgentModeEnabled {
		return false
	}

	return settings.OpenAIKeyStatus == models.OrganizationAgentOpenAIKeyStatusValid
}

func serializeAgentSettings(
	settings *models.OrganizationAgentSettings,
	credential *models.OrganizationAgentCredential,
) *pb.AgentSettings {
	configured := credential != nil

	openAIKey := &pb.AgentOpenAIKey{
		Configured: configured,
		Status:     settings.OpenAIKeyStatus,
	}

	if settings.OpenAIKeyLast4 != nil {
		openAIKey.Last4 = *settings.OpenAIKeyLast4
	} else if credential != nil {
		openAIKey.Last4 = credential.KeyLast4
	}
	if settings.OpenAIKeyValidationError != nil {
		openAIKey.ValidationError = *settings.OpenAIKeyValidationError
	}
	if settings.OpenAIKeyValidatedAt != nil {
		openAIKey.ValidatedAt = timestamppb.New(*settings.OpenAIKeyValidatedAt)
	}
	openAIKey.UpdatedAt = timestamppb.New(settings.UpdatedAt)
	if settings.UpdatedBy != nil {
		openAIKey.UpdatedBy = settings.UpdatedBy.String()
	}

	return &pb.AgentSettings{
		OrganizationId:    settings.OrganizationID.String(),
		AgentModeEnabled:  settings.AgentModeEnabled,
		AgentModeEffective: isAgentModeEffective(settings, credential),
		OpenaiKey:         openAIKey,
	}
}

func openAIKeyLast4(apiKey string) string {
	if len(apiKey) <= 4 {
		return apiKey
	}
	return apiKey[len(apiKey)-4:]
}

func optionalUUID(raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	return &parsed, nil
}
