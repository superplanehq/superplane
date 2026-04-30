package serviceaccounts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeServiceAccount(user *models.User, creator *models.User) *pb.ServiceAccount {
	sa := &pb.ServiceAccount{
		Id:             user.ID.String(),
		Name:           user.Name,
		OrganizationId: user.OrganizationID.String(),
		HasToken:       user.TokenHash != "",
		CreatedAt:      timestamppb.New(user.CreatedAt),
		UpdatedAt:      timestamppb.New(user.UpdatedAt),
	}

	if user.Description != nil {
		sa.Description = *user.Description
	}

	if creator != nil {
		sa.CreatedBy = &pb.UserRef{
			Id:   creator.ID.String(),
			Name: creator.Name,
		}
	}

	return sa
}

func serializeServiceAccounts(orgID string, users []models.User) ([]*pb.ServiceAccount, error) {
	userIDs := make([]uuid.UUID, 0)
	for i := range users {
		if users[i].CreatedBy != nil {
			userIDs = append(userIDs, *users[i].CreatedBy)
		}
	}

	creators, err := models.FindMaybeDeletedUsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	creatorsByID := make(map[string]models.User, len(creators))
	for _, c := range creators {
		if c.OrganizationID.String() != orgID {
			continue
		}
		creatorsByID[c.ID.String()] = c
	}

	out := make([]*pb.ServiceAccount, len(users))
	for i := range users {
		var creator *models.User
		if users[i].CreatedBy != nil {
			if u, ok := creatorsByID[users[i].CreatedBy.String()]; ok {
				creator = &u
			}
		}
		out[i] = serializeServiceAccount(&users[i], creator)
	}

	return out, nil
}
