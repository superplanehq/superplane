package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
)

const testWebhookBaseURL = "http://localhost:3000/api/v1"

func canvasSpecFromVersionYAML(ctx context.Context, t *testing.T, orgID, canvasID, versionID string) *pb.Canvas_Spec {
	t.Helper()
	yamlText, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	canvas, err := canvasFromYAMLText(yamlText)
	require.NoError(t, err)
	require.NotNil(t, canvas.GetSpec())
	return canvas.GetSpec()
}

func createDraftVersionID(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasID, displayName string) string {
	t.Helper()

	response, err := CreateCanvasVersion(ctx, r.GitProvider, r.Registry, r.Organization.ID.String(), canvasID, displayName)
	require.NoError(t, err)
	require.NotNil(t, response.GetVersion())
	require.NotNil(t, response.GetVersion().GetMetadata())

	versionID := strings.TrimSpace(response.GetVersion().GetMetadata().GetId())
	require.NotEmpty(t, versionID)

	return versionID
}

func createCanvasWithNoopNode(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasName string) string {
	t.Helper()

	return createGitCanvas(ctx, t, r, canvasName, []*componentpb.Node{
		{
			Id:        "node-1",
			Name:      "Initial Name",
			Component: "noop",
		},
	})
}

// createGitCanvasWithDraft creates an empty git-backed canvas and a registered
// git draft branch version for the current user, returning the canvas record and
// the draft version ID. It replaces the DB-native support.CreateCanvas +
// models.CreateDraftBranchFromLiveInTransaction pattern for git-first tests.
func createGitCanvasWithDraft(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	canvasName string,
) (*models.Canvas, string) {
	t.Helper()

	canvasID := createGitCanvas(ctx, t, r, canvasName, nil)
	created, err := CreateCanvasVersion(ctx, r.GitProvider, r.Registry, r.Organization.ID.String(), canvasID, "")
	require.NoError(t, err)

	canvas, err := models.FindCanvas(r.Organization.ID, uuid.MustParse(canvasID))
	require.NoError(t, err)

	return canvas, created.GetVersion().GetMetadata().GetId()
}

// createGitCanvas creates a git-backed canvas (seeded repository + materialized
// live version) with the given nodes, returning the canvas ID. Pass nil/empty
// nodes for an empty canvas.
func createGitCanvas(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	canvasName string,
	nodes []*componentpb.Node,
) string {
	t.Helper()

	createCanvasResponse, err := CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		r.GitProvider,
		testWebhookBaseURL,
		r.Organization.ID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: canvasName},
			Spec: &pb.Canvas_Spec{
				Nodes: nodes,
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		nil,
	)
	require.NoError(t, err)
	return createCanvasResponse.Canvas.Metadata.Id
}

func createCanvasWithChangeManagement(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasName string) string {
	t.Helper()
	require.NoError(
		t,
		database.Conn().
			Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("change_management_enabled", true).
			Error,
	)
	return createCanvasWithNoopNode(ctx, t, r, canvasName)
}

func createDraftVersion(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasID string, nodeName string) string {
	t.Helper()

	versionID := createDraftVersionID(ctx, t, r, canvasID, "")
	canvasUUID := uuid.MustParse(canvasID)
	version, err := models.FindCanvasVersion(canvasUUID, uuid.MustParse(versionID))
	require.NoError(t, err)

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvasUUID)
	require.NoError(t, err)

	nodes := append([]models.Node(nil), liveVersion.Nodes...)
	edges := append([]models.Edge(nil), liveVersion.Edges...)
	for i := range nodes {
		if nodes[i].ID == "node-1" {
			nodes[i].Name = nodeName
		}
	}

	canvas := materialize.CanvasYAMLFromVersion(liveVersion)
	canvas.Spec.Nodes = nodes
	canvas.Spec.Edges = edges
	canvasYAML, err := materialize.BuildCanvasYAMLFromCanvas(canvas)
	require.NoError(t, err)

	consoleYAML, err := materialize.BuildConsoleYAMLFromVersion(liveVersion)
	require.NoError(t, err)

	commitResp, err := CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		versionID,
		version.CommitSHA,
		"Update draft",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: materialize.CanvasFileName, Content: canvasYAML},
			{Path: materialize.ConsoleFileName, Content: consoleYAML},
		},
		nil,
		testWebhookBaseURL,
		r.AuthService,
	)
	require.NoError(t, err)
	require.NotEmpty(t, commitResp.GetCommitSha())

	return versionID
}

func commitDraftMetadataOnly(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	canvasID string,
	draftVersionID string,
	newCanvasName string,
	newDescription string,
) string {
	t.Helper()

	canvasUUID := uuid.MustParse(canvasID)
	version, err := models.FindCanvasVersion(canvasUUID, uuid.MustParse(draftVersionID))
	require.NoError(t, err)

	nodes := append([]models.Node(nil), version.Nodes...)
	edges := append([]models.Edge(nil), version.Edges...)

	canvas := materialize.CanvasYAMLFromVersion(version)
	canvas.Metadata.Name = newCanvasName
	canvas.Metadata.Description = newDescription
	canvas.Spec.Nodes = nodes
	canvas.Spec.Edges = edges
	canvasYAML, err := materialize.BuildCanvasYAMLFromCanvas(canvas)
	require.NoError(t, err)

	consoleYAML, err := materialize.BuildConsoleYAMLFromVersion(version)
	require.NoError(t, err)

	commitResp, err := CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		draftVersionID,
		version.CommitSHA,
		"Update metadata",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: materialize.CanvasFileName, Content: canvasYAML},
			{Path: materialize.ConsoleFileName, Content: consoleYAML},
		},
		nil,
		testWebhookBaseURL,
		r.AuthService,
	)
	require.NoError(t, err)
	require.NotEmpty(t, commitResp.GetCommitSha())
	return draftVersionID
}

func commitDraftConsoleOnly(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	canvasID string,
	draftVersionID string,
	panelTitle string,
) string {
	t.Helper()

	canvasUUID := uuid.MustParse(canvasID)
	version, err := models.FindCanvasVersion(canvasUUID, uuid.MustParse(draftVersionID))
	require.NoError(t, err)

	panels := append([]models.ConsolePanel(nil), version.ConsolePanels.Data()...)
	panels = append(panels, models.ConsolePanel{
		ID:   "notes",
		Type: models.ConsolePanelTypeMarkdown,
		Content: map[string]any{
			"body": panelTitle,
		},
	})
	layout := append([]models.ConsoleLayoutItem(nil), version.ConsoleLayout.Data()...)
	layout = append(layout, models.ConsoleLayoutItem{I: "notes", X: 0, Y: 0, W: 4, H: 2})

	consoleYAML, err := materialize.BuildConsoleYAMLFromDashboard(&models.ConsoleYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Metadata: models.ConsoleYAMLMetadata{
			CanvasID: canvasID,
			Name:     version.Name,
		},
		Spec: models.ConsoleYAMLSpec{
			Panels: panels,
			Layout: layout,
		},
	})
	require.NoError(t, err)

	canvasYAML, err := materialize.BuildCanvasYAMLFromCanvas(materialize.CanvasYAMLFromVersion(version))
	require.NoError(t, err)

	commitResp, err := CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		nil,
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		draftVersionID,
		version.CommitSHA,
		"Update console",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: materialize.CanvasFileName, Content: canvasYAML},
			{Path: materialize.ConsoleFileName, Content: consoleYAML},
		},
		nil,
		testWebhookBaseURL,
		r.AuthService,
	)
	require.NoError(t, err)
	require.NotEmpty(t, commitResp.GetCommitSha())
	return draftVersionID
}
