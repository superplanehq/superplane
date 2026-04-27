package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "version id is required")
	})

	t.Run("valid draft version id -> updates draft", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvas.ID, r.User, nil, nil)
		require.NoError(t, err)

		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		response, err := UpdateCanvasVersion(
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
		require.NotNil(t, response)
		require.NotNil(t, response.Version)
	})

	t.Run("usage limit violation blocks oversized draft", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvas.ID, r.User, nil, nil)
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
		)

		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.ResourceExhausted, s.Code())
		assert.Equal(t, "canvas node limit exceeded", s.Message())
	})

	t.Run("invalid source output channel -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		draftVersion, err := models.SaveCanvasDraftInTransaction(database.Conn(), canvas.ID, r.User, nil, nil)
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
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), `source node http-1 does not have output channel "default"`)
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
