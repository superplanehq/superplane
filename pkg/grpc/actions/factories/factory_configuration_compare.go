package factories

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func configurationAutomationPath(reference string) string {
	return path.Join(path.Dir(factoryConfigurationManifestPath), reference)
}

func configurationStagesByKey(target *factoryConfigurationTarget) map[string]*factoryConfigurationStageTarget {
	stages := make(map[string]*factoryConfigurationStageTarget, len(target.stages))
	for _, stage := range target.stages {
		stages[stage.key] = stage
	}
	return stages
}

func configurationAutomationByStableIdentity(stage *factoryConfigurationStageTarget, id string) *factoryConfigurationAutomationTarget {
	for _, current := range stage.automations {
		if current.id == id {
			return current
		}
		if current.templateID != "" && id == configurationAutomationKey(current)+"-"+configurationCanvasKey(current.canvas.ID) {
			return current
		}
		if current.canvas.FactoryID != nil && current.sourceID == configurationAutomationSourceID(*current.canvas.FactoryID, configurationStagePlacement(stage), id) {
			return current
		}
	}
	return nil
}

func configurationAutomationByIdentity(stage *factoryConfigurationStageTarget, desired factoryConfigurationAutomation, automations []factoryConfigurationAutomation) *factoryConfigurationAutomationTarget {
	if current := configurationAutomationByStableIdentity(stage, desired.ID); current != nil {
		return current
	}
	// Older manifests and new custom entries can use different IDs from export.
	for _, current := range stage.automations {
		// A filename or name must not take another entry's stable identity.
		if slices.ContainsFunc(automations, func(automation factoryConfigurationAutomation) bool {
			return configurationAutomationByStableIdentity(stage, automation.ID) == current
		}) {
			continue
		}
		if current.templateID == "" && current.uses == desired.Uses {
			return current
		}
		if current.canvas.Name != desired.Name {
			continue
		}
		if current.templateID != "" && current.uses == desired.Uses {
			return current
		}
		if current.templateID == "" && !strings.HasPrefix(desired.Uses, "@superplane/") {
			return current
		}
	}
	return nil
}

func factoryConfigurationStagesChanged(target *factoryConfigurationTarget, manifest *factoryConfigurationManifest) bool {
	for i, stage := range manifest.Stages {
		if stage.Key != target.stages[i].key || stage.Name != target.stages[i].stage.Name {
			return true
		}
	}
	return false
}

func factoryConfigurationAutomationChanged(current *factoryConfigurationAutomationTarget, desired factoryConfigurationAutomation, manifest *factoryConfigurationManifest) bool {
	return current.canvas.Name != desired.Name || factoryConfigurationAutomationContentChanged(current, desired, manifest)
}

func factoryConfigurationAutomationContentChanged(current *factoryConfigurationAutomationTarget, desired factoryConfigurationAutomation, manifest *factoryConfigurationManifest) bool {
	if current.templateID != "" {
		return !equalCanvasJSON(current.settings, desired.With) || factoryConfigurationBindingChanged(current, desired) || factoryConfigurationReviewSourceChanged(current, desired)
	}
	previous, err := automationspec.Parse(current.source)
	if err != nil {
		return true
	}
	requested := manifest.Definitions[configurationAutomationPath(desired.Uses)]
	previous.Definition.Name = requested.Name
	return !equalCanvasJSON(previous.Definition, requested)
}

func compareFactoryConfiguration(target *factoryConfigurationTarget, manifest *factoryConfigurationManifest) []*pb.FactoryConfigurationChange {
	var changes []*pb.FactoryConfigurationChange
	byKey := configurationStagesByKey(target)
	previousOrder, requestedOrder := []string{}, []string{}
	for _, stage := range target.stages {
		previousOrder = append(previousOrder, stage.key)
	}
	for _, stage := range manifest.Stages {
		requestedOrder = append(requestedOrder, stage.Key)
	}
	if !slices.Equal(previousOrder, requestedOrder) {
		changes = append(changes, configurationChange("stages", "stages", previousOrder, requestedOrder))
	}
	seen := map[string]bool{}
	for i, stage := range manifest.Stages {
		live := byKey[stage.Key]
		stagePath := fmt.Sprintf("stages[%d]", i)
		if stage.Name != live.stage.Name {
			changes = append(changes, configurationChange("stages."+stage.Key, stagePath+".name", live.stage.Name, stage.Name))
		}
		if live.stage.StepIndex != nil {
			previous := configurationMaxParallelism(target.line.Steps[*live.stage.StepIndex].MaxParallelism)
			if !equalCanvasJSON(previous, stage.MaxParallelism) {
				changes = append(changes, configurationChange("stages."+stage.Key, stagePath+".max_parallelism", previous, stage.MaxParallelism))
			}
		}
		retained := map[string]bool{}
		for j, automation := range stage.Automations {
			current := configurationAutomationByIdentity(live, automation, stage.Automations)
			resource := "stages." + stage.Key + ".automations." + automation.ID
			if current == nil {
				changes = append(changes, configurationChange(resource, fmt.Sprintf("%s.automations[%d]", stagePath, j), nil, automation))
				continue
			}
			retained[current.canvas.ID.String()] = true
			if seen[current.canvas.ID.String()] || !factoryConfigurationAutomationChanged(current, automation, manifest) {
				continue
			}
			seen[current.canvas.ID.String()] = true
			if current.canvas.Name != automation.Name {
				changes = append(changes, configurationChange(resource, fmt.Sprintf("%s.automations[%d].name", stagePath, j), current.canvas.Name, automation.Name))
			}
			if !factoryConfigurationAutomationContentChanged(current, automation, manifest) {
				continue
			}
			if current.templateID != "" {
				if factoryConfigurationBindingChanged(current, automation) {
					changes = append(changes, configurationChange(resource, fmt.Sprintf("%s.automations[%d].binding", stagePath, j), current.exportedBinding, automation.Binding))
				}
				if factoryConfigurationReviewSourceChanged(current, automation) {
					previous := findIntakeNode(current.version.Nodes, reviewRequestTriggerNodeID)
					requested := findIntakeNode(automation.prepared.Nodes, reviewRequestTriggerNodeID)
					changes = append(changes, configurationChange(resource, fmt.Sprintf("%s.automations[%d].with.template.merge_confidence", stagePath, j), previous.Configuration["app"], requested.Configuration["app"]))
				}
				changes = append(changes, configurationSettingChanges(resource, fmt.Sprintf("%s.automations[%d].with", stagePath, j), current.settings, automation.With)...)
				continue
			}
			filePath := configurationAutomationPath(automation.Uses)
			changes = append(changes, configurationChange(resource, filePath, current.source, manifest.Sources[filePath]))
		}
		for _, current := range live.automations {
			identity := current.canvas.ID.String()
			if retained[identity] || seen[identity] {
				continue
			}
			seen[identity] = true
			previous := factoryConfigurationAutomation{ID: current.id, Name: current.canvas.Name, Uses: current.uses, With: current.settings}
			resource := "stages." + stage.Key + ".automations." + current.id
			changes = append(changes, configurationChange(resource, stagePath+".automations", previous, nil))
		}
	}
	return changes
}

func configurationSettingChanges(resource, field string, previous, requested map[string]any) []*pb.FactoryConfigurationChange {
	keys := make([]string, 0, len(previous)+len(requested))
	for key := range previous {
		keys = append(keys, key)
	}
	for key := range requested {
		if _, present := previous[key]; !present {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	var changes []*pb.FactoryConfigurationChange
	for _, key := range keys {
		before, after := previous[key], requested[key]
		if equalCanvasJSON(before, after) {
			continue
		}
		beforeFields, beforeMap := before.(map[string]any)
		afterFields, afterMap := after.(map[string]any)
		if afterMap && (beforeMap || before == nil) {
			changes = append(changes, configurationSettingChanges(resource, field+"."+key, beforeFields, afterFields)...)
			continue
		}
		changes = append(changes, configurationChange(resource, field+"."+key, before, after))
	}
	return changes
}

func configurationChange(resource, field string, previous, requested any) *pb.FactoryConfigurationChange {
	return &pb.FactoryConfigurationChange{Resource: resource, FieldPath: field, PreviousValue: configurationChangeValue(previous), RequestedValue: configurationChangeValue(requested)}
}

func configurationChangeValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func configurationMaxParallelism(value *int) *int {
	if value == nil || *value == models.DefaultFactoryLineStepMaxParallelism {
		return nil
	}
	return value
}
