package factories

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func ImportFactoryIntakeItem(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.ImportFactoryIntakeItemRequest,
) (*pb.ImportFactoryIntakeItemResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	intakeID, err := parseIntakeID(req.GetIntakeId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	itemID := strings.TrimSpace(req.GetItemId())
	if itemID == "" {
		return nil, factoryErrorToStatus(invalidArgument("item id is required"), "failed to import factory intake item")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to import factory intake item")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	intake, err := factory.FindIntake(db, intakeID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	source, err := deps.itemSource(ctx, db, intake)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	item, err := source.Get(ctx, itemID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	origin := models.WorkOrderOrigin{URL: item.URL, Label: models.OriginLabelFromURL(item.URL)}
	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	err = db.Transaction(func(tx *gorm.DB) error {
		created, err := factory.CreateWorkOrderWithOrigin(
			tx,
			item.Title,
			item.Body,
			&createdByID,
			[]uuid.UUID{createdByID},
			nil,
			origin,
		)
		if err != nil {
			return err
		}
		order = created
		body := item.Body
		if fetcher, ok := source.(interface {
			RemoteFetch(context.Context, *http.Request) (*http.Response, error)
		}); ok {
			ingested, ingestErr := storedfiles.IngestRemoteImages(
				ctx,
				tx,
				blob.Current(),
				fetcher.RemoteFetch,
				blob.IsGitHubImageURL,
				orgID,
				factory.ID,
				order.ID,
				&createdByID,
				body,
			)
			bound.CopiedKeys = append(bound.CopiedKeys, ingested.ObjectKeys...)
			if ingestErr == nil {
				body = ingested.Markdown
			}
			if body != order.Description {
				if err := order.UpdateContent(tx, nil, &body); err != nil {
					return err
				}
			}
		}
		result, bindErr := storedfiles.BindDescriptionFiles(ctx, tx, blob.Current(), orgID, factory.ID, order.ID, order.Description)
		bound.StaleKeys = append(bound.StaleKeys, result.StaleKeys...)
		bound.CopiedKeys = append(bound.CopiedKeys, result.CopiedKeys...)
		return bindErr
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factory.ID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after bind")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	workersctx.EmitWorkOrderCreated(db, factory, order)

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import factory intake item")
	}

	return &pb.ImportFactoryIntakeItemResponse{Order: serialized}, nil
}
