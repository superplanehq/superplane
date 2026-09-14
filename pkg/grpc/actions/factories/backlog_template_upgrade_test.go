package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
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
	assert.Equal(t, BacklogTemplateUpgradeResult{}, result)

	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCreateWithAgent))
	result, err = UpgradeDefaultBacklogTemplates(ctx, deps, r.Organization.ID)
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
	assert.Equal(t, findModelNode(t, legacyNodes, intakeAnalysisNodeID).Position, findModelNode(t, liveVersion.Nodes, intakeAnalysisNodeID).Position)

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
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCreateWithAgent))

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

	created, err := canvases.CreateCanvasWithSeedFiles(
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
