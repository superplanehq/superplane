package licensing_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

type memorySource struct {
	raw []byte
	err error
}

func (m *memorySource) Kind() licensing.SourceKind {
	return licensing.SourceDatabase
}

func (m *memorySource) Read(context.Context) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}

	if m.raw == nil {
		return nil, licensing.ErrNotInstalled
	}

	return m.raw, nil
}

func (m *memorySource) Write(_ context.Context, raw []byte, _ uuid.UUID) error {
	m.raw = raw
	return nil
}

func (m *memorySource) Clear(context.Context) error {
	m.raw = nil
	return nil
}

func newService(t *testing.T, issuer *licensingtest.Issuer, source licensing.Source) *licensing.Service {
	t.Helper()
	return licensing.NewService(licensing.NewVerifier(licensingtest.KeySet(issuer)), source)
}

func writeLicenseFile(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "license.jws")
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return path
}

func TestServiceDefaultsToCommunity(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	service := newService(t, issuer, &memorySource{})
	require.NoError(t, service.Refresh(context.Background()))

	status := service.Status()
	assert.Equal(t, licensing.EditionCommunity, status.Edition)
	assert.Equal(t, licensing.StateNone, status.State)
	assert.Equal(t, licensing.SourceNone, status.Source)
	assert.True(t, status.Writable)
	assert.False(t, service.IsEntitled(licensing.FeatureGroups))
}

func TestServiceGrantsOnlyLicensedFeatures(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	source := &memorySource{raw: issuer.License(licensing.FeatureGroups)}
	service := newService(t, issuer, source)
	require.NoError(t, service.Refresh(context.Background()))

	status := service.Status()
	assert.Equal(t, licensing.EditionEnterprise, status.Edition)
	assert.Equal(t, licensing.StateActive, status.State)
	assert.True(t, service.IsEntitled(licensing.FeatureGroups))
	assert.False(t, service.IsEntitled(licensing.FeatureCustomRoles))
}

func TestServiceTreatsExpiredLicenseAsCommunity(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	claims := licensingtest.Claims(licensing.FeatureGroups)
	claims["iat"] = time.Now().Add(-48 * time.Hour).Unix()
	claims["nbf"] = time.Now().Add(-48 * time.Hour).Unix()
	claims["exp"] = time.Now().Add(-time.Hour).Unix()

	service := newService(t, issuer, &memorySource{raw: issuer.Sign(claims)})
	require.NoError(t, service.Refresh(context.Background()))

	status := service.Status()
	assert.Equal(t, licensing.StateExpired, status.State)
	assert.Equal(t, licensing.EditionCommunity, status.Edition)
	require.NotNil(t, status.License, "an expired license stays visible to administrators")
	assert.False(t, service.IsEntitled(licensing.FeatureGroups))
}

func TestServiceReportsInvalidLicense(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	other := licensingtest.NewIssuer("untrusted")
	service := newService(t, issuer, &memorySource{raw: other.License(licensing.FeatureGroups)})
	require.NoError(t, service.Refresh(context.Background()))

	status := service.Status()
	assert.Equal(t, licensing.StateInvalid, status.State)
	assert.Equal(t, licensing.ReasonUnknownKey, status.Reason)
	assert.Nil(t, status.License)
	assert.False(t, service.IsEntitled(licensing.FeatureGroups))
}

func TestServiceKeepsLastStateOnTemporaryDatabaseError(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	source := &memorySource{raw: issuer.License(licensing.FeatureGroups)}
	service := newService(t, issuer, source)
	require.NoError(t, service.Refresh(context.Background()))

	source.err = errors.New("connection refused")
	require.Error(t, service.Refresh(context.Background()))

	assert.True(t, service.IsEntitled(licensing.FeatureGroups))
}

func TestServiceAppliesChangesWhenReloadFails(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")

	t.Run("remove stops access", func(t *testing.T) {
		source := &memorySource{raw: issuer.License(licensing.FeatureGroups)}
		service := newService(t, issuer, source)
		require.NoError(t, service.Refresh(context.Background()))

		source.err = errors.New("connection refused")
		status, err := service.Remove(context.Background(), uuid.New())
		require.NoError(t, err)

		assert.Equal(t, licensing.EditionCommunity, status.Edition)
		assert.False(t, service.IsEntitled(licensing.FeatureGroups))
	})

	t.Run("replacement grants neither license until a read confirms it", func(t *testing.T) {
		source := &memorySource{raw: issuer.License(licensing.FeatureCustomRoles)}
		service := newService(t, issuer, source)
		require.NoError(t, service.Refresh(context.Background()))

		source.err = errors.New("connection refused")
		_, err := service.Install(context.Background(), issuer.License(licensing.FeatureGroups), uuid.New())
		require.Error(t, err)

		assert.False(t, service.IsEntitled(licensing.FeatureCustomRoles))
		assert.False(t, service.IsEntitled(licensing.FeatureGroups))

		source.err = nil
		require.NoError(t, service.Refresh(context.Background()))
		assert.True(t, service.IsEntitled(licensing.FeatureGroups))
	})
}

func TestServiceConcurrentChangesMatchStoredState(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	raw := issuer.License(licensing.FeatureGroups)
	source := &memorySource{}
	service := newService(t, issuer, source)
	require.NoError(t, service.Refresh(context.Background()))

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			status, err := service.Install(context.Background(), raw, uuid.New())
			assert.NoError(t, err)
			assert.Equal(t, licensing.EditionEnterprise, status.Edition, "install reports the license that it stored")
		}()
		go func() {
			defer wg.Done()
			status, err := service.Remove(context.Background(), uuid.New())
			assert.NoError(t, err)
			assert.Equal(t, licensing.EditionCommunity, status.Edition, "remove reports the removal")
		}()
	}
	wg.Wait()

	assert.Equal(t, source.raw != nil, service.IsEntitled(licensing.FeatureGroups))
}

func TestServiceRefreshPicksUpChangesFromOtherReplicas(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	source := &memorySource{}
	service := newService(t, issuer, source)
	require.NoError(t, service.Refresh(context.Background()))
	assert.False(t, service.IsEntitled(licensing.FeatureGroups))

	source.raw = issuer.License(licensing.FeatureGroups)
	assert.False(t, service.IsEntitled(licensing.FeatureGroups), "state changes only on refresh")
	require.NoError(t, service.Refresh(context.Background()))
	assert.True(t, service.IsEntitled(licensing.FeatureGroups))

	source.raw = nil
	require.NoError(t, service.Refresh(context.Background()))
	assert.False(t, service.IsEntitled(licensing.FeatureGroups))
	assert.Equal(t, licensing.StateNone, service.Status().State)
}

func TestServiceStopsEntitlingWhenLicenseExpiresBetweenRefreshes(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")
	claims := licensingtest.Claims(licensing.FeatureGroups)
	expiresAt := time.Now().Add(time.Hour)
	claims["exp"] = expiresAt.Unix()

	service := newService(t, issuer, &memorySource{raw: issuer.Sign(claims)})
	require.NoError(t, service.Refresh(context.Background()))
	assert.True(t, service.IsEntitled(licensing.FeatureGroups))

	service.SetClock(func() time.Time { return expiresAt.Add(licensing.ClockSkew - time.Second) })
	assert.True(t, service.IsEntitled(licensing.FeatureGroups), "clock skew tolerance applies")

	service.SetClock(func() time.Time { return expiresAt.Add(licensing.ClockSkew) })
	assert.False(t, service.IsEntitled(licensing.FeatureGroups))
	assert.Equal(t, licensing.StateExpired, service.Status().State)
}

func TestServiceLicenseFileIsAuthoritative(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")

	t.Run("valid file", func(t *testing.T) {
		path := writeLicenseFile(t, issuer.License(licensing.FeatureCustomRoles))
		service := newService(t, issuer, licensing.NewFileSource(path))
		require.NoError(t, service.Refresh(context.Background()))

		status := service.Status()
		assert.Equal(t, licensing.SourceFile, status.Source)
		assert.False(t, status.Writable)
		assert.True(t, service.IsEntitled(licensing.FeatureCustomRoles))
	})

	t.Run("invalid file does not fall back", func(t *testing.T) {
		path := writeLicenseFile(t, []byte("not-a-license"))
		service := newService(t, issuer, licensing.NewFileSource(path))
		require.NoError(t, service.Refresh(context.Background()))

		status := service.Status()
		assert.Equal(t, licensing.SourceFile, status.Source)
		assert.Equal(t, licensing.StateInvalid, status.State)
		assert.Equal(t, licensing.ReasonMalformed, status.Reason)
	})

	t.Run("missing file", func(t *testing.T) {
		service := newService(t, issuer, licensing.NewFileSource(filepath.Join(t.TempDir(), "missing")))
		require.NoError(t, service.Refresh(context.Background()))

		status := service.Status()
		assert.Equal(t, licensing.StateInvalid, status.State)
		assert.Equal(t, licensing.ReasonUnreadable, status.Reason)
	})

	t.Run("oversized file", func(t *testing.T) {
		path := writeLicenseFile(t, make([]byte, licensing.MaxLicenseBytes+1))
		service := newService(t, issuer, licensing.NewFileSource(path))
		require.NoError(t, service.Refresh(context.Background()))
		assert.Equal(t, licensing.ReasonMalformed, service.Status().Reason)
	})

	t.Run("install and remove are refused", func(t *testing.T) {
		path := writeLicenseFile(t, issuer.License(licensing.FeatureCustomRoles))
		service := newService(t, issuer, licensing.NewFileSource(path))

		_, err := service.Install(context.Background(), issuer.License(licensing.FeatureGroups), uuid.New())
		require.ErrorIs(t, err, licensing.ErrManagedByFile)

		_, err = service.Remove(context.Background(), uuid.New())
		require.ErrorIs(t, err, licensing.ErrManagedByFile)
	})
}

func TestSourceFromEnvironment(t *testing.T) {
	t.Setenv(licensing.LicensePathEnv, "")
	assert.Equal(t, licensing.SourceDatabase, licensing.SourceFromEnvironment(nil).Kind())

	t.Setenv(licensing.LicensePathEnv, "/etc/superplane/license.jws")
	assert.Equal(t, licensing.SourceFile, licensing.SourceFromEnvironment(nil).Kind())
}

func TestServiceInstall(t *testing.T) {
	issuer := licensingtest.NewIssuer("test-2026-01")

	t.Run("installs a valid license", func(t *testing.T) {
		source := &memorySource{}
		service := newService(t, issuer, source)
		token := issuer.License(licensing.FeatureGroups)

		status, err := service.Install(context.Background(), append(token, '\n'), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, licensing.StateActive, status.State)
		assert.Equal(t, token, source.raw, "stored value is trimmed")
	})

	rejected := map[string]struct {
		claims func() map[string]any
		reason licensing.Reason
	}{
		"expired": {func() map[string]any {
			c := licensingtest.Claims(licensing.FeatureGroups)
			c["iat"], c["nbf"], c["exp"] = time.Now().Add(-48*time.Hour).Unix(), time.Now().Add(-48*time.Hour).Unix(), time.Now().Add(-time.Hour).Unix()
			return c
		}, licensing.ReasonExpired},
		"not yet valid": {func() map[string]any {
			c := licensingtest.Claims(licensing.FeatureGroups)
			c["iat"], c["nbf"] = time.Now().Unix(), time.Now().Add(24*time.Hour).Unix()
			return c
		}, licensing.ReasonNotYetValid},
		"wrong audience": {func() map[string]any {
			c := licensingtest.Claims(licensing.FeatureGroups)
			c["aud"] = "other"
			return c
		}, licensing.ReasonInvalidClaims},
	}

	for name, tc := range rejected {
		t.Run("rejects "+name+" without replacing the current license", func(t *testing.T) {
			current := issuer.License(licensing.FeatureCustomRoles)
			source := &memorySource{raw: current}
			service := newService(t, issuer, source)
			require.NoError(t, service.Refresh(context.Background()))

			_, err := service.Install(context.Background(), issuer.Sign(tc.claims()), uuid.New())
			assert.Equal(t, tc.reason, licensing.ReasonOf(err))
			assert.Equal(t, current, source.raw)
			assert.True(t, service.IsEntitled(licensing.FeatureCustomRoles))
		})
	}

	t.Run("remove returns to Community", func(t *testing.T) {
		source := &memorySource{raw: issuer.License(licensing.FeatureGroups)}
		service := newService(t, issuer, source)
		require.NoError(t, service.Refresh(context.Background()))

		status, err := service.Remove(context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, licensing.StateNone, status.State)
		assert.False(t, service.IsEntitled(licensing.FeatureGroups))
	})
}

func TestAllowsFailsClosed(t *testing.T) {
	assert.False(t, licensing.Allows(nil, licensing.FeatureGroups))
	assert.False(t, licensing.Allows(licensing.Community, licensing.FeatureGroups))
}

type failingTransport struct {
	t *testing.T
}

func (f failingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.t.Errorf("license verification made a network request to %s", r.URL.Host)
	return nil, errors.New("network access is blocked in this test")
}

// TestVerificationWorksWithoutNetwork proves that license verification never
// contacts the issuer or any other host.
func TestVerificationWorksWithoutNetwork(t *testing.T) {
	originalTransport := http.DefaultTransport
	originalDial := net.DefaultResolver.Dial
	originalPreferGo := net.DefaultResolver.PreferGo
	http.DefaultTransport = failingTransport{t: t}
	net.DefaultResolver.PreferGo = true
	net.DefaultResolver.Dial = func(context.Context, string, string) (net.Conn, error) {
		t.Errorf("license verification resolved a host name")
		return nil, errors.New("network access is blocked in this test")
	}
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
		net.DefaultResolver.Dial = originalDial
		net.DefaultResolver.PreferGo = originalPreferGo
	})

	issuer := licensingtest.NewIssuer("test-2026-01")
	path := writeLicenseFile(t, issuer.License(licensing.FeatureGroups))
	service := newService(t, issuer, licensing.NewFileSource(path))
	require.NoError(t, service.Refresh(context.Background()))
	assert.True(t, service.IsEntitled(licensing.FeatureGroups))

	_, err := licensing.TrustedKeyStore(nil)
	require.NoError(t, err)
}
