package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	grpcCanvases "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func TestResolveCustomToolAutoLayout_DefaultsGraphUpdatesToFullCanvas(t *testing.T) {
	layout := resolveCustomToolAutoLayout(nil, true)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_FULL_CANVAS, layout.Scope)
	assert.Empty(t, layout.NodeIds)
}

func TestResolveCustomToolAutoLayout_SkipsConsoleOnlyUpdates(t *testing.T) {
	assert.Nil(t, resolveCustomToolAutoLayout(nil, false))
}

func TestResolveCustomToolAutoLayout_PreservesExplicitSettings(t *testing.T) {
	layout := resolveCustomToolAutoLayout(&superPlaneCanvasAutoLayoutInput{
		Scope:   "connected_component",
		NodeIDs: []string{"node-1"},
	}, true)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT, layout.Scope)
	assert.Equal(t, []string{"node-1"}, layout.NodeIds)
}

func TestSummarizeNodes_UsesYamlComponentFieldName(t *testing.T) {
	summary := summarizeNodes([]models.Node{
		{
			ID:   "node-1",
			Name: "Notify",
			Type: "TYPE_ACTION",
			Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "slack.sendTextMessage"}},
		},
	}, 20)

	require.Len(t, summary, 1)
	assert.Equal(t, "slack.sendTextMessage", summary[0].Component)

	data, err := json.Marshal(summary[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), `"component":"slack.sendTextMessage"`)
	assert.NotContains(t, string(data), `"ref"`)
}

func TestSelectedVersion_ReturnsLiveVersionLoadErrors(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	missingVersionID := uuid.New()
	canvas.LiveVersionID = &missingVersionID

	version, err := selectedVersion(canvas, nil, "live")

	require.Error(t, err)
	assert.Nil(t, version)
	assert.Contains(t, err.Error(), "load live canvas version summary")
}

func TestSuperPlaneCanvasTool_UpdateDraftStagesEdits(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	canvasYAML, err := grpcCanvases.ReadRepositorySpecFile(
		context.Background(),
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		grpcCanvases.CanvasYAMLRepositoryPath,
	)
	require.NoError(t, err)

	input, err := json.Marshal(map[string]string{
		"action":      "update_draft",
		"canvas_yaml": canvasYAML,
	})
	require.NoError(t, err)

	tool := NewSuperPlaneCanvasTool(SuperPlaneCanvasToolOptions{
		Encryptor:      r.Encryptor,
		Registry:       r.Registry,
		AuthService:    r.AuthService,
		WebhookBaseURL: "https://hooks.example.test",
	})

	result := tool.ExecuteCustomTool(context.Background(), agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, agents.CustomToolUse{
		ID:    "tool-1",
		Name:  SuperPlaneCanvasToolName,
		Input: string(input),
	})

	// Agents stage onto their private draft without committing; usage limits
	// are enforced when the user commits, not at stage time.
	require.False(t, result.IsError, result.Content)
	assert.Contains(t, result.Content, `"action":"update_draft"`)
}
