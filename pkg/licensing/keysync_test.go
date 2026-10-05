package licensing_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
)

type memoryKeyCache struct {
	mu       sync.Mutex
	document string
	version  int64
	saveErr  error
}

func (m *memoryKeyCache) Load(context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.document, nil
}

func (m *memoryKeyCache) Save(_ context.Context, list *licensing.KeyList) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	if list.Version > m.version {
		m.document, m.version = list.Document, list.Version
	}
	return nil
}

// keyServer serves a key list and counts the downloads.
type keyServer struct {
	*httptest.Server
	mu       sync.Mutex
	document []byte
	requests atomic.Int32
}

func newKeyServer(t *testing.T, document []byte) *keyServer {
	t.Helper()
	server := &keyServer{document: document}
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.requests.Add(1)
		server.mu.Lock()
		defer server.mu.Unlock()
		etag := fmt.Sprintf(`"%x"`, sha256.Sum256(server.document))
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		_, _ = w.Write(server.document)
	}))
	t.Cleanup(server.Close)
	return server
}

func (s *keyServer) serve(document []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.document = document
}

type keyFixture struct {
	root  *licensingtest.Issuer
	first *licensingtest.Issuer
	next  *licensingtest.Issuer
	store *licensing.KeyStore
	cache *memoryKeyCache
}

// newKeyFixture trusts version 1 with the first signing key.
func newKeyFixture(t *testing.T) *keyFixture {
	t.Helper()
	root := licensingtest.NewIssuer("root-2026")
	first := licensingtest.NewIssuer("signing-2026-01")
	store, err := licensing.NewKeyStore(licensingtest.KeySet(root), root.KeyList(1, first), nil)
	require.NoError(t, err)
	return &keyFixture{
		root:  root,
		first: first,
		next:  licensingtest.NewIssuer("signing-2026-02"),
		store: store,
		cache: &memoryKeyCache{},
	}
}

func TestKeySyncDownloadsNewerKeys(t *testing.T) {
	fixture := newKeyFixture(t)
	server := newKeyServer(t, fixture.root.KeyList(2, fixture.first, fixture.next))
	keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)
	assert.Equal(t, licensing.KeySyncSyncing, keySync.Status().State)

	changed, err := keySync.Sync(context.Background())
	require.NoError(t, err)
	assert.True(t, changed)
	_, ok := fixture.store.PublicKey(fixture.next.KeyID)
	assert.True(t, ok)
	assert.Equal(t, int64(2), fixture.cache.version, "the list is shared with other replicas")

	status := keySync.Status()
	assert.Equal(t, licensing.KeySyncSynced, status.State)
	assert.Equal(t, int64(2), status.Version)
	require.NotNil(t, status.SyncedAt)

	changed, err = keySync.Sync(context.Background())
	require.NoError(t, err)
	assert.False(t, changed, "an unchanged list is not modified")
}

func TestKeySyncKeepsTrustedKeysOnBadResponses(t *testing.T) {
	cases := map[string]func(f *keyFixture) []byte{
		"another root":    func(f *keyFixture) []byte { return licensingtest.NewIssuer("root-2026").KeyList(2, f.next) },
		"invalid version": func(f *keyFixture) []byte { return f.root.KeyList(0, f.next) },
		"not a list":      func(*keyFixture) []byte { return []byte("<html>maintenance</html>") },
	}

	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			fixture := newKeyFixture(t)
			server := newKeyServer(t, document(fixture))
			keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)

			changed, err := keySync.Sync(context.Background())
			require.Error(t, err)
			assert.False(t, changed)
			assert.Equal(t, int64(1), fixture.store.Current().Version)
			assert.Equal(t, licensing.KeySyncFailed, keySync.Status().State)
		})
	}

	t.Run("a replayed older list", func(t *testing.T) {
		fixture := newKeyFixture(t)
		server := newKeyServer(t, fixture.root.KeyList(3, fixture.next))
		keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)
		_, err := keySync.Sync(context.Background())
		require.NoError(t, err)

		server.serve(fixture.root.KeyList(2, fixture.first, fixture.next))
		_, err = keySync.Sync(context.Background())
		require.ErrorIs(t, err, licensing.ErrKeyListDowngrade)
		_, ok := fixture.store.PublicKey(fixture.first.KeyID)
		assert.False(t, ok)
	})

	t.Run("an unreachable issuer", func(t *testing.T) {
		fixture := newKeyFixture(t)
		server := newKeyServer(t, nil)
		server.Close()
		keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)

		_, err := keySync.Sync(context.Background())
		require.Error(t, err)
		_, ok := fixture.store.PublicKey(fixture.first.KeyID)
		assert.True(t, ok)
	})
}

func TestKeySyncReloadsTheSharedCache(t *testing.T) {
	fixture := newKeyFixture(t)
	keySync := licensing.NewKeySync(fixture.store, fixture.cache, "")
	assert.Equal(t, licensing.KeySyncDisabled, keySync.Status().State)

	t.Run("fills an empty cache", func(t *testing.T) {
		changed, err := keySync.Reload(context.Background())
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, int64(1), fixture.cache.version)
	})

	t.Run("rejects a tampered cache", func(t *testing.T) {
		fixture.cache.document = string(licensingtest.NewIssuer("root-2026").KeyList(5, fixture.next))
		_, err := keySync.Reload(context.Background())
		require.ErrorIs(t, err, licensing.ErrInvalidKeyList)
		assert.Equal(t, int64(1), fixture.store.Current().Version)
	})

	t.Run("trusts a newer list from another replica", func(t *testing.T) {
		fixture.cache.document = string(fixture.root.KeyList(2, fixture.next))
		changed, err := keySync.Reload(context.Background())
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, int64(2), fixture.store.Current().Version)
	})

	t.Run("replaces an older cached list", func(t *testing.T) {
		fixture.cache.document, fixture.cache.version = string(fixture.root.KeyList(1, fixture.first)), 1
		changed, err := keySync.Reload(context.Background())
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, int64(2), fixture.store.Current().Version)
		assert.Equal(t, int64(2), fixture.cache.version)
	})
}

func TestKeySyncInstallsAnUploadedList(t *testing.T) {
	fixture := newKeyFixture(t)
	keySync := licensing.NewKeySync(fixture.store, fixture.cache, "")

	_, err := keySync.Install(context.Background(), licensingtest.NewIssuer("root-2026").KeyList(2, fixture.next))
	require.ErrorIs(t, err, licensing.ErrInvalidKeyList)

	list, err := keySync.Install(context.Background(), fixture.root.KeyList(2, fixture.next))
	require.NoError(t, err)
	assert.Equal(t, int64(2), list.Version)
	assert.Equal(t, int64(2), fixture.cache.version)
}

func TestKeySyncRetriesAFailedCacheWrite(t *testing.T) {
	fixture := newKeyFixture(t)
	server := newKeyServer(t, fixture.root.KeyList(2, fixture.first, fixture.next))
	keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)

	fixture.cache.saveErr = errors.New("database is down")
	changed, err := keySync.Sync(context.Background())
	require.Error(t, err)
	assert.True(t, changed, "this replica already uses the new keys")
	assert.Equal(t, int64(0), fixture.cache.version)

	fixture.cache.saveErr = nil
	changed, err = keySync.Reload(context.Background())
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, int64(2), fixture.cache.version, "the next reload shares the list with other replicas")
}

func TestKeySyncOnDemandIsRateLimited(t *testing.T) {
	fixture := newKeyFixture(t)
	server := newKeyServer(t, fixture.root.KeyList(1, fixture.first))
	keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)
	now := time.Now()
	keySync.SetClock(func() time.Time { return now })

	for range 3 {
		_, err := keySync.SyncOnDemand(context.Background())
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), server.requests.Load())

	now = now.Add(time.Minute)
	_, err := keySync.SyncOnDemand(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(2), server.requests.Load())
}

func TestKeysURLFromEnvironment(t *testing.T) {
	cases := map[string]string{
		"":                            licensing.DefaultKeysURL,
		"none":                        "",
		"https://mirror.example/keys": "https://mirror.example/keys",
	}
	for value, want := range cases {
		t.Setenv(licensing.KeysURLEnv, value)
		got, err := licensing.KeysURLFromEnvironment()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}

	for _, value := range []string{"file:///etc/keys", "mirror.example/keys"} {
		t.Setenv(licensing.KeysURLEnv, value)
		_, err := licensing.KeysURLFromEnvironment()
		require.Error(t, err, value)
	}
}

func TestServiceSyncsKeysForALicenseFromANewKey(t *testing.T) {
	t.Run("on install", func(t *testing.T) {
		fixture := newKeyFixture(t)
		server := newKeyServer(t, fixture.root.KeyList(2, fixture.first, fixture.next))
		keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)
		service := licensing.NewService(licensing.NewVerifier(fixture.store), &memorySource{}, licensing.WithKeySync(keySync))

		status, err := service.Install(context.Background(), fixture.next.License(licensing.FeatureGroups), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, licensing.StateActive, status.State)
	})

	t.Run("on refresh", func(t *testing.T) {
		fixture := newKeyFixture(t)
		server := newKeyServer(t, fixture.root.KeyList(2, fixture.next))
		keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)
		source := &memorySource{raw: fixture.next.License(licensing.FeatureGroups)}
		service := licensing.NewService(licensing.NewVerifier(fixture.store), source, licensing.WithKeySync(keySync))

		require.NoError(t, service.Refresh(context.Background()))
		assert.True(t, service.IsEntitled(licensing.FeatureGroups))
	})

	t.Run("on install with a list that another replica saved", func(t *testing.T) {
		fixture := newKeyFixture(t)
		keySync := licensing.NewKeySync(fixture.store, fixture.cache, "")
		service := licensing.NewService(licensing.NewVerifier(fixture.store), &memorySource{}, licensing.WithKeySync(keySync))
		fixture.cache.document, fixture.cache.version = string(fixture.root.KeyList(2, fixture.first, fixture.next)), 2

		status, err := service.Install(context.Background(), fixture.next.License(licensing.FeatureGroups), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, licensing.StateActive, status.State)
	})

	t.Run("an unknown key stays invalid when the issuer does not list it", func(t *testing.T) {
		fixture := newKeyFixture(t)
		server := newKeyServer(t, fixture.root.KeyList(1, fixture.first))
		keySync := licensing.NewKeySync(fixture.store, fixture.cache, server.URL)
		service := licensing.NewService(licensing.NewVerifier(fixture.store), &memorySource{}, licensing.WithKeySync(keySync))

		_, err := service.Install(context.Background(), fixture.next.License(licensing.FeatureGroups), uuid.New())
		assert.Equal(t, licensing.ReasonUnknownKey, licensing.ReasonOf(err))
	})
}
