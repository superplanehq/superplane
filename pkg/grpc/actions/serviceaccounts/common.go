package serviceaccounts

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func serializeServiceAccounts(users []models.User) ([]*pb.ServiceAccount, error) {
	userIDs := []uuid.UUID{}
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
	for _, u := range creators {
		creatorsByID[u.ID.String()] = u
	}

	out := make([]*pb.ServiceAccount, len(users))
	for i := range users {
		out[i] = serializeServiceAccountWithCreator(&users[i], creatorsByID)
	}
	return out, nil
}

func serializeServiceAccount(user *models.User) *pb.ServiceAccount {
	return serializeServiceAccountWithCreator(user, nil)
}

func serializeServiceAccountWithCreator(user *models.User, creatorsByID map[string]models.User) *pb.ServiceAccount {
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
				sa.CreatedByUser = userRefFromCreator(&creator)
			}
		}
	}

	return sa
}

func userRefFromCreator(creator *models.User) *pb.UserRef {
	if creator == nil {
		return nil
	}

	name := creator.Name
	if creator.DeletedAt.Valid && name == "" {
		name = "Former member"
	}
	if name == "" {
		name = "Unknown"
	}

	return &pb.UserRef{
		Id:   creator.ID.String(),
		Name: name,
	}
}

func attachCreatorUserRef(sa *pb.ServiceAccount, orgID string) error {
	if sa == nil || sa.CreatedBy == "" || sa.CreatedByUser != nil {
		return nil
	}

	creator, err := models.FindMaybeDeletedUserByID(orgID, sa.CreatedBy)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	sa.CreatedByUser = userRefFromCreator(creator)
	return nil
}
