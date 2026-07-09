package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
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

func (s *fakeCanvasUsageService) SetupOrganization(context.Context, string, string, usage.SetupOrganizationDetails) (*usagepb.SetupOrganizationResponse, error) {
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

	baseURL := "https://example.com"
	_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, r.GitProvider, baseURL, r.Organization.ID, "Duplicate Canvas", "", nil)
	require.NoError(t, err)

	_, err = CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, r.GitProvider, baseURL, r.Organization.ID, "Duplicate Canvas", "", nil)
	require.Error(t, err)
	require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
}

func TestCreateCanvasRejectsWhitespaceOnlyName(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	baseURL := "https://example.com"
	_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, r.GitProvider, baseURL, r.Organization.ID, "   ", "", nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Equal(t, "canvas name is required", func() string {
		_, msg, ok := grpcerrors.HandlerStatus(err)
		if ok {
			return msg
		}
		return err.Error()
	}())
}

func TestCreateCanvasOnFreshOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	baseURL := "https://example.com"
	response, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, r.GitProvider, baseURL, r.Organization.ID, "Health Check Monitor", "Quick start canvas on a fresh organization", nil)
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

	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), persisted)
	require.NoError(t, err)
	require.Empty(t, liveVersion.Nodes)
	require.Empty(t, liveVersion.Edges)
}

func TestCreateCanvasWithUsageRejectsLimitViolation(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

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
	_, err := CreateCanvas(ctx, r.Registry, r.Encryptor, r.AuthService, r.GitProvider, baseURL, r.Organization.ID, "Limited Canvas", "", service)
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, grpcerrors.Code(err))
	assert.Equal(t, "organization canvas limit exceeded", status.Convert(err).Message())
}
