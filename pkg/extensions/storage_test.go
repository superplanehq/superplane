package extensions

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	artifactstorage "github.com/superplanehq/superplane/pkg/storage"
)

func TestStorageImplementationsManageVersions(t *testing.T) {
	t.Parallel()

	storages := map[string]*Storage{
		"in-memory": NewStorage(artifactstorage.NewInMemoryStorage()),
		"folder":    NewStorage(artifactstorage.NewLocalFolderStorage(t.TempDir())),
	}

	for name, storage := range storages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			extension := Extension{
				ID:          "ext-1",
				Name:        "acme.echo",
				Description: "Echo extension",
			}
			require.NoError(t, storage.CreateExtension("org-1", extension))

			filesV1 := newBundleFiles(t, Manifest{
				Integrations: []IntegrationManifest{{
					Name:        "acme.echo.integration",
					Label:       "Echo Integration",
					Description: "Integration",
					Icon:        "icon",
				}},
				Components: []ComponentManifest{{
					Name:        "acme.echo.component",
					Label:       "Echo Component",
					Description: "Component",
					Icon:        "icon",
					Color:       "#fff",
				}},
				Triggers: []TriggerManifest{{
					Name:        "acme.echo.trigger",
					Label:       "Echo Trigger",
					Description: "Trigger",
					Icon:        "icon",
					Color:       "#000",
				}},
			}, "console.log('v1');")

			version := Version{
				ID:           "ver-1",
				ExtensionID:  extension.ID,
				Digest:       "digest-v1",
				State:        "draft",
				Integrations: filesV1.Manifest.Integrations,
				Components:   filesV1.Manifest.Components,
				Triggers:     filesV1.Manifest.Triggers,
			}
			require.NoError(t, storage.CreateVersion("org-1", extension.ID, version, filesV1))

			versions, err := storage.ListVersions("org-1", extension.ID)
			require.NoError(t, err)
			require.Len(t, versions, 1)
			require.Equal(t, "draft", versions[0].State)
			require.Len(t, storage.ListComponents("org-1"), 1)
			require.Len(t, storage.ListTriggers("org-1"), 1)
			require.Len(t, storage.ListIntegrations("org-1"), 1)

			filesV2 := newBundleFiles(t, Manifest{
				Components: []ComponentManifest{{
					Name:        "acme.echo.component.v2",
					Label:       "Echo Component V2",
					Description: "Updated component",
					Icon:        "icon",
					Color:       "#123456",
				}},
			}, "console.log('v2');")

			updatedVersion := Version{
				ID:           version.ID,
				ExtensionID:  extension.ID,
				Digest:       "digest-v2",
				State:        "draft",
				Integrations: filesV2.Manifest.Integrations,
				Components:   filesV2.Manifest.Components,
				Triggers:     filesV2.Manifest.Triggers,
			}
			require.NoError(t, storage.UpdateVersion("org-1", extension.ID, version.ID, updatedVersion, filesV2))

			currentVersion, err := storage.FindVersionById("org-1", extension.ID, version.ID)
			require.NoError(t, err)
			require.Equal(t, "digest-v2", currentVersion.Digest)
			require.Equal(t, "acme.echo.component.v2", currentVersion.Components[0].Name)

			publishedVersion := *currentVersion
			publishedVersion.Version = "1.0.0"
			publishedVersion.State = "published"
			require.NoError(t, storage.UpdateVersion("org-1", extension.ID, version.ID, publishedVersion, nil))

			persistedVersion, err := storage.FindVersionById("org-1", extension.ID, version.ID)
			require.NoError(t, err)
			require.Equal(t, "1.0.0", persistedVersion.Version)
			require.Equal(t, "published", persistedVersion.State)
		})
	}
}

func TestFolderStorageStoresVersionFilesInVersionDirectory(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	storage := NewStorage(artifactstorage.NewLocalFolderStorage(rootDir))

	extension := Extension{
		ID:          "ext-1",
		Name:        "acme.echo",
		Description: "Echo extension",
	}
	require.NoError(t, storage.CreateExtension("org-1", extension))

	files := newBundleFiles(t, Manifest{
		Components: []ComponentManifest{{
			Name:        "acme.echo.component",
			Label:       "Echo Component",
			Description: "Component",
			Icon:        "icon",
			Color:       "#fff",
		}},
	}, "console.log('draft');")

	version := Version{
		ID:          "ver-1",
		ExtensionID: extension.ID,
		Digest:      "digest-v1",
		State:       "draft",
		Components:  files.Manifest.Components,
	}
	require.NoError(t, storage.CreateVersion("org-1", extension.ID, version, files))

	draftDir := filepath.Join(rootDir, "org-1", "acme.echo", "ver-1")
	assertVersionFiles(t, draftDir, files)

	publishedVersion := version
	publishedVersion.Version = "1.2.3"
	publishedVersion.State = "published"
	require.NoError(t, storage.UpdateVersion("org-1", extension.ID, version.ID, publishedVersion, nil))

	publishedDir := filepath.Join(rootDir, "org-1", "acme.echo", "1.2.3")
	assertVersionFiles(t, publishedDir, files)
}

func TestExtractBundleFilesRequiresBundleJS(t *testing.T) {
	t.Parallel()

	bundle := newBundleBytes(t, map[string][]byte{
		"dist/manifest.json": []byte(`{"components":[]}`),
	})

	_, err := ExtractBundleFiles(bundle)
	require.EqualError(t, err, "bundle.js not found in bundle")
}

func assertVersionFiles(t *testing.T, versionDir string, files *BundleFiles) {
	t.Helper()

	manifestJSON, err := os.ReadFile(filepath.Join(versionDir, "manifest.json"))
	require.NoError(t, err)
	require.Equal(t, files.ManifestJSON, manifestJSON)

	bundleJS, err := os.ReadFile(filepath.Join(versionDir, "bundle.js"))
	require.NoError(t, err)
	require.Equal(t, files.BundleJS, bundleJS)
}

func newBundleFiles(t *testing.T, manifest Manifest, bundleJS string) *BundleFiles {
	t.Helper()

	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	files, err := ExtractBundleFiles(newBundleBytes(t, map[string][]byte{
		"dist/manifest.json": manifestJSON,
		"dist/bundle.js":     []byte(bundleJS),
	}))
	require.NoError(t, err)

	return files
}

func newBundleBytes(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		require.NoError(t, tarWriter.WriteHeader(header))
		_, err := tarWriter.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())

	return buffer.Bytes()
}
