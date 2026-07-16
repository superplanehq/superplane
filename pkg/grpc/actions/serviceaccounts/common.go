package serviceaccounts

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func serializeServiceAccount(sa *models.User, creator *models.User) *pb.ServiceAccount {
	out := &pb.ServiceAccount{
		Id:             sa.ID.String(),
		Name:           sa.Name,
		OrganizationId: sa.OrganizationID.String(),
		HasToken:       sa.TokenHash != "",
		CreatedAt:      timestamppb.New(sa.CreatedAt),
		UpdatedAt:      timestamppb.New(sa.UpdatedAt),
	}

	if sa.Description != nil {
		out.Description = *sa.Description
	}

	if sa.CreatedBy != nil {
		out.CreatedBy = sa.CreatedBy.String()
	}

	if creator != nil {
		out.CreatedByName = creator.Name
		if creator.Email != nil {
			out.CreatedByEmail = *creator.Email
		}
	}

	if sa.ServiceAccountExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(*sa.ServiceAccountExpiresAt)
	}

	out.CanvasIds = append(out.CanvasIds, sa.ServiceAccountCanvasIDs...)

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

func validateServiceAccountCanvasIDs(db *gorm.DB, orgID string, rawCanvasIDs []string) ([]string, error) {
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

func creatorUserForServiceAccount(db *gorm.DB, orgID string, sa *models.User) (*models.User, error) {
	if sa.CreatedBy == nil {
		return nil, nil
	}

	users, err := models.FindUsersByIDsInOrganization(db, orgID, []string{sa.CreatedBy.String()})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return &users[0], nil
}

func creatorsByIDForServiceAccounts(db *gorm.DB, orgID string, sas []models.User) (map[string]*models.User, error) {
	idSet := make(map[string]struct{})
	for i := range sas {
		if sas[i].CreatedBy != nil {
			idSet[sas[i].CreatedBy.String()] = struct{}{}
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
