package serviceaccounts

import (
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeServiceAccounts(users []models.User) ([]*pb.ServiceAccount, error) {
	ids := make([]uuid.UUID, 0)
	seen := make(map[string]struct{})
	for i := range users {
		if users[i].CreatedBy == nil {
			continue
		}
		key := users[i].CreatedBy.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ids = append(ids, *users[i].CreatedBy)
	}

	creators, err := models.FindMaybeDeletedUsersByIDs(ids)
	if err != nil {
		return nil, err
	}

	creatorsByID := make(map[string]models.User, len(creators))
	for _, u := range creators {
		creatorsByID[u.ID.String()] = u
	}

	out := make([]*pb.ServiceAccount, len(users))
	for i := range users {
		out[i] = serializeServiceAccount(&users[i], creatorsByID)
	}
	return out, nil
}

func serializeServiceAccount(user *models.User, creatorsByID map[string]models.User) *pb.ServiceAccount {
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
			if creator, ok := creatorsByID[user.CreatedBy.String()]; ok {
				sa.CreatedByUser = creatorUserRef(&creator)
			} else {
				sa.CreatedByUser = &pb.UserRef{Id: user.CreatedBy.String(), Name: "Unknown"}
			}
		}
	}

	return sa
}

func creatorUserRef(u *models.User) *pb.UserRef {
	name := strings.TrimSpace(u.Name)
	if name == "" {
		if u.DeletedAt.Valid {
			name = "Former member"
		} else {
			name = "Unknown"
		}
	}
	return &pb.UserRef{Id: u.ID.String(), Name: name}
}

func serializeServiceAccountSingle(user *models.User) (*pb.ServiceAccount, error) {
	list, err := serializeServiceAccounts([]models.User{*user})
	if err != nil {
		return nil, err
	}
	return list[0], nil
}
