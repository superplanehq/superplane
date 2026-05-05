package oci

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	imagePayloadType           = "oci.image"
	imageDeletedPayloadType    = "oci.imageDeleted"
	imageStateAvailable        = "AVAILABLE"
	imageStateDeleted          = "DELETED"
	imageStateDisabled         = "DISABLED"
	createImagePollInterval    = 10 * time.Second
	createImageMaxPollAttempts = 180
)

type imageNodeMetadata struct {
	ImageID         string `json:"imageId,omitempty" mapstructure:"imageId"`
	ImageName       string `json:"imageName,omitempty" mapstructure:"imageName"`
	CompartmentID   string `json:"compartmentId,omitempty" mapstructure:"compartmentId"`
	CompartmentName string `json:"compartmentName,omitempty" mapstructure:"compartmentName"`
	InstanceID      string `json:"instanceId,omitempty" mapstructure:"instanceId"`
	InstanceName    string `json:"instanceName,omitempty" mapstructure:"instanceName"`
	DisplayName     string `json:"displayName,omitempty" mapstructure:"displayName"`
	SourceType      string `json:"sourceType,omitempty" mapstructure:"sourceType"`
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

func customImageIDField(required bool) configuration.Field {
	return imageResourceField(required, ResourceTypeCustomImage)
}

func imageResourceField(required bool, resourceType string) configuration.Field {
	return configuration.Field{
		Name:        "image",
		Label:       "Image",
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    required,
		Description: "OCI image OCID",
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: resourceType,
			},
		},
	}
}

func compartmentField() configuration.Field {
	return configuration.Field{
		Name:        "compartment",
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

func resolveImageNodeMetadata(ctx core.SetupContext, node imageNodeMetadata) imageNodeMetadata {
	node.ImageID = strings.TrimSpace(node.ImageID)
	node.ImageName = strings.TrimSpace(node.ImageName)
	node.CompartmentID = strings.TrimSpace(node.CompartmentID)
	node.CompartmentName = strings.TrimSpace(node.CompartmentName)
	node.InstanceID = strings.TrimSpace(node.InstanceID)
	node.InstanceName = strings.TrimSpace(node.InstanceName)
	node.DisplayName = strings.TrimSpace(node.DisplayName)
	node.SourceType = strings.TrimSpace(node.SourceType)

	if ctx.HTTP == nil || ctx.Integration == nil {
		return node
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return node
	}

	if node.ImageID != "" && node.ImageName == "" {
		if image, err := client.GetImage(node.ImageID); err == nil {
			node.ImageName = image.DisplayName
			if node.CompartmentID == "" {
				node.CompartmentID = image.CompartmentID
			}
		}
	}

	if node.InstanceID != "" && node.InstanceName == "" {
		if instance, err := client.GetInstance(node.InstanceID); err == nil {
			node.InstanceName = instance.DisplayName
		}
	}

	if node.CompartmentID != "" && node.CompartmentName == "" {
		node.CompartmentName = findCompartmentName(client, node.CompartmentID)
	}

	return node
}

func findCompartmentName(client *Client, compartmentID string) string {
	if compartmentID == "" {
		return ""
	}
	if compartmentID == client.tenancyOCID {
		return "Root (tenancy)"
	}

	compartments, err := client.ListCompartments()
	if err != nil {
		return ""
	}

	for _, compartment := range compartments {
		if compartment.ID == compartmentID {
			return compartment.Name
		}
	}

	return ""
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
	if len(image.FreeformTags) > 0 {
		out["freeformTags"] = image.FreeformTags
	}

	out["createImageAllowed"] = image.CreateImageAllowed
	return out
}

func validateImageID(imageID string) error {
	if strings.TrimSpace(imageID) == "" {
		return errors.New("image is required")
	}
	return nil
}

func isCustomImage(image Image) bool {
	return strings.TrimSpace(image.CompartmentID) != ""
}

func ensureCustomImage(client *Client, imageID string) (*Image, error) {
	return ensureCustomImageForOperation(client, imageID, "updated or deleted")
}

func ensureCustomImageForOperation(client *Client, imageID string, operation string) (*Image, error) {
	image, err := client.GetImage(imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	if !isCustomImage(*image) {
		return nil, fmt.Errorf("image %s is an OCI platform image; only custom images can be %s", imageID, operation)
	}

	return image, nil
}

func noWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func defaultProcessQueue(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
