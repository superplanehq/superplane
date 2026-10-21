package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	"gorm.io/gorm"
)

type productAnalyticsActor struct {
	distinctID string
	typeName   productanalytics.ActorType
}

func captureFactoryProductEvent(
	ctx context.Context,
	db *gorm.DB,
	factory *models.Factory,
	taskID uuid.UUID,
	name productanalytics.EventName,
	eventID uuid.UUID,
	properties productanalytics.Properties,
) {
	if !productanalytics.Enabled() || factory == nil {
		return
	}

	actor := resolveProductAnalyticsActor(ctx, db, factory.OrganizationID)
	productanalytics.Capture(productanalytics.Event{
		Name:           name,
		EventID:        eventID,
		DistinctID:     actor.distinctID,
		OrganizationID: factory.OrganizationID.String(),
		FactoryID:      factory.ID.String(),
		TaskID:         optionalUUID(taskID),
		ActorType:      actor.typeName,
		Source:         productanalytics.SourceFromContext(ctx, productanalytics.SourceAPI),
		Properties:     properties,
	})
}

func captureExternalFactoryProductEvent(
	db *gorm.DB,
	factory *models.Factory,
	taskID uuid.UUID,
	name productanalytics.EventName,
	eventID uuid.UUID,
	provider string,
	identity string,
	properties productanalytics.Properties,
) {
	if !productanalytics.Enabled() || factory == nil {
		return
	}

	actor := resolveExternalProductAnalyticsActor(db, factory.OrganizationID, provider, identity)
	productanalytics.Capture(productanalytics.Event{
		Name:           name,
		EventID:        eventID,
		DistinctID:     actor.distinctID,
		OrganizationID: factory.OrganizationID.String(),
		FactoryID:      factory.ID.String(),
		TaskID:         optionalUUID(taskID),
		ActorType:      actor.typeName,
		Source:         productanalytics.SourceGitHubWebhook,
		Properties:     properties,
	})
}

func resolveProductAnalyticsActor(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) productAnalyticsActor {
	if accountID, ok := authentication.GetAccountIDFromMetadata(ctx); ok {
		return productAnalyticsActor{distinctID: accountID, typeName: productanalytics.ActorUser}
	}

	if userID, ok := authentication.GetUserIdFromMetadata(ctx); ok {
		user, err := models.FindActiveUserByIDInTransaction(db, organizationID.String(), userID)
		if err == nil && user.AccountID != nil {
			return productAnalyticsActor{distinctID: user.AccountID.String(), typeName: productanalytics.ActorUser}
		}
	}

	organization := organizationID.String()
	return productAnalyticsActor{
		distinctID: productanalytics.SyntheticDistinctID(organization, productanalytics.ActorSystem, "factory"),
		typeName:   productanalytics.ActorSystem,
	}
}

func resolveExternalProductAnalyticsActor(
	db *gorm.DB,
	organizationID uuid.UUID,
	provider string,
	identity string,
) productAnalyticsActor {
	if provider == models.ProviderGitHub {
		members, err := models.ListFactoryVelocityMembers(db, organizationID)
		if err == nil {
			login := normalizedAnalyticsValue(identity)
			for _, member := range members {
				if normalizedAnalyticsValue(member.GitHubLogin) != login {
					continue
				}
				user, findErr := models.FindActiveUserByIDInTransaction(db, organizationID.String(), member.UserID.String())
				if findErr == nil && user.AccountID != nil {
					return productAnalyticsActor{distinctID: user.AccountID.String(), typeName: productanalytics.ActorUser}
				}
			}
		}
	}

	organization := organizationID.String()
	return productAnalyticsActor{
		distinctID: productanalytics.SyntheticDistinctID(organization, productanalytics.ActorExternal, provider+":"+identity),
		typeName:   productanalytics.ActorExternal,
	}
}

func optionalUUID(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

func factoryCloseEvent(result string) (productanalytics.EventName, productanalytics.Properties, bool) {
	switch result {
	case models.FactoryWorkOrderResultCompleted:
		return productanalytics.EventFactoryTaskApproved, productanalytics.Properties{"close_source": "manual"}, true
	case models.FactoryWorkOrderResultRejected, models.FactoryWorkOrderResultManualRejected:
		return productanalytics.EventFactoryTaskRejected, productanalytics.Properties{"rejection_source": "manual"}, true
	case models.FactoryWorkOrderResultPRClosed:
		return productanalytics.EventFactoryTaskRejected, productanalytics.Properties{"rejection_source": "pull_request_closed"}, true
	case models.FactoryWorkOrderResultArchived:
		return productanalytics.EventFactoryTaskArchived, productanalytics.Properties{"archive_source": "manual"}, true
	default:
		return "", nil, false
	}
}

func normalizedAnalyticsValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
