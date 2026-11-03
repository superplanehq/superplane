package github

import (
	"testing"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/ee/githubenterprise"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
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
