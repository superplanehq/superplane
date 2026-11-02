package sentry

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
)

// fakeHostedInstallGrants replaces the database store, so the hosted setup
// tests need no database. The stored behavior of a grant has its own test in
// pkg/models.
type fakeHostedInstallGrants struct {
	mu        sync.Mutex
	grants    map[string]hostedSentryInstall
	claimedBy map[string]string
	deleted   map[string]bool
}

func useFakeHostedInstallGrants(t *testing.T) *fakeHostedInstallGrants {
	t.Helper()

	fake := &fakeHostedInstallGrants{
		grants:    map[string]hostedSentryInstall{},
		claimedBy: map[string]string{},
		deleted:   map[string]bool{},
	}
	previous := hostedInstallGrants
	previousWait := hostedInstallGrantWait
	hostedInstallGrants = fake
	hostedInstallGrantWait.Timeout = 20 * time.Millisecond
	hostedInstallGrantWait.Interval = time.Millisecond
	t.Cleanup(func() {
		hostedInstallGrants = previous
		hostedInstallGrantWait = previousWait
	})
	return fake
}

func (f *fakeHostedInstallGrants) Remember(install hostedSentryInstall) error {
	install.InstallationUUID = strings.TrimSpace(install.InstallationUUID)
	install.Code = strings.TrimSpace(install.Code)
	if install.InstallationUUID == "" || install.Code == "" {
		return nil
	}
	if install.ExpiresAt.IsZero() {
		install.ExpiresAt = time.Now().Add(hostedInstallGrantTTL)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleted[install.InstallationUUID] {
		return models.ErrSentryAppInstallationDeleted
	}
	previous, exists := f.grants[install.InstallationUUID]
	sameGrant := exists && previous.Code == install.Code
	if sameGrant && install.AccessToken == "" {
		install.AccessToken = previous.AccessToken
		install.RefreshToken = previous.RefreshToken
		install.TokenExpiresAt = previous.TokenExpiresAt
	}
	f.grants[install.InstallationUUID] = install
	if !sameGrant {
		delete(f.claimedBy, install.InstallationUUID)
	}
	return nil
}

func (f *fakeHostedInstallGrants) Find(installationUUID, code string) (*hostedSentryInstall, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	install, ok := f.grants[strings.TrimSpace(installationUUID)]
	if !ok || install.Code != strings.TrimSpace(code) || !install.ExpiresAt.After(time.Now()) {
		return nil, nil
	}
	copied := install
	return &copied, nil
}

func (f *fakeHostedInstallGrants) Uninstall(installationUUID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	installationUUID = strings.TrimSpace(installationUUID)
	f.deleted[installationUUID] = true
	delete(f.grants, installationUUID)
	delete(f.claimedBy, installationUUID)
	return nil
}

func (f *fakeHostedInstallGrants) Deleted(installationUUID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deleted[strings.TrimSpace(installationUUID)], nil
}

func (f *fakeHostedInstallGrants) Take(installationUUID, code, integrationID string) (*hostedSentryInstall, error) {
	installationUUID = strings.TrimSpace(installationUUID)
	code = strings.TrimSpace(code)
	integrationID = strings.TrimSpace(integrationID)
	if installationUUID == "" || code == "" || integrationID == "" {
		return nil, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	install, ok := f.grants[installationUUID]
	if !ok || install.Code != code || !install.ExpiresAt.After(time.Now()) {
		return nil, nil
	}
	if claimed, exists := f.claimedBy[installationUUID]; exists && claimed != integrationID {
		return nil, nil
	}
	f.claimedBy[installationUUID] = integrationID
	copied := install
	return &copied, nil
}

func (f *fakeHostedInstallGrants) Forget(installationUUID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	installationUUID = strings.TrimSpace(installationUUID)
	delete(f.grants, installationUUID)
	delete(f.claimedBy, installationUUID)
	return nil
}
