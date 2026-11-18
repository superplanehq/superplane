package enterprisefeatures

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/enterprise_features"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

func ListOrganizationEnterpriseFeatures(ctx context.Context, features *enterprise.OrganizationFeatures, req *pb.ListOrganizationEnterpriseFeaturesRequest) (*pb.ListOrganizationEnterpriseFeaturesResponse, error) {
	if _, err := installationAdmin(ctx); err != nil {
		return nil, err
	}

	orgID, err := uuid.Parse(req.GetOrganizationId())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "Invalid organization ID.")
	}

	views, licenseActive, err := features.List(ctx, orgID)
	if err != nil {
		return nil, organizationFeatureError(err, "Could not list Enterprise features. Try again.")
	}

	return &pb.ListOrganizationEnterpriseFeaturesResponse{
		Features:      featureProtos(views),
		LicenseActive: licenseActive,
	}, nil
}

func SetOrganizationEnterpriseFeatures(ctx context.Context, features *enterprise.OrganizationFeatures, req *pb.SetOrganizationEnterpriseFeaturesRequest) (*pb.SetOrganizationEnterpriseFeaturesResponse, error) {
	admin, err := installationAdmin(ctx)
	if err != nil {
		return nil, err
	}

	orgID, err := uuid.Parse(req.GetOrganizationId())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "Invalid organization ID.")
	}

	added, removed, err := features.Set(ctx, orgID, req.GetFeatures(), admin.ID)
	if err != nil {
		return nil, organizationFeatureError(err, "Could not update Enterprise features. Try again.")
	}

	views, _, err := features.List(ctx, orgID)
	if err != nil {
		return nil, organizationFeatureError(err, "Could not list Enterprise features. Try again.")
	}

	return &pb.SetOrganizationEnterpriseFeaturesResponse{
		Features: featureProtos(views),
		Added:    added,
		Removed:  removed,
	}, nil
}

func ListEnterpriseFeatureOrganizations(ctx context.Context, req *pb.ListEnterpriseFeatureOrganizationsRequest) (*pb.ListEnterpriseFeatureOrganizationsResponse, error) {
	if _, err := installationAdmin(ctx); err != nil {
		return nil, err
	}

	key := req.GetFeatureKey()
	if !licensing.IsRecognizedFeature(key) {
		return nil, grpcerrors.InvalidArgument(errors.New(key), "The feature is not recognized.")
	}

	limit, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)
	ids, total, err := models.ListOrganizationsWithEnterpriseFeature(db, key, limit, offset)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not list organizations. Try again.")
	}

	organizations, err := enterpriseFeatureOrganizations(db, key, ids)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not list organizations. Try again.")
	}

	return &pb.ListEnterpriseFeatureOrganizationsResponse{
		Organizations: organizations,
		NextPageToken: nextPageToken(offset, len(organizations), total),
		Total:         total,
	}, nil
}

func ListLicenseActivations(ctx context.Context, req *pb.ListLicenseActivationsRequest) (*pb.ListLicenseActivationsResponse, error) {
	if _, err := installationAdmin(ctx); err != nil {
		return nil, err
	}

	limit, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}

	rows, total, err := models.ListLicenseActivations(database.DB(ctx), limit, offset)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not list license activations. Try again.")
	}

	names, err := organizationNames(removedGrantOrganizationIDs(rows))
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not list license activations. Try again.")
	}

	activations := make([]*pb.LicenseActivation, 0, len(rows))
	for _, row := range rows {
		activations = append(activations, activationProto(row, names))
	}

	return &pb.ListLicenseActivationsResponse{
		Activations:   activations,
		NextPageToken: nextPageToken(offset, len(activations), total),
	}, nil
}

func installationAdmin(ctx context.Context) (*models.Account, error) {
	accountID, ok := authentication.GetAccountIDFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "Account authentication is required.")
	}

	admin, err := models.FindAccountByID(accountID)
	if err != nil || !admin.IsInstallationAdmin() {
		return nil, grpcerrors.PermissionDenied(nil, "Installation admin access is required.")
	}

	return admin, nil
}

func organizationFeatureError(err error, fallback string) error {
	if err == nil {
		return nil
	}

	var unlicensed enterprise.UnlicensedFeatureError
	switch {
	case errors.As(err, &unlicensed):
		return grpcerrors.FailedPrecondition(err, unlicensed.Error())
	case errors.Is(err, enterprise.ErrActiveEnterpriseLicenseRequired):
		return grpcerrors.FailedPrecondition(err, enterprise.ActiveEnterpriseLicenseMessage)
	case errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.NotFound(err, "Organization not found.")
	default:
		return grpcerrors.Internal(err, fallback)
	}
}

func featureProtos(views []enterprise.OrganizationFeatureView) []*pb.OrganizationEnterpriseFeature {
	features := make([]*pb.OrganizationEnterpriseFeature, 0, len(views))
	for _, view := range views {
		feature := &pb.OrganizationEnterpriseFeature{
			Key:           view.Key,
			Licensed:      view.Licensed,
			Granted:       view.Granted,
			Effective:     view.Effective,
			Source:        view.Source,
			GrantedByName: view.GrantedByName,
		}
		if view.GrantedBy != nil {
			feature.GrantedBy = view.GrantedBy.String()
		}
		if view.Granted && !view.CreatedAt.IsZero() {
			feature.CreatedAt = timestamppb.New(view.CreatedAt)
		}
		features = append(features, feature)
	}

	return features
}

func enterpriseFeatureOrganizations(db *gorm.DB, key string, ids []uuid.UUID) ([]*pb.EnterpriseFeatureOrganization, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	stringIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		stringIDs = append(stringIDs, id.String())
	}

	organizations, err := models.ListOrganizationsByIDs(stringIDs)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]models.Organization{}
	for _, organization := range organizations {
		byID[organization.ID] = organization
	}

	var grants []models.OrganizationEnterpriseFeature
	if err := db.Where("feature_key = ? AND organization_id IN ?", key, ids).Find(&grants).Error; err != nil {
		return nil, err
	}
	grantByOrg := map[uuid.UUID]models.OrganizationEnterpriseFeature{}
	for _, grant := range grants {
		grantByOrg[grant.OrganizationID] = grant
	}

	result := make([]*pb.EnterpriseFeatureOrganization, 0, len(ids))
	for _, id := range ids {
		organization, ok := byID[id]
		grant, granted := grantByOrg[id]
		if !ok || !granted {
			continue
		}
		result = append(result, &pb.EnterpriseFeatureOrganization{
			OrganizationId: organization.ID.String(),
			Name:           organization.Name,
			Slug:           organization.Slug,
			Source:         grant.Source,
			CreatedAt:      timestamppb.New(grant.CreatedAt),
		})
	}

	return result, nil
}

func activationProto(row models.InstallationLicenseActivation, names map[uuid.UUID]string) *pb.LicenseActivation {
	activation := &pb.LicenseActivation{
		Id:                strconv.FormatInt(row.ID, 10),
		LicenseId:         row.LicenseID.String(),
		LicenseFeatures:   []string(row.LicenseFeatures),
		ActivatedAt:       timestamppb.New(row.ActivatedAt),
		OrganizationCount: int32(row.OrganizationCount),
		AutoGranted:       row.AutoGranted,
	}
	if row.DeactivatedAt != nil {
		activation.DeactivatedAt = timestamppb.New(*row.DeactivatedAt)
	}
	if row.DeactivationReason != nil {
		activation.DeactivationReason = *row.DeactivationReason
	}
	for _, grant := range row.RemovedGrants {
		activation.RemovedGrants = append(activation.RemovedGrants, &pb.RemovedEnterpriseGrant{
			OrganizationId:   grant.OrganizationID.String(),
			OrganizationName: names[grant.OrganizationID],
			FeatureKey:       grant.FeatureKey,
			Source:           grant.Source,
			RemovedBy:        grant.RemovedBy,
			RemovedAt:        timestamppb.New(grant.RemovedAt),
		})
	}

	return activation
}

func removedGrantOrganizationIDs(rows []models.InstallationLicenseActivation) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	var ids []uuid.UUID
	for _, row := range rows {
		for _, grant := range row.RemovedGrants {
			if _, ok := seen[grant.OrganizationID]; ok {
				continue
			}
			seen[grant.OrganizationID] = struct{}{}
			ids = append(ids, grant.OrganizationID)
		}
	}

	return ids
}

func organizationNames(ids []uuid.UUID) (map[uuid.UUID]string, error) {
	names := map[uuid.UUID]string{}
	if len(ids) == 0 {
		return names, nil
	}

	stringIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		stringIDs = append(stringIDs, id.String())
	}
	organizations, err := models.ListOrganizationsByIDs(stringIDs)
	if err != nil {
		return nil, err
	}
	for _, organization := range organizations {
		names[organization.ID] = organization.Name
	}

	return names, nil
}

func page(pageSize int32, pageToken string) (limit, offset int, err error) {
	if pageSize < 0 {
		return 0, 0, grpcerrors.InvalidArgument(errors.New("negative page size"), "Page size must be zero or greater.")
	}

	limit = int(pageSize)
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	if pageToken == "" {
		return limit, 0, nil
	}

	offset, err = strconv.Atoi(pageToken)
	if err != nil || offset < 0 {
		return 0, 0, grpcerrors.InvalidArgument(err, "Page token is not valid.")
	}

	return limit, offset, nil
}

func nextPageToken(offset, count int, total int64) string {
	next := int64(offset + count)
	if count == 0 || next >= total {
		return ""
	}

	return strconv.FormatInt(next, 10)
}
