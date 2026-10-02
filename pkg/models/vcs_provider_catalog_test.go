package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestVCSProviderCatalogListsRepositoriesByCollaborator(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	installation := VCSProviderInstallation{
		Provider:            ProviderGitHub,
		InstallationID:      101,
		AccountID:           int64Pointer(9001),
		AccountLogin:        "acme",
		AccountType:         "Organization",
		RepositorySelection: "selected",
	}
	require.NoError(t, UpsertVCSProviderInstallation(db, &installation))

	repositories := []VCSProviderRepository{
		{RepositoryID: 201, InstallationID: 101, FullName: "acme/api", Private: true, DefaultBranch: "main"},
		{RepositoryID: 202, InstallationID: 101, FullName: "acme/web", DefaultBranch: "trunk"},
	}
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, repositories))
	require.NoError(t, ReplaceVCSProviderRepositoryCollaborators(db, ProviderGitHub, 201, []VCSProviderRepositoryCollaborator{
		{RepositoryID: 201, ProviderUserID: 42, ProviderLogin: "octocat"},
	}))
	require.NoError(t, ReplaceVCSProviderRepositoryCollaborators(db, ProviderGitHub, 202, []VCSProviderRepositoryCollaborator{
		{RepositoryID: 202, ProviderUserID: 7, ProviderLogin: "other"},
	}))

	accessible, err := ListAccessibleVCSProviderRepositories(db, ProviderGitHub, 42)
	require.NoError(t, err)
	require.Len(t, accessible, 1)
	assert.Equal(t, int64(201), accessible[0].RepositoryID)
	assert.Equal(t, "acme/api", accessible[0].FullName)
	assert.Equal(t, "acme", accessible[0].AccountLogin)
	assert.Equal(t, "main", accessible[0].DefaultBranch)
}

func TestReplaceVCSProviderRepositoryCollaboratorsRemovesLostAccess(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{Provider: ProviderGitHub, InstallationID: 101}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, InstallationID: 101, FullName: "acme/api"},
	}))
	require.NoError(t, ReplaceVCSProviderRepositoryCollaborators(db, ProviderGitHub, 201, []VCSProviderRepositoryCollaborator{
		{RepositoryID: 201, ProviderUserID: 42, ProviderLogin: "octocat"},
		{RepositoryID: 201, ProviderUserID: 7, ProviderLogin: "other"},
	}))

	require.NoError(t, ReplaceVCSProviderRepositoryCollaborators(db, ProviderGitHub, 201, []VCSProviderRepositoryCollaborator{
		{RepositoryID: 201, ProviderUserID: 42, ProviderLogin: "octocat"},
	}))

	accessible, err := ListAccessibleVCSProviderRepositories(db, ProviderGitHub, 7)
	require.NoError(t, err)
	assert.Empty(t, accessible)
}

func TestVCSProviderCatalogIsolatesProvidersWithMatchingRemoteIDs(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	for _, provider := range []string{ProviderGitHub, "gitlab"} {
		require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
			Provider:       provider,
			InstallationID: 101,
			AccountLogin:   provider + "-account",
		}))
		require.NoError(t, ReplaceVCSProviderRepositories(db, provider, 101, []VCSProviderRepository{
			{RepositoryID: 201, FullName: provider + "-account/api"},
		}))
		require.NoError(t, ReplaceVCSProviderRepositoryCollaborators(
			db,
			provider,
			201,
			[]VCSProviderRepositoryCollaborator{{ProviderUserID: 42, ProviderLogin: "developer"}},
		))
	}

	githubRepositories, err := ListAccessibleVCSProviderRepositories(db, ProviderGitHub, 42)
	require.NoError(t, err)
	require.Len(t, githubRepositories, 1)
	assert.Equal(t, "github-account/api", githubRepositories[0].FullName)

	gitlabRepositories, err := ListAccessibleVCSProviderRepositories(db, "gitlab", 42)
	require.NoError(t, err)
	require.Len(t, gitlabRepositories, 1)
	assert.Equal(t, "gitlab-account/api", gitlabRepositories[0].FullName)
}

func TestVCSProviderBindingReferencesGlobalInstallation(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
		Provider:       ProviderGitHub,
		InstallationID: 101,
		AccountLogin:   "acme",
	}))

	firstOrganization, err := CreateOrganization("First "+uuid.NewString(), "")
	require.NoError(t, err)
	secondOrganization, err := CreateOrganization("Second "+uuid.NewString(), "")
	require.NoError(t, err)

	first, err := FindOrCreateVCSProviderBinding(db, firstOrganization.ID, ProviderGitHub, 101, "acme")
	require.NoError(t, err)
	same, err := FindOrCreateVCSProviderBinding(db, firstOrganization.ID, ProviderGitHub, 101, "acme")
	require.NoError(t, err)
	second, err := FindOrCreateVCSProviderBinding(db, secondOrganization.ID, ProviderGitHub, 101, "acme")
	require.NoError(t, err)

	assert.Equal(t, first.ID, same.ID)
	assert.NotEqual(t, first.ID, second.ID)
	assert.Equal(t, map[string]any{"hostedApp": true}, first.Metadata.Data())

	bindings, err := ListVCSProviderIntegrationBindings(db, ProviderGitHub, 101)
	require.NoError(t, err)
	assert.Len(t, bindings, 2)

	require.NoError(t, db.Delete(&VCSProviderIntegrationBinding{}, "integration_id = ?", first.ID).Error)
	_, err = FindVCSProviderInstallation(db, ProviderGitHub, 101)
	require.NoError(t, err)
}

func TestVCSProviderBindingListsOnlyGrantedRepositories(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
		Provider:       ProviderGitHub,
		InstallationID: 101,
		AccountLogin:   "acme",
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
		{RepositoryID: 202, FullName: "acme/private"},
	}))
	organization, err := CreateOrganization("Binding "+uuid.NewString(), "")
	require.NoError(t, err)
	integration, err := FindOrCreateVCSProviderBinding(db, organization.ID, ProviderGitHub, 101, "acme")
	require.NoError(t, err)

	require.NoError(t, GrantVCSProviderBindingRepository(db, integration.ID, ProviderGitHub, 201))
	require.NoError(t, GrantVCSProviderBindingRepository(db, integration.ID, ProviderGitHub, 201))

	repositories, err := ListVCSProviderBindingRepositories(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, repositories, 1)
	assert.Equal(t, int64(201), repositories[0].RepositoryID)
}

func TestSuspendedVCSProviderInstallationHidesEveryRepository(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()

	installation := VCSProviderInstallation{Provider: ProviderGitHub, InstallationID: 101, AccountLogin: "acme"}
	require.NoError(t, UpsertVCSProviderInstallation(db, &installation))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
	}))
	require.NoError(t, ReplaceVCSProviderRepositoryCollaborators(db, ProviderGitHub, 201, []VCSProviderRepositoryCollaborator{
		{ProviderUserID: 42, ProviderLogin: "octocat"},
	}))

	suspendedAt := time.Now()
	installation.SuspendedAt = &suspendedAt
	require.NoError(t, UpsertVCSProviderInstallation(db, &installation))
	accessible, err := ListAccessibleVCSProviderRepositories(db, ProviderGitHub, 42)
	require.NoError(t, err)
	assert.Empty(t, accessible)
}

func TestVCSProviderRepositorySyncJobClaim(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{Provider: ProviderGitHub, InstallationID: 101}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, InstallationID: 101, FullName: "acme/api"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		now.Add(-time.Second),
		VCSProviderRepositorySyncPriorityBackground,
	))

	job, err := ClaimVCSProviderRepositorySync(db, ProviderGitHub, now, now.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, int64(201), job.RepositoryID)

	none, err := ClaimVCSProviderRepositorySync(db, ProviderGitHub, now, now.Add(-time.Minute))
	require.NoError(t, err)
	assert.Nil(t, none)

	require.NoError(t, CompleteVCSProviderRepositorySync(db, ProviderGitHub, 201, *job.LockedAt))
}

func TestVCSProviderRepositorySyncKeepsRefreshEnqueuedDuringClaim(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{Provider: ProviderGitHub, InstallationID: 101}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		now.Add(-time.Second),
		VCSProviderRepositorySyncPriorityBackground,
	))

	job, err := ClaimVCSProviderRepositorySync(db, ProviderGitHub, now, now.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	require.NotNil(t, job.LockedAt)

	refreshAt := now.Add(time.Second)
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		refreshAt,
		VCSProviderRepositorySyncPriorityInteractive,
	))
	require.NoError(t, CompleteVCSProviderRepositorySync(db, ProviderGitHub, 201, *job.LockedAt))

	var queued VCSProviderRepositorySyncJob
	require.NoError(t, db.Where("provider = ? AND repository_id = ?", ProviderGitHub, 201).First(&queued).Error)
	assert.Nil(t, queued.LockedAt)
	assert.WithinDuration(t, refreshAt, queued.RunAt, time.Millisecond)
}

func TestVCSProviderRepositorySyncClaimsInteractiveJobsFirst(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
		Provider:       ProviderGitHub,
		InstallationID: 101,
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/background"},
		{RepositoryID: 202, FullName: "acme/interactive"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		now.Add(-time.Minute),
		VCSProviderRepositorySyncPriorityBackground,
	))
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		202,
		now,
		VCSProviderRepositorySyncPriorityInteractive,
	))

	job, err := ClaimVCSProviderRepositorySync(db, ProviderGitHub, now.Add(time.Second), now.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, int64(202), job.RepositoryID)
	assert.Equal(t, VCSProviderRepositorySyncPriorityInteractive, job.Priority)
}

func TestVCSProviderRepositorySyncReenqueueKeepsUrgency(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
		Provider:       ProviderGitHub,
		InstallationID: 101,
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		now,
		VCSProviderRepositorySyncPriorityInteractive,
	))
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		now.Add(time.Minute),
		VCSProviderRepositorySyncPriorityBackground,
	))

	var job VCSProviderRepositorySyncJob
	require.NoError(t, db.First(&job, "provider = ? AND repository_id = ?", ProviderGitHub, 201).Error)
	assert.Equal(t, VCSProviderRepositorySyncPriorityInteractive, job.Priority)
	assert.WithinDuration(t, now, job.RunAt, time.Millisecond)
}

func TestVCSProviderRepositorySyncDelayedRefreshMovesUnlockedJobLater(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
		Provider:       ProviderGitHub,
		InstallationID: 101,
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		now,
		VCSProviderRepositorySyncPriorityInteractive,
	))
	delayedUntil := now.Add(10 * time.Second)
	require.NoError(t, DelayVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		delayedUntil,
		VCSProviderRepositorySyncPriorityInteractive,
	))

	var job VCSProviderRepositorySyncJob
	require.NoError(t, db.First(&job, "provider = ? AND repository_id = ?", ProviderGitHub, 201).Error)
	assert.WithinDuration(t, delayedUntil, job.RunAt, time.Millisecond)
}

func TestVCSProviderCatalogSynchronizingIgnoresUnrelatedRepositoryJobs(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()
	organizationID := uuid.New()

	for _, installation := range []VCSProviderInstallation{
		{Provider: ProviderGitHub, InstallationID: 101, AccountLogin: "acme", AccountType: "Organization"},
		{Provider: ProviderGitHub, InstallationID: 102, AccountLogin: "other", AccountType: "Organization"},
	} {
		require.NoError(t, UpsertVCSProviderInstallation(db, &installation))
	}
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 102, []VCSProviderRepository{
		{RepositoryID: 202, FullName: "other/api"},
	}))
	require.NoError(t, ReplaceVCSProviderRepositoryCollaborators(db, ProviderGitHub, 201, []VCSProviderRepositoryCollaborator{
		{ProviderUserID: 42, ProviderLogin: "octocat"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		202,
		now,
		VCSProviderRepositorySyncPriorityBackground,
	))

	synchronizing, err := VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, organizationID)
	require.NoError(t, err)
	assert.False(t, synchronizing)

	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		now,
		VCSProviderRepositorySyncPriorityInteractive,
	))
	synchronizing, err = VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, organizationID)
	require.NoError(t, err)
	assert.True(t, synchronizing)
}

func TestVCSProviderCatalogSynchronizingIncludesRequestedOrganizationJobs(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()
	firstOrganizationID := uuid.New()
	secondOrganizationID := uuid.New()
	require.NoError(t, EnqueueVCSProviderInstallationReconciliation(
		db,
		ProviderGitHub,
		101,
		firstOrganizationID,
		now,
	))
	require.NoError(t, EnqueueVCSProviderInstallationReconciliation(
		db,
		ProviderGitHub,
		101,
		secondOrganizationID,
		now,
	))

	synchronizing, err := VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, firstOrganizationID)
	require.NoError(t, err)
	assert.True(t, synchronizing)

	synchronizing, err = VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, secondOrganizationID)
	require.NoError(t, err)
	assert.True(t, synchronizing)

	synchronizing, err = VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, uuid.New())
	require.NoError(t, err)
	assert.False(t, synchronizing)
}

// After an organization installs the App on a new GitHub organization, the
// install job finishes in about a second. The collaborator sync of its
// repositories runs much longer, and the user is not a collaborator yet.
func TestVCSProviderCatalogSynchronizingFollowsRequestedInstallationAfterItsJob(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()
	organizationID := uuid.New()
	require.NoError(t, EnqueueVCSProviderInstallationReconciliation(db, ProviderGitHub, 101, organizationID, now.Add(-time.Second)))
	job, err := ClaimVCSProviderInstallationReconciliation(db, ProviderGitHub, now, now.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
		Provider:       ProviderGitHub,
		InstallationID: 101,
		AccountLogin:   "acme",
		AccountType:    "Organization",
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(db, ProviderGitHub, 201, now, VCSProviderRepositorySyncPriorityInteractive))
	require.NoError(t, CompleteVCSProviderInstallationReconciliation(db, ProviderGitHub, 101, *job.LockedAt))

	synchronizing, err := VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, organizationID)
	require.NoError(t, err)
	assert.True(t, synchronizing)

	synchronizing, err = VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, uuid.New())
	require.NoError(t, err)
	assert.False(t, synchronizing)

	require.NoError(t, db.Model(&VCSProviderInstallationReconcileRequester{}).
		Where("installation_id = ?", 101).
		Update("created_at", now.Add(-24*time.Hour)).Error)
	synchronizing, err = VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, organizationID)
	require.NoError(t, err)
	assert.True(t, synchronizing)

	require.NoError(t, db.Where("repository_id = ?", 201).Delete(&VCSProviderRepositorySyncJob{}).Error)
	require.NoError(t, EnqueueVCSProviderRepositorySync(db, ProviderGitHub, 201, now, VCSProviderRepositorySyncPriorityBackground))
	synchronizing, err = VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, organizationID)
	require.NoError(t, err)
	assert.False(t, synchronizing)
}

func TestDeleteFinishedVCSProviderInstallationRequesters(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()
	finishedOrganizationID := uuid.New()
	syncingOrganizationID := uuid.New()
	newOrganizationID := uuid.New()
	for installationID, organizationID := range map[int64]uuid.UUID{
		101: finishedOrganizationID,
		102: syncingOrganizationID,
		103: newOrganizationID,
	} {
		require.NoError(t, EnqueueVCSProviderInstallationReconciliation(db, ProviderGitHub, installationID, organizationID, now))
	}
	require.NoError(t, db.Where("provider = ?", ProviderGitHub).Delete(&VCSProviderInstallationReconcileJob{}).Error)
	for installationID, login := range map[int64]string{101: "done", 102: "acme"} {
		require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
			Provider:       ProviderGitHub,
			InstallationID: installationID,
			AccountLogin:   login,
			AccountType:    "Organization",
		}))
	}
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "done/api"},
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 102, []VCSProviderRepository{
		{RepositoryID: 202, FullName: "acme/api"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(db, ProviderGitHub, 201, now, VCSProviderRepositorySyncPriorityBackground))
	require.NoError(t, EnqueueVCSProviderRepositorySync(db, ProviderGitHub, 202, now, VCSProviderRepositorySyncPriorityInteractive))
	require.NoError(t, db.Model(&VCSProviderInstallationReconcileRequester{}).
		Where("installation_id IN ?", []int64{101, 102}).
		Update("created_at", now.Add(-time.Hour)).Error)

	require.NoError(t, DeleteFinishedVCSProviderInstallationRequesters(db, ProviderGitHub, now.Add(-time.Minute)))

	var requesters []VCSProviderInstallationReconcileRequester
	require.NoError(t, db.Order("installation_id").Find(&requesters).Error)
	require.Len(t, requesters, 2)
	assert.Equal(t, syncingOrganizationID, requesters[0].OrganizationID)
	assert.Equal(t, newOrganizationID, requesters[1].OrganizationID)
}

func TestVCSProviderCatalogSynchronizingIncludesRelevantInstallationJobs(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
		Provider:       ProviderGitHub,
		InstallationID: 101,
		AccountID:      int64Pointer(42),
		AccountLogin:   "octocat",
		AccountType:    "User",
	}))
	require.NoError(t, EnqueueVCSProviderInstallationReconciliation(
		db,
		ProviderGitHub,
		101,
		uuid.Nil,
		now,
	))

	synchronizing, err := VCSProviderCatalogSynchronizing(db, ProviderGitHub, 42, uuid.New())
	require.NoError(t, err)
	assert.True(t, synchronizing)

	synchronizing, err = VCSProviderCatalogSynchronizing(db, ProviderGitHub, 7, uuid.New())
	require.NoError(t, err)
	assert.False(t, synchronizing)
}

func TestVCSProviderInstallationReconciliationJobClaim(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, EnqueueVCSProviderInstallationReconciliation(
		db,
		ProviderGitHub,
		101,
		uuid.New(),
		now.Add(-time.Second),
	))
	job, err := ClaimVCSProviderInstallationReconciliation(db, ProviderGitHub, now, now.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, int64(101), job.InstallationID)

	none, err := ClaimVCSProviderInstallationReconciliation(db, ProviderGitHub, now, now.Add(-time.Minute))
	require.NoError(t, err)
	assert.Nil(t, none)
	require.NoError(t, CompleteVCSProviderInstallationReconciliation(
		db,
		ProviderGitHub,
		job.InstallationID,
		*job.LockedAt,
	))
	var requesterCount int64
	require.NoError(t, db.Model(&VCSProviderInstallationReconcileRequester{}).Count(&requesterCount).Error)
	assert.Equal(t, int64(1), requesterCount)
}

func TestVCSProviderReconciliationJobClaim(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, EnqueueVCSProviderReconciliation(db, ProviderGitHub, now.Add(-time.Second)))
	job, err := ClaimVCSProviderReconciliation(db, ProviderGitHub, now, now.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, ProviderGitHub, job.Provider)

	none, err := ClaimVCSProviderReconciliation(db, ProviderGitHub, now, now.Add(-time.Minute))
	require.NoError(t, err)
	assert.Nil(t, none)
	require.NoError(t, CompleteVCSProviderReconciliation(db, ProviderGitHub, *job.LockedAt))
}

func int64Pointer(value int64) *int64 {
	return &value
}
