package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__UpgradeDefaultBacklogTemplates(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvasModel, legacyNodes, _ := createLegacyBacklogForUpgrade(ctx, t, r, factoryModel.ID, nil)
	canvasID := canvasModel.ID
	previousVersionID := *canvasModel.LiveVersionID
	run, err := models.CreateCanvasRunInTransaction(db, canvasID, backlogTriggerNodeID, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	assert.Equal(t, previousVersionID, run.VersionID)

	deps := backlogUpgradeDependencies(r)
	result, err := UpgradeDefaultBacklogTemplates(ctx, deps, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, BacklogTemplateUpgradeResult{Upgraded: 1}, result)

	upgradedCanvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvasID)
	require.NoError(t, err)
	require.NotNil(t, upgradedCanvas.LiveVersionID)
	assert.NotEqual(t, previousVersionID, *upgradedCanvas.LiveVersionID)
	assert.Equal(t, "Scores new tasks", upgradedCanvas.Name)

	versions, err := models.ListCanvasVersionsInTransaction(db, canvasID)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, previousVersionID, run.VersionID)

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvasID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryAppTemplateBacklogID, models.FactoryAppTemplateID(liveVersion.Nodes))
	require.Truef(t, models.IsBacklogFactoryApp(liveVersion.Nodes, liveVersion.Edges), "live nodes: %#v", liveVersion.Nodes)
	expectedDocument := buildBacklogCanvas(backlogCanvasRequest{
		Name:       upgradedCanvas.Name,
		Agent:      intakeAgentFromCanvasNodes(legacyNodes),
		GitHubName: backlogGitHubIntegrationName(legacyNodes),
	})
	expectedNodes, expectedEdges, err := expectedDocument.Parse(r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	preserveBacklogNodePositions(legacyNodes, expectedNodes)
	assert.Equal(t, backlogBehaviorNodes(expectedNodes), backlogBehaviorNodes(liveVersion.Nodes))
	assert.Equal(t, sortedBacklogEdges(expectedEdges), sortedBacklogEdges(liveVersion.Edges))
	assert.NotNil(t, findModelNode(t, liveVersion.Nodes, backlogRefinementNodeID))
	assert.Nil(t, findModelNodeOrNil(liveVersion.Nodes, intakeAnalysisNodeID))

	result, err = UpgradeDefaultBacklogTemplates(ctx, deps, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, BacklogTemplateUpgradeResult{Skipped: 1}, result)
	afterRetry, err := models.ListCanvasVersionsInTransaction(db, canvasID)
	require.NoError(t, err)
	assert.Len(t, afterRetry, 2)
}

func Test__UpgradeDefaultBacklogTemplatesSkipsCustomizedBacklog(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvasModel, _, _ := createLegacyBacklogForUpgrade(ctx, t, r, factoryModel.ID, func(nodes []models.Node) {
		analysis := findModelNode(t, nodes, intakeAnalysisNodeID)
		steps := analysis.Configuration["steps"].([]any)
		steps[1].(map[string]any)["prompt"] = "Use the team's custom scoring rules."
	})
	previousVersionID := *canvasModel.LiveVersionID

	result, err := UpgradeDefaultBacklogTemplates(ctx, backlogUpgradeDependencies(r), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, BacklogTemplateUpgradeResult{Skipped: 1}, result)
	reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvasModel.ID)
	require.NoError(t, err)
	assert.Equal(t, previousVersionID, *reloaded.LiveVersionID)
	versions, err := models.ListCanvasVersionsInTransaction(db, canvasModel.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 1)
}

func Test__UpgradeDefaultBacklogTemplatesRefreshesStaleRefinePrompt(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	stalePrompt := "## 3. Score\n\nScore Clarity from 1 through 5.\n\nTask:\n{{ root().data.workOrder }}"
	staleDigest := refinePromptDigest(stalePrompt)
	defaultRefinePromptDigests[staleDigest] = struct{}{}
	t.Cleanup(func() { delete(defaultRefinePromptDigests, staleDigest) })

	canvasModel, seededNodes, seededEdges := createCurrentBacklogForUpgrade(ctx, t, r, factoryModel.ID, func(nodes []models.Node) {
		backlogRefineStep(findModelNode(t, nodes, backlogRefinementNodeID).Configuration)["prompt"] = stalePrompt
	})
	previousVersionID := *canvasModel.LiveVersionID

	deps := backlogUpgradeDependencies(r)
	result, err := UpgradeDefaultBacklogTemplates(ctx, deps, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, BacklogTemplateUpgradeResult{Upgraded: 1}, result)

	reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvasModel.ID)
	require.NoError(t, err)
	assert.NotEqual(t, previousVersionID, *reloaded.LiveVersionID)
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvasModel.ID)
	require.NoError(t, err)
	refreshed := backlogRefineStep(findModelNode(t, liveVersion.Nodes, backlogRefinementNodeID).Configuration)
	assert.Equal(t, intakeRefinementPrompt(), refreshed["prompt"])
	assert.Contains(t, refreshed["prompt"], "## 4. Score Confidence")

	// Only the prompt changed. The rest of the graph is the user's.
	expected := cloneBacklogNodes(seededNodes)
	require.True(t, refreshBacklogRefinePrompt(expected))
	assert.Equal(t, backlogBehaviorNodes(expected), backlogBehaviorNodes(liveVersion.Nodes))
	assert.Equal(t, sortedBacklogEdges(seededEdges), sortedBacklogEdges(liveVersion.Edges))

	result, err = UpgradeDefaultBacklogTemplates(ctx, deps, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, BacklogTemplateUpgradeResult{Skipped: 1}, result)
}

func Test__UpgradeDefaultBacklogTemplatesKeepsEditedRefinePrompt(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvasModel, _, _ := createCurrentBacklogForUpgrade(ctx, t, r, factoryModel.ID, func(nodes []models.Node) {
		backlogRefineStep(findModelNode(t, nodes, backlogRefinementNodeID).Configuration)["prompt"] = "Use the team's scoring rules."
	})
	previousVersionID := *canvasModel.LiveVersionID

	result, err := UpgradeDefaultBacklogTemplates(ctx, backlogUpgradeDependencies(r), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, BacklogTemplateUpgradeResult{Skipped: 1}, result)
	reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvasModel.ID)
	require.NoError(t, err)
	assert.Equal(t, previousVersionID, *reloaded.LiveVersionID)
}

func Test__UpgradeDefaultBacklogTemplatesUpgradesUneditedV2(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvasModel, v2Nodes, _ := createV2BacklogForUpgrade(ctx, t, r, factoryModel.ID)
	previousVersionID := *canvasModel.LiveVersionID

	result, err := UpgradeDefaultBacklogTemplates(ctx, backlogUpgradeDependencies(r), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, BacklogTemplateUpgradeResult{Upgraded: 1}, result)

	reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvasModel.ID)
	require.NoError(t, err)
	assert.NotEqual(t, previousVersionID, *reloaded.LiveVersionID)
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvasModel.ID)
	require.NoError(t, err)
	assert.NotNil(t, findModelNodeOrNil(liveVersion.Nodes, backlogRefinementNodeID))
	assert.Nil(t, findModelNodeOrNil(liveVersion.Nodes, intakeAnalysisNodeID))
	assert.False(t, isDefaultLegacyBacklog(liveVersion.Nodes, liveVersion.Edges))
	_ = v2Nodes
}

// createCurrentBacklogForUpgrade seeds a Backlog from the current template, the
// way ensureBacklogCanvas does, then lets the test age or edit it.
func createCurrentBacklogForUpgrade(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
	customize func([]models.Node),
) (*models.Canvas, []models.Node, []models.Edge) {
	t.Helper()
	document := buildBacklogCanvas(backlogCanvasRequest{Name: "Backlog"})
	nodes, edges, err := document.Parse(r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	if customize != nil {
		customize(nodes)
	}

	created, err := canvases.CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		r.GitProvider,
		"http://localhost:8000",
		r.Organization.ID,
		document.Metadata.Name,
		document.Metadata.Description,
		&factoryID,
		nodes,
		edges,
		nil,
	)
	require.NoError(t, err)
	canvasID := uuid.MustParse(created.GetCanvas().GetMetadata().GetId())
	canvasModel, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, canvasID)
	require.NoError(t, err)
	return canvasModel, nodes, edges
}

func createV2BacklogForUpgrade(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
) (*models.Canvas, []models.Node, []models.Edge) {
	t.Helper()
	document := buildV2BacklogCanvas(backlogCanvasRequest{Name: "Backlog"})
	nodes, edges, err := document.Parse(r.Registry, r.Organization.ID.String())
	require.NoError(t, err)

	created, err := canvases.CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		r.GitProvider,
		"http://localhost:8000",
		r.Organization.ID,
		document.Metadata.Name,
		document.Metadata.Description,
		&factoryID,
		nodes,
		edges,
		nil,
	)
	require.NoError(t, err)
	canvasID := uuid.MustParse(created.GetCanvas().GetMetadata().GetId())
	canvasModel, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, canvasID)
	require.NoError(t, err)
	return canvasModel, nodes, edges
}

func createLegacyBacklogForUpgrade(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
	customize func([]models.Node),
) (*models.Canvas, []models.Node, []models.Edge) {
	t.Helper()
	legacyDocument := buildLegacyBacklogCanvas(backlogCanvasRequest{Name: "Scores new tasks"})
	legacyNodes, legacyEdges, err := legacyDocument.Parse(r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.Equal(t, models.FactoryAppTemplateBacklogID, models.FactoryAppTemplateID(legacyNodes))
	if customize != nil {
		customize(legacyNodes)
	}

	created, err := canvases.CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		r.GitProvider,
		"http://localhost:8000",
		r.Organization.ID,
		legacyDocument.Metadata.Name,
		legacyDocument.Metadata.Description,
		&factoryID,
		legacyNodes,
		legacyEdges,
		nil,
	)
	require.NoError(t, err)
	canvasID := uuid.MustParse(created.GetCanvas().GetMetadata().GetId())
	canvasModel, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, canvasID)
	require.NoError(t, err)
	return canvasModel, legacyNodes, legacyEdges
}

func backlogUpgradeDependencies(r *support.ResourceRegistry) IntakeDependencies {
	return IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		GitProvider:    r.GitProvider,
		WebhookBaseURL: "http://localhost:8000",
	}
}
