package extensions

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"path"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type Manifest struct {
	Integrations []IntegrationManifest `json:"integrations"`
	Components   []ComponentManifest   `json:"components"`
	Triggers     []TriggerManifest     `json:"triggers"`
}

type IntegrationManifest struct {
	Name          string                `json:"name"`
	Label         string                `json:"label"`
	Icon          string                `json:"icon"`
	Description   string                `json:"description"`
	Instructions  string                `json:"instructions,omitempty"`
	ResourceTypes []string              `json:"resourceTypes"`
	Configuration []configuration.Field `json:"configuration"`
	Actions       []core.Action         `json:"actions"`
}

type ComponentManifest struct {
	Name           string                `json:"name"`
	Integration    string                `json:"integration,omitempty"`
	Label          string                `json:"label"`
	Description    string                `json:"description"`
	Icon           string                `json:"icon"`
	Color          string                `json:"color"`
	OutputChannels []core.OutputChannel  `json:"outputChannels"`
	Configuration  []configuration.Field `json:"configuration"`
	Actions        []core.Action         `json:"actions"`

	//
	// TODO: Not sure about this.
	// I need it to be able to know which extension version
	// a component belongs to, so I can create the RunnerJob in ExtensionComponent.Execute()
	// ExtensionID and VersionID are not part of the manifest, but are dynamically set by LoadManifestInTransaction().
	//
	ExtensionID string `json:"-"`
	VersionID   string `json:"-"`
}

type TriggerManifest struct {
	Name          string                `json:"name"`
	Integration   string                `json:"integration,omitempty"`
	Label         string                `json:"label"`
	Description   string                `json:"description"`
	Icon          string                `json:"icon"`
	Color         string                `json:"color"`
	Configuration []configuration.Field `json:"configuration"`
	Actions       []core.Action         `json:"actions"`
}

type BundleFiles struct {
	Manifest     *Manifest
	ManifestJSON []byte
	BundleJS     []byte
}

func ExtractManifestFromBundle(bundle []byte) (*Manifest, error) {
	files, err := ExtractBundleFiles(bundle)
	if err != nil {
		return nil, err
	}

	return files.Manifest, nil
}

func ExtractBundleFiles(bundle []byte) (*BundleFiles, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(bundle))
	if err != nil {
		return nil, fmt.Errorf("open bundle gzip: %w", err)
	}
	defer gzipReader.Close()

	files := &BundleFiles{}
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read bundle tar: %w", err)
		}
		if header.FileInfo().IsDir() {
			continue
		}

		fileData, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, fmt.Errorf("read bundle file %s: %w", path.Base(header.Name), err)
		}

		switch path.Base(header.Name) {
		case "manifest.json":
			files.ManifestJSON = fileData
		case "bundle.js":
			files.BundleJS = fileData
		}
	}

	if len(files.ManifestJSON) == 0 {
		return nil, fmt.Errorf("manifest.json not found in bundle")
	}
	if len(files.BundleJS) == 0 {
		return nil, fmt.Errorf("bundle.js not found in bundle")
	}

	var manifest Manifest
	if err := json.Unmarshal(files.ManifestJSON, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest.json: %w", err)
	}

	files.Manifest = &manifest
	return files, nil
}
