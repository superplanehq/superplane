package extensions

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"path"

	"github.com/superplanehq/superplane/pkg/core"
)

type Manifest struct {
	Integrations []ManifestIntegration `json:"integrations"`
	Components   []ManifestComponent   `json:"components"`
	Triggers     []ManifestTrigger     `json:"triggers"`
}

type ManifestIntegration struct {
	Name          string   `json:"name"`
	Label         string   `json:"label"`
	Icon          string   `json:"icon"`
	Description   string   `json:"description"`
	Instructions  string   `json:"instructions,omitempty"`
	ResourceTypes []string `json:"resourceTypes"`

	// TODO: should use []configuration.Field
	Configuration []map[string]any `json:"configuration"`

	// TODO: should use []core.Action
	Actions []ManifestAction `json:"actions"`
}

type ManifestComponent struct {
	Name           string               `json:"name"`
	Integration    string               `json:"integration,omitempty"`
	Label          string               `json:"label"`
	Description    string               `json:"description"`
	Icon           string               `json:"icon"`
	Color          string               `json:"color"`
	OutputChannels []core.OutputChannel `json:"outputChannels"`

	// TODO: should use []configuration.Field
	Configuration []map[string]any `json:"configuration"`

	// TODO: should use []core.Action
	Actions []ManifestAction `json:"actions"`
}

type ManifestTrigger struct {
	Name        string `json:"name"`
	Integration string `json:"integration,omitempty"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`

	// TODO: should use []configuration.Field
	Configuration []map[string]any `json:"configuration"`

	// TODO: should use []core.Action
	Actions []ManifestAction `json:"actions"`
}

type ManifestAction struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Parameters  []map[string]any `json:"parameters"`
}

func extractManifestFromBundle(bundle []byte) (*Manifest, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(bundle))
	if err != nil {
		return nil, fmt.Errorf("open bundle gzip: %w", err)
	}
	defer gzipReader.Close()

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
		if path.Base(header.Name) != "manifest.json" {
			continue
		}

		manifestJSON, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, fmt.Errorf("read manifest.json: %w", err)
		}

		var manifest Manifest
		if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
			return nil, fmt.Errorf("parse manifest.json: %w", err)
		}

		return &manifest, nil
	}

	return nil, fmt.Errorf("manifest.json not found in bundle")
}
