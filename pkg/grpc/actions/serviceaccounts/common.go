package serviceaccounts

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	return out
}

func creatorUserForServiceAccount(orgID string, sa *models.User) (*models.User, error) {
	if sa.CreatedBy == nil {
		return nil, nil
	}

	users, err := models.FindUsersByIDsInOrganization(orgID, []string{sa.CreatedBy.String()})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return &users[0], nil
}

func creatorsByIDForServiceAccounts(orgID string, sas []models.User) (map[string]*models.User, error) {
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

	users, err := models.FindUsersByIDsInOrganization(orgID, ids)
	if err != nil {
		return nil, err
	}

	m := make(map[string]*models.User, len(users))
	for i := range users {
		m[users[i].ID.String()] = &users[i]
	}
	return m, nil
}
