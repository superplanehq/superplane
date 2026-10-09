package factories

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
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

type forkMode int

const (
	forkModeIntake forkMode = iota
	forkModePlan
)

var planningScoreCheckKeys = []string{
	models.PlanningClarityCheckKey,
	models.PlanningComplexityCheckKey,
	models.PlanningVerifiabilityCheckKey,
}

func ForkWorkOrder(ctx context.Context, organizationID string, req *pb.ForkWorkOrderRequest) (*pb.ForkWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to fork work order")
	}
	mode, err := parseForkMode(req.GetMode())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to fork work order")
	}
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to fork work order")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to fork work order")
	}
	source, err := findWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to fork work order")
	}

	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	err = db.Transaction(func(tx *gorm.DB) error {
		created, forkErr := forkWorkOrder(ctx, tx, factory, source, actorID, mode)
		order = created.order
		bound = created.bound
		return forkErr
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factory.ID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after fork")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to fork work order")
	}

	if mode == forkModeIntake {
		workersctx.EmitWorkOrderCreated(db, factory, order)
	}
	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to fork work order")
	}
	return &pb.ForkWorkOrderResponse{Order: serialized}, nil
}

type forkedWorkOrder struct {
	order *models.FactoryWorkOrder
	bound storedfiles.BindResult
}

func forkWorkOrder(
	ctx context.Context,
	tx *gorm.DB,
	factory *models.Factory,
	source *models.FactoryWorkOrder,
	actorID uuid.UUID,
	mode forkMode,
) (forkedWorkOrder, error) {
	plan, err := sourcePlan(tx, source, mode)
	if err != nil {
		return forkedWorkOrder{}, err
	}

	order, err := factory.CreateWorkOrder(tx, source.Title, source.Description, &actorID, []uuid.UUID{actorID}, nil)
	if err != nil {
		return forkedWorkOrder{}, err
	}
	if err := copyRepositorySnapshot(tx, source, order); err != nil {
		return forkedWorkOrder{}, err
	}

	copied, err := storedfiles.CopyReadyTaskFiles(
		ctx,
		tx,
		blob.Current(),
		source.ID,
		order.ID,
		actorID,
		order.Description,
		plan,
	)
	result := forkedWorkOrder{order: order, bound: storedfiles.BindResult{CopiedKeys: copied.CopiedKeys}}
	if err != nil {
		return result, err
	}
	if copied.Description != order.Description {
		if err := order.UpdateContent(tx, nil, &copied.Description); err != nil {
			return result, err
		}
	}
	bound, err := storedfiles.BindDescriptionFiles(
		ctx,
		tx,
		blob.Current(),
		factory.OrganizationID,
		factory.ID,
		order.ID,
		order.Description,
	)
	result.bound.StaleKeys = bound.StaleKeys
	result.bound.CopiedKeys = append(result.bound.CopiedKeys, bound.CopiedKeys...)
	if err != nil {
		return result, err
	}
	if mode == forkModePlan {
		if err := copyPlan(tx, source, order, copied); err != nil {
			return result, err
		}
	}
	return result, nil
}

func parseForkMode(mode pb.ForkWorkOrderRequest_Mode) (forkMode, error) {
	switch mode {
	case pb.ForkWorkOrderRequest_MODE_INTAKE:
		return forkModeIntake, nil
	case pb.ForkWorkOrderRequest_MODE_PLAN:
		return forkModePlan, nil
	default:
		return 0, invalidArgument("mode is required")
	}
}

func sourcePlan(tx *gorm.DB, source *models.FactoryWorkOrder, mode forkMode) (string, error) {
	if mode != forkModePlan {
		return "", nil
	}
	artifact, err := source.FindArtifactByKey(tx, models.PlanningSpecArtifactKey+":"+source.ID.String())
	if err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderArtifactNotFound) {
			return "", invalidArgument("this task has no plan")
		}
		return "", err
	}
	body := artifactMarkdownBody(artifact)
	if body == "" {
		return "", invalidArgument("this task has no plan")
	}
	return body, nil
}

func artifactMarkdownBody(artifact *models.FactoryWorkOrderArtifact) string {
	if artifact == nil || len(artifact.Data) == 0 {
		return ""
	}
	var data map[string]any
	if json.Unmarshal(artifact.Data, &data) != nil {
		return ""
	}
	body, _ := data["body"].(string)
	return strings.TrimSpace(body)
}

func copyRepositorySnapshot(tx *gorm.DB, source, order *models.FactoryWorkOrder) error {
	repository := trimmedString(source.Repository)
	branch := trimmedString(source.DefaultBranch)
	provider := trimmedString(source.VCSProvider)
	if repository == "" || branch == "" || provider == "" {
		return nil
	}
	if err := tx.Model(order).Updates(map[string]any{
		"repository":     repository,
		"default_branch": branch,
		"vcs_provider":   provider,
	}).Error; err != nil {
		return err
	}
	order.Repository = &repository
	order.DefaultBranch = &branch
	order.VCSProvider = &provider
	return nil
}

func trimmedString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func copyPlan(tx *gorm.DB, source, order *models.FactoryWorkOrder, copied storedfiles.CopiedTaskFiles) error {
	if err := order.StorePlanningSpec(tx, copied.Plan); err != nil {
		return err
	}
	checks, err := source.ListChecks(tx)
	if err != nil {
		return err
	}
	for i := range checks {
		if !isPlanningScoreCheck(checks[i].Key) {
			continue
		}
		if _, err := order.ReportCheck(tx, models.FactoryWorkOrderCheckParams{
			Key:      checks[i].Key,
			Name:     checks[i].Name,
			Score:    checks[i].Score,
			MaxScore: checks[i].MaxScore,
			Format:   checks[i].Format,
			Level:    checks[i].Level,
			Summary:  checks[i].Summary,
			Analysis: copied.Rewrite(checks[i].Analysis),
		}); err != nil {
			return err
		}
	}
	return nil
}

func isPlanningScoreCheck(key string) bool {
	return slices.Contains(planningScoreCheckKeys, key)
}
