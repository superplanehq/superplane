package extensions

import (
	"errors"
	"fmt"
)

// Temporary storage for extensions.
// This should replaced with a proper storage layer (database + bucket)
type ExtensionStorage struct {

	//
	// A map of extensions per organization.
	//
	extensions map[string][]Extension

	//
	// A map of versions per extension.
	//
	versions map[string][]Version
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
	Integrations []ManifestIntegration
	Components   []ManifestComponent
	Triggers     []ManifestTrigger
}

func NewExtensionStorage() *ExtensionStorage {
	return &ExtensionStorage{
		extensions: make(map[string][]Extension),
		versions:   make(map[string][]Version),
	}
}

func (s *ExtensionStorage) ListExtensions(organizationID string) ([]Extension, error) {
	return s.extensions[organizationID], nil
}

func (s *ExtensionStorage) ListVersions(organizationID string, extensionID string) ([]Version, error) {
	_, err := s.FindExtensionById(organizationID, extensionID)
	if err != nil {
		return []Version{}, nil
	}

	return s.versions[extensionID], nil
}

func (s *ExtensionStorage) CreateExtension(organizationID string, extension Extension) error {
	if _, ok := s.extensions[organizationID]; !ok {
		s.extensions[organizationID] = []Extension{}
	}

	if _, err := s.FindExtensionByName(organizationID, extension.Name); err == nil {
		return fmt.Errorf("extension %s already exists", extension.Name)
	}

	s.extensions[organizationID] = append(s.extensions[organizationID], extension)
	return nil
}

func (s *ExtensionStorage) CreateVersion(organizationID string, extensionID string, version Version) error {
	_, err := s.FindExtensionById(organizationID, extensionID)
	if err != nil {
		return err
	}

	if _, ok := s.versions[extensionID]; !ok {
		s.versions[extensionID] = []Version{}
	}

	s.versions[extensionID] = append(s.versions[extensionID], version)
	return nil
}

func (s *ExtensionStorage) FindExtensionById(organizationID string, extensionID string) (*Extension, error) {
	for _, extension := range s.extensions[organizationID] {
		if extension.ID == extensionID {
			return &extension, nil
		}
	}
	return nil, errors.New("extension not found")
}

func (s *ExtensionStorage) FindExtensionByName(organizationID string, name string) (*Extension, error) {
	for _, extension := range s.extensions[organizationID] {
		if extension.Name == name {
			return &extension, nil
		}
	}
	return nil, errors.New("extension not found")
}

func (s *ExtensionStorage) FindVersionById(organizationID string, extensionID string, versionID string) (*Version, error) {
	for _, version := range s.versions[extensionID] {
		if version.ID == versionID {
			return &version, nil
		}
	}
	return nil, errors.New("version not found")
}

func (s *ExtensionStorage) UpdateVersion(organizationID string, extensionID string, versionID string, updatedVersion Version) error {
	_, err := s.FindVersionById(organizationID, extensionID, versionID)
	if err != nil {
		return err
	}

	for i, v := range s.versions[extensionID] {
		if v.ID == versionID {
			s.versions[extensionID][i] = updatedVersion
			break
		}
	}

	return nil
}
