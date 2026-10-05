package licensing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	// KeysURLEnv overrides where SuperPlane downloads the license key list.
	// The value "none" turns downloads off, for example without network
	// access.
	KeysURLEnv     = "SUPERPLANE_LICENSE_KEYS_URL"
	DefaultKeysURL = "https://licensing.superplane.com/.well-known/license-keys.jws"

	KeySyncInterval = 6 * time.Hour

	keySyncTimeout          = 15 * time.Second
	onDemandKeySyncInterval = time.Minute
)

type KeySyncState string

const (
	KeySyncDisabled KeySyncState = "disabled"
	KeySyncSyncing  KeySyncState = "syncing"
	KeySyncSynced   KeySyncState = "synced"
	KeySyncFailed   KeySyncState = "failed"
)

// KeySyncStatus describes the trusted key list. Version is zero when no key
// list is trusted.
type KeySyncStatus struct {
	State    KeySyncState
	Version  int64
	SyncedAt *time.Time
}

// KeyListCache stores the newest key list for every replica. Load returns an
// empty document when nothing is cached.
type KeyListCache interface {
	Load(ctx context.Context) (string, error)
	Save(ctx context.Context, list *KeyList) error
}

// KeySync keeps the trusted license signing keys current. It downloads the
// key list, verifies it with the root keys, and shares it through the cache.
type KeySync struct {
	store  *KeyStore
	cache  KeyListCache
	url    string
	client *http.Client
	now    func() time.Time

	syncMu       sync.Mutex
	etag         string
	lastOnDemand time.Time

	statusMu sync.Mutex
	state    KeySyncState
	syncedAt *time.Time
}

// NewKeySync downloads from keysURL. An empty keysURL turns downloads off.
func NewKeySync(store *KeyStore, cache KeyListCache, keysURL string) *KeySync {
	state := KeySyncSyncing
	if keysURL == "" {
		state = KeySyncDisabled
	}

	return &KeySync{
		store:  store,
		cache:  cache,
		url:    keysURL,
		client: &http.Client{Timeout: keySyncTimeout, Transport: http.DefaultTransport.(*http.Transport).Clone()},
		now:    time.Now,
		state:  state,
	}
}

// KeysURLFromEnvironment returns the key list URL, or an empty string when
// downloads are off.
func KeysURLFromEnvironment() (string, error) {
	configured := os.Getenv(KeysURLEnv)
	switch configured {
	case "":
		return DefaultKeysURL, nil
	case "none":
		return "", nil
	}

	parsed, err := url.Parse(configured)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return "", fmt.Errorf("%s must be an http or https URL, or none", KeysURLEnv)
	}

	return configured, nil
}

func (k *KeySync) Status() KeySyncStatus {
	k.statusMu.Lock()
	defer k.statusMu.Unlock()

	status := KeySyncStatus{State: k.state, SyncedAt: k.syncedAt}
	if current := k.store.Current(); current != nil {
		status.Version = current.Version
	}

	return status
}

// Start syncs once and then every KeySyncInterval until the context ends. It
// calls onChange after the trusted keys change.
func (k *KeySync) Start(ctx context.Context, onChange func()) {
	if k.url == "" {
		return
	}

	go func() {
		ticker := time.NewTicker(KeySyncInterval)
		defer ticker.Stop()

		for {
			changed, err := k.Sync(ctx)
			if err != nil {
				log.WithError(err).Warn("Licensing: license key sync failed; keeping the trusted keys")
			}

			if changed {
				onChange()
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// Reload trusts the cached key list when it is newer. Another replica may have
// saved it. It reports whether the trusted keys changed.
func (k *KeySync) Reload(ctx context.Context) (bool, error) {
	document, err := k.cache.Load(ctx)
	if err != nil || document == "" {
		return false, err
	}

	if current := k.store.Current(); current != nil && current.Document == document {
		return false, nil
	}

	list, err := VerifyKeyList([]byte(document), k.store.Roots())
	if err != nil {
		return false, err
	}

	changed, err := k.store.Update(list)
	if errors.Is(err, ErrKeyListDowngrade) {
		return false, nil
	}

	return changed, err
}

// Sync downloads the key list and trusts it when it is newer. It reports
// whether the trusted keys changed.
func (k *KeySync) Sync(ctx context.Context) (bool, error) {
	if k.url == "" {
		return false, nil
	}

	k.syncMu.Lock()
	defer k.syncMu.Unlock()

	changed, err := k.download(ctx)
	k.recordAttempt(err)
	return changed, err
}

// SyncOnDemand runs Sync at most once per minute. Call it when a license uses
// an unknown key, which happens after the issuer adds a signing key.
func (k *KeySync) SyncOnDemand(ctx context.Context) (bool, error) {
	if k.url == "" {
		return false, nil
	}

	k.syncMu.Lock()
	if k.now().Sub(k.lastOnDemand) < onDemandKeySyncInterval {
		k.syncMu.Unlock()
		return false, nil
	}

	k.lastOnDemand = k.now()
	k.syncMu.Unlock()
	return k.Sync(ctx)
}

// Install trusts a key list that an administrator supplies, for example
// without network access.
func (k *KeySync) Install(ctx context.Context, raw []byte) (*KeyList, error) {
	list, err := VerifyKeyList(raw, k.store.Roots())
	if err != nil {
		return nil, err
	}

	if _, err := k.trust(ctx, list); err != nil {
		return nil, err
	}

	k.recordAttempt(nil)
	return list, nil
}

func (k *KeySync) download(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, keySyncTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, k.url, nil)
	if err != nil {
		return false, fmt.Errorf("build license key request: %w", err)
	}

	request.Header.Set("Accept", "application/jose")
	if k.etag != "" {
		request.Header.Set("If-None-Match", k.etag)
	}

	response, err := k.client.Do(request)
	if err != nil {
		return false, fmt.Errorf("download license keys: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotModified {
		return false, nil
	}

	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("download license keys: unexpected status %d", response.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(response.Body, MaxKeyListBytes+1))
	if err != nil {
		return false, fmt.Errorf("read license keys: %w", err)
	}

	list, err := VerifyKeyList(raw, k.store.Roots())
	if err != nil {
		return false, err
	}

	changed, err := k.trust(ctx, list)
	if err == nil {
		k.etag = response.Header.Get("ETag")
	}

	return changed, err
}

// trust reports a change even when the cache write fails, because this
// replica already uses the new keys.
func (k *KeySync) trust(ctx context.Context, list *KeyList) (bool, error) {
	changed, err := k.store.Update(list)
	if err != nil || !changed {
		return false, err
	}

	log.WithFields(log.Fields{"version": list.Version, "kids": list.Keys.KeyIDs()}).Info("Licensing: trusting a new license key list")
	if err := k.cache.Save(ctx, list); err != nil {
		return true, fmt.Errorf("cache license keys: %w", err)
	}

	return true, nil
}

func (k *KeySync) recordAttempt(err error) {
	k.statusMu.Lock()
	defer k.statusMu.Unlock()

	if err != nil {
		k.state = KeySyncFailed
		return
	}

	now := k.now().UTC()
	k.state = KeySyncSynced
	k.syncedAt = &now
}

// DatabaseKeyListCache shares the key list between replicas.
type DatabaseKeyListCache struct{}

func (DatabaseKeyListCache) Load(ctx context.Context) (string, error) {
	record, err := models.FindInstallationLicenseKeys(database.DB(ctx))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("find license keys: %w", err)
	}

	return record.Document, nil
}

func (DatabaseKeyListCache) Save(ctx context.Context, list *KeyList) error {
	return models.SaveInstallationLicenseKeys(database.DB(ctx), list.Document, list.Version)
}
