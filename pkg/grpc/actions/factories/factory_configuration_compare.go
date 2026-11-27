package factories

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automationspec"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func (manifest *factoryConfigurationManifest) automationPath(reference string) string {
	return path.Join(path.Dir(manifest.ManifestPath), reference)
}

func configurationStagesByKey(target *factoryConfigurationTarget) map[string]*factoryConfigurationStageTarget {
	stages := make(map[string]*factoryConfigurationStageTarget, len(target.stages))
	for _, stage := range target.stages {
		stages[stage.key] = stage
	}
	return stages
}

func resolveFactoryConfigurationIdentities(target *factoryConfigurationTarget, manifest *factoryConfigurationManifest) {
	byID := map[string]*factoryConfigurationStageTarget{}
	automations := map[string]*factoryConfigurationAutomationTarget{}
	for _, stage := range target.stages {
		byID[stage.stage.ConfigurationID] = stage
		for _, automation := range stage.automations {
			automations[automation.id] = automation
		}
	}
	target.removed = map[uuid.UUID]bool{}
	desiredIDs := map[string]bool{}
	for _, stage := range manifest.Stages {
		for _, desired := range stage.Automations {
			desiredIDs[desired.ID] = true
		}
	}
	for _, stage := range target.stages {
		for _, current := range stage.automations {
			if !desiredIDs[current.id] {
				target.removed[current.canvas.ID] = true
			}
		}
	}
	for i := range manifest.Stages {
		stage := &manifest.Stages[i]
		stage.retained = map[uuid.UUID]bool{}
		live := byID[stage.ID]
		if live == nil {
			continue
		}
		stage.Key = live.key
		for j := range stage.Automations {
			desired := &stage.Automations[j]
			desired.current = automations[desired.ID]
			desired.placement = configurationStagePlacement(live)
			if desired.current != nil {
				stage.retained[desired.current.canvas.ID] = true
				if live.runner == desired.current || slices.Contains(live.automations, desired.current) {
					desired.placement = desired.current.placement
				}
			}
		}
	}
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
	if current.placement != desired.placement {
		return true
	}
	if current.templateID != "" {
		return !equalCanvasJSON(current.settings, desired.With) || factoryConfigurationBindingChanged(current, desired) || factoryConfigurationReviewSourceChanged(current, desired)
	}
	previous, err := automationspec.Parse(current.source)
	if err != nil {
		return true
	}
	requested := manifest.Definitions[manifest.automationPath(desired.Uses)]
	previous.Definition.Name = requested.Name
	return !equalCanvasJSON(previous.Definition, requested)
}

func compareFactoryConfiguration(target *factoryConfigurationTarget, manifest *factoryConfigurationManifest) []*pb.FactoryConfigurationChange {
	var changes []*pb.FactoryConfigurationChange
	if manifest.Name != target.factory.Name {
		changes = append(changes, configurationChange("workspace", "name", target.factory.Name, manifest.Name))
	}
	byKey := configurationStagesByKey(target)
	previousOrder, requestedOrder := []string{}, []string{}
	for _, stage := range target.stages {
		previousOrder = append(previousOrder, stage.stage.ConfigurationID)
	}
	for _, stage := range manifest.Stages {
		requestedOrder = append(requestedOrder, stage.ID)
	}
	if !slices.Equal(previousOrder, requestedOrder) {
		changes = append(changes, configurationChange("stages", "stages", previousOrder, requestedOrder))
	}
	seen := map[string]bool{}
	allRetained := map[string]bool{}
	for _, stage := range manifest.Stages {
		for id := range stage.retained {
			allRetained[id.String()] = true
		}
	}
	for i, stage := range manifest.Stages {
		live := byKey[stage.Key]
		stagePath := fmt.Sprintf("stages[%d]", i)
		if stage.Name != live.stage.Name {
			changes = append(changes, configurationChange("stages."+stage.ID, stagePath+".name", live.stage.Name, stage.Name))
		}
		retained := map[string]bool{}
		for j, automation := range stage.Automations {
			current := automation.current
			resource := "stages." + stage.ID + ".automations." + automation.ID
			if current == nil {
				changes = append(changes, configurationChange(resource, fmt.Sprintf("%s.automations[%d]", stagePath, j), nil, automation))
				continue
			}
			retained[current.canvas.ID.String()] = true
			if seen[current.canvas.ID.String()] || !factoryConfigurationAutomationChanged(current, automation, manifest) {
				continue
			}
			seen[current.canvas.ID.String()] = true
			if current.placement != automation.placement {
				changes = append(changes, configurationChange(resource, fmt.Sprintf("%s.automations[%d].stage", stagePath, j), current.placement, automation.placement))
			}
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
			filePath := manifest.automationPath(automation.Uses)
			changes = append(changes, configurationChange(resource, filePath, current.source, manifest.Sources[filePath]))
		}
		for _, current := range live.automations {
			identity := current.canvas.ID.String()
			if retained[identity] || allRetained[identity] || seen[identity] {
				continue
			}
			seen[identity] = true
			previous := factoryConfigurationAutomation{ID: current.id, Name: current.canvas.Name, Uses: current.uses, With: current.settings}
			resource := "stages." + stage.ID + ".automations." + current.id
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
