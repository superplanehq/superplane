package extensions

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"

	storage "github.com/superplanehq/superplane/pkg/storage"
)

type Storage struct {
	underlying storage.Storage
	extensions map[string][]Extension
	versions   map[string][]Version
}

type Extension struct {
	ID          string
	Name        string
	Description string
}

type Version struct {
	ID           string
	Version      string
	ExtensionID  string
	Digest       string
	State        string
	Integrations []IntegrationManifest
	Components   []ComponentManifest
	Triggers     []TriggerManifest
}

func NewStorage(underlying storage.Storage) *Storage {
	return &Storage{
		underlying: underlying,
		extensions: make(map[string][]Extension),
		versions:   make(map[string][]Version),
	}
}

func (s *Storage) ListExtensions(organizationID string) ([]Extension, error) {
	return s.extensions[organizationID], nil
}

func (s *Storage) ListVersions(organizationID string, extensionID string) ([]Version, error) {
	_, err := s.FindExtensionById(organizationID, extensionID)
	if err != nil {
		return []Version{}, nil
	}

	return s.versions[extensionID], nil
}

func (s *Storage) CreateExtension(organizationID string, extension Extension) error {
	if _, ok := s.extensions[organizationID]; !ok {
		s.extensions[organizationID] = []Extension{}
	}

	if _, err := s.FindExtensionByName(organizationID, extension.Name); err == nil {
		return fmt.Errorf("extension %s already exists", extension.Name)
	}

	s.extensions[organizationID] = append(s.extensions[organizationID], extension)
	return nil
}

func (s *Storage) CreateVersion(organizationID string, extensionID string, version Version, files *BundleFiles) error {
	extension, err := s.FindExtensionById(organizationID, extensionID)
	if err != nil {
		return err
	}

	if files == nil {
		return errors.New("bundle files are required")
	}

	if _, ok := s.versions[extensionID]; !ok {
		s.versions[extensionID] = []Version{}
	}

	if err := s.writeVersionFiles(organizationID, extension.Name, version, files); err != nil {
		return err
	}

	s.versions[extensionID] = append(s.versions[extensionID], version)
	return nil
}

func (s *Storage) FindExtensionById(organizationID string, extensionID string) (*Extension, error) {
	for _, extension := range s.extensions[organizationID] {
		if extension.ID == extensionID {
			return &extension, nil
		}
	}

	return nil, errors.New("extension not found")
}

func (s *Storage) FindExtensionByName(organizationID string, name string) (*Extension, error) {
	for _, extension := range s.extensions[organizationID] {
		if extension.Name == name {
			return &extension, nil
		}
	}

	return nil, errors.New("extension not found")
}

func (s *Storage) FindVersionById(organizationID string, extensionID string, versionID string) (*Version, error) {
	for _, version := range s.versions[extensionID] {
		if version.ID == versionID {
			return &version, nil
		}
	}

	return nil, errors.New("version not found")
}

func (s *Storage) UpdateVersion(organizationID string, extensionID string, versionID string, updatedVersion Version, files *BundleFiles) error {
	extension, err := s.FindExtensionById(organizationID, extensionID)
	if err != nil {
		return err
	}

	currentVersion, err := s.FindVersionById(organizationID, extensionID, versionID)
	if err != nil {
		return err
	}

	if files != nil {
		if err := s.writeVersionFiles(organizationID, extension.Name, updatedVersion, files); err != nil {
			return err
		}
	}

	if files == nil && versionDirectoryName(*currentVersion) != versionDirectoryName(updatedVersion) {
		if err := s.copyVersionFiles(organizationID, extension.Name, *currentVersion, updatedVersion); err != nil {
			return err
		}
	}

	for i, version := range s.versions[extensionID] {
		if version.ID == versionID {
			s.versions[extensionID][i] = updatedVersion
			break
		}
	}

	return nil
}

func (s *Storage) ListComponents(organizationID string) []ComponentManifest {
	components := []ComponentManifest{}
	for _, extension := range s.extensions[organizationID] {
		for _, version := range s.versions[extension.ID] {
			components = append(components, version.Components...)
		}
	}

	return components
}

func (s *Storage) ListTriggers(organizationID string) []TriggerManifest {
	triggers := []TriggerManifest{}
	for _, extension := range s.extensions[organizationID] {
		for _, version := range s.versions[extension.ID] {
			triggers = append(triggers, version.Triggers...)
		}
	}

	return triggers
}

func (s *Storage) ListIntegrations(organizationID string) []IntegrationManifest {
	integrations := []IntegrationManifest{}
	for _, extension := range s.extensions[organizationID] {
		for _, version := range s.versions[extension.ID] {
			integrations = append(integrations, version.Integrations...)
		}
	}

	return integrations
}

func (s *Storage) writeVersionFiles(organizationID string, extensionName string, version Version, files *BundleFiles) error {
	manifestPath := versionFilePath(organizationID, extensionName, version, "manifest.json")
	if err := s.underlying.Write(manifestPath, bytes.NewReader(files.ManifestJSON)); err != nil {
		return fmt.Errorf("write manifest.json: %w", err)
	}

	bundlePath := versionFilePath(organizationID, extensionName, version, "bundle.js")
	if err := s.underlying.Write(bundlePath, bytes.NewReader(files.BundleJS)); err != nil {
		return fmt.Errorf("write bundle.js: %w", err)
	}

	return nil
}

func (s *Storage) copyVersionFiles(organizationID string, extensionName string, currentVersion Version, updatedVersion Version) error {
	for _, filename := range []string{"manifest.json", "bundle.js"} {
		content, err := s.readVersionFile(organizationID, extensionName, currentVersion, filename)
		if err != nil {
			return err
		}

		targetPath := versionFilePath(organizationID, extensionName, updatedVersion, filename)
		if err := s.underlying.Write(targetPath, bytes.NewReader(content)); err != nil {
			return fmt.Errorf("copy %s: %w", filename, err)
		}
	}

	return nil
}

func (s *Storage) readVersionFile(organizationID string, extensionName string, version Version, filename string) ([]byte, error) {
	reader, err := s.underlying.Read(versionFilePath(organizationID, extensionName, version, filename))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filename, err)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read %s content: %w", filename, err)
	}

	return content, nil
}

func versionFilePath(organizationID string, extensionName string, version Version, filename string) string {
	return path.Join(organizationID, extensionName, versionDirectoryName(version), filename)
}

func versionDirectoryName(version Version) string {
	if version.Version != "" {
		return version.Version
	}

	return version.ID
}
