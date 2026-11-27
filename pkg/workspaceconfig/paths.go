// Package workspaceconfig defines paths shared by workspace bundle loaders.
package workspaceconfig

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

const (
	ManifestPath = ".superplane/main.yml"
	Coverage     = "Workspace name, repositories, stages, and automations"
)

// NormalizePath returns a YAML file path relative to the bundle root.
func NormalizePath(name string) (string, error) {
	if name == "" || path.IsAbs(name) || strings.ContainsAny(name, "\\:\x00") {
		return "", fmt.Errorf("use a relative YAML path inside the bundle root")
	}
	normalized := path.Clean(name)
	if !fs.ValidPath(normalized) || normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("keep the file inside the bundle root")
	}
	if ext := path.Ext(normalized); ext != ".yaml" && ext != ".yml" {
		return "", fmt.Errorf("use a .yaml or .yml file")
	}
	return normalized, nil
}

// ResolveAutomationPath resolves a reference relative to the selected manifest.
func ResolveAutomationPath(manifestPath, reference string) (string, error) {
	manifest, err := NormalizePath(manifestPath)
	if err != nil {
		return "", err
	}
	if reference == "" || path.IsAbs(reference) || strings.ContainsAny(reference, "\\:\x00") {
		return "", fmt.Errorf("use a relative YAML path inside the bundle root")
	}
	resolved, err := NormalizePath(path.Join(path.Dir(manifest), reference))
	if err != nil {
		return "", err
	}
	if resolved == manifest {
		return "", fmt.Errorf("an automation cannot reference the Factory manifest")
	}
	return resolved, nil
}
