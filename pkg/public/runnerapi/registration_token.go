package runnerapi

import (
	"fmt"

	jwtlib "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

const RegistrationAudience = "superplane-runner-registration"

type RegistrationClaims struct {
	FleetID string `json:"fleet_id"`
	TaskID  string `json:"task_id,omitempty"`
	jwtlib.RegisteredClaims
}

func MintRegistrationToken(
	signer *jwt.Signer,
	runner *models.Runner,
	registration *models.RunnerRegistration,
	fleetID string,
	taskID *uuid.UUID,
) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("JWT signer is required")
	}

	claims := RegistrationClaims{
		FleetID: fleetID,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Audience:  jwtlib.ClaimStrings{RegistrationAudience},
			ExpiresAt: jwtlib.NewNumericDate(registration.ExpiresAt),
			ID:        registration.JTI.String(),
			IssuedAt:  jwtlib.NewNumericDate(registration.CreatedAt),
			NotBefore: jwtlib.NewNumericDate(registration.CreatedAt),
			Subject:   runner.ID.String(),
		},
	}
	if taskID != nil {
		claims.TaskID = taskID.String()
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	value, err := token.SignedString([]byte(signer.Secret))
	if err != nil {
		return "", fmt.Errorf("sign runner registration token: %w", err)
	}
	return value, nil
}
