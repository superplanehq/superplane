package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func resolveModelCatalogOwner(ctx context.Context, tx *gorm.DB, f *models.Factory, orderRef string) (uuid.UUID, error) {
	if !f.OnboardingConfigValue().CodexSubscription {
		return uuid.Nil, nil
	}
	callerID, err := codexCallerID(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return f.ResolveSubscriptionCatalogOwner(tx, callerID, orderRef)
}

func subscriptionOrderOwnerID(tx *gorm.DB, order *models.FactoryWorkOrder) uuid.UUID {
	owners, err := order.ListAssignees(tx)
	if err != nil || len(owners) != 1 {
		return uuid.Nil
	}
	return owners[0].UserID
}

func ListFactoryOwnerEligibility(ctx context.Context, organizationID string, req *pb.ListFactoryOwnerEligibilityRequest) (*pb.ListFactoryOwnerEligibilityResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load owner eligibility")
	}
	response := &pb.ListFactoryOwnerEligibilityResponse{Subscription: factory.OnboardingConfigValue().CodexSubscription, OwnershipChangeAllowed: true}
	if req.GetOrderId() != "" && response.Subscription {
		order, err := findWorkOrder(db, factory, req.GetOrderId())
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to load task")
		}
		blocked, err := order.SubscriptionOwnershipBlocked(db)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to load task ownership restrictions")
		}
		response.OwnershipChangeAllowed = !blocked
		if blocked {
			response.RestrictionReason = models.ErrSubscriptionOwnerBusy.Error()
		}
	}
	users, err := models.ListHumanUsersInOrganization(db, orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load workspace members")
	}
	for _, user := range users {
		member := &pb.FactoryOwnerEligibility{UserId: user.ID.String(), Eligible: true}
		if response.Subscription {
			member.Eligible, err = models.SubscriptionOwnerEligible(db, orgID, factory.ID, user.ID)
			if err != nil && !errors.Is(err, models.ErrCodexConnectionUnavailable) {
				return nil, factoryErrorToStatus(err, "failed to load owner eligibility")
			}
			if !member.Eligible {
				member.Reason = "Connect a ChatGPT subscription in this workspace."
			}
		}
		response.Members = append(response.Members, member)
	}
	return response, nil
}
