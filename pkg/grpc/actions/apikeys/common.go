package apikeys

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/api_keys"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func serializeAPIKey(apiKey *models.User, creator *models.User) *pb.APIKey {
	out := &pb.APIKey{
		Id:             apiKey.ID.String(),
		Name:           apiKey.Name,
		OrganizationId: apiKey.OrganizationID.String(),
		HasToken:       apiKey.TokenHash != "",
		CreatedAt:      timestamppb.New(apiKey.CreatedAt),
		UpdatedAt:      timestamppb.New(apiKey.UpdatedAt),
	}

	if apiKey.Description != nil {
		out.Description = *apiKey.Description
	}

	if apiKey.CreatedBy != nil {
		out.CreatedBy = apiKey.CreatedBy.String()
	}

	if creator != nil {
		out.CreatedByName = creator.Name
		if creator.Email != nil {
			out.CreatedByEmail = *creator.Email
		}
	}

	if apiKey.APIKeyExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(*apiKey.APIKeyExpiresAt)
	}

	out.CanvasIds = append(out.CanvasIds, apiKey.APIKeyCanvasIDs...)

	return out
}

func normalizedCanvasIDs(rawCanvasIDs []string) []string {
	canvasIDs := make([]string, 0, len(rawCanvasIDs))
	for _, rawCanvasID := range rawCanvasIDs {
		canvasID := strings.TrimSpace(rawCanvasID)
		if canvasID == "" || slices.Contains(canvasIDs, canvasID) {
			continue
		}

		canvasIDs = append(canvasIDs, canvasID)
	}

	return canvasIDs
}

func validateAPIKeyCanvasIDs(db *gorm.DB, orgID string, rawCanvasIDs []string) ([]string, error) {
	canvasIDs := normalizedCanvasIDs(rawCanvasIDs)
	if len(canvasIDs) == 0 {
		return []string{}, nil
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization ID")
	}

	for _, canvasID := range canvasIDs {
		canvasUUID, err := uuid.Parse(canvasID)
		if err != nil {
			return nil, grpcerrors.InvalidArgument(err, fmt.Sprintf("invalid canvas ID %q", canvasID))
		}

		exists, err := models.CheckCanvasExistence(db, orgUUID, canvasUUID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("canvas %q does not exist in this organization", canvasID))
		}
	}

	return canvasIDs, nil
}

func creatorUserForAPIKey(db *gorm.DB, orgID string, apiKey *models.User) (*models.User, error) {
	if apiKey.CreatedBy == nil {
		return nil, nil
	}

	users, err := models.FindUsersByIDsInOrganization(db, orgID, []string{apiKey.CreatedBy.String()})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return &users[0], nil
}

func creatorsByIDForAPIKeys(db *gorm.DB, orgID string, apiKeys []models.User) (map[string]*models.User, error) {
	idSet := make(map[string]struct{})
	for i := range apiKeys {
		if apiKeys[i].CreatedBy != nil {
			idSet[apiKeys[i].CreatedBy.String()] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return map[string]*models.User{}, nil
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	users, err := models.FindUsersByIDsInOrganization(db, orgID, ids)
	if err != nil {
		return nil, err
	}

	m := make(map[string]*models.User, len(users))
	for i := range users {
		m[users[i].ID.String()] = &users[i]
	}
	return m, nil
}
