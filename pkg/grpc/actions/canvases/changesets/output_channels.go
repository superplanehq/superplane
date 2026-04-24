package changesets

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

var errUnresolvableSourceNodeOutputChannels = errors.New("source node output channels are not resolvable")

func ValidateSourceNodeOutputChannel(
	tx *gorm.DB,
	registry *registry.Registry,
	organizationID uuid.UUID,
	sourceNode models.Node,
	channel string,
) error {
	outputChannels, err := listSourceNodeOutputChannels(tx, registry, organizationID, sourceNode)
	if err != nil {
		if errors.Is(err, errUnresolvableSourceNodeOutputChannels) {
			return nil
		}

		return fmt.Errorf("failed to resolve output channels for source node %s: %w", sourceNode.ID, err)
	}

	if slices.ContainsFunc(outputChannels, func(outputChannel core.OutputChannel) bool {
		return outputChannel.Name == channel
	}) {
		return nil
	}

	available := make([]string, 0, len(outputChannels))
	for _, outputChannel := range outputChannels {
		available = append(available, outputChannel.Name)
	}

	return fmt.Errorf(
		"source node %s does not have output channel %q (available: %v)",
		sourceNode.ID,
		channel,
		available,
	)
}

func listSourceNodeOutputChannels(
	tx *gorm.DB,
	registry *registry.Registry,
	organizationID uuid.UUID,
	sourceNode models.Node,
) ([]core.OutputChannel, error) {
	if sourceNode.Type == models.NodeTypeComponent {
		return listComponentOutputChannels(registry, sourceNode)
	}

	if sourceNode.Type == models.NodeTypeTrigger {
		return []core.OutputChannel{core.DefaultOutputChannel}, nil
	}

	if sourceNode.Type == models.NodeTypeBlueprint {
		return listBlueprintOutputChannels(tx, organizationID, sourceNode)
	}

	return nil, fmt.Errorf("node type %s is not supported", sourceNode.Type)
}

func listComponentOutputChannels(registry *registry.Registry, sourceNode models.Node) ([]core.OutputChannel, error) {
	if sourceNode.Ref.Component == nil || sourceNode.Ref.Component.Name == "" {
		return nil, fmt.Errorf("%w: component reference is required", errUnresolvableSourceNodeOutputChannels)
	}

	component, err := registry.GetComponent(sourceNode.Ref.Component.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUnresolvableSourceNodeOutputChannels, err)
	}

	outputChannels := component.OutputChannels(sourceNode.Configuration)
	if len(outputChannels) > 0 {
		return outputChannels, nil
	}

	return []core.OutputChannel{core.DefaultOutputChannel}, nil
}

func listBlueprintOutputChannels(
	tx *gorm.DB,
	organizationID uuid.UUID,
	sourceNode models.Node,
) ([]core.OutputChannel, error) {
	if sourceNode.Ref.Blueprint == nil || sourceNode.Ref.Blueprint.ID == "" {
		return nil, fmt.Errorf("%w: blueprint reference is required", errUnresolvableSourceNodeOutputChannels)
	}

	blueprint, err := models.FindBlueprintInTransaction(tx, organizationID.String(), sourceNode.Ref.Blueprint.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUnresolvableSourceNodeOutputChannels, err)
	}

	outputChannels := make([]core.OutputChannel, 0, len(blueprint.OutputChannels))
	for _, outputChannel := range blueprint.OutputChannels {
		outputChannels = append(outputChannels, core.OutputChannel{Name: outputChannel.Name})
	}

	return outputChannels, nil
}
