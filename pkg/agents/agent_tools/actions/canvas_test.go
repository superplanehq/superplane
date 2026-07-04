package actions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/authentication"
	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestResolveToolAutoLayoutInput_DefaultsNodeIDsToConnectedComponent(t *testing.T) {
	layout := resolveToolAutoLayoutInput(&AutoLayoutInput{NodeIDs: []string{"node-1"}})

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT, layout.Scope)
	assert.Equal(t, []string{"node-1"}, layout.NodeIds)
}

func TestResolveToolAutoLayoutInput_PreservesExplicitSettings(t *testing.T) {
	layout := resolveToolAutoLayoutInput(&AutoLayoutInput{
		Scope:   "connected_component",
		NodeIDs: []string{"node-1"},
	})

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT, layout.Scope)
	assert.Equal(t, []string{"node-1"}, layout.NodeIds)
}

func TestResolvePatchDraftAutoLayout_DefaultsToAffectedConnectedComponents(t *testing.T) {
	changeset := requireDraftChangeset(t, []PatchOperation{
		{
			Op: "add_node",
			Node: &PatchNode{
				ID:        "new-node",
				Name:      "New node",
				Component: "noop",
			},
		},
		{
			Op:   "delete_node",
			Node: &PatchNode{ID: "deleted-node"},
		},
	})

	layout := resolvePatchStagingAutoLayout(
		nil,
		changeset,
		[]models.Edge{{SourceID: "kept-node", TargetID: "deleted-node", Channel: "default"}},
		[]models.Node{{ID: "new-node"}, {ID: "kept-node"}},
	)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT, layout.Scope)
	assert.Equal(t, []string{"kept-node", "new-node"}, layout.NodeIds)
}

func TestResolvePatchDraftAutoLayout_PreservesExplicitSettings(t *testing.T) {
	layout := resolvePatchStagingAutoLayout(
		&AutoLayoutInput{Scope: "full_canvas"},
		nil,
		nil,
		nil,
	)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_FULL_CANVAS, layout.Scope)
	assert.Empty(t, layout.NodeIds)
}

func TestResolvePatchDraftAutoLayout_TreatsEmptyInputLikeOmitted(t *testing.T) {
	changeset := requireDraftChangeset(t, []PatchOperation{
		{
			Op: "add_node",
			Node: &PatchNode{
				ID:        "new-node",
				Name:      "New node",
				Component: "noop",
			},
		},
	})

	layout := resolvePatchStagingAutoLayout(
		&AutoLayoutInput{},
		changeset,
		nil,
		[]models.Node{{ID: "new-node"}},
	)

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT, layout.Scope)
	assert.Equal(t, []string{"new-node"}, layout.NodeIds)
}

func TestResolvePatchDraftAutoLayout_DefaultsLayoutOnlyUpdatesToFullCanvas(t *testing.T) {
	layout := resolvePatchStagingAutoLayout(&AutoLayoutInput{}, nil, nil, []models.Node{{ID: "node-1"}})

	require.NotNil(t, layout)
	assert.Equal(t, pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL, layout.Algorithm)
	assert.Equal(t, pb.CanvasAutoLayout_SCOPE_FULL_CANVAS, layout.Scope)
	assert.Empty(t, layout.NodeIds)
}

func TestResolveLiveCanvasVersion_ResolvesLiveVersion(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	liveVersion := requireLiveVersion(t, canvas.ID)

	resolved, err := resolveLiveCanvasVersion(canvas.ID, Input{Action: "patch_staging"})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, liveVersion.ID, resolved.ID)

	resolved, err = resolveLiveCanvasVersion(canvas.ID, Input{
		Action:    "patch_staging",
		VersionID: liveVersion.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, liveVersion.ID, resolved.ID)

	wrongVersionID := uuid.New()
	_, err = resolveLiveCanvasVersion(canvas.ID, Input{
		Action:    "patch_staging",
		VersionID: wrongVersionID.String(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not the current live version")
}

func requireDraftChangeset(t *testing.T, operations []PatchOperation) *changesets.CanvasChangeset {
	t.Helper()

	changeset, err := buildDraftChangeset(operations)
	require.NoError(t, err)
	return changeset
}

func requireLiveVersion(t *testing.T, canvasID uuid.UUID) models.CanvasVersion {
	t.Helper()
	live, err := models.FindLiveCanvasVersion(canvasID)
	require.NoError(t, err)
	require.NotNil(t, live)
	return *live
}

func upsertUserStagingYAML(t *testing.T, canvas *models.Canvas, userID uuid.UUID, content string) {
	t.Helper()
	require.NotNil(t, canvas.LiveVersionID)
	_, err := models.UpsertStagedFile(database.DB(t.Context()), canvas.ID, userID, *canvas.LiveVersionID, canvas.OrganizationID, canvasRepository.CanvasYAMLRepositoryPath, content)
	require.NoError(t, err)
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

func TestAppAgentTool_PatchStagingStagesSmallGraphEdits(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	liveVersion := requireLiveVersion(t, canvas.ID)

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
		Action: "patch_staging",
		PatchOperations: []PatchOperation{
			{
				Op: "add_node",
				Node: &PatchNode{
					ID:        "first-node",
					Name:      "First",
					Component: "noop",
					Position:  &PatchPosition{X: 10, Y: 20},
				},
			},
			{
				Op: "add_node",
				Node: &PatchNode{
					ID:        "second-node",
					Name:      "Second",
					Component: "noop",
				},
			},
			{
				Op: "add_edge",
				Edge: &PatchEdge{
					SourceID: "first-node",
					TargetID: "second-node",
				},
			},
		},
	})

	require.NoError(t, err)
	update, ok := result.(updateResult)
	require.True(t, ok)
	assert.Equal(t, "patch_staging", update.Action)
	assert.Equal(t, liveVersion.ID.String(), update.VersionID)
	assert.Equal(t, 2, update.Summary.NodeCount)
	assert.Equal(t, 1, update.Summary.EdgeCount)

	staging, err := canvasRepository.GetCanvasStaging(ctx, r.Organization.ID.String(), canvas.ID.String())
	require.NoError(t, err)
	assert.True(t, staging.GetHasStaging())
	assert.Contains(t, staging.GetStagedPaths(), canvasRepository.CanvasYAMLRepositoryPath)

	staged, err := canvasRepository.ReadRepositorySpecFileStaged(
		ctx,
		r.Organization.ID.String(),
		canvas.ID.String(),
		update.VersionID,
		canvasRepository.CanvasYAMLRepositoryPath,
	)
	require.NoError(t, err)

	patched, err := canvasyaml.ParseCanvasResource([]byte(staged))
	require.NoError(t, err)
	require.Len(t, patched.GetSpec().GetNodes(), 2)
	require.Len(t, patched.GetSpec().GetEdges(), 1)
	assert.Equal(t, "first-node", patched.GetSpec().GetNodes()[0].GetId())
	assert.Equal(t, "second-node", patched.GetSpec().GetNodes()[1].GetId())
	assert.Equal(t, "default", patched.GetSpec().GetEdges()[0].GetChannel())
}

func TestAppAgentTool_PatchStagingAddsIntegrationBackedNode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	liveVersion := requireLiveVersion(t, canvas.ID)
	integration := support.CreateIntegrationWithCapabilities(t, r.Organization.ID, []models.CapabilityState{
		{Name: "github.createIssue", State: core.IntegrationCapabilityStateEnabled},
	})

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
		Action:    "patch_staging",
		VersionID: liveVersion.ID.String(),
		PatchOperations: []PatchOperation{
			{
				Op: "add_node",
				Node: &PatchNode{
					ID:            "create-issue",
					Name:          "Create issue",
					Component:     "github.createIssue",
					IntegrationID: integration.ID.String(),
					Configuration: map[string]any{
						"repository": "superplanehq/superplane",
						"title":      "Generated issue",
					},
				},
			},
		},
	})

	require.NoError(t, err)
	update, ok := result.(updateResult)
	require.True(t, ok)
	assert.Empty(t, update.NodeIssues)

	staged, err := canvasRepository.ReadRepositorySpecFileStaged(
		ctx,
		r.Organization.ID.String(),
		canvas.ID.String(),
		update.VersionID,
		canvasRepository.CanvasYAMLRepositoryPath,
	)
	require.NoError(t, err)

	patched, err := canvasyaml.ParseCanvasResource([]byte(staged))
	require.NoError(t, err)
	require.Len(t, patched.GetSpec().GetNodes(), 1)
	node := patched.GetSpec().GetNodes()[0]
	assert.Equal(t, "github.createIssue", node.GetComponent())
	require.NotNil(t, node.GetIntegration())
	assert.Equal(t, integration.ID.String(), *node.GetIntegration().Id)
}

func TestAppAgentTool_PatchStagingStagesConsoleYAML(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	liveVersion := requireLiveVersion(t, canvas.ID)
	consoleYAML, err := canvasRepository.ReadRepositorySpecFile(
		context.Background(),
		r.Organization.ID.String(),
		canvas.ID.String(),
		"",
		canvasRepository.ConsoleYAMLRepositoryPath,
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
		Action:      "patch_staging",
		VersionID:   liveVersion.ID.String(),
		ConsoleYAML: consoleYAML,
	})

	require.NoError(t, err)
	update, ok := result.(updateResult)
	require.True(t, ok)
	assert.Equal(t, "patch_staging", update.Action)
	assert.Equal(t, liveVersion.ID.String(), update.VersionID)

	// patch_staging writes to the UI staging layer instead of committing into the
	// live version row, so the edit shows up as pending staging that the user
	// reviews and publishes, exactly like an edit made in the UI editor.
	staging, err := canvasRepository.GetCanvasStaging(ctx, r.Organization.ID.String(), canvas.ID.String())
	require.NoError(t, err)
	assert.True(t, staging.GetHasStaging())
	assert.Contains(t, staging.GetStagedPaths(), canvasRepository.ConsoleYAMLRepositoryPath)

	// The agent reads back the same staged content it wrote through the staged
	// read path the `read` action now uses.
	staged, err := canvasRepository.ReadRepositorySpecFileStaged(
		ctx,
		r.Organization.ID.String(),
		canvas.ID.String(),
		update.VersionID,
		canvasRepository.ConsoleYAMLRepositoryPath,
	)
	require.NoError(t, err)
	assert.Equal(t, consoleYAML, staged)
}

func TestAppAgentTool_ListResources(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		ListResources: func(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
			assert.Equal(t, "repository", resourceType)
			assert.Equal(t, "repository", ctx.Parameters["type"])
			assert.Equal(t, "superplanehq", ctx.Parameters["owner"])
			return []core.IntegrationResource{
				{Type: "repository", ID: "superplanehq/superplane", Name: "superplane"},
				{Type: "repository", ID: "superplanehq/docs", Name: "docs"},
			}, nil
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)

	registry := NewDefaultRegistry(Dependencies{
		Registry:    r.Registry,
		AuthService: r.AuthService,
	})
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	result, err := registry.Execute(ctx, agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       uuid.NewString(),
	}, Input{
		Action:        "list_resources",
		IntegrationID: integration.ID.String(),
		ResourceType:  "repository",
		Parameters:    map[string]string{"owner": "superplanehq"},
		Limit:         1,
	})

	require.NoError(t, err)
	resources, ok := result.(resourcesResult)
	require.True(t, ok)
	assert.Equal(t, "list_resources", resources.Action)
	assert.Equal(t, integration.ID.String(), resources.IntegrationID)
	assert.Equal(t, "repository", resources.ResourceType)
	assert.Equal(t, 2, resources.Count)
	assert.True(t, resources.Truncated)
	require.Len(t, resources.Resources, 1)
	assert.Equal(t, integrationResourceResult{
		Type: "repository",
		ID:   "superplanehq/superplane",
		Name: "superplane",
	}, resources.Resources[0])
}

func TestAppAgentTool_PatchStagingWithLiveVersionIDStagesChanges(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	liveVersion := requireLiveVersion(t, canvas.ID)

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
		Action:    "patch_staging",
		VersionID: liveVersion.ID.String(),
		PatchOperations: []PatchOperation{{
			Op: "add_node",
			Node: &PatchNode{
				ID:        "live-version-node",
				Name:      "Live version node",
				Component: "noop",
			},
		}},
	})

	require.NoError(t, err)
	update, ok := result.(updateResult)
	require.True(t, ok)
	assert.Equal(t, liveVersion.ID.String(), update.VersionID)

	hasStaging, err := models.HasStagedFilesForUser(database.DB(t.Context()), canvas.ID, r.User)
	require.NoError(t, err)
	assert.True(t, hasStaging)
}

func TestAppAgentTool_PatchStagingWithoutVersionIDSucceeds(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	liveVersion := requireLiveVersion(t, canvas.ID)

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
		Action: "patch_staging",
		PatchOperations: []PatchOperation{{
			Op: "add_node",
			Node: &PatchNode{
				ID:        "omitted-version-node",
				Name:      "Omitted version node",
				Component: "noop",
			},
		}},
	})

	require.NoError(t, err)
	update, ok := result.(updateResult)
	require.True(t, ok)
	assert.Equal(t, liveVersion.ID.String(), update.VersionID)

	hasStaging, err := models.HasStagedFilesForUser(database.DB(t.Context()), canvas.ID, r.User)
	require.NoError(t, err)
	assert.True(t, hasStaging)
}

func TestAppAgentTool_ReadReturnsUserStagingOnLiveVersion(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	liveVersion := requireLiveVersion(t, canvas.ID)
	upsertUserStagingYAML(t, canvas, r.User, "draft: staged\n")

	registry := NewDefaultRegistry(Dependencies{
		Encryptor:      r.Encryptor,
		Registry:       r.Registry,
		AuthService:    r.AuthService,
		WebhookBaseURL: "https://hooks.example.test",
	})

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	result, err := registry.Execute(ctx, agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{
		Action:            "read",
		VersionID:         liveVersion.ID.String(),
		IncludeCanvasYAML: true,
	})

	require.NoError(t, err)
	read, ok := result.(readResult)
	require.True(t, ok)
	assert.Equal(t, "staging", read.Source)
	assert.Equal(t, liveVersion.ID.String(), read.VersionID)
	assert.Equal(t, "draft: staged\n", read.CanvasYAML)
	assert.Equal(t, len("draft: staged\n"), read.CanvasYAMLBytes)
}

func TestAppAgentTool_ReadOmitsCanvasYAMLByDefault(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	liveVersion := requireLiveVersion(t, canvas.ID)
	upsertUserStagingYAML(t, canvas, r.User, "draft: compact\n")

	registry := NewDefaultRegistry(Dependencies{
		Encryptor:      r.Encryptor,
		Registry:       r.Registry,
		AuthService:    r.AuthService,
		WebhookBaseURL: "https://hooks.example.test",
	})

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	result, err := registry.Execute(ctx, agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{
		Action:    "read",
		VersionID: liveVersion.ID.String(),
	})

	require.NoError(t, err)
	read, ok := result.(readResult)
	require.True(t, ok)
	assert.Empty(t, read.CanvasYAML)
	assert.True(t, read.CanvasYAMLOmitted)
	assert.Equal(t, len("draft: compact\n"), read.CanvasYAMLBytes)
	assert.Equal(t, "staging", read.Source)
	assert.Equal(t, liveVersion.ID.String(), read.VersionID)
}

func TestAppAgentTool_ListFilesReportsContextFiles(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	head, err := r.GitProvider.Head(context.Background(), repository.RepoID, "")
	require.NoError(t, err)
	_, err = r.GitProvider.Commit(context.Background(), repository.RepoID, gitprovider.CommitOptions{
		Branch:          "main",
		BaseBranch:      "main",
		ExpectedHeadSHA: head,
		Message:         "Add context",
		Author:          gitprovider.CommitAuthor{Name: "Test", Email: "test@example.com"},
		Operations: []gitprovider.FileOperation{
			{Path: "AGENTS.md", Content: strings.NewReader("Use pnpm.\n"), SizeBytes: int64(len("Use pnpm.\n"))},
			{Path: "scripts/run.py", Content: strings.NewReader("print('ok')\n"), SizeBytes: int64(len("print('ok')\n"))},
		},
	})
	require.NoError(t, err)

	registry := NewDefaultRegistry(Dependencies{GitProvider: r.GitProvider})
	result, err := registry.Execute(context.Background(), agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{Action: "list_files"})

	require.NoError(t, err)
	list, ok := result.(fileListResult)
	require.True(t, ok)
	assert.Contains(t, list.Files, "AGENTS.md")
	assert.Contains(t, list.Files, "README.md")
	assert.Contains(t, list.ContextFiles, "AGENTS.md")
	assert.Contains(t, list.ContextFiles, "README.md")
}

func TestAppAgentTool_ReadFileReturnsStagedDraftContent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	liveVersion := requireLiveVersion(t, canvas.ID)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	registry := NewDefaultRegistry(Dependencies{GitProvider: r.GitProvider})

	_, err := registry.Execute(ctx, agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{
		Action:    "write_file",
		VersionID: liveVersion.ID.String(),
		Path:      "README.md",
		Content:   "draft readme\n",
	})
	require.NoError(t, err)

	result, err := registry.Execute(ctx, agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{
		Action:    "read_file",
		VersionID: liveVersion.ID.String(),
		Path:      "README.md",
	})

	require.NoError(t, err)
	read, ok := result.(fileReadResult)
	require.True(t, ok)
	require.Len(t, read.Files, 1)
	assert.Equal(t, "README.md", read.Files[0].Path)
	assert.Equal(t, "draft readme\n", read.Files[0].Content)
	assert.Equal(t, "staging", read.Files[0].Source)
}

func TestAppAgentTool_ReadFileUsesLiveVersionWhenVersionOmitted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	liveVersion := requireLiveVersion(t, canvas.ID)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	registry := NewDefaultRegistry(Dependencies{GitProvider: r.GitProvider})
	session := agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}

	_, err := registry.Execute(ctx, session, Input{
		Action:    "write_file",
		VersionID: liveVersion.ID.String(),
		Path:      "README.md",
		Content:   "draft readme\n",
	})
	require.NoError(t, err)

	result, err := registry.Execute(ctx, session, Input{
		Action: "read_file",
		Path:   "README.md",
	})

	require.NoError(t, err)
	read, ok := result.(fileReadResult)
	require.True(t, ok)
	require.Len(t, read.Files, 1)
	assert.Equal(t, liveVersion.ID.String(), read.Files[0].VersionID)
	assert.Equal(t, "draft readme\n", read.Files[0].Content)
	assert.Equal(t, "staging", read.Files[0].Source)
}

func TestAppAgentTool_WriteFileRejectsSpecFiles(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	liveVersion := requireLiveVersion(t, canvas.ID)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	registry := NewDefaultRegistry(Dependencies{GitProvider: r.GitProvider})
	_, err := registry.Execute(ctx, agents.AgentSessionContext{
		SessionID:      "session-1",
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{
		Action:    "write_file",
		VersionID: liveVersion.ID.String(),
		Path:      "canvas.yaml",
		Content:   "name: invalid\n",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "use patch_staging")
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

	assert.Contains(t, accessible, "GET /api/v1/canvases/{canvas_id}/runs")
	assert.Equal(t, []string{canvasID}, accessible["GET /api/v1/canvases/{canvas_id}/runs"].Resources)
	assert.Contains(t, accessible, "PUT /api/v1/canvases/{canvas_id}/staging")
	assert.Contains(t, unavailable, "POST /api/v1/canvases/{canvas_id}/staging/commit")
	assert.Contains(t, unavailable, "GET /api/v1/canvases")
	assert.Contains(t, unavailable, "POST /api/v1/canvases")

	toolActions := toolAccessByAction(result.ToolActions)
	require.Contains(t, toolActions, "list_resources")
	assert.True(t, toolActions["list_resources"].Allowed)
	require.Contains(t, toolActions, "patch_staging")
	assert.True(t, toolActions["patch_staging"].Allowed)
	require.Contains(t, toolActions, "read_runtime")
	assert.True(t, toolActions["read_runtime"].Allowed)
	require.Contains(t, toolActions, "list_files")
	assert.True(t, toolActions["list_files"].Allowed)
	require.Contains(t, toolActions, "read_file")
	assert.True(t, toolActions["read_file"].Allowed)
	require.Contains(t, toolActions, "write_file")
	assert.True(t, toolActions["write_file"].Allowed)
	require.Contains(t, toolActions, "delete_file")
	assert.True(t, toolActions["delete_file"].Allowed)
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

func TestReadRuntimeAction_ReadsRunnerLogsByExecutionID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/v1/tasks/task-agent-logs/live-logs", req.URL.Path)
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"type":"line","text":"agent log line"}` + "\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-req.Context().Done()
	}))
	defer broker.Close()
	t.Setenv("TASK_BROKER_BASE_URL", broker.URL)
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "trigger-1",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
		},
		{
			NodeID: "runner-1",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "runnerBash"},
			}),
		},
	}, nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)

	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, event)
		if err != nil {
			return err
		}
		return event.RoutedInTransaction(tx)
	}))

	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      "runner-1",
		RootEventID: event.ID,
		RunID:       run.ID,
		EventID:     event.ID,
		State:       models.CanvasNodeExecutionStateStarted,
		Metadata: datatypes.NewJSONType(map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-agent-logs",
		}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)

	action := readRuntimeAction{
		registry: r.Registry,
		auth:     allowingPermissionChecker{},
	}

	result, err := action.Execute(context.Background(), agents.AgentSessionContext{
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{
		Resource:    "runner_logs",
		ExecutionID: execution.ID.String(),
		Limit:       10,
	})

	require.NoError(t, err)
	read, ok := result.(runtimeReadResult)
	require.True(t, ok)
	payload, ok := read.Payload.(runnerLogsPayload)
	require.True(t, ok)
	require.Len(t, payload.Logs, 1)
	assert.Equal(t, execution.ID.String(), payload.Logs[0].ExecutionID)
	assert.Equal(t, "runner-1", payload.Logs[0].NodeID)
	assert.Equal(t, "task-agent-logs", payload.Logs[0].BrokerTaskID)
	require.Len(t, payload.Logs[0].Records, 1)
	assert.Equal(t, "agent log line", payload.Logs[0].Records[0].Text)
}

func TestReadRuntimeAction_ReadsRunnerLogsByRunIDWithMissingNodeExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/v1/tasks/task-agent-logs/live-logs", req.URL.Path)
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"type":"line","text":"agent log line"}` + "\n"))
	}))
	defer broker.Close()
	t.Setenv("TASK_BROKER_BASE_URL", broker.URL)
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "trigger-1",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
		},
		{
			NodeID: "runner-1",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "runnerBash"},
			}),
		},
		{
			NodeID: "missing-node",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "runnerBash"},
			}),
		},
	}, nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)

	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, event)
		if err != nil {
			return err
		}
		return event.RoutedInTransaction(tx)
	}))

	now := time.Now()
	executions := []models.CanvasNodeExecution{
		{
			ID:            uuid.New(),
			WorkflowID:    canvas.ID,
			NodeID:        "missing-node",
			RootEventID:   event.ID,
			RunID:         run.ID,
			EventID:       event.ID,
			State:         models.CanvasNodeExecutionStateStarted,
			Metadata:      datatypes.NewJSONType(map[string]any{}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			CreatedAt:     &now,
			UpdatedAt:     &now,
		},
		{
			ID:          uuid.New(),
			WorkflowID:  canvas.ID,
			NodeID:      "runner-1",
			RootEventID: event.ID,
			RunID:       run.ID,
			EventID:     event.ID,
			State:       models.CanvasNodeExecutionStateStarted,
			Metadata: datatypes.NewJSONType(map[string]any{
				runneraction.ExecutionMetadataBrokerTaskID: "task-agent-logs",
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			CreatedAt:     &now,
			UpdatedAt:     &now,
		},
	}
	require.NoError(t, database.Conn().Create(&executions).Error)
	require.NoError(t, database.Conn().
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", "missing-node").
		Delete(&models.CanvasNode{}).
		Error)

	action := readRuntimeAction{
		registry: r.Registry,
		auth:     allowingPermissionChecker{},
	}

	result, err := action.Execute(context.Background(), agents.AgentSessionContext{
		OrganizationID: r.Organization.ID.String(),
		UserID:         r.User.String(),
		CanvasID:       canvas.ID.String(),
	}, Input{
		Resource: "runner_logs",
		RunID:    run.ID.String(),
		Limit:    10,
	})

	require.NoError(t, err)
	read, ok := result.(runtimeReadResult)
	require.True(t, ok)
	payload, ok := read.Payload.(runnerLogsPayload)
	require.True(t, ok)
	require.Len(t, payload.Logs, 1)
	assert.Equal(t, executions[1].ID.String(), payload.Logs[0].ExecutionID)
	assert.Equal(t, "runner-1", payload.Logs[0].NodeID)
	assert.Equal(t, "agent log line", payload.Logs[0].Records[0].Text)
}

type allowingPermissionChecker struct{}

func (allowingPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
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
