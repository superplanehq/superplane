package organizations

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const agentCredentialEncryptionKeyID = "default"

func SetAgentOpenAIKey(
	ctx context.Context,
	encryptor crypto.Encryptor,
	orgID string,
	requesterUserID string,
	apiKey string,
	validate bool,
) (*pb.SetAgentOpenAIKeyResponse, error) {
	apiKey = strings.TrimSpace(apiKey)
	if !isOpenAIKeyFormatValid(apiKey) {
		return nil, status.Error(codes.InvalidArgument, "invalid OpenAI API key format")
	}

	updatedBy, err := optionalUUID(requesterUserID)
	if err != nil {
		return nil, err
	}

	validationStatus := models.OrganizationAgentOpenAIKeyStatusUnchecked
	var validationError *string
	var validatedAt *time.Time
	if validate && shouldValidateOpenAIKeyLive() {
		validationStatus, validationError, validatedAt = validateOpenAIKeyLive(ctx, apiKey)
	}
	if validationStatus == models.OrganizationAgentOpenAIKeyStatusInvalid {
		msg := "OpenAI rejected the provided API key"
		if validationError != nil && *validationError != "" {
			msg = *validationError
		}
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	ciphertext, err := encryptor.Encrypt(ctx, []byte(apiKey), []byte(agentOpenAIKeyCredentialName))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encrypt OpenAI API key")
	}

	last4 := openAIKeyLast4(apiKey)
	now := time.Now()
	var settings *models.OrganizationAgentSettings
	encryptionKeyID := agentCredentialEncryptionKeyID

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error

		settings, txErr = findOrCreateOrganizationAgentSettingsInTransaction(tx, orgID)
		if txErr != nil {
			return txErr
		}

		settings.OpenAIApiKeyCiphertext = ciphertext
		settings.OpenAIKeyEncryptionKeyID = &encryptionKeyID
		settings.OpenAIKeyLast4 = &last4
		settings.OpenAIKeyStatus = validationStatus
		settings.OpenAIKeyValidatedAt = validatedAt
		settings.OpenAIKeyValidationError = validationError
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

	return &pb.SetAgentOpenAIKeyResponse{
		AgentSettings: serializeAgentSettings(settings),
	}, nil
}

func isOpenAIKeyFormatValid(apiKey string) bool {
	return strings.TrimSpace(apiKey) != ""
}

func validateOpenAIKeyLive(
	ctx context.Context,
	apiKey string,
) (string, *string, *time.Time) {
	now := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		msg := "unable to validate OpenAI key right now"
		return models.OrganizationAgentOpenAIKeyStatusUnchecked, &msg, &now
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		msg := "unable to validate OpenAI key right now"
		return models.OrganizationAgentOpenAIKeyStatusUnchecked, &msg, &now
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return models.OrganizationAgentOpenAIKeyStatusValid, nil, &now
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		msg := "OpenAI rejected the provided API key"
		return models.OrganizationAgentOpenAIKeyStatusInvalid, &msg, &now
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		msg := "unable to validate OpenAI key right now"
		return models.OrganizationAgentOpenAIKeyStatusUnchecked, &msg, &now
	}

	msg := "OpenAI rejected the provided API key"
	return models.OrganizationAgentOpenAIKeyStatusInvalid, &msg, &now
}

func shouldValidateOpenAIKeyLive() bool {
	// E2E runs against superplane_test and should not depend on external OpenAI availability.
	return strings.TrimSpace(os.Getenv("DB_NAME")) != "superplane_test"
}
