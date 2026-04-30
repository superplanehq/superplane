package oci

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	imagePayloadType           = "oci.image"
	imageDeletedPayloadType    = "oci.imageDeleted"
	imageStateAvailable        = "AVAILABLE"
	imageStateDeleted          = "DELETED"
	imageStateDisabled         = "DISABLED"
	imageStateImporting        = "IMPORTING"
	imageStateProvisioning     = "PROVISIONING"
	createImagePollInterval    = 10 * time.Second
	createImageMaxPollAttempts = 180
)

type imageNodeMetadata struct {
	ImageID       string `json:"imageId,omitempty" mapstructure:"imageId"`
	CompartmentID string `json:"compartmentId,omitempty" mapstructure:"compartmentId"`
	DisplayName   string `json:"displayName,omitempty" mapstructure:"displayName"`
}

type imageExecutionMetadata struct {
	ImageID       string `json:"imageId" mapstructure:"imageId"`
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	DisplayName   string `json:"displayName" mapstructure:"displayName"`
	State         string `json:"state" mapstructure:"state"`
	StartedAt     string `json:"startedAt" mapstructure:"startedAt"`
	PollErrors    int    `json:"pollErrors" mapstructure:"pollErrors"`
	PollAttempts  int    `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func imageIDField(required bool) configuration.Field {
	return configuration.Field{
		Name:        "imageId",
		Label:       "Image",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    required,
		Description: "OCI image OCID",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: ResourceTypeImage,
			},
		},
	}
}

func compartmentField() configuration.Field {
	return configuration.Field{
		Name:        "compartmentId",
		Label:       "Compartment",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: "The compartment that contains the image or source instance",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: ResourceTypeCompartment,
			},
		},
	}
}

func displayNameField(required bool) configuration.Field {
	return configuration.Field{
		Name:        "displayName",
		Label:       "Display Name",
		Type:        configuration.FieldTypeString,
		Required:    required,
		Description: "Human-readable image name",
		Placeholder: "app-golden-image",
	}
}

func trimImageNodeMetadata(config any, metadata core.MetadataWriter) error {
	var node imageNodeMetadata
	if err := mapstructure.WeakDecode(config, &node); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	node.ImageID = strings.TrimSpace(node.ImageID)
	node.CompartmentID = strings.TrimSpace(node.CompartmentID)
	node.DisplayName = strings.TrimSpace(node.DisplayName)
	return metadata.Set(node)
}

func emitImage(state core.ExecutionStateContext, image *Image) error {
	return state.Emit(core.DefaultOutputChannel.Name, imagePayloadType, []any{
		map[string]any{"image": imageToMap(image)},
	})
}

func imageToMap(image *Image) map[string]any {
	if image == nil {
		return map[string]any{}
	}

	out := map[string]any{
		"id":             image.ID,
		"displayName":    image.DisplayName,
		"lifecycleState": image.LifecycleState,
	}

	if image.CompartmentID != "" {
		out["compartmentId"] = image.CompartmentID
	}
	if image.BaseImageID != "" {
		out["baseImageId"] = image.BaseImageID
	}
	if image.OperatingSystem != "" {
		out["operatingSystem"] = image.OperatingSystem
	}
	if image.OperatingSystemVersion != "" {
		out["operatingSystemVersion"] = image.OperatingSystemVersion
	}
	if image.LaunchMode != "" {
		out["launchMode"] = image.LaunchMode
	}
	if image.SizeInMBs > 0 {
		out["sizeInMBs"] = image.SizeInMBs
	}
	if image.TimeCreated != "" {
		out["timeCreated"] = image.TimeCreated
	}

	out["createImageAllowed"] = image.CreateImageAllowed
	return out
}

func validateImageID(imageID string) error {
	if strings.TrimSpace(imageID) == "" {
		return errors.New("imageId is required")
	}
	return nil
}

func noWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func defaultProcessQueue(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
