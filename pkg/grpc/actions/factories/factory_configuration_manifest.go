package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workspaceconfig"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const (
	factoryConfigurationManifestPath = workspaceconfig.ManifestPath
	factoryConfigurationCoverage     = workspaceconfig.Coverage
)

type factoryConfigurationManifest struct {
	ManifestPath string `yaml:"-" json:"-"`
	root         *yaml.Node

	APIVersion   string                               `yaml:"apiVersion"`
	Kind         string                               `yaml:"kind"`
	Name         string                               `yaml:"name"`
	Repositories *factoryConfigurationRepositories    `yaml:"repositories,omitempty"`
	Stages       []factoryConfigurationStage          `yaml:"stages"`
	Sources      map[string]string                    `yaml:"-" json:"-"`
	Definitions  map[string]automationspec.Definition `yaml:"-" json:"definitions,omitempty"`
}

type factoryConfigurationStage struct {
	retained    map[uuid.UUID]bool               `yaml:"-" json:"-"`
	ID          string                           `yaml:"id"`
	Key         string                           `yaml:"-" json:"-"`
	Name        string                           `yaml:"name"`
	Run         string                           `yaml:"run,omitempty"`
	Automations []factoryConfigurationAutomation `yaml:"automations"`
}

type factoryConfigurationAutomation struct {
	placement         string                                `yaml:"-" json:"-"`
	current           *factoryConfigurationAutomationTarget `yaml:"-" json:"-"`
	ID                string                                `yaml:"id"`
	Name              string                                `yaml:"name,omitempty"`
	Uses              string                                `yaml:"uses"`
	Binding           *factoryConfigurationBinding          `yaml:"binding,omitempty"`
	With              map[string]any                        `yaml:"with,omitempty"`
	prepared          *preparedFactoryConfigurationBuiltIn
	dependency        any
	dependencyPresent bool
}

type factoryConfigurationBinding struct {
	IntegrationName string `yaml:"integration_name,omitempty" json:"integration_name,omitempty"`
	IntegrationID   string `yaml:"integration_id,omitempty" json:"integration_id,omitempty"`
	ResourceID      string `yaml:"resource_id,omitempty" json:"resource_id,omitempty"`
	TeamID          string `yaml:"team_id,omitempty" json:"team_id,omitempty"`
}

func configurationDiagnostic(field, message string) *pb.FactoryConfigurationDiagnostic {
	return &pb.FactoryConfigurationDiagnostic{Field: field, Message: message}
}

func parseFactoryConfigurationBundle(bundle *pb.FactoryConfigurationBundle) (manifestResult *factoryConfigurationManifest, diagnosticResult *pb.FactoryConfigurationDiagnostic) {
	manifestPath, err := workspaceconfig.NormalizePath(bundle.GetManifestPath())
	if err != nil {
		diagnostic := configurationDiagnostic("manifest_path", err.Error())
		diagnostic.Path = bundle.GetManifestPath()
		if diagnostic.Path == "" {
			diagnostic.Path = factoryConfigurationManifestPath
		}
		return nil, diagnostic
	}
	var rootNode *yaml.Node
	defer func() {
		if diagnosticResult != nil && diagnosticResult.Path == "" {
			diagnosticResult.Path = manifestPath
			addConfigurationDiagnosticLocation(diagnosticResult, rootNode)
		}
	}()
	files := map[string]string{}
	for _, file := range bundle.GetFiles() {
		name, err := workspaceconfig.NormalizePath(file.GetPath())
		if err != nil {
			return nil, configurationDiagnostic("files", err.Error())
		}
		if _, exists := files[name]; exists {
			return nil, configurationDiagnostic("files", "A file path occurs more than once.")
		}
		files[name] = file.GetContent()
	}
	content, exists := files[manifestPath]
	if !exists {
		return nil, configurationDiagnostic("files", "The bundle must contain the selected manifest.")
	}

	decoder := yaml.NewDecoder(strings.NewReader(content))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, configurationDiagnostic("$", "Invalid YAML: "+err.Error())
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, configurationDiagnostic("$", "The manifest must contain one YAML document.")
	}
	if len(document.Content) != 1 {
		return nil, configurationDiagnostic("$", "The manifest must contain a Factory document.")
	}
	rootNode = document.Content[0]
	if diagnostic := validateConfigurationYAML(document.Content[0], ""); diagnostic != nil {
		return nil, diagnostic
	}
	root, diagnostic := configurationFields(document.Content[0], "", []string{"apiVersion", "kind", "name", "repositories", "stages"}, []string{"apiVersion", "kind", "name", "stages"})
	if diagnostic != nil {
		return nil, diagnostic
	}
	for _, field := range []string{"apiVersion", "kind", "name"} {
		if diagnostic := configurationStringNode(root[field], field); diagnostic != nil {
			return nil, diagnostic
		}
	}
	if node := root["repositories"]; node != nil {
		if diagnostic := validateConfigurationRepositoriesYAML(node); diagnostic != nil {
			return nil, diagnostic
		}
	}
	if root["stages"].Kind != yaml.SequenceNode || len(root["stages"].Content) == 0 {
		return nil, configurationDiagnostic("stages", "Use a nonempty stage list.")
	}
	for i, stage := range root["stages"].Content {
		stagePath := fmt.Sprintf("stages[%d]", i)
		fields, diagnostic := configurationFields(stage, stagePath, []string{"id", "name", "run", "automations"}, []string{"id", "name", "automations"})
		if diagnostic != nil {
			return nil, diagnostic
		}
		for _, field := range []string{"id", "name", "run"} {
			if fields[field] != nil {
				if diagnostic := configurationStringNode(fields[field], configurationField(stagePath, field)); diagnostic != nil {
					return nil, diagnostic
				}
			}
		}
		if fields["automations"].Kind != yaml.SequenceNode {
			return nil, configurationDiagnostic(stagePath+".automations", "Use an automation list, including an empty list for an empty stage.")
		}
		for j, automation := range fields["automations"].Content {
			automationPath := fmt.Sprintf("%s.automations[%d]", stagePath, j)
			fields, diagnostic := configurationFields(automation, automationPath, []string{"id", "name", "uses", "binding", "with"}, []string{"id", "uses"})
			if diagnostic != nil {
				return nil, diagnostic
			}
			for _, field := range []string{"id", "name", "uses"} {
				if fields[field] == nil {
					continue
				}
				if diagnostic := configurationStringNode(fields[field], automationPath+"."+field); diagnostic != nil {
					return nil, diagnostic
				}
			}
			if node := fields["with"]; node != nil && node.Kind != yaml.MappingNode {
				return nil, configurationDiagnostic(automationPath+".with", "Use a mapping of supported settings.")
			}
			if node := fields["binding"]; node != nil {
				if _, diagnostic := validateConfigurationBindingYAML(node, automationPath+".binding", []string{"integration_id", "integration_name", "resource_id", "team_id"}); diagnostic != nil {
					return nil, diagnostic
				}
			}
		}
	}
	var manifest factoryConfigurationManifest
	if err := document.Decode(&manifest); err != nil {
		return nil, configurationDiagnostic("$", "Invalid Factory manifest: "+err.Error())
	}
	for i := range manifest.Stages {
		for j := range manifest.Stages[i].Automations {
			automation := &manifest.Stages[i].Automations[j]
			if automation.Binding != nil && *automation.Binding == (factoryConfigurationBinding{}) {
				automation.Binding = nil
			}
		}
	}
	manifest.ManifestPath = manifestPath
	manifest.root = document.Content[0]
	manifest.Sources = map[string]string{}
	manifest.Definitions = map[string]automationspec.Definition{}
	referenced := map[string]bool{manifestPath: true}
	for i, stage := range manifest.Stages {
		for j, automation := range stage.Automations {
			if strings.HasPrefix(automation.Uses, "@superplane/") {
				continue
			}
			filePath, err := workspaceconfig.ResolveAutomationPath(manifestPath, automation.Uses)
			if err != nil {
				return nil, configurationDiagnostic(fmt.Sprintf("stages[%d].automations[%d].uses", i, j), err.Error())
			}

			source, exists := files[filePath]
			if !exists {
				return nil, configurationDiagnostic(fmt.Sprintf("stages[%d].automations[%d].uses", i, j), "The referenced automation file is missing from the bundle.")
			}
			parsed, err := automationspec.Parse(source)
			if err != nil {
				return nil, configurationSourceDiagnostic(filePath, err)
			}
			referenced[filePath] = true
			manifest.Sources[filePath] = source
			manifest.Definitions[filePath] = parsed.Definition
			if manifest.Stages[i].Automations[j].Name == "" {
				manifest.Stages[i].Automations[j].Name = parsed.Name
			}
		}
	}
	if len(referenced) != len(files) {
		return nil, configurationDiagnostic("files", "The bundle contains an unreferenced file.")
	}
	return &manifest, nil
}

func validateConfigurationYAML(node *yaml.Node, field string) *pb.FactoryConfigurationDiagnostic {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return configurationDiagnostic(configurationField(field, ""), "YAML aliases and anchors are not supported.")
	}
	if node.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			child := configurationField(field, key.Value)
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return configurationDiagnostic(child, "Use string mapping keys.")
			}
			if seen[key.Value] {
				return configurationDiagnostic(child, "This field occurs more than once.")
			}
			seen[key.Value] = true
			if diagnostic := validateConfigurationYAML(value, child); diagnostic != nil {
				return diagnostic
			}
		}
	}
	if node.Kind == yaml.SequenceNode {
		for i, child := range node.Content {
			if diagnostic := validateConfigurationYAML(child, fmt.Sprintf("%s[%d]", field, i)); diagnostic != nil {
				return diagnostic
			}
		}
	}
	return nil
}

func configurationFields(node *yaml.Node, field string, allowed, required []string) (map[string]*yaml.Node, *pb.FactoryConfigurationDiagnostic) {
	if node.Kind != yaml.MappingNode {
		return nil, configurationDiagnostic(configurationField(field, ""), "Use an explicit mapping.")
	}
	values := map[string]*yaml.Node{}
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if !slices.Contains(allowed, key.Value) {
			return nil, configurationDiagnostic(configurationField(field, key.Value), configurationRemovedFieldMessage(key.Value))
		}
		values[key.Value] = value
	}
	for _, key := range required {
		if values[key] == nil {
			return nil, configurationDiagnostic(configurationField(field, key), "This field is required.")
		}
	}
	return values, nil
}

func configurationStringNode(node *yaml.Node, field string) *pb.FactoryConfigurationDiagnostic {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return configurationDiagnostic(field, "Use an explicit string value.")
	}
	return nil
}

func configurationField(field, child string) string {
	if field == "" {
		if child == "" {
			return "$"
		}
		return child
	}
	if child == "" {
		return field
	}
	return field + "." + child
}

func configurationSourceDiagnostic(filePath string, err error) *pb.FactoryConfigurationDiagnostic {
	var diagnostic *automationspec.Diagnostic
	if errors.As(err, &diagnostic) {
		return &pb.FactoryConfigurationDiagnostic{Path: filePath, Field: diagnostic.Path, Message: fmt.Sprintf("%s (line %d, column %d)", diagnostic.Message, diagnostic.Line, diagnostic.Column)}
	}
	return &pb.FactoryConfigurationDiagnostic{Path: filePath, Field: "$", Message: err.Error()}
}

func exportFactoryConfigurationManifest(target *factoryConfigurationTarget) factoryConfigurationManifest {
	manifest := factoryConfigurationManifest{ManifestPath: factoryConfigurationManifestPath, APIVersion: "v2", Kind: "Factory", Name: target.factory.Name, Repositories: target.repositories, Stages: []factoryConfigurationStage{}, Sources: map[string]string{}, Definitions: map[string]automationspec.Definition{}}
	for _, stage := range target.stages {
		entry := factoryConfigurationStage{ID: stage.stage.ConfigurationID, Key: stage.key, Name: stage.stage.Name, Run: stage.run, Automations: []factoryConfigurationAutomation{}}
		for _, automation := range stage.automations {
			entry.Automations = append(entry.Automations, factoryConfigurationAutomation{ID: automation.id, Name: automation.canvas.Name, Uses: automation.uses, Binding: automation.exportedBinding, With: factoryConfigurationOverrides(automation.settings, automation.defaultSettings)})
			if automation.templateID == "" {
				filePath := path.Join(path.Dir(factoryConfigurationManifestPath), automation.uses)
				manifest.Sources[filePath] = automation.source
				parsed, err := automationspec.Parse(automation.source)
				if err == nil {
					manifest.Definitions[filePath] = parsed.Definition
				}
			}
		}
		manifest.Stages = append(manifest.Stages, entry)
	}
	return manifest
}

func validateFactoryConfigurationManifest(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, manifest *factoryConfigurationManifest, target *factoryConfigurationTarget) (diagnosticResult *pb.FactoryConfigurationDiagnostic, resultError error) {
	defer func() {
		if diagnosticResult != nil && diagnosticResult.Path == "" {
			diagnosticResult.Path = manifest.ManifestPath
			addConfigurationDiagnosticLocation(diagnosticResult, manifest.root)
		}
	}()
	if strings.TrimSpace(manifest.Name) == "" {
		return configurationDiagnostic("name", models.ErrFactoryNameRequired.Error()), nil
	}
	manifest.Name = strings.TrimSpace(manifest.Name)

	for _, check := range []struct{ field, value, want string }{{"apiVersion", manifest.APIVersion, "v2"}, {"kind", manifest.Kind, "Factory"}} {
		if check.value != check.want {
			return configurationDiagnostic(check.field, fmt.Sprintf("This field must match the selected live resource: %q.", check.want)), nil
		}
	}
	if len(manifest.Stages) != len(target.stages) {
		return configurationDiagnostic("stages", "Include every existing stage. This POC does not add or remove stages."), nil
	}
	if diagnostic, err := validateFactoryConfigurationRepositories(tx, target, manifest.Repositories); diagnostic != nil || err != nil {
		return diagnostic, err
	}
	tx = withFactoryAutomationReadInputs(tx, target.reads)
	resolveFactoryConfigurationIdentities(target, manifest)
	validation, err := loadFactoryConfigurationValidationInputs(tx, target, manifest)
	if err != nil {
		return nil, err
	}
	target.validation = validation
	byKey := map[string]*factoryConfigurationStageTarget{}
	for _, stage := range target.stages {
		byKey[stage.stage.ConfigurationID] = stage
	}
	seen := map[string]bool{}
	names := map[string]bool{}
	desiredByCanvas := map[string]string{}

	newNames := map[string]bool{}
	desiredNames := map[string]string{}
	sourceNames := map[string]string{}
	ids := map[string]bool{}
	for i := range manifest.Stages {
		stage := &manifest.Stages[i]
		stagePath := fmt.Sprintf("stages[%d]", i)
		live := byKey[stage.ID]
		if live != nil {
			stage.Key = live.key
		}
		if live == nil || seen[stage.ID] {
			return configurationDiagnostic(stagePath+".id", "Use each existing stage ID exactly once."), nil
		}
		seen[stage.ID] = true
		if strings.TrimSpace(stage.Name) == "" || strings.TrimSpace(stage.Name) != stage.Name || utf8.RuneCountInString(stage.Name) > factoryLineStageNameMaxLength || names[strings.ToLower(stage.Name)] {
			return configurationDiagnostic(stagePath+".name", "Use a unique stage name of 1 through 64 characters without surrounding spaces."), nil
		}
		names[strings.ToLower(stage.Name)] = true
		if stage.Run != live.run {
			return configurationDiagnostic(stagePath+".run", "Keep the existing phase runner binding."), nil
		}

		automationNames, retained := map[string]bool{}, map[string]bool{}

		for j := range stage.Automations {
			automation := &stage.Automations[j]
			automationPath := fmt.Sprintf("%s.automations[%d]", stagePath, j)
			if strings.TrimSpace(automation.ID) == "" || automation.ID != strings.TrimSpace(automation.ID) || ids[automation.ID] {
				return configurationDiagnostic(automationPath+".id", "Use a unique nonempty automation ID without surrounding spaces."), nil
			}
			ids[automation.ID] = true
			if strings.TrimSpace(automation.Name) == "" || automation.Name != strings.TrimSpace(automation.Name) || automationNames[automation.Name] {
				return configurationDiagnostic(automationPath+".name", "Use each automation name once per stage without surrounding spaces."), nil
			}
			automationNames[automation.Name] = true
			current := automation.current
			identity := "new:" + stage.Key + "." + automation.ID
			if current != nil {
				identity = current.canvas.ID.String()
			}
			if owner, exists := desiredNames[automation.Name]; exists && owner != identity {
				return configurationDiagnostic(automationPath+".name", "Use a unique automation name in this workspace."), nil
			}
			desiredNames[automation.Name] = identity
			if current == nil {
				for _, canvas := range target.reads.canvases {
					if canvas.ConfigurationID == automation.ID {
						return configurationDiagnostic(automationPath+".id", "This automation ID already belongs to another workspace automation."), nil
					}
				}
				for _, existingStage := range target.stages {
					for _, existing := range existingStage.automations {
						if existing.canvas.Name == automation.Name && !target.removed[existing.canvas.ID] {
							return configurationDiagnostic(automationPath+".name", "Keep existing automations in their current stages with their existing identities."), nil
						}
					}
				}
				if !models.ValidCanvasColumnKey(configurationStagePlacement(live)) {
					return configurationDiagnostic(automationPath, "New automations cannot attach to this phase. Use an existing board stage."), nil
				}
				if !strings.HasPrefix(automation.Uses, "@superplane/") {
					sourceID := configurationAutomationSourceID(target.factory.ID, configurationStagePlacement(live), automation.ID)
					authored := validation.sources[sourceID]
					if authored != nil {
						canvasID, err := uuid.Parse(authored.ArtifactID)
						if err != nil {
							return nil, err
						}
						if validation.existingArtifacts[canvasID] || authored.FactoryID != target.factory.ID {
							return configurationDiagnostic(automationPath+".id", "Use an automation ID that is not already assigned in this workspace."), nil
						}
					}
				}
				if newNames[automation.Name] {
					return configurationDiagnostic(automationPath+".name", "Use a unique name for each new automation."), nil
				}
				newNames[automation.Name] = true
			} else {
				phaseRunner := slices.ContainsFunc(target.stages, func(stage *factoryConfigurationStageTarget) bool {
					return stage.runner == current
				})
				if phaseRunner && live.runner != current {
					return configurationDiagnostic(automationPath+".id", "Declare a phase runner in a phase that uses it."), nil
				}
				identity := current.canvas.ID.String()
				if retained[identity] {
					return configurationDiagnostic(automationPath+".id", "Include each automation only once per stage."), nil
				}
				retained[identity] = true
				if automation.Name != current.canvas.Name {
					existing, err := target.findCanvasByName(tx, automation.Name)
					if err != nil {
						return nil, err
					}
					if existing != nil && existing.ID != current.canvas.ID {
						return configurationDiagnostic(automationPath+".name", "Use a unique automation name in this workspace."), nil
					}
				}
				if (current.templateID != "" || strings.HasPrefix(automation.Uses, "@superplane/")) && automation.Uses != current.uses {
					return configurationDiagnostic(automationPath+".uses", "Keep the existing automation binding."), nil
				}
			}
			if strings.HasPrefix(automation.Uses, "@superplane/") {
				if section, ok := automation.With["template"].(map[string]any); ok {
					automation.dependency, automation.dependencyPresent = section["merge_confidence"]
				}
				prepared, err := prepareFactoryConfigurationBuiltIn(ctx, tx, deps, target, live, *automation)
				if err != nil {
					var setting *factoryConfigurationSettingError
					if errors.As(err, &setting) {
						return configurationDiagnostic(automationPath+"."+setting.Field, setting.Message), nil
					}
					return nil, err
				}
				automation.prepared = prepared
				automation.With = prepared.Settings

			} else {
				if automation.Binding != nil {
					return configurationDiagnostic(automationPath+".binding", "Custom automation bindings belong in the referenced Automation file."), nil
				}
				if len(automation.With) != 0 {
					return configurationDiagnostic(automationPath+".with", "Custom automation settings belong in the referenced Automation file."), nil
				}
				filePath := manifest.automationPath(automation.Uses)
				if previous, exists := sourceNames[filePath]; exists && previous != automation.Name {
					return configurationDiagnostic(automationPath+".uses", "Use separate YAML files for automations with different names."), nil
				}
				sourceNames[filePath] = automation.Name
				source, err := automationspec.ReplaceName(manifest.Sources[filePath], automation.Name)
				if err != nil {
					return configurationSourceDiagnostic(filePath, err), nil
				}
				manifest.Sources[filePath] = source
				placement := configurationStagePlacement(live)
				if current != nil {
					placement = automation.placement
				}
				document, err := validateFactoryConfigurationCustom(ctx, tx, deps, target, current, placement, manifest.Sources[filePath])
				if err != nil {
					var diagnostic *automationspec.Diagnostic
					if errors.As(err, &diagnostic) {
						return configurationSourceDiagnostic(filePath, err), nil
					}
					return nil, err
				}
				manifest.Definitions[filePath] = document.Definition
			}
			if current == nil {
				continue
			}
			encoded, _ := json.Marshal(struct {
				Name       string
				Settings   map[string]any
				Binding    *factoryConfigurationBinding
				Definition automationspec.Definition
			}{automation.Name, automation.With, automation.Binding, manifest.Definitions[manifest.automationPath(automation.Uses)]})
			if previous, exists := desiredByCanvas[identity]; exists && previous != string(encoded) {
				return configurationDiagnostic(automationPath, "This shared automation must have the same configuration in every stage."), nil
			}
			desiredByCanvas[identity] = string(encoded)
		}
	}
	for i, stage := range manifest.Stages {
		if stage.Run != "" && !ids[stage.Run] {
			return configurationDiagnostic(fmt.Sprintf("stages[%d].run", i), "Include the automation bound to this phase runner."), nil
		}
	}
	if diagnostic := resolveFactoryConfigurationDependencies(manifest, target); diagnostic != nil {
		return diagnostic, nil
	}
	checkedRemovals := map[string]bool{}
	for i, stage := range manifest.Stages {
		for _, current := range byKey[stage.ID].automations {
			identity := current.canvas.ID.String()
			if stage.retained[current.canvas.ID] {
				continue
			}
			field := fmt.Sprintf("stages[%d].automations", i)
			if _, retained := desiredByCanvas[identity]; retained {
				continue
			}
			if checkedRemovals[identity] {
				continue
			}
			checkedRemovals[identity] = true
			if err := validateFactoryConfigurationAutomationRemoval(target, current); err != nil {
				return configurationDiagnostic(field, fmt.Sprintf("Cannot remove automation %q: %s", current.canvas.Name, err)), nil
			}
		}
	}
	if byKey[manifest.Stages[0].ID].stage.Kind != models.FactoryLineStageBacklog {
		return configurationDiagnostic("stages[0]", "Backlog must remain first."), nil
	}
	if byKey[manifest.Stages[len(manifest.Stages)-1].ID].stage.Kind != models.FactoryLineStageDone {
		return configurationDiagnostic("stages", "Done must remain last."), nil
	}
	return nil, nil
}

func configurationRemovedFieldMessage(field string) string {
	switch field {
	case "key":
		return "Use id instead of key."
	case "max_parallelism":
		return "Set concurrency in workspace settings. max_parallelism is not supported in YAML."
	default:
		return "This field is not supported."
	}
}

func addConfigurationDiagnosticLocation(diagnostic *pb.FactoryConfigurationDiagnostic, root *yaml.Node) {
	if root == nil {
		return
	}
	segments := strings.FieldsFunc(diagnostic.Field, func(r rune) bool { return r == '.' || r == '[' || r == ']' })
	node := root
	for _, segment := range segments {
		var next *yaml.Node
		if node.Kind == yaml.MappingNode {
			for i := 0; i < len(node.Content); i += 2 {
				if node.Content[i].Value == segment {
					next = node.Content[i+1]
					break
				}
			}
		}
		if node.Kind == yaml.SequenceNode {
			for i, child := range node.Content {
				if fmt.Sprint(i) == segment {
					next = child
					break
				}
			}
		}
		if next == nil {
			break
		}
		node = next
	}
	diagnostic.Message += fmt.Sprintf(" (line %d, column %d)", node.Line, node.Column)
}
