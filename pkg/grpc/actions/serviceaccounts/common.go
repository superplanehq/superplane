package serviceaccounts

import (
	"github.com/google/uuid"
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

	out.CreatedByUser = creatorRefFromUser(creator)

	return out
}

func creatorRefFromUser(creator *models.User) *pb.UserRef {
	if creator == nil {
		return nil
	}

	name := creator.Name
	if creator.DeletedAt.Valid {
		name = name + " (removed)"
	}

	return &pb.UserRef{
		Id:   creator.ID.String(),
		Name: name,
	}
}

func distinctCreatedByIDs(users []models.User) []uuid.UUID {
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
