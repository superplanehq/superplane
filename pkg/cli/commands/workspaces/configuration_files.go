package workspaces

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/superplanehq/superplane/pkg/workspaceconfig"
	"gopkg.in/yaml.v3"
)

type configurationManifestConnection struct {
	IntegrationID   *string `yaml:"integration_id"`
	IntegrationName *string `yaml:"integration_name"`
}

type configurationManifestRepository struct {
	Repository string `yaml:"repository"`
	Ref        string `yaml:"ref"`
}

type configurationManifestRepositories struct {
	Binding           *configurationManifestConnection `yaml:"binding"`
	Primary           *configurationManifestRepository `yaml:"primary"`
	BacklogRepository string                           `yaml:"backlog_repository"`
	Additional        []struct {
		Repository string `yaml:"repository"`
		Ref        string `yaml:"ref"`
		Access     string `yaml:"access"`
		Enabled    *bool  `yaml:"enabled"`
		Provider   string `yaml:"provider"`
	} `yaml:"additional"`
}

// The server validates live stage kinds, built-in aliases, and settings.
// The CLI validates bundle structure before reading references or replacing local files.
func configurationManifestReferences(content, manifestPath string) ([]string, error) {
	var manifest struct {
		APIVersion   string                             `yaml:"apiVersion"`
		Kind         string                             `yaml:"kind"`
		Name         string                             `yaml:"name"`
		Repositories *configurationManifestRepositories `yaml:"repositories"`
		Stages       []struct {
			ID          string `yaml:"id"`
			Name        string `yaml:"name"`
			Run         string `yaml:"run"`
			Automations []struct {
				ID      string `yaml:"id"`
				Name    string `yaml:"name"`
				Uses    string `yaml:"uses"`
				Binding *struct {
					IntegrationID   *string `yaml:"integration_id"`
					IntegrationName *string `yaml:"integration_name"`
					ResourceID      string  `yaml:"resource_id"`
					TeamID          string  `yaml:"team_id"`
				} `yaml:"binding"`
				With map[string]any `yaml:"with"`
			} `yaml:"automations"`
		} `yaml:"stages"`
	}
	decoder := yaml.NewDecoder(strings.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		message := err.Error()
		if strings.Contains(message, "field key not found") {
			message += "; replace stage key with id"
		}
		if strings.Contains(message, "field max_parallelism not found") {
			message += "; edit stage concurrency in workspace settings"
		}
		return nil, fmt.Errorf("invalid Factory manifest: %s", message)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid Factory manifest: expected one YAML document")
	}
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return nil, fmt.Errorf("invalid Factory manifest: %w", err)
	}
	if configurationYAMLHasAlias(&document) {
		return nil, fmt.Errorf("invalid Factory manifest: YAML aliases are not supported")
	}
	if manifest.APIVersion != "v2" || manifest.Kind != "Factory" || strings.TrimSpace(manifest.Name) == "" || len(manifest.Stages) == 0 {
		return nil, fmt.Errorf("invalid Factory manifest: expected a named v2 Factory with stages")
	}
	if repositories := manifest.Repositories; repositories != nil {
		if repositories.Binding == nil || repositories.Binding.IntegrationID == nil && repositories.Binding.IntegrationName == nil {
			return nil, fmt.Errorf("invalid Factory manifest: repositories.binding requires an integration_name or integration_id")
		}
		if err := validateConfigurationConnection(repositories.Binding.IntegrationID, repositories.Binding.IntegrationName); err != nil {
			return nil, fmt.Errorf("invalid Factory manifest: repositories.binding: %w", err)
		}
		if repositories.Primary == nil || strings.TrimSpace(repositories.Primary.Repository) == "" {
			return nil, fmt.Errorf("invalid Factory manifest: repositories.primary.repository is required")
		}
		for index, repository := range repositories.Additional {
			if strings.TrimSpace(repository.Repository) == "" || repository.Access != "read" && repository.Access != "write" {
				return nil, fmt.Errorf("invalid Factory manifest: repositories.additional[%d] requires a repository and access of read or write", index)
			}
		}
	}
	var references []string
	seenReferences := map[string]bool{}
	seenStages := map[string]bool{}
	seenAutomations := map[string]bool{}
	for _, stage := range manifest.Stages {
		if stage.ID == "" || strings.TrimSpace(stage.Name) == "" || seenStages[stage.ID] || stage.Automations == nil {
			return nil, fmt.Errorf("invalid Factory manifest: stages require unique IDs, names, and an automations list")
		}
		seenStages[stage.ID] = true

		for _, automation := range stage.Automations {
			if automation.ID == "" || seenAutomations[automation.ID] || automation.Uses == "" {
				return nil, fmt.Errorf("invalid Factory manifest: automations require unique IDs and uses")
			}
			seenAutomations[automation.ID] = true
			if binding := automation.Binding; binding != nil {
				if err := validateConfigurationConnection(binding.IntegrationID, binding.IntegrationName); err != nil {
					return nil, fmt.Errorf("invalid Factory manifest: automation %q binding: %w", automation.ID, err)
				}
			}
			if strings.HasPrefix(automation.Uses, "@superplane/") {
				continue
			}
			resolved, err := workspaceconfig.ResolveAutomationPath(manifestPath, automation.Uses)
			if err != nil {
				return nil, fmt.Errorf("unsafe custom automation path %q: %w", automation.Uses, err)
			}
			if !seenReferences[resolved] {
				seenReferences[resolved] = true
				references = append(references, resolved)
			}
		}
	}
	for _, stage := range manifest.Stages {
		if stage.Run != "" && !seenAutomations[stage.Run] {
			return nil, fmt.Errorf("invalid Factory manifest: phase run must identify a declared workspace automation")
		}
	}
	return references, nil
}

func validateConfigurationConnection(id, name *string) error {
	if id != nil && name != nil {
		return fmt.Errorf("use either integration_name or integration_id")
	}
	if id != nil && strings.TrimSpace(*id) == "" || name != nil && strings.TrimSpace(*name) == "" {
		return fmt.Errorf("use a nonempty integration_name or integration_id")
	}
	return nil
}

func configurationYAMLHasAlias(node *yaml.Node) bool {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return true
	}
	for _, child := range node.Content {
		if configurationYAMLHasAlias(child) {
			return true
		}
	}
	return false
}

func readConfigurationReference(directory *os.Root, reference string) (string, error) {
	// Root confines reads even if a path changes after these symlink checks.
	parts := strings.Split(reference, "/")
	var expected os.FileInfo
	for index := range parts {
		info, err := directory.Lstat(strings.Join(parts[:index+1], "/"))
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symbolic links are not supported")
		}
		if index == len(parts)-1 {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("the reference must be a regular file")
			}
			expected = info
		}
	}
	file, err := directory.Open(reference)
	if err != nil {
		return "", err
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !actual.Mode().IsRegular() || !os.SameFile(expected, actual) {
		return "", fmt.Errorf("the input changed while opening; use a regular file without symbolic links")
	}
	content, err := io.ReadAll(file)
	return string(content), err
}

// Walk from the filesystem root so selected directories cannot hide symlinks.
func openConfigurationInputDirectory(directory string) (*os.Root, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	rootPath := filepath.VolumeName(absolute) + string(filepath.Separator)
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(absolute, rootPath), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		info, err := root.Lstat(part)
		if err != nil {
			root.Close()
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, fmt.Errorf("%s must be a directory without symbolic links", part)
		}
		next, err := root.OpenRoot(part)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = next
	}
	return root, nil
}
