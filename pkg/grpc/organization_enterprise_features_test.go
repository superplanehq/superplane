package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

func TestDescribeOrganizationReturnsEffectiveFeatures(t *testing.T) {
	r := support.Setup(t)
	license := licensingtest.EnterpriseService(licensing.FeatureGroups, licensing.FeatureCustomRoles)
	service := &OrganizationService{enterpriseFeatures: enterprise.NewOrganizationFeatures(license)}
	ctx := organizationContext(r.Organization.ID.String(), r.User.String())

	response, err := service.DescribeOrganization(ctx, &pb.DescribeOrganizationRequest{})
	require.NoError(t, err)
	assert.Empty(t, response.GetOrganization().GetEnterpriseFeatures())

	_, err = models.GrantEnterpriseFeatures(
		database.Conn(),
		r.Organization.ID,
		[]string{string(licensing.FeatureGroups)},
		models.EnterpriseFeatureSourceAdmin,
		&r.Account.ID,
	)
	require.NoError(t, err)

	response, err = service.DescribeOrganization(ctx, &pb.DescribeOrganizationRequest{})
	require.NoError(t, err)
	assert.Equal(t, []string{string(licensing.FeatureGroups)}, response.GetOrganization().GetEnterpriseFeatures())

	other := support.CreateOrganization(t, r, r.User)
	otherCtx := organizationContext(other.ID.String(), r.User.String())
	response, err = service.DescribeOrganization(otherCtx, &pb.DescribeOrganizationRequest{})
	require.NoError(t, err)
	assert.Empty(t, response.GetOrganization().GetEnterpriseFeatures())
}

func TestDescribeOrganizationDevelopmentLicenseNeedsNoGrant(t *testing.T) {
	r := support.Setup(t)
	issuer := licensingtest.NewIssuer(licensing.DevelopmentKeyID)
	license := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(issuer)),
		licensingtest.StaticSource{Raw: issuer.License(licensing.FeatureGroups)},
	)
	require.NoError(t, license.Refresh(t.Context()))
	service := &OrganizationService{enterpriseFeatures: enterprise.NewOrganizationFeatures(license)}

	response, err := service.DescribeOrganization(
		organizationContext(r.Organization.ID.String(), r.User.String()),
		&pb.DescribeOrganizationRequest{},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{string(licensing.FeatureGroups)}, response.GetOrganization().GetEnterpriseFeatures())
}
