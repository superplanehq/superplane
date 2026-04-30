package serviceaccounts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func enrichServiceAccountCreators(serviceAccountUsers []models.User) (map[uuid.UUID]models.User, error) {
	seen := make(map[uuid.UUID]struct{})
	var ids []uuid.UUID
	for i := range serviceAccountUsers {
		cb := serviceAccountUsers[i].CreatedBy
		if cb == nil {
			continue
		}
		if _, ok := seen[*cb]; ok {
			continue
		}
		seen[*cb] = struct{}{}
		ids = append(ids, *cb)
	}
	if len(ids) == 0 {
		return map[uuid.UUID]models.User{}, nil
	}

	users, err := models.FindMaybeDeletedUsersByIDs(ids)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]models.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	return byID, nil
}

func serializeServiceAccount(user *models.User, creatorsByID map[uuid.UUID]models.User) *pb.ServiceAccount {
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

	if user.CreatedBy != nil {
		sa.CreatedBy = user.CreatedBy.String()
		if creatorsByID != nil {
			if creator, ok := creatorsByID[*user.CreatedBy]; ok {
				sa.CreatedByUser = &pb.UserRef{
					Id:   creator.ID.String(),
					Name: creator.Name,
				}
			} else {
				sa.CreatedByUser = &pb.UserRef{
					Id: user.CreatedBy.String(),
				}
			}
		}
	}

	return sa
}
