package github

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/ee/githubenterprise"
	"github.com/superplanehq/superplane/pkg/enterprise"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func licensedGitHubEnterprise() enterprise.GitHubEnterprise {
	return githubenterprise.NewService(licensingtest.EnterpriseService(licensing.FeatureGitHubEnterprise))
}

func communityGitHubEnterprise() enterprise.GitHubEnterprise {
	return githubenterprise.NewService(licensing.Community)
}

func withLicensedGitHubEnterprise(t *testing.T) {
	t.Helper()
	t.Cleanup(withGitHubEnterpriseForTest(licensedGitHubEnterprise()))
}

func enableGitHubEnterpriseFlag(t *testing.T) {
	t.Helper()
	t.Cleanup(withGitHubEnterpriseFlagForTest(func(uuid.UUID) bool { return true }))
}

func withGitHubEnterpriseCloudAllowed(t *testing.T) {
	t.Helper()
	withLicensedGitHubEnterprise(t)
	enableGitHubEnterpriseFlag(t)
}

func TestGitHubEnterpriseHostRequiresTheOrganizationGrant(t *testing.T) {
	orgID := uuid.New()
	granted := uuid.New()
	t.Cleanup(withGitHubEnterpriseForTest(githubenterprise.NewService(
		licensingtest.EnterpriseServiceForOrganizations(
			map[uuid.UUID][]licensing.Feature{
				granted: {licensing.FeatureGitHubEnterprise},
			},
			licensing.FeatureGitHubEnterprise,
		),
	)))
	enableGitHubEnterpriseFlag(t)

	_, err := WizardCreateConfiguration(orgID, map[string]any{"host": "acme.ghe.com"}, nil)
	require.Error(t, err)
	assert.Equal(t, licensing.ErrNotLicensed.Error(), grpcerrors.StatusMessage(err))

	_, err = WizardCreateConfiguration(orgID, map[string]any{"host": "github.com"}, nil)
	require.NoError(t, err)

	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}}}
	got, err := WizardCreateConfiguration(granted, map[string]any{"host": "https://acme.ghe.com"}, httpCtx)
	require.NoError(t, err)
	assert.Equal(t, "acme.ghe.com", got[common.ConfigHost])
}
