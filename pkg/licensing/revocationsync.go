package licensing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const RevocationSyncInterval = 5 * time.Minute

const revocationSyncTimeout = 15 * time.Second

// RevocationListCache stores the newest revocation list for every replica.
// Load returns an empty document when nothing is cached.
type RevocationListCache interface {
	Load(ctx context.Context) (string, error)
	Save(ctx context.Context, list *RevocationList) error
}

// RevocationSync downloads the revocation list and keeps the last list that
// verified. A failed download does not change that list.
type RevocationSync struct {
	roots  *KeySet
	cache  RevocationListCache
	url    string
	client *http.Client

	syncMu  sync.Mutex
	listMu  sync.Mutex
	etag    string
	current atomic.Pointer[RevocationList]
}

// NewRevocationSync downloads from revocationsURL. An empty URL turns
// downloads off.
func NewRevocationSync(roots *KeySet, cache RevocationListCache, revocationsURL string) *RevocationSync {
	return &RevocationSync{
		roots:  roots,
		cache:  cache,
		url:    revocationsURL,
		client: &http.Client{Timeout: revocationSyncTimeout, Transport: http.DefaultTransport.(*http.Transport).Clone()},
	}
}

// RevocationsURL returns the revocation list URL for a key-list URL. An empty
// key-list URL, which disables key downloads, also disables this download.
func RevocationsURL(keysURL string) string {
	if keysURL == "" {
		return ""
	}

	parsed, err := url.Parse(keysURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return ""
	}

	parsed.Path = "/.well-known/license-revocations.jws"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

// Contains reports whether the last verified list names the license id. No
// verified list names nothing.
func (k *RevocationSync) Contains(id uuid.UUID) bool {
	return k.current.Load().Contains(id)
}

// Start syncs once and then every RevocationSyncInterval until the context ends.
func (k *RevocationSync) Start(ctx context.Context) {
	if k.url == "" {
		return
	}

	go func() {
		ticker := time.NewTicker(RevocationSyncInterval)
		defer ticker.Stop()

		for {
			if _, err := k.Sync(ctx); err != nil {
				log.WithError(err).Warn("Licensing: revocation list sync failed; keeping the last verified list")
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// Reload trusts the cached list when it is newer. Another replica may have
// saved it. When the cache is older, Reload saves the trusted list.
func (k *RevocationSync) Reload(ctx context.Context) (bool, error) {
	document, err := k.cache.Load(ctx)
	if err != nil {
		return false, err
	}

	current := k.current.Load()
	if current != nil && current.Document == document {
		return false, nil
	}

	if document == "" {
		return false, k.share(ctx, current)
	}

	list, err := VerifyRevocationList([]byte(document), k.roots)
	if err != nil {
		return false, err
	}

	changed, err := k.trust(ctx, list)
	if errors.Is(err, ErrRevocationListDowngrade) {
		return false, k.share(ctx, current)
	}

	return changed, err
}

func (k *RevocationSync) share(ctx context.Context, list *RevocationList) error {
	if list == nil {
		return nil
	}

	if err := k.cache.Save(ctx, list); err != nil {
		return fmt.Errorf("cache revocation list: %w", err)
	}

	return nil
}

// Sync downloads the revocation list and trusts it when it is newer. It
// reports whether the trusted list changed.
func (k *RevocationSync) Sync(ctx context.Context) (bool, error) {
	if k.url == "" {
		return false, nil
	}

	k.syncMu.Lock()
	defer k.syncMu.Unlock()
	return k.download(ctx)
}

func (k *RevocationSync) download(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, revocationSyncTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, k.url, nil)
	if err != nil {
		return false, fmt.Errorf("build revocation list request: %w", err)
	}

	request.Header.Set("Accept", "application/jose")
	if k.etag != "" {
		request.Header.Set("If-None-Match", k.etag)
	}

	response, err := k.client.Do(request)
	if err != nil {
		return false, fmt.Errorf("download revocation list: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotModified {
		return false, nil
	}

	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("download revocation list: unexpected status %d", response.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(response.Body, MaxRevocationListBytes+1))
	if err != nil {
		return false, fmt.Errorf("read revocation list: %w", err)
	}

	list, err := VerifyRevocationList(raw, k.roots)
	if err != nil {
		return false, err
	}

	changed, err := k.trust(ctx, list)
	if err == nil {
		k.etag = response.Header.Get("ETag")
	}

	return changed, err
}

// trust saves the list even when this replica already trusts it, so a retry
// repairs an earlier failed cache write.
func (k *RevocationSync) trust(ctx context.Context, list *RevocationList) (bool, error) {
	k.listMu.Lock()
	defer k.listMu.Unlock()

	current := k.current.Load()
	if current != nil && list.Version < current.Version {
		return false, ErrRevocationListDowngrade
	}

	changed := current == nil || list.Version > current.Version
	if changed {
		k.current.Store(list)
		log.WithFields(log.Fields{"version": list.Version}).Info("Licensing: trusting a new license revocation list")
	}

	return changed, k.share(ctx, list)
}

// DatabaseRevocationListCache shares the revocation list between replicas.
type DatabaseRevocationListCache struct{}

func (DatabaseRevocationListCache) Load(ctx context.Context) (string, error) {
	record, err := models.FindInstallationLicenseRevocations(database.DB(ctx))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("find revocation list: %w", err)
	}

	return record.Document, nil
}

func (DatabaseRevocationListCache) Save(ctx context.Context, list *RevocationList) error {
	return models.SaveInstallationLicenseRevocations(database.DB(ctx), list.Document, list.Version)
}
