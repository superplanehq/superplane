package grpc

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	grpcApps "github.com/superplanehq/superplane/pkg/grpc/actions/apps"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/apps"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// nonAlphanumeric matches any character that isn't a lowercase letter, digit, or underscore.
var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

type AppsService struct {
	registry *registry.Registry
}

func NewAppsService(reg *registry.Registry) *AppsService {
	return &AppsService{registry: reg}
}

// orgSlugFromName derives a slug from an org's Name field.
// It lowercases, collapses non-alphanumeric runs to underscores, and trims underscores.
func orgSlugFromName(name string) string {
	lower := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return '_'
	}, name)
	slug := nonAlphanumeric.ReplaceAllString(lower, "_")
	return strings.Trim(slug, "_")
}

func (s *AppsService) ListApps(ctx context.Context, req *pb.ListAppsRequest) (*pb.ListAppsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.ListApps(ctx, organizationID)
}

func (s *AppsService) DescribeApp(ctx context.Context, req *pb.DescribeAppRequest) (*pb.DescribeAppResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.DescribeApp(ctx, organizationID, req.Id)
}

func (s *AppsService) CreateApp(ctx context.Context, req *pb.CreateAppRequest) (*pb.CreateAppResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	org, err := models.FindOrganizationByID(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load organization: %v", err)
	}

	return grpcApps.CreateApp(
		ctx,
		uuid.MustParse(organizationID),
		orgSlugFromName(org.Name),
		req.DisplayName,
		req.AppSlug,
		req.Description,
	)
}

func (s *AppsService) DeleteApp(ctx context.Context, req *pb.DeleteAppRequest) (*pb.DeleteAppResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.DeleteApp(ctx, uuid.MustParse(organizationID), req.Id)
}

func (s *AppsService) SyncApp(ctx context.Context, req *pb.SyncAppRequest) (*pb.SyncAppResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.SyncApp(ctx, uuid.MustParse(organizationID), req.Id)
}

func (s *AppsService) GetAppDashboard(ctx context.Context, req *pb.GetAppDashboardRequest) (*pb.GetAppDashboardResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.GetAppDashboard(ctx, organizationID, req.AppId)
}

func (s *AppsService) UpdateAppDashboard(ctx context.Context, req *pb.UpdateAppDashboardRequest) (*pb.UpdateAppDashboardResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.UpdateAppDashboard(ctx, organizationID, req.AppId, req.Panels, req.Layout)
}

func (s *AppsService) GetAppCanvas(ctx context.Context, req *pb.GetAppCanvasRequest) (*pb.GetAppCanvasResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.GetAppCanvas(ctx, s.registry, organizationID, req.AppId)
}

func (s *AppsService) ListAppDocs(ctx context.Context, req *pb.ListAppDocsRequest) (*pb.ListAppDocsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.ListAppDocs(ctx, organizationID, req.AppId)
}

func (s *AppsService) GetAppDoc(ctx context.Context, req *pb.GetAppDocRequest) (*pb.GetAppDocResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.GetAppDoc(ctx, organizationID, req.AppId, req.Path)
}

func (s *AppsService) UpdateAppDoc(ctx context.Context, req *pb.UpdateAppDocRequest) (*pb.UpdateAppDocResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return grpcApps.UpdateAppDoc(ctx, organizationID, req.AppId, req.Path, req.Content)
}
