package factories

import (
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var intakeSourcesWithDefaultOwner = []string{
	models.FactoryIntakeSourceGitHubIssues,
	models.FactoryIntakeSourceSentryExceptions,
	models.FactoryIntakeSourceJiraIssues,
	models.FactoryIntakeSourceDependabotAlerts,
	models.FactoryIntakeSourcePagerDutyIncidents,
	models.FactoryIntakeSourceProductiveTasks,
	models.FactoryIntakeSourceDatadog,
	models.FactoryIntakeSourceLinearIssues,
}

func intakeSourceSupportsDefaultOwner(source string) bool {
	return slices.Contains(intakeSourcesWithDefaultOwner, source)
}

func validateIntakeDefaultOwner(tx *gorm.DB, factory *models.Factory, source, currentOwner string, settings *pb.FactoryIntake_Settings) error {
	subscription := factory.OnboardingConfigValue().CodexSubscription
	ownerID := currentOwner
	if settings != nil && settings.DefaultOwnerId != nil {
		if !intakeSourceSupportsDefaultOwner(source) {
			return invalidArgument("default task owner is not supported for this intake")
		}
		ownerID = settings.GetDefaultOwnerId()
	}
	if ownerID == "" {
		if subscription {
			return invalidArgument("Select a default task owner with Codex connected")
		}
		return nil
	}
	if ownerID == currentOwner && !subscription {
		return nil
	}
	id, err := uuid.Parse(ownerID)
	if err != nil {
		return invalidArgument("invalid default task owner id")
	}
	user, err := models.FindActiveUserByIDInTransaction(tx, factory.OrganizationID.String(), id.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return invalidArgument("default task owner must be an active organization member")
	}
	if err != nil {
		return err
	}
	if !user.IsHuman() {
		return invalidArgument("default task owner must be a person")
	}
	if !subscription || ownerID == currentOwner {
		return nil
	}
	connected, err := models.SubscriptionOwnerEligible(tx, factory.OrganizationID, factory.ID, id)
	if err != nil {
		return err
	}
	if !connected {
		return invalidArgument("Select a default task owner with Codex connected")
	}
	return nil
}

// Explicit source confirmation replaces every intake default owner. The actor
// remains the audit author even when another member supplies the subscription.
func replaceSubscriptionIntakeOwners(tx *gorm.DB, factory *models.Factory, ownerID, authorID uuid.UUID) ([]uuid.UUID, error) {
	intakes, err := factory.ListIntakes(tx)
	if err != nil {
		return nil, err
	}
	sources := map[uuid.UUID]string{}
	for _, intake := range intakes {
		sources[intake.CanvasID] = intake.Source
	}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	changed := []uuid.UUID{}
	for i := range canvases {
		canvas := &canvases[i]
		if canvas.LiveVersionID == nil {
			continue
		}
		live, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if err != nil {
			return nil, err
		}
		source, intake := sources[canvas.ID]
		scheduled := models.FactoryAppTemplateID(live.Nodes) == scheduledIntakeTemplateID
		if !intake && !scheduled {
			continue
		}
		target := "create-task"
		if intake {
			target = resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges}).CreateNodeID
		}
		creators := 0
		for _, candidate := range live.Nodes {
			if candidate.ComponentName() == "createWorkOrder" {
				creators++
			}
		}
		if creators != 1 {
			return nil, invalidArgument("intake automation must have exactly one task creation node")
		}
		node := findIntakeNode(live.Nodes, target)
		if node == nil || node.ComponentName() != "createWorkOrder" {
			return nil, invalidArgument("intake automation has no task creation node to update")
		}

		nodes := slices.Clone(live.Nodes)
		for j := range nodes {
			if nodes[j].ID != target {
				continue
			}
			nodes[j].Configuration = maps.Clone(nodes[j].Configuration)
			if nodes[j].Configuration == nil {
				nodes[j].Configuration = map[string]any{}
			}
			nodes[j].Configuration["defaultOwnerId"] = ownerID.String()
		}
		version, err := models.CreateCommitVersionWithSpecInTransaction(tx, canvas.ID, authorID, "Set intake task owner", nodes, nil)
		if err != nil {
			return nil, err
		}
		if err := models.PromoteToLiveInTransaction(tx, version, nodes, live.Edges); err != nil {
			return nil, err
		}
		if err := tx.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, target).Updates(map[string]any{"configuration": datatypes.NewJSONType(findIntakeNode(nodes, target).Configuration), "updated_at": time.Now()}).Error; err != nil {
			return nil, err
		}
		changed = append(changed, canvas.ID)
	}
	return changed, nil
}

// seedSubscriptionIntakeOwner preserves an explicit choice. An unavailable
// saved default requires a new selection rather than another member's account.
func seedSubscriptionIntakeOwner(factory *models.Factory, settings *pb.FactoryIntake_Settings) *pb.FactoryIntake_Settings {
	if !factory.OnboardingConfigValue().CodexSubscription {
		return settings
	}
	if settings == nil {
		settings = &pb.FactoryIntake_Settings{}
	}
	if settings.DefaultOwnerId == nil {
		ownerID := factory.OnboardingConfigValue().CodexDefaultOwnerID
		settings.DefaultOwnerId = &ownerID
	}
	return settings
}
