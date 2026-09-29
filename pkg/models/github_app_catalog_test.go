package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestGitHubAppCatalogListsRepositoriesByCollaborator(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	installation := GitHubAppInstallation{
		InstallationID:      101,
		AccountID:           int64Pointer(9001),
		AccountLogin:        "acme",
		AccountType:         "Organization",
		RepositorySelection: "selected",
	}
	require.NoError(t, UpsertGitHubAppInstallation(db, &installation))

	repositories := []GitHubAppRepository{
		{RepositoryID: 201, InstallationID: 101, FullName: "acme/api", Private: true, DefaultBranch: "main"},
		{RepositoryID: 202, InstallationID: 101, FullName: "acme/web", DefaultBranch: "trunk"},
	}
	require.NoError(t, ReplaceGitHubAppRepositories(db, 101, repositories))
	require.NoError(t, ReplaceGitHubAppRepositoryCollaborators(db, 201, []GitHubAppRepositoryCollaborator{
		{RepositoryID: 201, GitHubUserID: 42, GitHubLogin: "octocat"},
	}))
	require.NoError(t, ReplaceGitHubAppRepositoryCollaborators(db, 202, []GitHubAppRepositoryCollaborator{
		{RepositoryID: 202, GitHubUserID: 7, GitHubLogin: "other"},
	}))

	accessible, err := ListAccessibleGitHubAppRepositories(db, 42)
	require.NoError(t, err)
	require.Len(t, accessible, 1)
	assert.Equal(t, int64(201), accessible[0].RepositoryID)
	assert.Equal(t, "acme/api", accessible[0].FullName)
	assert.Equal(t, "acme", accessible[0].AccountLogin)
	assert.Equal(t, "main", accessible[0].DefaultBranch)
}

func TestReplaceGitHubAppRepositoryCollaboratorsRemovesLostAccess(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	require.NoError(t, UpsertGitHubAppInstallation(db, &GitHubAppInstallation{InstallationID: 101}))
	require.NoError(t, ReplaceGitHubAppRepositories(db, 101, []GitHubAppRepository{
		{RepositoryID: 201, InstallationID: 101, FullName: "acme/api"},
	}))
	require.NoError(t, ReplaceGitHubAppRepositoryCollaborators(db, 201, []GitHubAppRepositoryCollaborator{
		{RepositoryID: 201, GitHubUserID: 42, GitHubLogin: "octocat"},
		{RepositoryID: 201, GitHubUserID: 7, GitHubLogin: "other"},
	}))

	require.NoError(t, ReplaceGitHubAppRepositoryCollaborators(db, 201, []GitHubAppRepositoryCollaborator{
		{RepositoryID: 201, GitHubUserID: 42, GitHubLogin: "octocat"},
	}))

	accessible, err := ListAccessibleGitHubAppRepositories(db, 7)
	require.NoError(t, err)
	assert.Empty(t, accessible)
}

func TestHostedGitHubBindingReferencesGlobalInstallation(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	require.NoError(t, UpsertGitHubAppInstallation(db, &GitHubAppInstallation{
		InstallationID: 101,
		AccountLogin:   "acme",
	}))

	firstOrganization, err := CreateOrganization("First "+uuid.NewString(), "")
	require.NoError(t, err)
	secondOrganization, err := CreateOrganization("Second "+uuid.NewString(), "")
	require.NoError(t, err)

	first, err := FindOrCreateHostedGitHubBinding(db, firstOrganization.ID, 101, "acme")
	require.NoError(t, err)
	same, err := FindOrCreateHostedGitHubBinding(db, firstOrganization.ID, 101, "acme")
	require.NoError(t, err)
	second, err := FindOrCreateHostedGitHubBinding(db, secondOrganization.ID, 101, "acme")
	require.NoError(t, err)

	assert.Equal(t, first.ID, same.ID)
	assert.NotEqual(t, first.ID, second.ID)
	assert.Equal(t, map[string]any{"hostedApp": true}, first.Metadata.Data())

	bindings, err := ListGitHubAppIntegrationBindings(db, 101)
	require.NoError(t, err)
	assert.Len(t, bindings, 2)

	require.NoError(t, db.Delete(&GitHubAppIntegrationBinding{}, "integration_id = ?", first.ID).Error)
	_, err = FindGitHubAppInstallation(db, 101)
	require.NoError(t, err)
}

func TestSuspendedGitHubAppInstallationHidesEveryRepository(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	installation := GitHubAppInstallation{InstallationID: 101, AccountLogin: "acme"}
	require.NoError(t, UpsertGitHubAppInstallation(db, &installation))
	require.NoError(t, ReplaceGitHubAppRepositories(db, 101, []GitHubAppRepository{
		{RepositoryID: 201, FullName: "acme/api"},
	}))
	require.NoError(t, ReplaceGitHubAppRepositoryCollaborators(db, 201, []GitHubAppRepositoryCollaborator{
		{GitHubUserID: 42, GitHubLogin: "octocat"},
	}))

	suspendedAt := time.Now()
	installation.SuspendedAt = &suspendedAt
	require.NoError(t, UpsertGitHubAppInstallation(db, &installation))
	accessible, err := ListAccessibleGitHubAppRepositories(db, 42)
	require.NoError(t, err)
	assert.Empty(t, accessible)
}

func TestGitHubAppRepositorySyncJobClaim(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertGitHubAppInstallation(db, &GitHubAppInstallation{InstallationID: 101}))
	require.NoError(t, ReplaceGitHubAppRepositories(db, 101, []GitHubAppRepository{
		{RepositoryID: 201, InstallationID: 101, FullName: "acme/api"},
	}))
	require.NoError(t, EnqueueGitHubAppRepositorySync(db, 201, now.Add(-time.Second)))

	job, err := ClaimGitHubAppRepositorySync(db, now, now.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, int64(201), job.RepositoryID)

	none, err := ClaimGitHubAppRepositorySync(db, now, now.Add(-time.Minute))
	require.NoError(t, err)
	assert.Nil(t, none)

	require.NoError(t, CompleteGitHubAppRepositorySync(db, 201))
}

func TestGitHubAppReconciliationJobClaim(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, EnqueueGitHubAppReconciliation(db, now.Add(-time.Second)))
	job, err := ClaimGitHubAppReconciliation(db, now, now.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, int16(1), job.ID)

	none, err := ClaimGitHubAppReconciliation(db, now, now.Add(-time.Minute))
	require.NoError(t, err)
	assert.Nil(t, none)
	require.NoError(t, CompleteGitHubAppReconciliation(db))
}

func int64Pointer(value int64) *int64 {
	return &value
}
