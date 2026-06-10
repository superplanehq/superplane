package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	compute "google.golang.org/api/compute/v1"
)

type UpdateImage struct{}

type UpdateImageSpec struct {
	Image            string       `mapstructure:"image"`
	DeprecationState string       `mapstructure:"deprecationState"`
	Replacement      string       `mapstructure:"replacement"`
	Labels           []LabelEntry `mapstructure:"labels"`
}

func (u *UpdateImage) Name() string {
	return "gcp.updateImage"
}

func (u *UpdateImage) Label() string {
	return "Compute • Update Image"
}

func (u *UpdateImage) Description() string {
	return "Update a Google Compute Engine image: deprecate or obsolete it, and update its labels"
}

func (u *UpdateImage) Documentation() string {
	return `The Update Image component updates an existing Compute Engine custom image. It can change the image's deprecation status (including marking it deprecated or obsolete) and update its labels.

## Use Cases

- **Release lifecycle**: Mark older images deprecated or obsolete as newer ones ship
- **Pointing consumers forward**: Set a replacement image so users are guided to the current version
- **Inventory hygiene**: Keep image labels accurate for billing, environment, and ownership

## Configuration

- **Image**: The custom image to update.
- **Deprecation state**: Optionally change the image lifecycle state:
  - **No change**: Leave the current deprecation state untouched.
  - **Active**: Clear any deprecation (make the image fully usable again).
  - **Deprecated**: Mark the image deprecated (still usable; warns on use).
  - **Obsolete**: Mark the image obsolete (cannot be used to create new resources).
  - **Deleted**: Mark the image as logically deleted.
- **Replacement image**: Optional image to recommend in place of this one (name or URL). Applies when setting a deprecation state.
- **Labels**: Optional key-value labels to add or update on the image. Labels you provide are merged with the image's existing labels; existing labels you don't list are left unchanged.

## Output

Emits the updated image: name, selfLink, family, status, diskSizeGb, labels, deprecationState, replacement, creationTimestamp.

## Important Notes

- Setting the deprecation state to **Active** clears an existing deprecation.
- Updating labels merges with the image's existing labels; labels you don't list are preserved.
- The component waits for each underlying global operation to complete before emitting.`
}

func (u *UpdateImage) Icon() string {
	return "image"
}

func (u *UpdateImage) Color() string {
	return "blue"
}

func (u *UpdateImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateImage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "image",
			Label:       "Image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The custom image to update.",
			Placeholder: "Select image",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCustomImages,
				},
			},
		},
		{
			Name:        "deprecationState",
			Label:       "Deprecation state",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optionally change the image's lifecycle state.",
			Default:     ImageStateNoChange,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "No change", Value: ImageStateNoChange},
						{Label: "Active", Value: ImageStateActive},
						{Label: "Deprecated", Value: ImageStateDeprecated},
						{Label: "Obsolete", Value: ImageStateObsolete},
						{Label: "Deleted", Value: ImageStateDeleted},
					},
				},
			},
		},
		{
			Name:        "replacement",
			Label:       "Replacement image",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional image to recommend in place of this one. Users of the deprecated image are pointed here.",
			Placeholder: "Select replacement image",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCustomImages,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "deprecationState", Values: []string{ImageStateDeprecated, ImageStateObsolete, ImageStateDeleted}},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Key-value labels to add or update. Merged with the image's existing labels; labels you don't list are kept.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Label:       "Key",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label key (e.g. env, team, cost-center).",
								Placeholder: "e.g. env",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Label value.",
								Placeholder: "e.g. production",
							},
						},
					},
				},
			},
		},
	}
}

func (u *UpdateImage) Setup(ctx core.SetupContext) error {
	spec := UpdateImageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Image) == "" {
		return errors.New("image is required")
	}

	if err := validateDeprecationState(spec.DeprecationState); err != nil {
		return err
	}

	return resolveImageNodeMetadata(ctx, spec.Image)
}

func validateDeprecationState(state string) error {
	switch strings.TrimSpace(state) {
	case "", ImageStateNoChange, ImageStateActive, ImageStateDeprecated, ImageStateObsolete, ImageStateDeleted:
		return nil
	default:
		return fmt.Errorf("invalid deprecation state %q", state)
	}
}

// desiredDeprecationState normalizes the configured value into the actual state
// to apply, returning "" when no change should be made.
func desiredDeprecationState(state string) string {
	s := strings.TrimSpace(state)
	if s == ImageStateNoChange {
		return ""
	}
	return s
}

func (u *UpdateImage) Execute(ctx core.ExecutionContext) error {
	spec := UpdateImageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if err := validateDeprecationState(spec.DeprecationState); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	labelUpdates := imageLabelsFromEntries(spec.Labels)
	desiredState := desiredDeprecationState(spec.DeprecationState)
	if labelUpdates == nil && desiredState == "" {
		return ctx.ExecutionState.Fail("error", "nothing to update: set a deprecation state or labels")
	}

	urlProject, name, err := parseImagePath(spec.Image)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	if urlProject != "" && urlProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"image belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			urlProject, project,
		))
	}

	callCtx := context.Background()

	// Read the current image to obtain the label fingerprint (required for
	// setLabels) and the current deprecation state.
	body, err := GetImage(callCtx, client, project, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read image: %v", err))
	}
	var current imageGetResp
	if err := json.Unmarshal(body, &current); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse image: %v", err))
	}

	if labelUpdates != nil {
		merged := mergeImageLabels(current.Labels, labelUpdates)
		if err := setImageLabels(callCtx, client, project, name, merged, current.LabelFingerprint); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to update image labels: %v", err))
		}
	}

	if desiredState != "" {
		if err := deprecateImage(callCtx, client, project, name, desiredState, strings.TrimSpace(spec.Replacement)); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to update image deprecation state: %v", err))
		}
	}

	updated, err := GetImage(callCtx, client, project, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read image after update: %v", err))
	}
	payload, err := ImagePayloadFromGetResponse(updated)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse updated image: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.image.updated",
		[]any{payload},
	)
}

// setImageLabels issues the images.setLabels POST and waits for the global operation.
func setImageLabels(ctx context.Context, client Client, project, name string, labels map[string]string, fingerprint string) error {
	path := fmt.Sprintf("projects/%s/global/images/%s/setLabels", project, name)
	reqBody := &compute.GlobalSetLabelsRequest{
		Labels:           labels,
		LabelFingerprint: fingerprint,
	}
	body, err := client.Post(ctx, path, reqBody)
	if err != nil {
		return err
	}
	opName, err := operationNameFromResponse(body, "set labels")
	if err != nil {
		return err
	}
	return WaitForGlobalOperation(ctx, client, project, opName)
}

// deprecateImage issues the images.deprecate POST and waits for the global operation.
func deprecateImage(ctx context.Context, client Client, project, name, state, replacement string) error {
	path := fmt.Sprintf("projects/%s/global/images/%s/deprecate", project, name)
	status := &compute.DeprecationStatus{State: state}
	// A replacement only applies to deprecating states. Never attach it when
	// clearing deprecation (ACTIVE), where a stale value (e.g. left over from a
	// previous Deprecated selection) would be sent and rejected.
	if replacement != "" && state != ImageStateActive {
		status.Replacement = resolveImageURL(project, replacement)
	}
	body, err := client.Post(ctx, path, status)
	if err != nil {
		return err
	}
	opName, err := operationNameFromResponse(body, "deprecate")
	if err != nil {
		return err
	}
	return WaitForGlobalOperation(ctx, client, project, opName)
}

func (u *UpdateImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateImage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateImage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (u *UpdateImage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (u *UpdateImage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
