package licensing_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

type memoryRevocationCache struct {
	mu       sync.Mutex
	document string
	version  int64
}

func (m *memoryRevocationCache) Load(context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.document, nil
}

func (m *memoryRevocationCache) Save(_ context.Context, list *licensing.RevocationList) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if list.Version > m.version {
		m.document, m.version = list.Document, list.Version
	}
	return nil
}

func TestRevocationListStopsOnlyTheListedLicense(t *testing.T) {
	root := licensingtest.NewIssuer("root-2026")
	signer := licensingtest.NewIssuer("signing-2026")
	revokedRaw := signer.License(licensing.FeatureGroups)
	revokedLicense, err := licensing.NewVerifier(licensingtest.KeySet(signer)).Verify(revokedRaw)
	require.NoError(t, err)
	otherRaw := signer.License(licensing.FeatureGroups)

	source := &memorySource{raw: revokedRaw}
	server := newKeyServer(t, root.RevocationList(1, revokedLicense.ID))
	revocations := licensing.NewRevocationSync(licensingtest.KeySet(root), &memoryRevocationCache{}, server.URL)
	service := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(signer)),
		source,
		licensing.WithRevocationSync(revocations),
	)
	require.NoError(t, service.Refresh(context.Background()))
	assert.True(t, service.IsEntitled(licensing.FeatureGroups), "a license stays active before a revocation list arrives")

	changed, err := revocations.Sync(context.Background())
	require.NoError(t, err)
	assert.True(t, changed)

	status := service.Status()
	assert.Equal(t, licensing.StateRevoked, status.State)
	assert.Equal(t, licensing.EditionCommunity, status.Edition)
	require.NotNil(t, status.License)
	assert.Equal(t, revokedLicense.ID, status.License.ID)
	assert.False(t, service.IsEntitled(licensing.FeatureGroups))

	_, err = service.Install(context.Background(), revokedRaw, uuid.New())
	assert.Equal(t, licensing.ReasonRevoked, licensing.ReasonOf(err))
	assert.Equal(t, revokedRaw, source.raw, "a revoked license does not replace the stored license")

	status, err = service.Install(context.Background(), otherRaw, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, licensing.StateActive, status.State)
	assert.True(t, service.IsEntitled(licensing.FeatureGroups))
}

func TestFailedRevocationDownloadKeepsLastResult(t *testing.T) {
	root := licensingtest.NewIssuer("root-2026")
	signer := licensingtest.NewIssuer("signing-2026")
	raw := signer.License(licensing.FeatureGroups)
	license, err := licensing.NewVerifier(licensingtest.KeySet(signer)).Verify(raw)
	require.NoError(t, err)

	t.Run("a failed check does not revoke a working license", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(server.Close)

		revocations := licensing.NewRevocationSync(licensingtest.KeySet(root), &memoryRevocationCache{}, server.URL)
		service := licensing.NewService(
			licensing.NewVerifier(licensingtest.KeySet(signer)),
			&memorySource{raw: raw},
			licensing.WithRevocationSync(revocations),
		)
		require.NoError(t, service.Refresh(context.Background()))
		_, err := revocations.Sync(context.Background())
		require.Error(t, err)
		assert.True(t, service.IsEntitled(licensing.FeatureGroups))
	})

	t.Run("a failed check keeps a revocation that already arrived", func(t *testing.T) {
		server := newKeyServer(t, root.RevocationList(1, license.ID))
		revocations := licensing.NewRevocationSync(licensingtest.KeySet(root), &memoryRevocationCache{}, server.URL)
		service := licensing.NewService(
			licensing.NewVerifier(licensingtest.KeySet(signer)),
			&memorySource{raw: raw},
			licensing.WithRevocationSync(revocations),
		)
		require.NoError(t, service.Refresh(context.Background()))
		_, err := revocations.Sync(context.Background())
		require.NoError(t, err)
		assert.Equal(t, licensing.StateRevoked, service.Status().State)

		server.Close()
		_, err = revocations.Sync(context.Background())
		require.Error(t, err)
		assert.Equal(t, licensing.StateRevoked, service.Status().State)
		assert.False(t, service.IsEntitled(licensing.FeatureGroups))
	})
}

func TestNewerRevocationListCanClearARevocation(t *testing.T) {
	root := licensingtest.NewIssuer("root-2026")
	signer := licensingtest.NewIssuer("signing-2026")
	raw := signer.License(licensing.FeatureGroups)
	license, err := licensing.NewVerifier(licensingtest.KeySet(signer)).Verify(raw)
	require.NoError(t, err)

	server := newKeyServer(t, root.RevocationList(1, license.ID))
	revocations := licensing.NewRevocationSync(licensingtest.KeySet(root), &memoryRevocationCache{}, server.URL)
	service := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(signer)),
		&memorySource{raw: raw},
		licensing.WithRevocationSync(revocations),
	)
	require.NoError(t, service.Refresh(context.Background()))
	_, err = revocations.Sync(context.Background())
	require.NoError(t, err)
	assert.Equal(t, licensing.StateRevoked, service.Status().State)

	server.serve(root.RevocationList(2))
	changed, err := revocations.Sync(context.Background())
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, licensing.StateActive, service.Status().State)
	assert.True(t, service.IsEntitled(licensing.FeatureGroups))

	server.serve(root.RevocationList(1, license.ID))
	_, err = revocations.Sync(context.Background())
	require.ErrorIs(t, err, licensing.ErrRevocationListDowngrade)
	assert.True(t, service.IsEntitled(licensing.FeatureGroups), "an older list cannot restore a cleared revocation")
}

func TestRevocationDownloadsStayOffWithoutAKeyListURL(t *testing.T) {
	signer := licensingtest.NewIssuer("signing-2026")
	raw := signer.License(licensing.FeatureGroups)
	revocations := licensing.NewRevocationSync(licensingtest.KeySet(licensingtest.NewIssuer("root-2026")), &memoryRevocationCache{}, "")
	service := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(signer)),
		&memorySource{raw: raw},
		licensing.WithRevocationSync(revocations),
	)
	require.NoError(t, service.Refresh(context.Background()))

	changed, err := revocations.Sync(context.Background())
	require.NoError(t, err)
	assert.False(t, changed)
	assert.True(t, service.IsEntitled(licensing.FeatureGroups))
}

func TestRefreshAppliesACachedRevocationList(t *testing.T) {
	root := licensingtest.NewIssuer("root-2026")
	signer := licensingtest.NewIssuer("signing-2026")
	raw := signer.License(licensing.FeatureGroups)
	license, err := licensing.NewVerifier(licensingtest.KeySet(signer)).Verify(raw)
	require.NoError(t, err)

	cache := &memoryRevocationCache{}
	list, err := licensing.VerifyRevocationList(root.RevocationList(1, license.ID), licensingtest.KeySet(root))
	require.NoError(t, err)
	require.NoError(t, cache.Save(context.Background(), list))

	service := licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(signer)),
		&memorySource{raw: raw},
		licensing.WithRevocationSync(licensing.NewRevocationSync(licensingtest.KeySet(root), cache, "")),
	)
	require.NoError(t, service.Refresh(context.Background()))
	assert.Equal(t, licensing.StateRevoked, service.Status().State)
	assert.False(t, service.IsEntitled(licensing.FeatureGroups))
}

func TestRevocationsURL(t *testing.T) {
	assert.Empty(t, licensing.RevocationsURL(""))
	assert.Equal(t,
		"https://licensing.superplane.com/.well-known/license-revocations.jws",
		licensing.RevocationsURL(licensing.DefaultKeysURL),
	)
	assert.Equal(t,
		"http://localhost:8100/.well-known/license-revocations.jws",
		licensing.RevocationsURL("http://localhost:8100/.well-known/license-keys.jws?cache=1"),
	)
}

func TestVerifyRevocationListRejectsABadDocument(t *testing.T) {
	root := licensingtest.NewIssuer("root-2026")
	id := uuid.New()
	roots := licensingtest.KeySet(root)

	_, err := licensing.VerifyRevocationList(root.RevocationList(1, id, id), roots)
	require.ErrorIs(t, err, licensing.ErrInvalidRevocationList)

	other := licensingtest.NewIssuer("other-root")
	_, err = licensing.VerifyRevocationList(other.RevocationList(1, id), roots)
	require.ErrorIs(t, err, licensing.ErrInvalidRevocationList)

	list, err := licensing.VerifyRevocationList(root.RevocationList(1), roots)
	require.NoError(t, err)
	assert.False(t, list.Contains(id))
}
