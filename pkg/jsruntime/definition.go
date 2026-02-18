package jsruntime

import (
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// ComponentDefinition holds the metadata extracted from a JS component file.
type ComponentDefinition struct {
	Name          string
	Label         string
	Description   string
	Documentation string
	Icon          string
	Color         string
	HasExecute    bool
	HasSetup      bool

	RawConfiguration  any
	RawOutputChannels any
}

// ParseConfiguration converts the raw JS configuration array into typed configuration.Field
// values. Returns nil if no configuration was defined.
func (d *ComponentDefinition) ParseConfiguration() ([]configuration.Field, error) {
	if d.RawConfiguration == nil {
		return nil, nil
	}

	data, err := json.Marshal(d.RawConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal configuration: %w", err)
	}

	var fields []configuration.Field
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("failed to parse configuration fields: %w", err)
	}

	return fields, nil
}

// ParseOutputChannels converts the raw JS output channels array into typed
// core.OutputChannel values. Returns a single "default" channel if none were defined.
func (d *ComponentDefinition) ParseOutputChannels() ([]core.OutputChannel, error) {
	if d.RawOutputChannels == nil {
		return []core.OutputChannel{core.DefaultOutputChannel}, nil
	}

	data, err := json.Marshal(d.RawOutputChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output channels: %w", err)
	}

	var channels []core.OutputChannel
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("failed to parse output channels: %w", err)
	}

	if len(channels) == 0 {
		return []core.OutputChannel{core.DefaultOutputChannel}, nil
	}

	return channels, nil
}
