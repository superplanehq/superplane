package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__RewriteWorkspaceAgentNode__PlanningNodeKeepsPlanningModel(t *testing.T) {
	t.Parallel()

	node := models.Node{
		ID:   planningAgentNodeID,
		Type: models.NodeTypeComponent,
		Ref:  models.NodeRef{Component: &models.ComponentRef{Name: models.SuperPlaneRunnerComponent}},
		Configuration: map[string]any{
			"model": "old",
		},
	}
	rewrite, err := workspaceAgentRewriteFor(modelSourceAnthropic, "claude")
	require.NoError(t, err)
	require.True(t, rewriteWorkspaceAgentNode(&node, rewrite))
	assert.Equal(t, "runnerClaudeCode", node.ComponentName())
	assert.Equal(t, "opus", node.Configuration["model"])
}

func Test__SwitchFactoryModelSource__RewritesAgentCanvases(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	hosted := createLineAppWithRunner(t, r, factory.ID, models.SuperPlaneRunnerComponent, "hosted", "")
	provider := createLineAppWithRunner(t, r, factory.ID, "runnerCodex", "integration", "gpt-5")
	untouched := createTriggerOnlyCanvas(t, r, factory.ID)
	hostedLive := *hosted.LiveVersionID
	providerLive := *provider.LiveVersionID
	untouchedLive := *untouched.LiveVersionID
	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", hosted.ID, "agent").
		Update("state", models.CanvasNodeStateError).Error)

	changed, _, err := SwitchFactoryModelSourceInTransaction(
		t.Context(),
		crypto.NewNoOpEncryptor(),
		r.Organization.ID.String(),
		factory.ID.String(),
		modelSourceAnthropic,
		"sk-test",
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{hosted.ID, provider.ID}, changed)

	assertAgentNode(t, db, hosted.ID, "runnerClaudeCode", "sonnet", "claude")
	assertAgentNode(t, db, provider.ID, "runnerClaudeCode", "sonnet", "claude")
	assertCanvasNodeReady(t, db, hosted.ID, "agent")
	assertCanvasLiveVersion(t, db, untouched.ID, untouchedLive)
	assertCanvasLiveVersionChanged(t, db, hosted.ID, hostedLive)
	assertCanvasLiveVersionChanged(t, db, provider.ID, providerLive)

	reloaded, err := findFactory(db, r.Organization.ID, factory.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.FactoryOnboardingAgentHarnessClaudeCode, reloaded.OnboardingConfigValue().AgentHarness)
	assert.NotEmpty(t, reloaded.OnboardingConfigValue().AgentIntegrationID)

	_, _, err = SwitchFactoryModelSourceInTransaction(
		t.Context(),
		crypto.NewNoOpEncryptor(),
		r.Organization.ID.String(),
		factory.ID.String(),
		modelSourceHosted,
		"",
	)
	require.NoError(t, err)
	assertHostedAgentNode(t, db, hosted.ID)
	reloaded, err = findFactory(db, r.Organization.ID, factory.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.FactoryOnboardingAgentHarnessSuperPlane, reloaded.OnboardingConfigValue().AgentHarness)
	assert.Empty(t, reloaded.OnboardingConfigValue().AgentIntegrationID)
}

func createTriggerOnlyCanvas(t *testing.T, r *support.ResourceRegistry, factoryID uuid.UUID) *models.Canvas {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           support.RandomName("filter"),
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		return tx.Create(&models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &r.User,
			Nodes: datatypes.NewJSONSlice([]models.Node{{
				ID:   "start",
				Name: "start",
				Type: models.NodeTypeTrigger,
				Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}},
			}}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}).Error
	}))
	return canvas
}

func assertAgentNode(t *testing.T, db *gorm.DB, canvasID uuid.UUID, component, model, installation string) {
	t.Helper()
	live, err := models.FindLiveCanvasVersionInTransaction(db, canvasID)
	require.NoError(t, err)
	var agent *models.Node
	for i := range live.Nodes {
		if live.Nodes[i].ID == "agent" {
			agent = &live.Nodes[i]
		}
	}
	require.NotNil(t, agent)
	assert.Equal(t, component, agent.ComponentName())
	assert.Equal(t, model, agent.Configuration["model"])
	credentials, _ := agent.Configuration["credentials"].(map[string]any)
	require.NotNil(t, credentials)
	assert.Equal(t, "integration", credentials["source"])
	integration, _ := credentials["integration"].(map[string]any)
	require.NotNil(t, integration)
	assert.Equal(t, installation, integration["name"])
}

func assertHostedAgentNode(t *testing.T, db *gorm.DB, canvasID uuid.UUID) {
	t.Helper()
	live, err := models.FindLiveCanvasVersionInTransaction(db, canvasID)
	require.NoError(t, err)
	for i := range live.Nodes {
		if live.Nodes[i].ID != "agent" {
			continue
		}
		assert.Equal(t, models.SuperPlaneRunnerComponent, live.Nodes[i].ComponentName())
		_, hasCredentials := live.Nodes[i].Configuration["credentials"]
		_, hasModel := live.Nodes[i].Configuration["model"]
		assert.False(t, hasCredentials)
		assert.False(t, hasModel)
		return
	}
	t.Fatal("agent node not found")
}

func assertCanvasLiveVersion(t *testing.T, db *gorm.DB, canvasID, versionID uuid.UUID) {
	t.Helper()
	var canvas models.Canvas
	require.NoError(t, db.First(&canvas, "id = ?", canvasID).Error)
	require.NotNil(t, canvas.LiveVersionID)
	assert.Equal(t, versionID, *canvas.LiveVersionID)
}

func assertCanvasLiveVersionChanged(t *testing.T, db *gorm.DB, canvasID, previous uuid.UUID) {
	t.Helper()
	var canvas models.Canvas
	require.NoError(t, db.First(&canvas, "id = ?", canvasID).Error)
	require.NotNil(t, canvas.LiveVersionID)
	assert.NotEqual(t, previous, *canvas.LiveVersionID)
}

func assertCanvasNodeReady(t *testing.T, db *gorm.DB, canvasID uuid.UUID, nodeID string) {
	t.Helper()
	var node models.CanvasNode
	require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).First(&node).Error)
	assert.Equal(t, models.CanvasNodeStateReady, node.State)
	assert.Nil(t, node.StateReason)
}

func Test__SwitchFactoryModelSource__ReusesReadyIntegration(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	installation, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "claude", "claude", map[string]any{"apiKey": "kept"})
	require.NoError(t, err)
	require.NoError(t, db.Model(installation).Update("state", models.IntegrationStateReady).Error)
	createLineAppWithRunner(t, r, factory.ID, models.SuperPlaneRunnerComponent, "hosted", "")

	_, integrationID, err := SwitchFactoryModelSourceInTransaction(
		t.Context(),
		crypto.NewNoOpEncryptor(),
		r.Organization.ID.String(),
		factory.ID.String(),
		modelSourceAnthropic,
		"sk-new",
	)
	require.NoError(t, err)
	assert.Equal(t, installation.ID.String(), integrationID)

	var saved models.Integration
	require.NoError(t, db.First(&saved, "id = ?", installation.ID).Error)
	assert.Equal(t, "kept", saved.Configuration.Data()["apiKey"])
}
