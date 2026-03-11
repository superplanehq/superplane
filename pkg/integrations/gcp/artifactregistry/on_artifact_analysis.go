package artifactregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ArtifactAnalysisEmittedEventType = "gcp.artifactregistry.artifact.analysis"
	ArtifactAnalysisSubscriptionType = "containeranalysis.occurrence"

	occurrenceKindVulnerability = "VULNERABILITY"
	occurrenceKindBuild         = "BUILD"
	occurrenceKindAttestation   = "ATTESTATION"
	occurrenceKindSBOM          = "SBOM"
)

type OnArtifactAnalysis struct{}

type OnArtifactAnalysisConfiguration struct {
	Kinds      []string `json:"kinds" mapstructure:"kinds"`
	Location   string   `json:"location" mapstructure:"location"`
	Repository string   `json:"repository" mapstructure:"repository"`
	Package    string   `json:"package" mapstructure:"package"`
}

type OnArtifactAnalysisMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

func (t *OnArtifactAnalysis) Name() string {
	return "gcp.artifactregistry.onArtifactAnalysis"
}

func (t *OnArtifactAnalysis) Label() string {
	return "Artifact Registry • On Artifact Analysis"
}

func (t *OnArtifactAnalysis) Description() string {
	return "Trigger a workflow when a Container Analysis occurrence is published for an artifact"
}

func (t *OnArtifactAnalysis) Documentation() string {
	return `The On Artifact Analysis trigger starts a workflow execution when Google Container Analysis publishes a new occurrence (e.g. vulnerability finding, build provenance, or attestation) for an artifact.

**Trigger behavior:** SuperPlane subscribes to the ` + "`container-analysis-occurrences-v1`" + ` Pub/Sub topic that Container Analysis automatically publishes to.

## Use Cases

- **Security automation**: React to new vulnerability findings for your container images
- **Compliance workflows**: Trigger policy enforcement when attestations are created
- **Build provenance**: React to new build provenance records

## Setup

**Required GCP setup:** Ensure the **Container Analysis API** (` + "`containeranalysis.googleapis.com`" + `) and **Pub/Sub API** are enabled in your project. The service account must have ` + "`roles/pubsub.admin`" + ` and ` + "`roles/containeranalysis.occurrences.viewer`" + `.

## Configuration

- **Occurrence Kinds**: Filter by occurrence type. Leave empty to receive only **DISCOVERY** occurrences (one event per completed scan — recommended). Set explicitly to receive other types such as VULNERABILITY (one event per CVE found).
- **Location / Repository / Package**: Optional filters to scope events to a specific artifact.

## Event Data

Each event contains the full Container Analysis Occurrence resource, including ` + "`kind`" + `, ` + "`resourceUri`" + `, ` + "`noteName`" + `, and the occurrence-specific data (e.g. ` + "`vulnerability`" + ` for vulnerability findings).`
}

func (t *OnArtifactAnalysis) Icon() string  { return "gcp" }
func (t *OnArtifactAnalysis) Color() string { return "gray" }

func (t *OnArtifactAnalysis) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "kinds",
			Label:       "Occurrence Kinds",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter by occurrence kind. Leave empty to receive events for all kinds.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Vulnerability", Value: occurrenceKindVulnerability},
						{Label: "Build Provenance", Value: occurrenceKindBuild},
						{Label: "Attestation", Value: occurrenceKindAttestation},
						{Label: "SBOM", Value: occurrenceKindSBOM},
					},
				},
			},
		},
		{
			Name:        "location",
			Label:       "Location",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional filter by Artifact Registry location. Leave empty to receive events for all locations.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeLocation,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional filter by repository. Leave empty to receive events for all repositories.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeRepository,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
					},
				},
			},
		},
		{
			Name:        "package",
			Label:       "Package",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional filter by package (image). Leave empty to receive events for all packages.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "location", Values: []string{"*"}},
				{Field: "repository", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePackage,
					Parameters: []configuration.ParameterRef{
						{Name: "location", ValueFrom: &configuration.ParameterValueFrom{Field: "location"}},
						{Name: "repository", ValueFrom: &configuration.ParameterValueFrom{Field: "repository"}},
					},
				},
			},
		},
	}
}

func (t *OnArtifactAnalysis) Setup(ctx core.TriggerContext) error {
	if _, err := decodeOnArtifactAnalysisConfiguration(ctx.Configuration); err != nil {
		return err
	}

	if ctx.Integration == nil {
		return fmt.Errorf("connect the GCP integration to this trigger to enable automatic event routing")
	}

	if err := scheduleArtifactRegistrySetupIfNeeded(ctx.Integration); err != nil {
		return err
	}

	subscriptionID, err := ctx.Integration.Subscribe(map[string]any{"type": ArtifactAnalysisSubscriptionType})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return ctx.Metadata.Set(OnArtifactAnalysisMetadata{
		SubscriptionID: subscriptionID.String(),
	})
}

func (t *OnArtifactAnalysis) Actions() []core.Action {
	return nil
}

func (t *OnArtifactAnalysis) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnArtifactAnalysis) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config, err := decodeOnArtifactAnalysisConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	var msg struct {
		Name string `mapstructure:"name"`
		Kind string `mapstructure:"kind"`
	}
	if err := mapstructure.Decode(ctx.Message, &msg); err != nil {
		return fmt.Errorf("failed to decode occurrence message: %w", err)
	}

	// The Pub/Sub message only contains name/kind/notificationTime.
	// Fetch the full occurrence to get resourceUri and other fields.
	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}
	occURL := fmt.Sprintf("https://containeranalysis.googleapis.com/v1/%s", msg.Name)
	body, err := client.GetURL(context.Background(), occURL)
	if err != nil {
		return fmt.Errorf("failed to fetch occurrence: %w", err)
	}
	var fullOccurrence map[string]any
	if err := json.Unmarshal(body, &fullOccurrence); err != nil {
		return fmt.Errorf("failed to parse occurrence: %w", err)
	}

	var occurrence struct {
		Kind        string `mapstructure:"kind"`
		ResourceURI string `mapstructure:"resourceUri"`
	}
	if err := mapstructure.Decode(fullOccurrence, &occurrence); err != nil {
		return fmt.Errorf("failed to decode occurrence: %w", err)
	}

	if len(config.Kinds) > 0 {
		matched := false
		for _, k := range config.Kinds {
			if strings.EqualFold(k, occurrence.Kind) {
				matched = true
				break
			}
		}
		if !matched {
			ctx.Logger.Infof("gcp artifact registry: occurrence kind %q does not match configured kinds, skipping", occurrence.Kind)
			return nil
		}
	} else {
		// Default: only emit on DISCOVERY occurrences (scan completion signal).
		// Without this, every individual vulnerability occurrence fires a separate event.
		if !strings.EqualFold(occurrence.Kind, "DISCOVERY") {
			return nil
		}
	}

	if config.Location != "" && !strings.Contains(occurrence.ResourceURI, config.Location+"-docker.pkg.dev") {
		ctx.Logger.Infof("gcp artifact registry: resource URI %q does not match location filter %q, skipping", occurrence.ResourceURI, config.Location)
		return nil
	}

	if config.Repository != "" || config.Package != "" {
		uri := strings.TrimPrefix(occurrence.ResourceURI, "https://")
		uri = strings.TrimPrefix(uri, "http://")
		// uri: LOCATION-docker.pkg.dev/PROJECT/REPOSITORY/PACKAGE@...
		parts := strings.SplitN(uri, "/", 4)
		if config.Repository != "" && (len(parts) < 3 || parts[2] != config.Repository) {
			ctx.Logger.Infof("gcp artifact registry: resource URI %q does not match repository filter %q, skipping", occurrence.ResourceURI, config.Repository)
			return nil
		}
		if config.Package != "" {
			if len(parts) < 4 {
				ctx.Logger.Infof("gcp artifact registry: resource URI %q does not contain a package path, skipping", occurrence.ResourceURI)
				return nil
			}

			imagePart := parts[3]
			if idx := strings.IndexAny(imagePart, "@:"); idx >= 0 {
				imagePart = imagePart[:idx]
			}
			if imagePart != config.Package {
				ctx.Logger.Infof("gcp artifact registry: resource URI %q does not match package filter %q, skipping", occurrence.ResourceURI, config.Package)
				return nil
			}
		}
	}

	return ctx.Events.Emit(ArtifactAnalysisEmittedEventType, fullOccurrence)
}

func (t *OnArtifactAnalysis) Cleanup(_ core.TriggerContext) error { return nil }

func (t *OnArtifactAnalysis) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func decodeOnArtifactAnalysisConfiguration(raw any) (OnArtifactAnalysisConfiguration, error) {
	var config OnArtifactAnalysisConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return OnArtifactAnalysisConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.Location = strings.TrimSpace(config.Location)
	config.Repository = strings.TrimSpace(config.Repository)
	config.Package = strings.TrimSpace(config.Package)
	return config, nil
}
