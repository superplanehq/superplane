package sentry

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeHostedInstallGrants replaces the database store, so the hosted setup
// tests need no database. The stored behavior of a grant has its own test in
// pkg/models.
type fakeHostedInstallGrants struct {
	mu     sync.Mutex
	grants map[string]hostedSentryInstall
}

func useFakeHostedInstallGrants(t *testing.T) *fakeHostedInstallGrants {
	t.Helper()

	fake := &fakeHostedInstallGrants{grants: map[string]hostedSentryInstall{}}
	previous := hostedInstallGrants
	hostedInstallGrants = fake
	t.Cleanup(func() {
		hostedInstallGrants = previous
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
	f.grants[install.InstallationUUID] = install
	return nil
}

func (f *fakeHostedInstallGrants) Take(installationUUID, code string) (*hostedSentryInstall, error) {
	installationUUID = strings.TrimSpace(installationUUID)
	code = strings.TrimSpace(code)
	if installationUUID == "" || code == "" {
		return nil, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	install, ok := f.grants[installationUUID]
	if !ok || install.Code != code || !install.ExpiresAt.After(time.Now()) {
		return nil, nil
	}
	copied := install
	return &copied, nil
}

func (f *fakeHostedInstallGrants) Forget(installationUUID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.grants, strings.TrimSpace(installationUUID))
	return nil
}
