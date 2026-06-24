package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func canvasSpecFromVersionYAML(ctx context.Context, t *testing.T, orgID, canvasID, versionID string) *pb.Canvas_Spec {
	t.Helper()
	yamlText, err := ReadRepositorySpecFile(ctx, orgID, canvasID, versionID, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	canvas, err := canvasFromYAMLText(yamlText)
	require.NoError(t, err)
	require.NotNil(t, canvas.GetSpec())
	return canvas.GetSpec()
}

func Test__UpdateCanvasVersion(t *testing.T) {
	r := support.Setup(t)

	t.Run("no version id -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			"",
			testPbCanvas(canvas.Name),
			nil,
			"",
			r.AuthService,
		)

		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Contains(t, msg, "version id is required")
	})

	t.Run("valid draft version id -> updates draft", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		version, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			draftVersion.ID.String(),
			testPbCanvas(canvas.Name),
			nil,
			"",
			r.AuthService,
		)

		require.NoError(t, err)
		require.NotNil(t, version)
	})

	t.Run("usage limit violation blocks oversized draft", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
		require.NoError(t, err)

		service := &fakeCanvasUsageService{
			checkOrganizationResp: &usagepb.CheckOrganizationLimitsResponse{
				Allowed: false,
				Violations: []*usagepb.LimitViolation{
					{
						Limit:           usagepb.LimitName_LIMIT_NAME_MAX_NODES_PER_CANVAS,
						ConfiguredLimit: 1,
						CurrentValue:    2,
					},
				},
			},
		}

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err = UpdateCanvasVersionWithUsage(
			ctx,
			service,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			draftVersion.ID.String(),
			testPbCanvas(canvas.Name),
			nil,
			"",
			r.AuthService,
			false,
		)

		require.Error(t, err)
		assert.Equal(t, codes.ResourceExhausted, grpcerrors.Code(err))
		assert.Equal(t, "canvas node limit exceeded", status.Convert(err).Message())
	})

	t.Run("invalid source output channel -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		_, err = UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			draftVersion.ID.String(),
			&pb.Canvas{
				Metadata: &pb.Canvas_Metadata{
					Name: canvas.Name,
				},
				Spec: &pb.Canvas_Spec{
					Nodes: []*componentpb.Node{
						{
							Id:        "http-1",
							Name:      "HTTP Request",
							Component: "http",
							Configuration: structFromAnyMap(t, map[string]any{
								"method": "GET",
								"url":    "https://example.com",
							}),
						},
						{
							Id:        "if-1",
							Name:      "If",
							Component: "if",
							Configuration: structFromAnyMap(t, map[string]any{
								"expression": "true",
							}),
						},
					},
					Edges: []*componentpb.Edge{
						{
							SourceId: "http-1",
							TargetId: "if-1",
							Channel:  "default",
						},
					},
				},
			},
			nil,
			"",
			r.AuthService,
		)

		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Contains(t, msg, `source node http-1 does not have output channel "default"`)
	})

	t.Run("invalid node field type -> serialized node carries error_message", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		version, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			draftVersion.ID.String(),
			&pb.Canvas{
				Metadata: &pb.Canvas_Metadata{
					Name: canvas.Name,
				},
				Spec: &pb.Canvas_Spec{
					Nodes: []*componentpb.Node{
						{
							Id:        "wait-1",
							Name:      "Wait",
							Component: "wait",
							Configuration: structFromAnyMap(t, map[string]any{
								"mode":    "interval",
								"waitFor": 30,
								"unit":    "seconds",
							}),
						},
					},
					Edges: []*componentpb.Edge{},
				},
			},
			nil,
			"",
			r.AuthService,
		)

		require.NoError(t, err)
		require.NotNil(t, version)
		spec := canvasSpecFromVersionYAML(ctx, t, r.Organization.ID.String(), canvas.ID.String(), version.ID.String())
		require.Len(t, spec.Nodes, 1)
		errMsg := spec.Nodes[0].GetErrorMessage()
		require.NotEmpty(t, errMsg)
		assert.Contains(t, errMsg, "waitFor")
	})

	t.Run("integration component with enabled capability -> updates without node error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
		require.NoError(t, err)

		integration := support.CreateIntegrationWithCapabilities(t, r.Organization.ID, []models.CapabilityState{
			{Name: "github.getIssue", State: core.IntegrationCapabilityStateEnabled},
		})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		version, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			draftVersion.ID.String(),
			testPbCanvasWithGitHubIssueNode(t, canvas.Name, integration.ID.String()),
			nil,
			"",
			r.AuthService,
		)

		require.NoError(t, err)
		require.NotNil(t, version)
		spec := canvasSpecFromVersionYAML(ctx, t, r.Organization.ID.String(), canvas.ID.String(), version.ID.String())
		require.Len(t, spec.Nodes, 1)

		node := spec.Nodes[0]
		assert.Empty(t, node.GetErrorMessage())
		require.NotNil(t, node.GetIntegration())
		require.NotNil(t, node.GetIntegration().Id)
		assert.Equal(t, integration.ID.String(), *node.GetIntegration().Id)
	})

	t.Run("integration component with disabled capability -> serialized node carries error_message", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.CreateDraftBranchFromLiveInTransaction(database.Conn(), canvas.ID, r.User, "", nil, nil)
		require.NoError(t, err)

		integration := support.CreateIntegrationWithCapabilities(t, r.Organization.ID, []models.CapabilityState{
			{Name: "github.getIssue", State: core.IntegrationCapabilityStateDisabled},
		})

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		version, err := UpdateCanvasVersion(
			ctx,
			r.Encryptor,
			r.Registry,
			r.Organization.ID.String(),
			canvas.ID.String(),
			draftVersion.ID.String(),
			testPbCanvasWithGitHubIssueNode(t, canvas.Name, integration.ID.String()),
			nil,
			"",
			r.AuthService,
		)

		require.NoError(t, err)
		require.NotNil(t, version)
		spec := canvasSpecFromVersionYAML(ctx, t, r.Organization.ID.String(), canvas.ID.String(), version.ID.String())
		require.Len(t, spec.Nodes, 1)

		errMsg := spec.Nodes[0].GetErrorMessage()
		require.NotEmpty(t, errMsg)
		assert.Contains(t, errMsg, "github.getIssue is not enabled for integration "+integration.InstallationName)
		assert.Contains(t, errMsg, integration.InstallationName)
	})
}

func testPbCanvas(name string) *pb.Canvas {
	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: name,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}
}

func testPbCanvasWithGitHubIssueNode(t *testing.T, name string, integrationID string) *pb.Canvas {
	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: name,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:        "github-issue",
					Name:      "Get Issue",
					Component: "github.getIssue",
					Integration: &componentpb.IntegrationRef{
						Id: &integrationID,
					},
					Configuration: structFromAnyMap(t, map[string]any{
						"repository":  "superplanehq/superplane",
						"issueNumber": "1",
					}),
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}
}
