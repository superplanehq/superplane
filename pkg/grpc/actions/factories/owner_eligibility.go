package factories

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func requireCodexConnection(tx *gorm.DB, factory *models.Factory, callerID uuid.UUID) error {
	if _, err := models.ActiveCodexMember(tx, factory.OrganizationID, callerID); err != nil {
		return err
	}
	eligible, err := models.SubscriptionOwnerEligible(tx, factory.OrganizationID, factory.ID, callerID)
	if err != nil {
		return err
	}
	if !eligible {
		return errCodexConnectionRequired
	}
	return nil
}

func requireCallerCodexConnection(tx *gorm.DB, factory *models.Factory, callerID uuid.UUID) error {
	if !factory.OnboardingConfigValue().CodexSubscription {
		return nil
	}
	return requireCodexConnection(tx, factory, callerID)
}

// Check ownership restrictions before offering connection setup. A reconnect
// must never allow a member to take a task from an active runner.
func updateInteractiveWorkOrderAssignees(tx *gorm.DB, factory *models.Factory, order *models.FactoryWorkOrder, ids []uuid.UUID, callerID uuid.UUID) error {
	err := order.UpdateAssignees(tx, ids, callerID)
	if err != nil && !errors.Is(err, models.ErrSubscriptionOwnerInvalid) {
		return err
	}
	if slices.Contains(ids, callerID) {
		if connectionErr := requireCallerCodexConnection(tx, factory, callerID); connectionErr != nil {
			return connectionErr
		}
	}
	return err
}

// assignWorkOrderStarter makes the caller the only owner of the task. Only
// subscription workspaces need this, because the caller's Codex connection
// runs the task. Other workspaces keep the saved owners.
func assignWorkOrderStarter(tx *gorm.DB, factory *models.Factory, order *models.FactoryWorkOrder, actor *uuid.UUID) error {
	if actor == nil || !factory.OnboardingConfigValue().CodexSubscription {
		return nil
	}
	owners, err := order.ListAssignees(tx)
	if err != nil {
		return err
	}
	if len(owners) == 1 && owners[0].UserID == *actor {
		return requireCallerCodexConnection(tx, factory, *actor)
	}
	return updateInteractiveWorkOrderAssignees(tx, factory, order, []uuid.UUID{*actor}, *actor)
}

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
	response := &pb.ListFactoryOwnerEligibilityResponse{Subscription: (factory.OnboardingConfigValue().CodexSubscription || req.GetProspectiveSubscription()), OwnershipChangeAllowed: true}
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
			if err != nil {
				return nil, factoryErrorToStatus(err, "failed to load owner eligibility")
			}
			if !member.Eligible {
				member.Reason = "Connect Codex in this organization."
			}
		}
		response.Members = append(response.Members, member)
	}
	return response, nil
}
