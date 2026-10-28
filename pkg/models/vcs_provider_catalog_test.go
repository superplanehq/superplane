package models

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
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
		{RepositoryID: 201, FullName: "acme/oldest"},
		{RepositoryID: 202, FullName: "acme/newer"},
		{RepositoryID: 203, FullName: "acme/interactive"},
		{RepositoryID: 204, FullName: "acme/future"},
	}))
	for repositoryID, job := range map[int64]struct {
		runAt    time.Time
		priority VCSProviderRepositorySyncPriority
	}{
		201: {now.Add(-time.Minute), VCSProviderRepositorySyncPriorityBackground},
		202: {now.Add(-time.Second), VCSProviderRepositorySyncPriorityBackground},
		203: {now, VCSProviderRepositorySyncPriorityInteractive},
		204: {now.Add(time.Minute), VCSProviderRepositorySyncPriorityInteractive},
	} {
		require.NoError(t, EnqueueVCSProviderRepositorySync(db, ProviderGitHub, repositoryID, job.runAt, job.priority))
	}

	claimed, err := ClaimVCSProviderRepositorySyncs(db, ProviderGitHub, now, now.Add(-time.Minute), 2)
	require.NoError(t, err)
	require.Len(t, claimed, 2)
	assert.ElementsMatch(t, []int64{203, 201}, []int64{claimed[0].RepositoryID, claimed[1].RepositoryID})
	for _, job := range claimed {
		require.NotNil(t, job.LockedAt)
		assert.Equal(t, 1, job.Attempts)
	}

	rest, err := ClaimVCSProviderRepositorySyncs(db, ProviderGitHub, now, now.Add(-time.Minute), 8)
	require.NoError(t, err)
	require.Len(t, rest, 1, "locked and future jobs must not be claimed")
	assert.Equal(t, int64(202), rest[0].RepositoryID)

	reclaimAt := now.Add(2 * time.Minute)
	reclaimed, err := ClaimVCSProviderRepositorySyncs(db, ProviderGitHub, reclaimAt, reclaimAt.Add(-time.Minute), 8)
	require.NoError(t, err)
	reclaimedAttempts := map[int64]int{}
	for _, job := range reclaimed {
		reclaimedAttempts[job.RepositoryID] = job.Attempts
	}
	assert.Equal(t, map[int64]int{201: 2, 202: 2, 203: 2, 204: 1}, reclaimedAttempts,
		"jobs whose lock expired must be claimed again")

	for _, job := range reclaimed {
		require.NoError(t, CompleteVCSProviderRepositorySync(db, ProviderGitHub, job.RepositoryID, *job.LockedAt))
	}
	var remaining int64
	require.NoError(t, db.Model(&VCSProviderRepositorySyncJob{}).Count(&remaining).Error)
	assert.Zero(t, remaining)
}

func TestVCSProviderRepositorySyncConcurrentClaimsSkipLockedJobs(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{Provider: ProviderGitHub, InstallationID: 101}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
		{RepositoryID: 202, FullName: "acme/web"},
		{RepositoryID: 203, FullName: "acme/docs"},
	}))
	for _, repositoryID := range []int64{201, 202, 203} {
		require.NoError(t, EnqueueVCSProviderRepositorySync(
			db, ProviderGitHub, repositoryID, now.Add(-time.Second), VCSProviderRepositorySyncPriorityBackground,
		))
	}

	held := db.Begin()
	t.Cleanup(func() { held.Rollback() })
	first, err := ClaimVCSProviderRepositorySyncs(held, ProviderGitHub, now, now.Add(-time.Minute), 2)
	require.NoError(t, err)
	require.Len(t, first, 2)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	second, err := ClaimVCSProviderRepositorySyncs(db.WithContext(ctx), ProviderGitHub, now, now.Add(-time.Minute), 8)
	require.NoError(t, err, "a second replica must not wait for rows that another replica holds")
	require.Len(t, second, 1)
	assert.NotContains(t, []int64{first[0].RepositoryID, first[1].RepositoryID}, second[0].RepositoryID)
}

func claimVCSProviderRepositorySync(t *testing.T, db *gorm.DB, now, claimableBefore time.Time) *VCSProviderRepositorySyncJob {
	t.Helper()
	jobs, err := ClaimVCSProviderRepositorySyncs(db, ProviderGitHub, now, claimableBefore, 1)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	return &jobs[0]
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

	job := claimVCSProviderRepositorySync(t, db, now, now.Add(-time.Minute))
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

	job := claimVCSProviderRepositorySync(t, db, now.Add(time.Second), now.Add(-time.Minute))
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

func TestVCSProviderRepositorySyncReenqueueDuringRunKeepsUrgency(t *testing.T) {
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
		VCSProviderRepositorySyncPriorityInteractive,
	))
	claimVCSProviderRepositorySync(t, db, now, now.Add(-time.Minute))

	require.NoError(t, EnqueueVCSProviderRepositorySync(
		db,
		ProviderGitHub,
		201,
		now,
		VCSProviderRepositorySyncPriorityBackground,
	))

	var running VCSProviderRepositorySyncJob
	require.NoError(t, db.First(&running, "provider = ? AND repository_id = ?", ProviderGitHub, 201).Error)
	assert.Equal(t, VCSProviderRepositorySyncPriorityInteractive, running.Priority)
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

func TestVCSProviderCatalogScheduleHasOneWinnerPerInterval(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	claimed, err := ClaimVCSProviderCatalogSchedule(db, ProviderGitHub, VCSProviderCatalogScheduleReconcile, now, 5*time.Minute)
	require.NoError(t, err)
	assert.True(t, claimed)

	claimed, err = ClaimVCSProviderCatalogSchedule(db, ProviderGitHub, VCSProviderCatalogScheduleReconcile, now.Add(time.Minute), 5*time.Minute)
	require.NoError(t, err)
	assert.False(t, claimed, "a second replica must not run the reconcile inside the interval")

	claimed, err = ClaimVCSProviderCatalogSchedule(db, ProviderGitHub, VCSProviderCatalogScheduleSweep, now.Add(time.Minute), 10*time.Minute)
	require.NoError(t, err)
	assert.True(t, claimed, "the sweep schedule is independent of the reconcile schedule")

	claimed, err = ClaimVCSProviderCatalogSchedule(db, ProviderGitHub, VCSProviderCatalogScheduleReconcile, now.Add(5*time.Minute), 5*time.Minute)
	require.NoError(t, err)
	assert.True(t, claimed)
}

func TestEnqueueUnsyncedVCSProviderRepositoriesSkipsSyncedRepositories(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{Provider: ProviderGitHub, InstallationID: 101}))
	repositories := []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/synced"},
		{RepositoryID: 202, FullName: "acme/new"},
		{RepositoryID: 203, FullName: "acme/queued"},
	}
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, repositories))
	require.NoError(t, ReplaceVCSProviderRepositoryCollaborators(db, ProviderGitHub, 201, []VCSProviderRepositoryCollaborator{
		{ProviderUserID: 42, ProviderLogin: "octocat"},
	}))
	require.NoError(t, EnqueueVCSProviderRepositorySync(db, ProviderGitHub, 203, now, VCSProviderRepositorySyncPriorityBackground))

	// A later reconcile rewrites the repository list. It must keep the sync time.
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, repositories))
	require.NoError(t, EnqueueUnsyncedVCSProviderRepositories(db, ProviderGitHub, 101, now, VCSProviderRepositorySyncPriorityInteractive))

	var jobs []VCSProviderRepositorySyncJob
	require.NoError(t, db.Order("repository_id").Find(&jobs).Error)
	require.Len(t, jobs, 2)
	assert.Equal(t, int64(202), jobs[0].RepositoryID)
	assert.Equal(t, VCSProviderRepositorySyncPriorityInteractive, jobs[0].Priority)
	assert.Equal(t, int64(203), jobs[1].RepositoryID)
	assert.Equal(t, VCSProviderRepositorySyncPriorityInteractive, jobs[1].Priority, "a requested reconcile raises the priority of a queued job")
}

func TestEnqueueStaleVCSProviderRepositorySyncsQueuesOldestFirst(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	now := time.Now()
	suspendedAt := now.Add(-time.Hour)

	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{Provider: ProviderGitHub, InstallationID: 101}))
	require.NoError(t, UpsertVCSProviderInstallation(db, &VCSProviderInstallation{
		Provider:       ProviderGitHub,
		InstallationID: 102,
		SuspendedAt:    &suspendedAt,
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 101, []VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/oldest"},
		{RepositoryID: 202, FullName: "acme/older"},
		{RepositoryID: 203, FullName: "acme/old"},
		{RepositoryID: 204, FullName: "acme/fresh"},
		{RepositoryID: 205, FullName: "acme/already-queued"},
	}))
	require.NoError(t, ReplaceVCSProviderRepositories(db, ProviderGitHub, 102, []VCSProviderRepository{
		{RepositoryID: 301, FullName: "suspended/api"},
	}))
	for repositoryID, age := range map[int64]time.Duration{
		201: 72 * time.Hour,
		202: 48 * time.Hour,
		203: 30 * time.Hour,
		204: time.Hour,
		205: 96 * time.Hour,
		301: 96 * time.Hour,
	} {
		require.NoError(t, db.Model(&VCSProviderRepository{}).
			Where("provider = ? AND repository_id = ?", ProviderGitHub, repositoryID).
			UpdateColumn("collaborators_synced_at", now.Add(-age)).Error)
	}
	require.NoError(t, EnqueueVCSProviderRepositorySync(db, ProviderGitHub, 205, now, VCSProviderRepositorySyncPriorityInteractive))

	queued, err := EnqueueStaleVCSProviderRepositorySyncs(db, ProviderGitHub, now.Add(-24*time.Hour), 2, now)
	require.NoError(t, err)
	assert.Equal(t, int64(2), queued)

	var jobs []VCSProviderRepositorySyncJob
	require.NoError(t, db.Order("repository_id").Find(&jobs).Error)
	require.Len(t, jobs, 3)
	assert.Equal(t, []int64{201, 202, 205}, []int64{jobs[0].RepositoryID, jobs[1].RepositoryID, jobs[2].RepositoryID})
	assert.Equal(t, VCSProviderRepositorySyncPriorityBackground, jobs[0].Priority)
	assert.Equal(t, VCSProviderRepositorySyncPriorityInteractive, jobs[2].Priority, "the sweep must not lower a queued job")
}

func int64Pointer(value int64) *int64 {
	return &value
}
