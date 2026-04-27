package changesets

import (
	"errors"
	"fmt"
	"slices"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

var errUnresolvableSourceNodeOutputChannels = errors.New("source node output channels are not resolvable")

func ValidateSourceNodeOutputChannel(
	registry *registry.Registry,
	sourceNode models.Node,
	channel string,
) error {
	outputChannels, err := listSourceNodeOutputChannels(registry, sourceNode)
	if err != nil {
		if errors.Is(err, errUnresolvableSourceNodeOutputChannels) {
			return nil
		}

		return fmt.Errorf("failed to resolve output channels for source node %s: %w", sourceNode.ID, err)
	}

	if len(outputChannels) == 0 {
		return nil
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
	registry *registry.Registry,
	sourceNode models.Node,
) ([]core.OutputChannel, error) {
	if sourceNode.Type == models.NodeTypeComponent {
		return listComponentOutputChannels(registry, sourceNode)
	}

	if sourceNode.Type == models.NodeTypeTrigger {
		return []core.OutputChannel{core.DefaultOutputChannel}, nil
	}

	if sourceNode.Type == models.NodeTypeBlueprint {
		// TODO: Validate blueprint output channels without doing a blueprint lookup per edge.
		return nil, nil
	}

	return nil, fmt.Errorf("node type %s is not supported", sourceNode.Type)
}

func listComponentOutputChannels(registry *registry.Registry, sourceNode models.Node) ([]core.OutputChannel, error) {
	if sourceNode.Ref.Component == nil || sourceNode.Ref.Component.Name == "" {
		return nil, fmt.Errorf("%w: component reference is required", errUnresolvableSourceNodeOutputChannels)
	}

	action, err := registry.GetAction(sourceNode.Ref.Component.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUnresolvableSourceNodeOutputChannels, err)
	}

	outputChannels := action.OutputChannels(sourceNode.Configuration)
	if len(outputChannels) > 0 {
		return outputChannels, nil
	}

	return []core.OutputChannel{core.DefaultOutputChannel}, nil
}
