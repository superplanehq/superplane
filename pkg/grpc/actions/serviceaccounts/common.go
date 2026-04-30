package serviceaccounts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func collectDistinctCreatedByIDs(users []models.User) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	var ids []uuid.UUID
	for i := range users {
		if users[i].CreatedBy == nil {
			continue
		}
		id := *users[i].CreatedBy
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func loadCreatedByUsersByID(ids []uuid.UUID) (map[uuid.UUID]models.User, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]models.User{}, nil
	}
	users, err := models.FindMaybeDeletedUsersByIDs(ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]models.User, len(users))
	for i := range users {
		byID[users[i].ID] = users[i]
	}
	return byID, nil
}

func createdByUserRef(createdBy *uuid.UUID, byID map[uuid.UUID]models.User) *pb.UserRef {
	if createdBy == nil {
		return nil
	}
	u, ok := byID[*createdBy]
	name := ""
	if ok {
		name = u.Name
		if name == "" && u.DeletedAt.Valid {
			name = "Former member"
		}
	}
	if !ok {
		name = "Unknown user"
	}
	return &pb.UserRef{
		Id:   createdBy.String(),
		Name: name,
	}
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

	sa.CreatedByUser = createdByUserRef(user.CreatedBy, creatorsByID)

	return sa
}
