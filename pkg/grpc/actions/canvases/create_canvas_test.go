package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCanvasUsageService struct {
	checkOrganizationResp *usagepb.CheckOrganizationLimitsResponse
}

func (s *fakeCanvasUsageService) Enabled() bool {
	return true
}

func (s *fakeCanvasUsageService) SetupAccount(context.Context, string) (*usagepb.SetupAccountResponse, error) {
	return &usagepb.SetupAccountResponse{}, nil
}

func (s *fakeCanvasUsageService) SetupOrganization(context.Context, string, string) (*usagepb.SetupOrganizationResponse, error) {
	return &usagepb.SetupOrganizationResponse{}, nil
}

func (s *fakeCanvasUsageService) DescribeAccountLimits(context.Context, string) (*usagepb.DescribeAccountLimitsResponse, error) {
	return &usagepb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakeCanvasUsageService) DescribeOrganizationLimits(context.Context, string) (*usagepb.DescribeOrganizationLimitsResponse, error) {
	return &usagepb.DescribeOrganizationLimitsResponse{}, nil
}

func (s *fakeCanvasUsageService) DescribeOrganizationUsage(context.Context, string) (*usagepb.DescribeOrganizationUsageResponse, error) {
	return &usagepb.DescribeOrganizationUsageResponse{}, nil
}

func (s *fakeCanvasUsageService) CheckAccountLimits(
	context.Context,
	string,
	*usagepb.AccountState,
) (*usagepb.CheckAccountLimitsResponse, error) {
	return &usagepb.CheckAccountLimitsResponse{Allowed: true}, nil
}

func (s *fakeCanvasUsageService) CheckOrganizationLimits(
	context.Context,
	string,
	*usagepb.OrganizationState,
	*usagepb.CanvasState,
) (*usagepb.CheckOrganizationLimitsResponse, error) {
	if s.checkOrganizationResp != nil {
		return s.checkOrganizationResp, nil
	}

	return &usagepb.CheckOrganizationLimitsResponse{Allowed: true}, nil
}

var _ usage.Service = (*fakeCanvasUsageService)(nil)

func TestCreateCanvasDuplicateName(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Duplicate Canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	baseURL := "https://example.com"
	_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, workflow, nil, nil)
	require.NoError(t, err)

	_, err = CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, workflow, nil, nil)
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestCreateCanvasInheritsOrganizationChangeManagementWhenEnabled(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	nowEnabled := true
	require.NoError(t, database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("change_management_enabled", nowEnabled).Error)

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Change management default canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	baseURL := "https://example.com"
	response, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, workflow, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)
	// New canvases inherit organization change management setting.
	require.NotNil(t, response.Canvas.Spec)
	require.NotNil(t, response.Canvas.Spec.ChangeManagement)
	require.True(t, response.Canvas.Spec.ChangeManagement.Enabled)

	require.NotEmpty(t, response.Canvas.Metadata.Id)
	createdCanvasUUID, parseErr := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, parseErr)
	createdCanvas, findErr := models.FindCanvas(r.Organization.ID, createdCanvasUUID)
	require.NoError(t, findErr)
	require.True(t, createdCanvas.ChangeManagementEnabled)
}

func TestCreateCanvasOnFreshOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        "Health Check Monitor",
			Description: "Quick start canvas on a fresh organization",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	baseURL := "https://example.com"
	response, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, canvas, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Metadata)
	require.Equal(t, "Health Check Monitor", response.Canvas.Metadata.Name)
	require.Equal(t, r.Organization.ID.String(), response.Canvas.Metadata.OrganizationId)
	require.NotEmpty(t, response.Canvas.Metadata.Id)

	canvasID, err := uuid.Parse(response.Canvas.Metadata.Id)
	require.NoError(t, err)
	persisted, err := models.FindCanvas(r.Organization.ID, canvasID)
	require.NoError(t, err)
	require.Equal(t, "Health Check Monitor", persisted.Name)
	require.Equal(t, r.Organization.ID, persisted.OrganizationID)
}

func TestCreateCanvasRejectsInvalidEdgeChannel(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Invalid HTTP Channel",
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
	}

	baseURL := "https://example.com"
	_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, canvas, nil, nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), `source node http-1 does not have output channel "default"`)
}

func TestCreateCanvasWithUsageRejectsLimitViolation(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	workflow := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name: "Limited Canvas",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.Node{},
			Edges: []*componentpb.Edge{},
		},
	}

	service := &fakeCanvasUsageService{
		checkOrganizationResp: &usagepb.CheckOrganizationLimitsResponse{
			Allowed: false,
			Violations: []*usagepb.LimitViolation{
				{
					Limit:           usagepb.LimitName_LIMIT_NAME_MAX_CANVASES,
					ConfiguredLimit: 1,
					CurrentValue:    2,
				},
			},
		},
	}

	baseURL := "https://example.com"
	_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, baseURL, r.Organization.ID, workflow, nil, service)
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Equal(t, "organization canvas limit exceeded", status.Convert(err).Message())
}
