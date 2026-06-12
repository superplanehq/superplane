package actions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/authentication"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
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
	layout := resolveCustomToolAutoLayout(&AutoLayoutInput{
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

func TestAppAgentTool_UpdateDraftStagesEdits(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	canvasYAML, err := canvasRepository.ReadRepositorySpecFile(
		context.Background(),
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		canvasRepository.CanvasYAMLRepositoryPath,
	)
	require.NoError(t, err)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	registry := NewDefaultRegistry(Dependencies{
		Encryptor:      r.Encryptor,
		Registry:       r.Registry,
		AuthService:    r.AuthService,
		WebhookBaseURL: "https://hooks.example.test",
	})

	result, err := registry.Execute(ctx, agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{
		Action:     "update_draft",
		CanvasYAML: canvasYAML,
	})

	require.NoError(t, err)
	update, ok := result.(updateResult)
	require.True(t, ok)
	assert.Equal(t, "update_draft", update.Action)
	require.NotEmpty(t, update.VersionID)

	// update_draft writes to the UI staging layer instead of committing into the
	// draft version row, so the edit shows up as pending staging that the user
	// reviews and publishes, exactly like an edit made in the UI editor.
	described, err := canvasRepository.DescribeCanvasVersion(
		ctx,
		r.Organization.ID.String(),
		canvas.ID.String(),
		update.VersionID,
	)
	require.NoError(t, err)
	assert.True(t, described.GetStagingState().GetHasStaging())
	assert.Contains(t, described.GetStagingState().GetStagedPaths(), canvasRepository.CanvasYAMLRepositoryPath)

	// The agent reads back the same staged content it wrote through the staged
	// read path the `read` action now uses.
	staged, err := canvasRepository.ReadRepositorySpecFileStaged(
		ctx,
		r.Organization.ID.String(),
		canvas.ID.String(),
		update.VersionID,
		canvasRepository.CanvasYAMLRepositoryPath,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, staged)
}

func TestAccessAction_ReportsInterceptorBackedAgentTokenAccess(t *testing.T) {
	organizationID := uuid.NewString()
	userID := uuid.NewString()
	canvasID := uuid.NewString()
	action := accessAction{auth: allowingPermissionChecker{}}

	payload, err := action.Execute(context.Background(), agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: organizationID,
		UserID:         userID,
		CanvasID:       canvasID,
	}, Input{})

	require.NoError(t, err)
	result, ok := payload.(accessResult)
	require.True(t, ok)

	assert.Equal(t, "access", result.Action)
	assert.Equal(t, canvasID, result.CanvasID)
	assert.ElementsMatch(t, []string{
		"org:read",
		"integrations:read",
		"canvases:read:" + canvasID,
		"canvases:update_version:" + canvasID,
	}, result.TokenScopes)

	accessible := apiAccessByMethod(result.Accessible)
	unavailable := apiAccessByMethod(result.Unavailable)

	assert.Contains(t, accessible, pb.Canvases_ListRuns_FullMethodName)
	assert.Equal(t, []string{canvasID}, accessible[pb.Canvases_ListRuns_FullMethodName].Resources)
	assert.Contains(t, accessible, pb.Canvases_CreateCanvasVersion_FullMethodName)
	assert.Contains(t, unavailable, pb.Canvases_ListCanvases_FullMethodName)
	assert.Contains(t, unavailable, pb.Canvases_PublishCanvasVersion_FullMethodName)

	toolActions := toolAccessByAction(result.ToolActions)
	require.Contains(t, toolActions, "update_draft")
	assert.True(t, toolActions["update_draft"].Allowed)
	require.Contains(t, toolActions, "read_runtime")
	assert.True(t, toolActions["read_runtime"].Allowed)
}

func TestReadRuntimeAction_ParseFilters(t *testing.T) {
	runStates, err := parseRunStates([]string{"started", "STATE_FINISHED"})
	require.NoError(t, err)
	assert.Equal(t, []pb.CanvasRun_State{pb.CanvasRun_STATE_STARTED, pb.CanvasRun_STATE_FINISHED}, runStates)

	runResults, err := parseRunResults([]string{"passed", "RESULT_CANCELLED"})
	require.NoError(t, err)
	assert.Equal(t, []pb.CanvasRun_Result{pb.CanvasRun_RESULT_PASSED, pb.CanvasRun_RESULT_CANCELLED}, runResults)

	executionStates, err := parseExecutionStates([]string{"pending", "STATE_STARTED", "finished"})
	require.NoError(t, err)
	assert.Equal(t, []pb.CanvasNodeExecution_State{
		pb.CanvasNodeExecution_STATE_PENDING,
		pb.CanvasNodeExecution_STATE_STARTED,
		pb.CanvasNodeExecution_STATE_FINISHED,
	}, executionStates)

	executionResults, err := parseExecutionResults([]string{"passed", "RESULT_FAILED"})
	require.NoError(t, err)
	assert.Equal(t, []pb.CanvasNodeExecution_Result{pb.CanvasNodeExecution_RESULT_PASSED, pb.CanvasNodeExecution_RESULT_FAILED}, executionResults)
}

func TestReadRuntimeAction_RejectsUnknownResource(t *testing.T) {
	action := readRuntimeAction{
		registry: &registry.Registry{},
		auth:     allowingPermissionChecker{},
	}

	_, err := action.Execute(context.Background(), agents.AgentSessionContext{
		OrganizationID: uuid.NewString(),
		UserID:         uuid.NewString(),
		CanvasID:       uuid.NewString(),
	}, Input{Resource: "secrets"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported runtime resource")
}

type allowingPermissionChecker struct{}

func (allowingPermissionChecker) CheckOrganizationPermission(_, _, _, _ string) (bool, error) {
	return true, nil
}

func apiAccessByMethod(entries []apiAccessResult) map[string]apiAccessResult {
	result := make(map[string]apiAccessResult, len(entries))
	for _, entry := range entries {
		result[entry.Method] = entry
	}
	return result
}

func toolAccessByAction(entries []toolAccessResult) map[string]toolAccessResult {
	result := make(map[string]toolAccessResult, len(entries))
	for _, entry := range entries {
		result[entry.Action] = entry
	}
	return result
}
