package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const (
	factoryConfigurationManifestPath = ".superplane/main.yml"
	factoryConfigurationCoverage     = "All stages and automations"
)

type factoryConfigurationManifest struct {
	APIVersion  string                               `yaml:"apiVersion"`
	Kind        string                               `yaml:"kind"`
	Name        string                               `yaml:"name"`
	Stages      []factoryConfigurationStage          `yaml:"stages"`
	Sources     map[string]string                    `yaml:"-" json:"-"`
	Definitions map[string]automationspec.Definition `yaml:"-" json:"definitions,omitempty"`
}

type factoryConfigurationStage struct {
	Key            string                           `yaml:"key"`
	Name           string                           `yaml:"name"`
	Kind           string                           `yaml:"kind"`
	Run            string                           `yaml:"run,omitempty"`
	MaxParallelism *int                             `yaml:"max_parallelism,omitempty"`
	Automations    []factoryConfigurationAutomation `yaml:"automations"`
}

type factoryConfigurationAutomation struct {
	ID   string         `yaml:"id"`
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with,omitempty"`
}

func configurationDiagnostic(field, message string) *pb.FactoryConfigurationDiagnostic {
	return &pb.FactoryConfigurationDiagnostic{Path: factoryConfigurationManifestPath, Field: field, Message: message}
}

func parseFactoryConfigurationBundle(bundle *pb.FactoryConfigurationBundle) (*factoryConfigurationManifest, *pb.FactoryConfigurationDiagnostic) {
	if bundle.GetManifestPath() != factoryConfigurationManifestPath {
		return nil, configurationDiagnostic("manifest_path", "The manifest path must be .superplane/main.yml.")
	}
	files := map[string]string{}
	for _, file := range bundle.GetFiles() {
		name := file.GetPath()
		if !fs.ValidPath(name) || !strings.HasPrefix(name, ".superplane/") || strings.ContainsAny(name, "\\:") {
			return nil, configurationDiagnostic("files", "Use unique relative file paths inside .superplane.")
		}
		if _, exists := files[name]; exists {
			return nil, configurationDiagnostic("files", "A file path occurs more than once.")
		}
		files[name] = file.GetContent()
	}
	content, exists := files[factoryConfigurationManifestPath]
	if !exists {
		return nil, configurationDiagnostic("files", "The bundle must contain .superplane/main.yml.")
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
	if diagnostic := validateConfigurationYAML(document.Content[0], ""); diagnostic != nil {
		return nil, diagnostic
	}
	root, diagnostic := configurationFields(document.Content[0], "", []string{"apiVersion", "kind", "name", "stages"}, []string{"apiVersion", "kind", "name", "stages"})
	if diagnostic != nil {
		return nil, diagnostic
	}
	for _, field := range []string{"apiVersion", "kind", "name"} {
		if diagnostic := configurationStringNode(root[field], field); diagnostic != nil {
			return nil, diagnostic
		}
	}
	if root["stages"].Kind != yaml.SequenceNode || len(root["stages"].Content) == 0 {
		return nil, configurationDiagnostic("stages", "Use a nonempty stage list.")
	}
	for i, stage := range root["stages"].Content {
		stagePath := fmt.Sprintf("stages[%d]", i)
		fields, diagnostic := configurationFields(stage, stagePath, []string{"key", "name", "kind", "run", "max_parallelism", "automations"}, []string{"key", "name", "kind", "automations"})
		if diagnostic != nil {
			return nil, diagnostic
		}
		for _, field := range []string{"key", "name", "kind", "run"} {
			if fields[field] != nil {
				if diagnostic := configurationStringNode(fields[field], configurationField(stagePath, field)); diagnostic != nil {
					return nil, diagnostic
				}
			}
		}
		if node := fields["max_parallelism"]; node != nil {
			value, err := strconv.Atoi(node.Value)
			if node.Kind != yaml.ScalarNode || node.Tag != "!!int" || err != nil || value < 1 || value > math.MaxInt32 {
				return nil, configurationDiagnostic(stagePath+".max_parallelism", "Use an integer from 1 through 2147483647.")
			}
		}
		if fields["automations"].Kind != yaml.SequenceNode {
			return nil, configurationDiagnostic(stagePath+".automations", "Use an automation list, including an empty list for an empty stage.")
		}
		for j, automation := range fields["automations"].Content {
			automationPath := fmt.Sprintf("%s.automations[%d]", stagePath, j)
			fields, diagnostic := configurationFields(automation, automationPath, []string{"id", "name", "uses", "with"}, []string{"id", "name", "uses"})
			if diagnostic != nil {
				return nil, diagnostic
			}
			for _, field := range []string{"id", "name", "uses"} {
				if diagnostic := configurationStringNode(fields[field], automationPath+"."+field); diagnostic != nil {
					return nil, diagnostic
				}
			}
			if node := fields["with"]; node != nil && node.Kind != yaml.MappingNode {
				return nil, configurationDiagnostic(automationPath+".with", "Use a mapping of supported settings.")
			}
		}
	}
	var manifest factoryConfigurationManifest
	if err := document.Decode(&manifest); err != nil {
		return nil, configurationDiagnostic("$", "Invalid Factory manifest: "+err.Error())
	}
	manifest.Sources = map[string]string{}
	manifest.Definitions = map[string]automationspec.Definition{}
	referenced := map[string]bool{factoryConfigurationManifestPath: true}
	for i, stage := range manifest.Stages {
		for j, automation := range stage.Automations {
			if strings.HasPrefix(automation.Uses, "@superplane/") {
				continue
			}
			reference := automation.Uses
			if !fs.ValidPath(reference) || strings.ContainsAny(reference, "\\:") || reference == "main.yml" || (path.Ext(reference) != ".yaml" && path.Ext(reference) != ".yml") {
				return nil, configurationDiagnostic(fmt.Sprintf("stages[%d].automations[%d].uses", i, j), "Use a relative YAML path inside the manifest directory.")
			}
			filePath := path.Join(path.Dir(factoryConfigurationManifestPath), reference)
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
			return nil, configurationDiagnostic(configurationField(field, key.Value), "This field is not supported.")
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
	manifest := factoryConfigurationManifest{APIVersion: "v2", Kind: "Factory", Name: target.factory.Name, Stages: []factoryConfigurationStage{}, Sources: map[string]string{}, Definitions: map[string]automationspec.Definition{}}
	for _, stage := range target.stages {
		entry := factoryConfigurationStage{Key: stage.key, Name: stage.stage.Name, Kind: stage.stage.Kind, Run: stage.run, Automations: []factoryConfigurationAutomation{}}
		if stage.stage.StepIndex != nil {
			entry.MaxParallelism = configurationMaxParallelism(target.line.Steps[*stage.stage.StepIndex].MaxParallelism)
		}
		for _, automation := range stage.automations {
			entry.Automations = append(entry.Automations, factoryConfigurationAutomation{ID: automation.id, Name: automation.canvas.Name, Uses: automation.uses, With: factoryConfigurationOverrides(automation.settings, automation.defaultSettings)})
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

func validateFactoryConfigurationManifest(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, manifest *factoryConfigurationManifest, target *factoryConfigurationTarget) (*pb.FactoryConfigurationDiagnostic, error) {
	for _, check := range []struct{ field, value, want string }{{"apiVersion", manifest.APIVersion, "v2"}, {"kind", manifest.Kind, "Factory"}, {"name", manifest.Name, target.factory.Name}} {
		if check.value != check.want {
			return configurationDiagnostic(check.field, fmt.Sprintf("This field must match the selected live resource: %q.", check.want)), nil
		}
	}
	if len(manifest.Stages) != len(target.stages) {
		return configurationDiagnostic("stages", "Include every existing stage. This POC does not add or remove stages."), nil
	}
	byKey := map[string]*factoryConfigurationStageTarget{}
	for _, stage := range target.stages {
		byKey[stage.key] = stage
	}
	seen := map[string]bool{}
	names := map[string]bool{}
	desiredByCanvas := map[string]string{}
	retainedByStage := map[string]map[string]bool{}
	newNames := map[string]bool{}
	desiredNames := map[string]string{}
	sourceNames := map[string]string{}
	for i := range manifest.Stages {
		stage := &manifest.Stages[i]
		stagePath := fmt.Sprintf("stages[%d]", i)
		live := byKey[stage.Key]
		if live == nil || seen[stage.Key] {
			return configurationDiagnostic(stagePath+".key", "Use each existing stage key exactly once."), nil
		}
		seen[stage.Key] = true
		if strings.TrimSpace(stage.Name) == "" || strings.TrimSpace(stage.Name) != stage.Name || utf8.RuneCountInString(stage.Name) > factoryLineStageNameMaxLength || names[strings.ToLower(stage.Name)] {
			return configurationDiagnostic(stagePath+".name", "Use a unique stage name of 1 through 64 characters without surrounding spaces."), nil
		}
		names[strings.ToLower(stage.Name)] = true
		if stage.Kind != live.stage.Kind {
			return configurationDiagnostic(stagePath+".kind", "The existing stage kind cannot change."), nil
		}
		if stage.Run != live.run {
			return configurationDiagnostic(stagePath+".run", "Keep the existing phase runner binding."), nil
		}
		if live.stage.StepIndex == nil && stage.MaxParallelism != nil {
			return configurationDiagnostic(stagePath+".max_parallelism", "Only stages with a runner binding support max_parallelism."), nil
		}
		stage.MaxParallelism = configurationMaxParallelism(stage.MaxParallelism)
		ids, automationNames, retained := map[string]bool{}, map[string]bool{}, map[string]bool{}
		retainedByStage[stage.Key] = retained
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
			current := configurationAutomationByIdentity(live, *automation, stage.Automations)
			identity := "new:" + stage.Key + "." + automation.ID
			if current != nil {
				identity = current.canvas.ID.String()
			}
			if owner, exists := desiredNames[automation.Name]; exists && owner != identity {
				return configurationDiagnostic(automationPath+".name", "Use a unique automation name in this workspace."), nil
			}
			desiredNames[automation.Name] = identity
			if current == nil {
				if strings.HasPrefix(automation.Uses, "@superplane/") {
					return configurationDiagnostic(automationPath+".uses", "Install this built-in automation in the workspace before adding it to the manifest."), nil
				}
				for _, existingStage := range target.stages {
					for _, existing := range existingStage.automations {
						if existing.canvas.Name == automation.Name {
							return configurationDiagnostic(automationPath+".name", "Keep existing automations in their current stages with their existing identities."), nil
						}
					}
				}
				if !models.ValidCanvasColumnKey(configurationStagePlacement(live)) {
					return configurationDiagnostic(automationPath, "New automations cannot attach to this phase. Use an existing board stage."), nil
				}
				sourceID := configurationAutomationSourceID(target.factory.ID, configurationStagePlacement(live), automation.ID)
				authored, err := models.FindAutomationSource(tx, target.factory.OrganizationID, sourceID)
				if err == nil {
					canvasID, err := uuid.Parse(authored.ArtifactID)
					if err != nil {
						return nil, err
					}
					inUse, err := models.CheckCanvasExistence(tx, target.factory.OrganizationID, canvasID)
					if err != nil {
						return nil, err
					}
					if inUse || authored.FactoryID != target.factory.ID {
						return configurationDiagnostic(automationPath+".id", "Use an automation ID that is not already assigned in this workspace."), nil
					}
				}
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, err
				}
				if newNames[automation.Name] {
					return configurationDiagnostic(automationPath+".name", "Use a unique name for each new automation."), nil
				}
				newNames[automation.Name] = true
			} else {
				identity := current.canvas.ID.String()
				if retained[identity] {
					return configurationDiagnostic(automationPath+".id", "Include each automation only once per stage."), nil
				}
				retained[identity] = true
				if automation.Name != current.canvas.Name {
					existing, err := models.FindCanvasByName(tx, target.factory.OrganizationID, &target.factory.ID, automation.Name)
					if err == nil && existing.ID != current.canvas.ID {
						return configurationDiagnostic(automationPath+".name", "Use a unique automation name in this workspace."), nil
					}
					if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
						return nil, err
					}
				}
				if (current.templateID != "" || strings.HasPrefix(automation.Uses, "@superplane/")) && automation.Uses != current.uses {
					return configurationDiagnostic(automationPath+".uses", "Keep the existing automation binding."), nil
				}
			}
			if current != nil && current.templateID != "" {
				nodes, edges, err := copyFactoryConfigurationBuiltIn(current.version, current.defaults, current.templateID, automation.With)
				if err != nil {
					var setting *factoryConfigurationSettingError
					if errors.As(err, &setting) {
						return configurationDiagnostic(automationPath+".with."+setting.Field, setting.Message), nil
					}
					return configurationDiagnostic(automationPath+".with", err.Error()), nil
				}
				effective := *current.version
				effective.Nodes, effective.Edges = nodes, edges
				automation.With, err = exportFactoryConfigurationBuiltIn(&effective, current.templateID)
				if err != nil {
					return nil, err
				}
				if !equalCanvasJSON(current.settings, automation.With) {
					if err := validateFactoryConfigurationBuiltInNodes(deps, nodes); err != nil {
						var setting *factoryConfigurationSettingError
						if errors.As(err, &setting) {
							return configurationDiagnostic(automationPath+".with."+setting.Field, setting.Message), nil
						}
						return nil, err
					}
				}
			} else {
				if len(automation.With) != 0 {
					return configurationDiagnostic(automationPath+".with", "Custom automation settings belong in the referenced Automation file."), nil
				}
				filePath := path.Join(path.Dir(factoryConfigurationManifestPath), automation.Uses)
				if previous, exists := sourceNames[filePath]; exists && previous != automation.Name {
					return configurationDiagnostic(automationPath+".uses", "Use separate YAML files for automations with different names."), nil
				}
				sourceNames[filePath] = automation.Name
				source, err := automationspec.ReplaceName(manifest.Sources[filePath], automation.Name)
				if err != nil {
					return configurationSourceDiagnostic(filePath, err), nil
				}
				manifest.Sources[filePath] = source
				var canvas *models.Canvas
				var version *models.CanvasVersion
				placement := configurationStagePlacement(live)
				if current != nil {
					canvas, version, placement = current.canvas, current.version, current.placement
				}
				document, err := validateFactoryConfigurationCustom(ctx, tx, deps, target.factory, canvas, version, placement, manifest.Sources[filePath])
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
				Definition automationspec.Definition
			}{automation.Name, automation.With, manifest.Definitions[path.Join(path.Dir(factoryConfigurationManifestPath), automation.Uses)]})
			if previous, exists := desiredByCanvas[identity]; exists && previous != string(encoded) {
				return configurationDiagnostic(automationPath, "This shared automation must have the same configuration in every stage."), nil
			}
			desiredByCanvas[identity] = string(encoded)
		}
		if stage.Run != "" && !slices.ContainsFunc(stage.Automations, func(automation factoryConfigurationAutomation) bool {
			return automation.ID == stage.Run
		}) {
			return configurationDiagnostic(stagePath+".automations", "Keep the automation bound to this phase runner."), nil
		}
	}
	checkedRemovals := map[string]bool{}
	for i, stage := range manifest.Stages {
		for _, current := range byKey[stage.Key].automations {
			identity := current.canvas.ID.String()
			if retainedByStage[stage.Key][identity] {
				continue
			}
			field := fmt.Sprintf("stages[%d].automations", i)
			if _, retained := desiredByCanvas[identity]; retained {
				return configurationDiagnostic(field, "Keep shared automations in every existing stage, or remove them from every stage."), nil
			}
			if checkedRemovals[identity] {
				continue
			}
			checkedRemovals[identity] = true
			if err := validateFactoryConfigurationAutomationRemoval(tx, target, current); err != nil {
				return configurationDiagnostic(field, fmt.Sprintf("Cannot remove automation %q: %s", current.canvas.Name, err)), nil
			}
		}
	}
	if manifest.Stages[0].Kind != models.FactoryLineStageBacklog {
		return configurationDiagnostic("stages[0]", "Backlog must remain first."), nil
	}
	if manifest.Stages[len(manifest.Stages)-1].Kind != models.FactoryLineStageDone {
		return configurationDiagnostic("stages", "Done must remain last."), nil
	}
	if factoryConfigurationStagesChanged(target, manifest) {
		enabled, err := models.OrganizationHasExperimentalFeatures(tx, target.factory.OrganizationID, features.FeatureFactoryCustomAutomations)
		if err != nil {
			return nil, err
		}
		if !enabled {
			return configurationDiagnostic("stages", "Enable custom stages before changing stage names or order."), nil
		}
	}
	return nil, nil
}
