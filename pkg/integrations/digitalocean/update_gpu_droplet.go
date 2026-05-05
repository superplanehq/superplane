package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const gpuDropletUpdatePollInterval = 10 * time.Second

type UpdateGPUDroplet struct{}

type UpdateGPUDropletSpec struct {
	GPUDroplet string  `json:"droplet" mapstructure:"gpuDroplet"`
	Name       *string `json:"name,omitempty" mapstructure:"name"`
	GPUSize    *string `json:"size,omitempty" mapstructure:"gpuSize"`
}

func (u *UpdateGPUDroplet) Name() string {
	return "digitalocean.updateGPUDroplet"
}

func (u *UpdateGPUDroplet) Label() string {
	return "Update GPU Droplet"
}

func (u *UpdateGPUDroplet) Description() string {
	return "Rename or upsize a DigitalOcean GPU Droplet"
}

func (u *UpdateGPUDroplet) Documentation() string {
	return `The Update GPU Droplet component allows renaming and upsizing an existing GPU droplet.

## Use Cases

- **Renaming**: Update the hostname of a GPU droplet
- **Upsizing**: Scale up a GPU droplet to a larger GPU size for more compute power
- **Combined updates**: Rename and upsize in a single operation

## Configuration

- **Droplet**: The GPU droplet to update (required, supports expressions)
- **New Name**: The new hostname for the GPU droplet (optional, supports expressions)
- **New GPU Size**: The new GPU size to upsize the droplet to (optional, only upsizing is supported)

## Output

Returns the updated GPU droplet object including:
- **id**: Droplet ID
- **name**: Updated droplet hostname
- **status**: Current droplet status
- **size_slug**: New GPU size identifier (if resized)
- **region**: Region information
- **networks**: Network information including IP addresses

## Important Notes

- At least one of **New Name** or **New GPU Size** must be provided
- Only **upsizing** is supported — you cannot downsize a GPU droplet
- The GPU droplet must be powered off before resizing
- If both rename and resize are specified, rename is performed first
- The component waits for each operation to complete before proceeding`
}

func (u *UpdateGPUDroplet) Icon() string {
	return "gpu"
}

func (u *UpdateGPUDroplet) Color() string {
	return "purple"
}

func (u *UpdateGPUDroplet) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateGPUDroplet) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "gpuDroplet",
			Label:       "GPU Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The GPU droplet to update",
			Placeholder: "Select GPU droplet",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "gpu_droplet",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:        "name",
			Label:       "New Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "The new hostname for the GPU droplet",
		},
		{
			Name:        "gpuSize",
			Label:       "New GPU Size",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "The new GPU size to upsize the droplet to",
			Placeholder: "Select a GPU size",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "gpu_size",
				},
			},
		},
	}
}

func (u *UpdateGPUDroplet) Setup(ctx core.SetupContext) error {
	spec := UpdateGPUDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.GPUDroplet == "" {
		return errors.New("GPU droplet is required")
	}

	hasName := spec.Name != nil && *spec.Name != ""
	hasSize := spec.GPUSize != nil && *spec.GPUSize != ""

	if !hasName && !hasSize {
		return errors.New("at least one of name or size must be provided")
	}

	err = resolveDropletMetadata(ctx, spec.GPUDroplet)
	if err != nil {
		return fmt.Errorf("error resolving GPU droplet metadata: %v", err)
	}

	return nil
}

func (u *UpdateGPUDroplet) Execute(ctx core.ExecutionContext) error {
	spec := UpdateGPUDropletSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	dropletID, err := parseDropletID(spec.GPUDroplet)
	if err != nil {
		return fmt.Errorf("invalid GPU droplet ID %q: %w", spec.GPUDroplet, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	hasName := spec.Name != nil && *spec.Name != ""
	hasSize := spec.GPUSize != nil && *spec.GPUSize != ""

	// Start with rename if specified, otherwise go directly to resize
	if hasName {
		if err := validateHostname(*spec.Name); err != nil {
			return err
		}

		action, err := client.RenameDroplet(dropletID, *spec.Name)
		if err != nil {
			return fmt.Errorf("failed to rename GPU droplet: %v", err)
		}

		state := "renaming"
		metadata := map[string]any{
			"actionID":  action.ID,
			"dropletID": dropletID,
			"state":     state,
			"newName":   *spec.Name,
		}

		if hasSize {
			metadata["newSize"] = *spec.GPUSize
		} else {
			state = "renaming_only"
			metadata["state"] = state
		}

		err = ctx.Metadata.Set(metadata)
		if err != nil {
			return fmt.Errorf("failed to store metadata: %v", err)
		}

		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, gpuDropletUpdatePollInterval)
	}

	// No rename, power off the droplet before resizing
	if !hasSize {
		return fmt.Errorf("new GPU size is required for resizing")
	}

	action, err := client.PostDropletAction(dropletID, "power_off")
	if err != nil {
		return fmt.Errorf("failed to power off GPU droplet before resize: %v", err)
	}

	err = ctx.Metadata.Set(map[string]any{
		"actionID":  action.ID,
		"dropletID": dropletID,
		"state":     "powering_off_for_resize",
		"newSize":   *spec.GPUSize,
	})
	if err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, gpuDropletUpdatePollInterval)
}

func (u *UpdateGPUDroplet) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateGPUDroplet) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateGPUDroplet) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: "poll",
			Type: core.HookTypeInternal,
		},
	}
}

func (u *UpdateGPUDroplet) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata struct {
		ActionID  int    `mapstructure:"actionID"`
		DropletID int    `mapstructure:"dropletID"`
		State     string `mapstructure:"state"`
		NewSize   string `mapstructure:"newSize"`
	}

	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	action, err := client.GetAction(metadata.ActionID)
	if err != nil {
		return fmt.Errorf("failed to get action status: %v", err)
	}

	switch action.Status {
	case "completed":
		return u.handleActionCompleted(ctx, client, metadata.DropletID, metadata.State, metadata.NewSize)
	case "in-progress":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, gpuDropletUpdatePollInterval)
	case "errored":
		return fmt.Errorf("GPU droplet update action failed with status: errored (state: %s)", metadata.State)
	default:
		return fmt.Errorf("action reached unexpected status %q", action.Status)
	}
}

func (u *UpdateGPUDroplet) handleActionCompleted(
	ctx core.ActionHookContext,
	client *Client,
	dropletID int,
	state string,
	newSize string,
) error {
	switch state {
	case "renaming":
		// Rename completed, now power off before resizing
		action, err := client.PostDropletAction(dropletID, "power_off")
		if err != nil {
			return fmt.Errorf("failed to power off GPU droplet after rename: %v", err)
		}

		err = ctx.Metadata.Set(map[string]any{
			"actionID":  action.ID,
			"dropletID": dropletID,
			"state":     "powering_off_for_resize",
			"newSize":   newSize,
		})
		if err != nil {
			return fmt.Errorf("failed to store metadata: %v", err)
		}

		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, gpuDropletUpdatePollInterval)

	case "powering_off_for_resize":
		// Power off completed, now start resize
		action, err := client.ResizeDroplet(dropletID, newSize, true)
		if err != nil {
			return fmt.Errorf("failed to resize GPU droplet after power off: %v", err)
		}

		err = ctx.Metadata.Set(map[string]any{
			"actionID":  action.ID,
			"dropletID": dropletID,
			"state":     "resizing",
			"newSize":   newSize,
		})
		if err != nil {
			return fmt.Errorf("failed to store metadata: %v", err)
		}

		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, gpuDropletUpdatePollInterval)

	case "renaming_only", "powering_on_after_resize":
		// Final operation completed, fetch and emit the updated droplet
		droplet, err := client.GetDroplet(dropletID)
		if err != nil {
			return fmt.Errorf("failed to get updated GPU droplet: %v", err)
		}

		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.gpuDroplet.updated",
			[]any{droplet},
		)

	case "resizing":
		// Resize completed, power the droplet back on
		action, err := client.PostDropletAction(dropletID, "power_on")
		if err != nil {
			return fmt.Errorf("failed to power on GPU droplet after resize: %v", err)
		}

		err = ctx.Metadata.Set(map[string]any{
			"actionID":  action.ID,
			"dropletID": dropletID,
			"state":     "powering_on_after_resize",
			"newSize":   newSize,
		})
		if err != nil {
			return fmt.Errorf("failed to store metadata: %v", err)
		}

		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, gpuDropletUpdatePollInterval)

	default:
		return fmt.Errorf("unexpected state %q", state)
	}
}

func (u *UpdateGPUDroplet) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateGPUDroplet) Cleanup(ctx core.SetupContext) error {
	return nil
}
