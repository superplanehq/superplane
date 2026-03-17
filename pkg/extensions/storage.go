package extensions

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	storage "github.com/superplanehq/superplane/pkg/storage"
)

type Storage struct {
	underlying    storage.Storage
	manifestCache *ristretto.Cache[string, *Manifest]
	loadManifest  func(organizationID string) (*Manifest, error)
}

func NewStorage(underlying storage.Storage, loadManifest func(organizationID string) (*Manifest, error)) (*Storage, error) {
	if loadManifest == nil {
		return nil, fmt.Errorf("loadManifest is required")
	}

	//
	// MaxCost=100 and cost for every manifest is 1,
	// so we can keep at most 100 manifests in memory.
	//
	cache, err := ristretto.NewCache(&ristretto.Config[string, *Manifest]{
		MaxCost:     100,
		NumCounters: 1000,
		BufferItems: 64,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating manifest cache: %w", err)
	}

	return &Storage{
		underlying:    underlying,
		manifestCache: cache,
		loadManifest:  loadManifest,
	}, nil
}

func (s *Storage) ListIntegrations(organizationID string) ([]IntegrationManifest, error) {
	manifest, ok := s.manifestCache.Get(organizationID)
	if ok {
		return manifest.Integrations, nil
	}

	manifest, err := s.loadManifest(organizationID)
	if err != nil {
		return nil, err
	}

	s.manifestCache.SetWithTTL(organizationID, manifest, 1, time.Minute)
	return manifest.Integrations, nil
}

func (s *Storage) ListComponents(organizationID string) ([]ComponentManifest, error) {
	manifest, ok := s.manifestCache.Get(organizationID)
	if ok {
		return manifest.Components, nil
	}

	manifest, err := s.loadManifest(organizationID)
	if err != nil {
		return nil, err
	}

	s.manifestCache.SetWithTTL(organizationID, manifest, 1, time.Minute)
	return manifest.Components, nil
}

func (s *Storage) ListTriggers(organizationID string) ([]TriggerManifest, error) {
	manifest, ok := s.manifestCache.Get(organizationID)
	if ok {
		return manifest.Triggers, nil
	}

	manifest, err := s.loadManifest(organizationID)
	if err != nil {
		return nil, err
	}

	s.manifestCache.SetWithTTL(organizationID, manifest, 1, time.Minute)
	return manifest.Triggers, nil
}

func (s *Storage) UploadVersion(organizationID string, extensionName string, versionName string, files *BundleFiles) error {
	return s.writeVersionFiles(organizationID, extensionName, versionName, files)
}

func (s *Storage) ReadVersionManifestJSON(organizationID string, extensionName string, versionName string) ([]byte, error) {
	return s.readVersionNamedFile(organizationID, extensionName, versionName, "manifest.json")
}

func (s *Storage) ReadVersionBundleJS(organizationID string, extensionName string, versionName string) ([]byte, error) {
	return s.readVersionNamedFile(organizationID, extensionName, versionName, "bundle.js")
}

func (s *Storage) readVersionNamedFile(organizationID string, extensionName string, versionName string, filename string) ([]byte, error) {
	return s.readVersionFile(organizationID, extensionName, versionName, filename)
}

func (s *Storage) writeVersionFiles(organizationID string, extensionName string, versionName string, files *BundleFiles) error {
	manifestPath := versionFilePath(organizationID, extensionName, versionName, "manifest.json")
	if err := s.underlying.Write(manifestPath, bytes.NewReader(files.ManifestJSON)); err != nil {
		return fmt.Errorf("write manifest.json: %w", err)
	}

	bundlePath := versionFilePath(organizationID, extensionName, versionName, "bundle.js")
	if err := s.underlying.Write(bundlePath, bytes.NewReader(files.BundleJS)); err != nil {
		return fmt.Errorf("write bundle.js: %w", err)
	}

	return nil
}

func (s *Storage) readVersionFile(organizationID string, extensionName string, versionName string, filename string) ([]byte, error) {
	reader, err := s.underlying.Read(versionFilePath(organizationID, extensionName, versionName, filename))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filename, err)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read %s content: %w", filename, err)
	}

	return content, nil
}

func versionFilePath(organizationID string, extensionName string, versionName string, filename string) string {
	return path.Join(organizationID, extensionName, versionName, filename)
}
