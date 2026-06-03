package canvases

import (
	"context"
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
const missingCommitSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func createDraftVersion(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasID string, nodeName string) string {
	t.Helper()

	canvasUUID := uuid.MustParse(canvasID)
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvasUUID)
	require.NoError(t, err)

	branchResp, err := CreateDraftBranch(ctx, r.GitProvider, r.Registry, r.Organization.ID.String(), canvasID, nodeName)
	require.NoError(t, err)

	nodes := append([]models.Node(nil), liveVersion.Nodes...)
	edges := append([]models.Edge(nil), liveVersion.Edges...)
	for i := range nodes {
		if nodes[i].ID == "node-1" {
			nodes[i].Name = nodeName
		}
	}

	canvasYAML, err := materialize.BuildCanvasYAML(
		liveVersion.Name,
		liveVersion.Description,
		nodes,
		edges,
		liveVersion.ChangeManagementEnabled,
		liveVersion.EffectiveChangeRequestApprovers(),
	)
	require.NoError(t, err)

	consoleYAML, err := materialize.BuildConsoleYAMLFromVersion(liveVersion)
	require.NoError(t, err)

	commitResp, err := CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		branchResp.GetBranch().GetBranchName(),
		branchResp.GetBranch().GetTipSha(),
		"Update draft",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: materialize.CanvasFileName, Content: canvasYAML},
			{Path: materialize.ConsoleFileName, Content: consoleYAML},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, commitResp.GetCommitSha())

	return commitResp.GetCommitSha()
}

func commitDraftMetadataOnly(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	canvasID string,
	draftVersionSHA string,
	newCanvasName string,
	newDescription string,
) string {
	t.Helper()

	canvasUUID := uuid.MustParse(canvasID)
	version, err := models.FindCanvasVersion(canvasUUID, draftVersionSHA)
	require.NoError(t, err)

	draftBranch, err := models.FindDraftBranch(canvasUUID, version.GitBranch)
	require.NoError(t, err)

	nodes := append([]models.Node(nil), version.Nodes...)
	edges := append([]models.Edge(nil), version.Edges...)

	canvasYAML, err := materialize.BuildCanvasYAML(
		newCanvasName,
		newDescription,
		nodes,
		edges,
		version.ChangeManagementEnabled,
		version.EffectiveChangeRequestApprovers(),
	)
	require.NoError(t, err)

	consoleYAML, err := materialize.BuildConsoleYAMLFromVersion(version)
	require.NoError(t, err)

	commitResp, err := CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		draftBranch.BranchName,
		draftBranch.TipSHA,
		"Update metadata",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: materialize.CanvasFileName, Content: canvasYAML},
			{Path: materialize.ConsoleFileName, Content: consoleYAML},
		},
	)
	require.NoError(t, err)
	return commitResp.GetCommitSha()
}

func commitDraftConsoleOnly(
	ctx context.Context,
	t *testing.T,
	r *support.ResourceRegistry,
	canvasID string,
	draftVersionSHA string,
	panelTitle string,
) string {
	t.Helper()

	canvasUUID := uuid.MustParse(canvasID)
	version, err := models.FindCanvasVersion(canvasUUID, draftVersionSHA)
	require.NoError(t, err)

	draftBranch, err := models.FindDraftBranch(canvasUUID, version.GitBranch)
	require.NoError(t, err)

	panels := append([]models.DashboardPanel(nil), version.ConsolePanels.Data()...)
	panels = append(panels, models.DashboardPanel{
		ID:   "notes",
		Type: models.DashboardPanelTypeMarkdown,
		Content: map[string]any{
			"body": panelTitle,
		},
	})
	layout := append([]models.DashboardLayoutItem(nil), version.ConsoleLayout.Data()...)
	layout = append(layout, models.DashboardLayoutItem{I: "notes", X: 0, Y: 0, W: 4, H: 2})

	consoleYAML, err := materialize.BuildConsoleYAMLFromDashboard(&models.DashboardYAML{
		APIVersion: models.DashboardAPIVersion,
		Kind:       models.ConsoleKind,
		Metadata: models.DashboardYAMLMetadata{
			CanvasID: canvasID,
			Name:     version.Name,
		},
		Spec: models.DashboardYAMLSpec{
			Panels: panels,
			Layout: layout,
		},
	})
	require.NoError(t, err)

	canvasYAML, err := materialize.BuildCanvasYAML(
		version.Name,
		version.Description,
		version.Nodes,
		version.Edges,
		version.ChangeManagementEnabled,
		version.EffectiveChangeRequestApprovers(),
	)
	require.NoError(t, err)

	commitResp, err := CommitCanvasRepositoryFiles(
		ctx,
		r.GitProvider,
		r.Registry,
		r.Organization.ID.String(),
		canvasID,
		draftBranch.BranchName,
		draftBranch.TipSHA,
		"Update console",
		[]*pb.CanvasRepositoryFileOperation{
			{Path: materialize.CanvasFileName, Content: canvasYAML},
			{Path: materialize.ConsoleFileName, Content: consoleYAML},
		},
	)
	require.NoError(t, err)
	return commitResp.GetCommitSha()
}

func createCanvasWithNoopNode(ctx context.Context, t *testing.T, r *support.ResourceRegistry, canvasName string) string {
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
				Nodes: []*componentpb.Node{
					{
						Id:        "node-1",
						Name:      "Initial Name",
						Component: "noop",
					},
				},
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
