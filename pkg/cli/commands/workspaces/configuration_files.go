package workspaces

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// The server validates live stage kinds, built-in aliases, and settings.
// The CLI validates bundle structure before reading references or replacing local files.
func configurationManifestReferences(content string) ([]string, error) {
	var manifest struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Name       string `yaml:"name"`
		Stages     []struct {
			Key            string `yaml:"key"`
			Name           string `yaml:"name"`
			Run            string `yaml:"run"`
			MaxParallelism *int   `yaml:"max_parallelism"`
			Automations    []struct {
				ID      string `yaml:"id"`
				Name    string `yaml:"name"`
				Uses    string `yaml:"uses"`
				Binding *struct {
					IntegrationID string `yaml:"integration_id"`
					ResourceID    string `yaml:"resource_id"`
					TeamID        string `yaml:"team_id"`
				} `yaml:"binding"`
				With map[string]any `yaml:"with"`
			} `yaml:"automations"`
		} `yaml:"stages"`
	}
	decoder := yaml.NewDecoder(strings.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid Factory manifest: %w", err)
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
	var references []string
	seenReferences := map[string]bool{}
	seenStages := map[string]bool{}
	for _, stage := range manifest.Stages {
		if stage.Key == "" || strings.TrimSpace(stage.Name) == "" || seenStages[stage.Key] || stage.Automations == nil {
			return nil, fmt.Errorf("invalid Factory manifest: stages require unique keys, names, and an automations list")
		}
		seenStages[stage.Key] = true
		if stage.MaxParallelism != nil && *stage.MaxParallelism < 1 {
			return nil, fmt.Errorf("invalid Factory manifest: max_parallelism must be positive")
		}
		seenAutomations := map[string]bool{}
		for _, automation := range stage.Automations {
			if automation.ID == "" || seenAutomations[automation.ID] || automation.Uses == "" {
				return nil, fmt.Errorf("invalid Factory manifest: automations require unique IDs and uses")
			}
			seenAutomations[automation.ID] = true
			if strings.HasPrefix(automation.Uses, "@superplane/") {
				continue
			}
			if !configurationReferenceAllowed(automation.Uses) {
				return nil, fmt.Errorf("unsafe custom automation path %q: use a relative YAML path inside the manifest directory", automation.Uses)
			}
			if !seenReferences[automation.Uses] {
				seenReferences[automation.Uses] = true
				references = append(references, automation.Uses)
			}
		}
		if stage.Run != "" && !seenAutomations[stage.Run] {
			return nil, fmt.Errorf("invalid Factory manifest: phase run must identify one of its automations")
		}
	}
	return references, nil
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

func configurationReferenceAllowed(reference string) bool {
	return fs.ValidPath(reference) && reference != "main.yml" &&
		!strings.ContainsAny(reference, "\\:") &&
		(path.Ext(reference) == ".yaml" || path.Ext(reference) == ".yml")
}

func readConfigurationReference(directory *os.Root, reference string) (string, error) {
	// Root confines reads even if a path changes after these symlink checks.
	parts := strings.Split(reference, "/")
	for index := range parts {
		info, err := directory.Lstat(strings.Join(parts[:index+1], "/"))
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symbolic links are not supported")
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return "", fmt.Errorf("the reference must be a regular file")
		}
	}
	file, err := directory.Open(reference)
	if err != nil {
		return "", err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	return string(content), err
}
