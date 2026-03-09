package artifactregistry

import (
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
	Kinds       []string `json:"kinds" mapstructure:"kinds"`
	ResourceURI string   `json:"resourceUri" mapstructure:"resourceUri"`
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

- **Occurrence Kinds**: Optional filter by occurrence type (VULNERABILITY, BUILD, ATTESTATION, SBOM). Leave empty to receive all kinds.
- **Resource URI**: Optional filter by artifact resource URI. Leave empty to receive events for all artifacts.

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
			Name:        "resourceUri",
			Label:       "Resource URI",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional filter by artifact resource URI. Leave empty to receive events for all artifacts.",
			Placeholder: "e.g. https://us-central1-docker.pkg.dev/project/repo/image@sha256:...",
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

	var occurrence struct {
		Kind        string `mapstructure:"kind"`
		ResourceURI string `mapstructure:"resourceUri"`
	}
	if err := mapstructure.Decode(ctx.Message, &occurrence); err != nil {
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
	}

	if config.ResourceURI != "" && !strings.Contains(occurrence.ResourceURI, config.ResourceURI) {
		ctx.Logger.Infof("gcp artifact registry: resource URI %q does not match filter %q, skipping", occurrence.ResourceURI, config.ResourceURI)
		return nil
	}

	return ctx.Events.Emit(ArtifactAnalysisEmittedEventType, ctx.Message)
}

func (t *OnArtifactAnalysis) Cleanup(_ core.TriggerContext) error { return nil }

func (t *OnArtifactAnalysis) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func decodeOnArtifactAnalysisConfiguration(raw any) (OnArtifactAnalysisConfiguration, error) {
	var config OnArtifactAnalysisConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return OnArtifactAnalysisConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	config.ResourceURI = strings.TrimSpace(config.ResourceURI)
	return config, nil
}
