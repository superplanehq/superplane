package githubenterprise

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"google.golang.org/grpc/codes"
)

func assertNotLicensed(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	assert.Equal(t, licensing.ErrNotLicensed.Error(), grpcerrors.StatusMessage(err))
}

func TestServiceRequiresLicense(t *testing.T) {
	t.Run("Community denies a GitHub Enterprise host", func(t *testing.T) {
		service := NewService(licensing.Community)
		assertNotLicensed(t, service.AuthorizeHost(context.Background(), uuid.New(), "acme.ghe.com"))
	})

	t.Run("nil entitlements deny", func(t *testing.T) {
		service := NewService(nil)
		assertNotLicensed(t, service.AuthorizeHost(context.Background(), uuid.New(), "acme.ghe.com"))
	})

	t.Run("a different feature does not unlock GitHub Enterprise", func(t *testing.T) {
		service := NewService(licensingtest.EnterpriseService(licensing.FeatureGroups))
		assertNotLicensed(t, service.AuthorizeHost(context.Background(), uuid.New(), "acme.ghe.com"))
	})

	t.Run("Enterprise allows a GitHub Enterprise host", func(t *testing.T) {
		service := NewService(licensingtest.EnterpriseService(licensing.FeatureGitHubEnterprise))
		require.NoError(t, service.AuthorizeHost(context.Background(), uuid.New(), "acme.ghe.com"))
	})

	t.Run("a grant for another organization does not allow the host", func(t *testing.T) {
		granted := uuid.New()
		service := NewService(licensingtest.EnterpriseServiceForOrganizations(
			map[uuid.UUID][]licensing.Feature{granted: {licensing.FeatureGitHubEnterprise}},
			licensing.FeatureGitHubEnterprise,
		))
		assertNotLicensed(t, service.AuthorizeHost(context.Background(), uuid.New(), "acme.ghe.com"))
		require.NoError(t, service.AuthorizeHost(context.Background(), granted, "acme.ghe.com"))
	})
}
