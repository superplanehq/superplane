package organizations

import (
	"context"
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/grpc/organizationref"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListOrganizationCreditGrants(
	ctx context.Context,
	orgID string,
	_ *pb.ListOrganizationCreditGrantsRequest,
) (*pb.ListOrganizationCreditGrantsResponse, error) {
	organizationID, err := organizationref.Resolve(ctx, orgID)
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)
	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list organization credit grants")
	}

	actorNames, err := billingmodels.CreditGrantActorNames(db, grants)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list organization credit grants")
	}

	out := make([]*pb.OrganizationCreditGrant, 0, len(grants))
	for _, grant := range grants {
		out = append(out, serializeOrganizationCreditGrant(grant, actorNames))
	}

	return &pb.ListOrganizationCreditGrantsResponse{Grants: out}, nil
}

func serializeOrganizationCreditGrant(
	grant billingmodels.OrganizationLLMCreditGrant,
	actorNames map[uuid.UUID]string,
) *pb.OrganizationCreditGrant {
	item := &pb.OrganizationCreditGrant{
		Id:          grant.ID.String(),
		Kind:        grant.Kind,
		AmountCents: billingmodels.SignedMicrosToCents(grant.AmountMicros),
		Note:        grant.Note,
		CreatedAt:   timestamppb.New(grant.CreatedAt),
		ExpiresAt:   protoTimestamp(grant.ExpiresAt),
	}
	if grant.ActorAccountID != nil {
		item.ActorName = actorNames[*grant.ActorAccountID]
	}
	if grant.PolarOrderID != nil {
		item.PolarOrderId = *grant.PolarOrderID
	}
	return item
}

func protoTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil || value.IsZero() {
		return nil
	}
	return timestamppb.New(*value)
}
