package factories

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// BacklogTemplateUpgradeResult reports the canvases changed or left alone.
type BacklogTemplateUpgradeResult struct {
	Upgraded int
	Skipped  int
}

// UpgradeDefaultBacklogTemplates upgrades generated version 1 and 2 Backlog
// graphs to the current template, and refreshes a Refine Task prompt that
// still matches an earlier default. Version 1 and 2 graphs still have the
// Analyze / intent.md path. SuperPlane replaces that path. A graph that no
// longer matches those node sets is user-owned. An edited Refine Task prompt
// stays as the user wrote it.
func UpgradeDefaultBacklogTemplates(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID uuid.UUID,
) (BacklogTemplateUpgradeResult, error) {
	db := database.DB(ctx)
	factoryModels, err := models.ListFactories(db, organizationID)
	if err != nil {
		return BacklogTemplateUpgradeResult{}, err
	}

	result := BacklogTemplateUpgradeResult{}
	for i := range factoryModels {
		factoryResult, err := upgradeFactoryBacklogTemplates(ctx, deps, &factoryModels[i])
		result.Upgraded += factoryResult.Upgraded
		result.Skipped += factoryResult.Skipped
		if err != nil {
			return result, fmt.Errorf("upgrade Backlog templates for factory %s: %w", factoryModels[i].ID, err)
		}
	}
	return result, nil
}

func upgradeFactoryBacklogTemplates(
	ctx context.Context,
	deps IntakeDependencies,
	factoryModel *models.Factory,
) (BacklogTemplateUpgradeResult, error) {
	canvasModels, err := factoryModel.ListCanvases(database.DB(ctx))
	if err != nil {
		return BacklogTemplateUpgradeResult{}, err
	}

	result := BacklogTemplateUpgradeResult{}
	for i := range canvasModels {
		upgraded, backlog, err := upgradeBacklogTemplate(ctx, deps, factoryModel, &canvasModels[i])
		if err != nil {
			return result, err
		}
		if upgraded {
			result.Upgraded++
		} else if backlog {
			result.Skipped++
		}
	}
	return result, nil
}

func upgradeBacklogTemplate(
	ctx context.Context,
	deps IntakeDependencies,
	factoryModel *models.Factory,
	canvasModel *models.Canvas,
) (upgraded bool, backlog bool, err error) {
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockCanvasForUpdate(tx, canvasModel.OrganizationID, canvasModel.ID)
		if err != nil {
			return err
		}
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, locked)
		if err != nil {
			return err
		}
		backlog = models.IsBacklogFactoryApp(liveVersion.Nodes, liveVersion.Edges)
		if !backlog {
			return nil
		}
		if isDefaultLegacyBacklog(liveVersion.Nodes, liveVersion.Edges) {
			upgraded = true
			return upgradeLegacyBacklog(ctx, tx, deps, locked, liveVersion)
		}
		nodes := cloneBacklogNodes(liveVersion.Nodes)
		if !refreshBacklogRefinePrompt(nodes) {
			return nil
		}
		upgraded = true
		return publishBacklogUpgrade(ctx, tx, deps, locked, liveVersion.OwnerID, "Refresh Backlog refine prompt", nodes, liveVersion.Edges)
	})
	if err != nil {
		return false, backlog, err
	}
	return upgraded, backlog, nil
}

// upgradeLegacyBacklog replaces an untouched version 1 graph with the current
// template and stamps the new version on the trigger.
func upgradeLegacyBacklog(
	ctx context.Context,
	tx *gorm.DB,
	deps IntakeDependencies,
	locked *models.Canvas,
	liveVersion *models.CanvasVersion,
) error {
	next := buildBacklogCanvas(backlogCanvasRequest{
		Name:       locked.Name,
		Agent:      intakeAgentFromCanvasNodes(liveVersion.Nodes),
		GitHubName: backlogGitHubIntegrationName(liveVersion.Nodes),
	})
	nodes, edges, err := next.Parse(deps.Registry, locked.OrganizationID.String())
	if err != nil {
		return fmt.Errorf("parse Backlog template version %d: %w", backlogTemplateVersion, err)
	}
	preserveBacklogNodePositions(liveVersion.Nodes, nodes)

	if err := publishBacklogUpgrade(ctx, tx, deps, locked, liveVersion.OwnerID, "Upgrade Backlog task refinement", nodes, edges); err != nil {
		return err
	}
	return locked.StampFactoryAppTemplate(
		tx,
		backlogTriggerNodeID,
		models.FactoryAppTemplateBacklogID,
		backlogTemplateVersion,
	)
}

func publishBacklogUpgrade(
	ctx context.Context,
	tx *gorm.DB,
	deps IntakeDependencies,
	locked *models.Canvas,
	ownerID *uuid.UUID,
	message string,
	nodes []models.Node,
	edges []models.Edge,
) error {
	return canvases.PublishGeneratedCanvasNodesWithOwner(
		ctx,
		tx,
		locked,
		ownerID,
		message,
		nodes,
		edges,
		changesets.CanvasPublisherOptions{
			Registry:       deps.Registry,
			OrgID:          locked.OrganizationID,
			Encryptor:      deps.Encryptor,
			AuthService:    deps.AuthService,
			WebhookBaseURL: deps.WebhookBaseURL,
			GitProvider:    deps.GitProvider,
		},
	)
}

// cloneBacklogNodes deep-copies nodes so a prompt refresh does not write into
// the live version that is still referenced by the caller.
func cloneBacklogNodes(nodes []models.Node) []models.Node {
	encoded, _ := json.Marshal(nodes)
	var cloned []models.Node
	_ = json.Unmarshal(encoded, &cloned)
	return cloned
}

func isDefaultLegacyBacklog(nodes []models.Node, edges []models.Edge) bool {
	if !models.IsBacklogFactoryApp(nodes, edges) || backlogTemplateVersionFrom(nodes) >= backlogTemplateVersion {
		return false
	}
	return models.IsLegacyAnalyzeBacklog(nodes)
}

func backlogTemplateVersionFrom(nodes []models.Node) int {
	for _, node := range nodes {
		metadata, ok := node.Metadata[factoryTemplateMetadataKey].(map[string]any)
		if !ok || metadata["id"] != models.FactoryAppTemplateBacklogID {
			continue
		}
		switch version := metadata["version"].(type) {
		case int:
			return version
		case float64:
			return int(version)
		case json.Number:
			value, _ := version.Int64()
			return int(value)
		}
	}
	return 0
}

func backlogBehaviorNodes(nodes []models.Node) []models.Node {
	encoded, _ := json.Marshal(nodes)
	var normalized []models.Node
	_ = json.Unmarshal(encoded, &normalized)
	for i := range normalized {
		normalized[i].Metadata = nil
		normalized[i].Position = models.Position{}
		normalized[i].IsCollapsed = false
		normalized[i].ErrorMessage = nil
		normalized[i].WarningMessage = nil
	}
	slices.SortFunc(normalized, func(left, right models.Node) int {
		return strings.Compare(left.ID, right.ID)
	})
	return normalized
}

func sortedBacklogEdges(edges []models.Edge) []models.Edge {
	result := slices.Clone(edges)
	slices.SortFunc(result, func(left, right models.Edge) int {
		leftKey := left.SourceID + "\x00" + left.Channel + "\x00" + left.TargetID
		rightKey := right.SourceID + "\x00" + right.Channel + "\x00" + right.TargetID
		return strings.Compare(leftKey, rightKey)
	})
	return result
}

func backlogGitHubIntegrationName(nodes []models.Node) string {
	for _, node := range nodes {
		if node.ID != intakeAnalysisNodeID && node.ID != backlogRefinementNodeID {
			continue
		}
		environment, _ := node.Configuration["environmentFrom"].([]any)
		for _, entry := range environment {
			item, _ := entry.(map[string]any)
			integration, _ := item["integration"].(map[string]any)
			name, _ := integration["name"].(string)
			if strings.TrimSpace(name) != "" {
				return name
			}
		}
	}
	return intakeGitHubAppName
}

func preserveBacklogNodePositions(current, next []models.Node) {
	positions := make(map[string]models.Position, len(current))
	for _, node := range current {
		positions[node.ID] = node.Position
	}
	for i := range next {
		if position, ok := positions[next[i].ID]; ok {
			next[i].Position = position
		}
	}
}
