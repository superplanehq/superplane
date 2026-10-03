package me

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

type recordingInstallationVerifier struct {
	mu            sync.Mutex
	verified      []int64
	errors        []error
	cancelRequest context.CancelFunc
}

func (v *recordingInstallationVerifier) VerifyInstallation(ctx context.Context, installationID int64) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.cancelRequest != nil {
		v.cancelRequest()
	}
	v.verified = append(v.verified, installationID)
	v.errors = append(v.errors, ctx.Err())
	return nil
}

func TestVerifyVCSProviderInstallationsFinishesChecksAfterThePageCancelsTheRequest(t *testing.T) {
	r := support.Setup(t)
	setVCSProviderGitHubAppEnvironment(t)
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.Conn(),
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "101", "octocat", "", ""),
	))
	saveInstallationWithRepositories(t, 301, []int64{401}, 101)
	ctx, cancel := context.WithCancel(notificationSettingsContext(r.User.String(), r.Organization.ID.String()))
	defer cancel()
	verifier := &recordingInstallationVerifier{cancelRequest: cancel}

	_, err := VerifyVCSProviderInstallations(ctx, models.ProviderGitHub, NewVCSProviderInstallationChecks(time.Minute), verifier)
	require.NoError(t, err)

	assert.Equal(t, []int64{301}, verifier.verified)
	assert.Equal(t, []error{nil}, verifier.errors)
}

func TestVerifyVCSProviderInstallationsChecksVisibleInstallationsOncePerInterval(t *testing.T) {
	r := support.Setup(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())
	setVCSProviderGitHubAppEnvironment(t)
	db := database.Conn()
	require.NoError(t, models.SaveAccountLinkedAccount(
		db,
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "101", "octocat", "", ""),
	))
	saveInstallationWithRepositories(t, 301, []int64{401, 402}, 101)
	saveInstallationWithRepositories(t, 302, []int64{403}, 999)

	checks := NewVCSProviderInstallationChecks(time.Minute)
	verifier := &recordingInstallationVerifier{}

	_, err := VerifyVCSProviderInstallations(ctx, models.ProviderGitHub, checks, verifier)
	require.NoError(t, err)
	assert.Equal(t, []int64{301}, verifier.verified)

	_, err = VerifyVCSProviderInstallations(ctx, models.ProviderGitHub, checks, verifier)
	require.NoError(t, err)
	assert.Equal(t, []int64{301}, verifier.verified)
}

func saveInstallationWithRepositories(t *testing.T, installationID int64, repositoryIDs []int64, collaboratorID int64) {
	t.Helper()
	db := database.Conn()
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: installationID,
		AccountLogin:   "acme",
		AccountType:    "Organization",
	}))
	repositories := make([]models.VCSProviderRepository, 0, len(repositoryIDs))
	for _, repositoryID := range repositoryIDs {
		repositories = append(repositories, models.VCSProviderRepository{
			RepositoryID: repositoryID,
			FullName:     fmt.Sprintf("acme/repository-%d", repositoryID),
		})
	}
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, installationID, repositories))
	for _, repositoryID := range repositoryIDs {
		require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(
			db,
			models.ProviderGitHub,
			repositoryID,
			[]models.VCSProviderRepositoryCollaborator{{ProviderUserID: collaboratorID, ProviderLogin: "user"}},
		))
	}
}

func TestVCSProviderInstallationChecksAllowAnotherCheckAfterTheInterval(t *testing.T) {
	checks := NewVCSProviderInstallationChecks(10 * time.Second)
	now := time.Now()

	assert.Equal(t, []int64{301}, checks.due([]int64{301}, now))
	assert.Empty(t, checks.due([]int64{301}, now.Add(9*time.Second)))
	assert.Equal(t, []int64{301}, checks.due([]int64{301}, now.Add(10*time.Second)))
}
