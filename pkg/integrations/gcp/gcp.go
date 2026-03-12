package gcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/artifactregistry"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/cloudbuild"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/cloudfunctions"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/compute"
	gcppubsub "github.com/superplanehq/superplane/pkg/integrations/gcp/pubsub"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("gcp", &GCP{})
	compute.SetClientFactory(func(ctx core.ExecutionContext) (compute.Client, error) {
		return gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	})
	cloudbuild.SetClientFactory(func(httpCtx core.HTTPContext, integration core.IntegrationContext) (cloudbuild.Client, error) {
		return gcpcommon.NewClient(httpCtx, integration)
	})
	cloudfunctions.SetClientFactory(func(httpCtx core.HTTPContext, integration core.IntegrationContext) (cloudfunctions.Client, error) {
		return gcpcommon.NewClient(httpCtx, integration)
	})
	artifactregistry.SetClientFactory(func(httpCtx core.HTTPContext, integration core.IntegrationContext) (artifactregistry.Client, error) {
		return gcpcommon.NewClient(httpCtx, integration)
	})
}

type GCP struct{}

const (
	ConnectionMethodServiceAccountKey = "serviceAccountKey"
	ConnectionMethodWIF               = "workloadIdentityFederation"

	PubSubSecretName            = "pubsub.events.secret"
	CloudBuildSecretName        = "cloudbuild.events.secret"
	ArtifactPushSecretName      = "artifactregistry.push.secret"
	ContainerAnalysisSecretName = "containeranalysis.occurrences.secret"
	CloudBuildTopicID           = "cloud-builds"
	ArtifactPushTopicID         = "gcr"
	ContainerAnalysisTopicID    = "container-analysis-occurrences-v1"
)

type Configuration struct {
	ConnectionMethod          string `json:"connectionMethod" mapstructure:"connectionMethod"`
	ServiceAccountKey         string `json:"serviceAccountKey" mapstructure:"serviceAccountKey"`
	WorkloadIdentityProvider  string `json:"workloadIdentityProvider" mapstructure:"workloadIdentityProvider"`
	WorkloadIdentityProjectID string `json:"workloadIdentityProjectId" mapstructure:"workloadIdentityProjectId"`
}

func (g *GCP) Name() string {
	return "gcp"
}

func (g *GCP) Label() string {
	return "Google Cloud"
}

func (g *GCP) Icon() string {
	return "gcp"
}

func (g *GCP) Description() string {
	return "Manage and use Google Cloud resources in your workflows"
}

func (g *GCP) Instructions() string {
	return `## Connection method

### Service Account Key

1. Go to [IAM & Admin → Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts) in the Google Cloud Console.
2. Select a service account → **Keys** → **Add Key** → **JSON**.
3. Paste the downloaded JSON below.

### Workload Identity Federation (keyless)

1. Create a [Workload Identity Pool](https://cloud.google.com/iam/docs/workload-identity-federation) with an OIDC provider.
2. Set the **Issuer URL** to this SuperPlane instance's URL.
3. Set the **Audience** to the pool provider resource name.
4. Grant the federated identity permission to [impersonate a service account](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-providers#mapping) with the roles your workflows need.
5. Enter the **pool provider resource name** and **Project ID** below.

## Required IAM roles

- ` + "`roles/logging.configWriter`" + ` — create logging sinks for event triggers
- ` + "`roles/pubsub.admin`" + ` — manage Pub/Sub topics, subscriptions, and IAM policies for event delivery
- Additional roles depending on which components you use (e.g. ` + "`roles/compute.admin`" + ` for VM management)`
}

func (g *GCP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "connectionMethod",
			Label:       "Connection method",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Authenticate with a service account key (JSON) or Workload Identity Federation (keyless).",
			Default:     ConnectionMethodServiceAccountKey,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Service Account Key", Value: ConnectionMethodServiceAccountKey},
						{Label: "Workload Identity Federation", Value: ConnectionMethodWIF},
					},
				},
			},
		},
		{
			Name:        "serviceAccountKey",
			Label:       "Service Account Key (JSON)",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Paste the full contents of your GCP service account JSON key file",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "connectionMethod", Values: []string{ConnectionMethodServiceAccountKey}},
			},
		},
		{
			Name:        "workloadIdentityProvider",
			Label:       "Workload Identity Pool Provider Resource Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full resource name of the OIDC provider. Must match the audience configured in the provider.",
			Placeholder: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/superplane",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "connectionMethod", Values: []string{ConnectionMethodWIF}},
			},
		},
		{
			Name:        "workloadIdentityProjectId",
			Label:       "Project ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "GCP project ID",
			Placeholder: "e.g. my-project",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "connectionMethod", Values: []string{ConnectionMethodWIF}},
			},
		},
	}
}

func (g *GCP) Components() []core.Component {
	return []core.Component{
		&compute.CreateVM{},
		&cloudbuild.CreateBuild{},
		&cloudbuild.GetBuild{},
		&cloudbuild.RunTrigger{},
		&cloudfunctions.InvokeFunction{},
		&artifactregistry.GetArtifact{},
		&artifactregistry.GetArtifactAnalysis{},
		&gcppubsub.PublishMessage{},
		&gcppubsub.CreateTopicComponent{},
		&gcppubsub.DeleteTopicComponent{},
		&gcppubsub.CreateSubscriptionComponent{},
		&gcppubsub.DeleteSubscriptionComponent{},
	}
}

func (g *GCP) Triggers() []core.Trigger {
	return []core.Trigger{
		&compute.OnVMInstance{},
		&cloudbuild.OnBuildComplete{},
		&artifactregistry.OnArtifactPush{},
		&artifactregistry.OnArtifactAnalysis{},
		&gcppubsub.OnMessage{},
	}
}

func (g *GCP) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	switch strings.TrimSpace(config.ConnectionMethod) {
	case ConnectionMethodServiceAccountKey:
		return g.syncServiceAccountKey(ctx, config)
	case ConnectionMethodWIF:
		return g.syncWIF(ctx, config)
	default:
		return fmt.Errorf("unknown connection method: %s", config.ConnectionMethod)
	}
}

func (g *GCP) syncWIF(ctx core.SyncContext, config Configuration) error {
	provider := strings.TrimSpace(config.WorkloadIdentityProvider)
	if provider == "" {
		return fmt.Errorf("Workload Identity Pool provider resource name is required")
	}
	projectID := strings.TrimSpace(config.WorkloadIdentityProjectID)
	if projectID == "" {
		return fmt.Errorf("Project ID is required for Workload Identity Federation")
	}

	subject := fmt.Sprintf("app-installation:%s", ctx.Integration.ID())
	oidcToken, err := ctx.OIDC.Sign(subject, 5*time.Minute, provider, nil)
	if err != nil {
		return fmt.Errorf("failed to generate OIDC token: %w", err)
	}

	callCtx := context.Background()
	accessToken, expiresIn, err := ExchangeToken(callCtx, ctx.HTTP, oidcToken, provider)
	if err != nil {
		return fmt.Errorf("Workload Identity Federation token exchange failed. Ensure your SuperPlane instance URL is set as the OIDC issuer in GCP, the audience matches the provider resource name, and the URL is reachable by Google: %w", err)
	}

	if err := ctx.Integration.SetSecret(gcpcommon.SecretNameAccessToken, []byte(accessToken)); err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}

	expiresAt := time.Now().Add(expiresIn)
	refreshAfter := expiresIn / 2
	if refreshAfter < time.Minute {
		refreshAfter = time.Minute
	}

	metadata := gcpcommon.Metadata{
		ProjectID:            projectID,
		ClientEmail:          "",
		AuthMethod:           gcpcommon.AuthMethodWIF,
		AccessTokenExpiresAt: expiresAt.Format(time.RFC3339),
	}
	ctx.Integration.SetMetadata(metadata)

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client after token exchange: %w", err)
	}
	crmURL := fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", projectID)
	if _, err := client.GetURL(callCtx, crmURL); err != nil {
		return fmt.Errorf("connection failed. Ensure the 'Cloud Resource Manager API' is enabled and the federated identity has 'Viewer' (or equivalent) on the project: %w", err)
	}

	if err := g.configurePubSub(ctx, client, &metadata); err != nil {
		return fmt.Errorf("failed to configure Pub/Sub event bus: %w", err)
	}
	if err := g.configureCloudBuild(ctx, client, &metadata); err != nil {
		ctx.Logger.Warnf("failed to configure Cloud Build subscription: %v", err)
	}
	if err := g.configureArtifactRegistry(ctx, client, &metadata); err != nil {
		ctx.Logger.Warnf("failed to configure Artifact Registry subscription: %v", err)
	}
	ctx.Integration.SetMetadata(metadata)

	if err := ctx.Integration.ScheduleResync(refreshAfter); err != nil {
		ctx.Logger.Warnf("could not schedule GCP WIF resync: %v", err)
	}
	ctx.Integration.Ready()
	return nil
}

func (g *GCP) syncServiceAccountKey(ctx core.SyncContext, config Configuration) error {
	keyJSON, err := ctx.Integration.GetConfig("serviceAccountKey")
	if err != nil {
		return fmt.Errorf("failed to read service account key: %w", err)
	}

	if len(keyJSON) == 0 {
		return fmt.Errorf("service account key is required")
	}

	metadata, err := validateAndParseServiceAccountKey(keyJSON)
	if err != nil {
		return fmt.Errorf("invalid service account key: %w", err)
	}
	metadata.AuthMethod = gcpcommon.AuthMethodServiceAccountKey

	if err := ctx.Integration.SetSecret(gcpcommon.SecretNameServiceAccountKey, keyJSON); err != nil {
		return fmt.Errorf("failed to store service account key: %w", err)
	}

	ctx.Integration.SetMetadata(metadata)
	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	crmURL := fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", metadata.ProjectID)
	if _, err := client.GetURL(context.Background(), crmURL); err != nil {
		return fmt.Errorf("connection failed. Ensure the 'Cloud Resource Manager API' is enabled on your project and the service account has 'Viewer' permissions: %w", err)
	}

	if err := g.configurePubSub(ctx, client, &metadata); err != nil {
		return fmt.Errorf("failed to configure Pub/Sub event bus: %w", err)
	}
	if err := g.configureCloudBuild(ctx, client, &metadata); err != nil {
		ctx.Logger.Warnf("failed to configure Cloud Build subscription: %v", err)
	}
	if err := g.configureArtifactRegistry(ctx, client, &metadata); err != nil {
		ctx.Logger.Warnf("failed to configure Artifact Registry subscription: %v", err)
	}
	ctx.Integration.SetMetadata(metadata)

	ctx.Integration.Ready()
	return nil
}

func validateAndParseServiceAccountKey(keyJSON []byte) (gcpcommon.Metadata, error) {
	var raw map[string]any
	if err := json.Unmarshal(keyJSON, &raw); err != nil {
		return gcpcommon.Metadata{}, fmt.Errorf("invalid JSON: %w", err)
	}

	var projectID, clientEmail string

	if v, ok := raw["project_id"].(string); ok {
		projectID = strings.TrimSpace(v)
	}

	if v, ok := raw["client_email"].(string); ok {
		clientEmail = strings.TrimSpace(v)
	}

	if projectID == "" {
		return gcpcommon.Metadata{}, fmt.Errorf("missing required field project_id in service account key")
	}

	if clientEmail == "" {
		return gcpcommon.Metadata{}, fmt.Errorf("missing required field client_email in service account key")
	}

	return gcpcommon.Metadata{
		ProjectID:   projectID,
		ClientEmail: clientEmail,
	}, nil
}

func (g *GCP) configurePubSub(ctx core.SyncContext, client *gcpcommon.Client, metadata *gcpcommon.Metadata) error {
	if metadata.PubSubTopic != "" {
		secret, err := g.eventsSecret(ctx.Integration)
		if err != nil {
			return fmt.Errorf("generate events secret: %w", err)
		}
		pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/events?token=%s", ctx.WebhooksBaseURL, ctx.Integration.ID(), secret)
		return gcppubsub.UpdatePushEndpoint(context.Background(), client, client.ProjectID(), metadata.PubSubSubscription, pushEndpoint)
	}

	projectID := client.ProjectID()
	reqCtx := context.Background()

	enabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "pubsub.googleapis.com")
	if err != nil {
		return fmt.Errorf("check Pub/Sub API: %w", err)
	}
	if !enabled {
		return fmt.Errorf("Pub/Sub API is not enabled in project %s. Enable it at https://console.cloud.google.com/apis/library/pubsub.googleapis.com?project=%s", projectID, projectID)
	}

	secret, err := g.eventsSecret(ctx.Integration)
	if err != nil {
		return fmt.Errorf("generate events secret: %w", err)
	}

	sanitized := sanitizeID(ctx.Integration.ID().String())
	topicID := "sp-events-" + sanitized
	subscriptionID := "sp-sub-" + sanitized

	if err := gcppubsub.CreateTopic(reqCtx, client, projectID, topicID); err != nil {
		return fmt.Errorf("create Pub/Sub topic: %w", err)
	}

	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/events?token=%s", ctx.WebhooksBaseURL, ctx.Integration.ID(), secret)
	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, subscriptionID, topicID, pushEndpoint); err != nil {
		return fmt.Errorf("create Pub/Sub push subscription: %w", err)
	}

	ctx.Logger.Infof("Created Pub/Sub topic %s and subscription %s for event routing", topicID, subscriptionID)

	metadata.PubSubTopic = topicID
	metadata.PubSubSubscription = subscriptionID
	return nil
}

func (g *GCP) configureCloudBuild(ctx core.SyncContext, client *gcpcommon.Client, metadata *gcpcommon.Metadata) error {
	return g.ensureCloudBuildSetup(context.Background(), client, ctx.Integration, ctx.WebhooksBaseURL, metadata)
}

func (g *GCP) configureArtifactRegistry(ctx core.SyncContext, client *gcpcommon.Client, metadata *gcpcommon.Metadata) error {
	return g.ensureArtifactRegistrySetup(context.Background(), client, ctx.Integration, ctx.WebhooksBaseURL, metadata)
}

func (g *GCP) ensureCloudBuildSetup(
	reqCtx context.Context,
	client *gcpcommon.Client,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	metadata *gcpcommon.Metadata,
) error {
	projectID := client.ProjectID()

	if metadata.CloudBuildSubscription != "" {
		secret, err := g.cloudBuildSecret(integration)
		if err != nil {
			return fmt.Errorf("generate cloud build secret: %w", err)
		}
		pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/cloud-build-events?token=%s", webhooksBaseURL, integration.ID(), secret)
		return gcppubsub.UpdatePushEndpoint(reqCtx, client, projectID, metadata.CloudBuildSubscription, pushEndpoint)
	}

	enabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "pubsub.googleapis.com")
	if err != nil {
		return fmt.Errorf("check Pub/Sub API: %w", err)
	}
	if !enabled {
		return fmt.Errorf(
			"Pub/Sub API is not enabled in project %s. Enable it at https://console.cloud.google.com/apis/library/pubsub.googleapis.com?project=%s",
			projectID,
			projectID,
		)
	}

	enabled, err = gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "cloudbuild.googleapis.com")
	if err != nil {
		return fmt.Errorf("check Cloud Build API: %w", err)
	}
	if !enabled {
		return fmt.Errorf(
			"Cloud Build API is not enabled in project %s. Enable it at https://console.cloud.google.com/apis/library/cloudbuild.googleapis.com?project=%s",
			projectID,
			projectID,
		)
	}

	secret, err := g.cloudBuildSecret(integration)
	if err != nil {
		return fmt.Errorf("generate cloud build secret: %w", err)
	}

	sanitized := sanitizeID(integration.ID().String())
	subscriptionID := "sp-cb-sub-" + sanitized
	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/cloud-build-events?token=%s", webhooksBaseURL, integration.ID(), secret)

	if err := gcppubsub.CreateTopic(reqCtx, client, projectID, CloudBuildTopicID); err != nil {
		return fmt.Errorf("create Cloud Build topic: %w", err)
	}

	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, subscriptionID, CloudBuildTopicID, pushEndpoint); err != nil {
		return fmt.Errorf("create Cloud Build push subscription: %w", err)
	}

	metadata.CloudBuildSubscription = subscriptionID
	return nil
}

func (g *GCP) ensureArtifactRegistrySetup(
	reqCtx context.Context,
	client *gcpcommon.Client,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	metadata *gcpcommon.Metadata,
) error {
	projectID := client.ProjectID()

	if metadata.ArtifactPushSubscription != "" {
		synced, err := g.syncArtifactRegistrySubscriptions(reqCtx, client, integration, webhooksBaseURL, metadata, projectID)
		if err != nil {
			return err
		}
		if synced {
			return nil
		}
	}

	return g.bootstrapArtifactRegistrySubscriptions(reqCtx, client, integration, webhooksBaseURL, metadata, projectID)
}

func (g *GCP) syncArtifactRegistrySubscriptions(
	reqCtx context.Context,
	client *gcpcommon.Client,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	metadata *gcpcommon.Metadata,
	projectID string,
) (bool, error) {
	secret, err := g.artifactPushSecret(integration)
	if err != nil {
		return false, fmt.Errorf("generate artifact push secret: %w", err)
	}

	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/artifact-push-events?token=%s", webhooksBaseURL, integration.ID(), secret)
	updateErr := gcppubsub.UpdatePushEndpoint(reqCtx, client, projectID, metadata.ArtifactPushSubscription, pushEndpoint)
	if updateErr != nil {
		if !gcpcommon.IsNotFoundError(updateErr) {
			return false, fmt.Errorf("update artifact push endpoint: %w", updateErr)
		}
		// Subscription no longer exists in GCP — recreate everything.
		metadata.ArtifactPushSubscription = ""
		metadata.ContainerAnalysisSubscription = ""
		return false, nil
	}

	if metadata.ContainerAnalysisSubscription != "" {
		caSecret, err := g.containerAnalysisSecret(integration)
		if err != nil {
			return false, fmt.Errorf("generate container analysis secret: %w", err)
		}
		caPushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/artifact-analysis-events?token=%s", webhooksBaseURL, integration.ID(), caSecret)
		caUpdateErr := gcppubsub.UpdatePushEndpoint(reqCtx, client, projectID, metadata.ContainerAnalysisSubscription, caPushEndpoint)
		if caUpdateErr == nil {
			return true, nil
		}
		if !gcpcommon.IsNotFoundError(caUpdateErr) {
			return false, fmt.Errorf("update container analysis endpoint: %w", caUpdateErr)
		}
		// Subscription no longer exists in GCP — recreate it below.
		metadata.ContainerAnalysisSubscription = ""
	}

	caEnabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "containeranalysis.googleapis.com")
	if err != nil {
		return false, fmt.Errorf("check Container Analysis API: %w", err)
	}
	if !caEnabled {
		return true, nil
	}

	sanitized := sanitizeID(integration.ID().String())
	caSubscriptionID := "sp-ca-sub-" + sanitized
	if err := g.createContainerAnalysisSubscription(reqCtx, client, projectID, integration, webhooksBaseURL, caSubscriptionID); err != nil {
		return false, err
	}
	metadata.ContainerAnalysisSubscription = caSubscriptionID
	return true, nil
}

func (g *GCP) bootstrapArtifactRegistrySubscriptions(
	reqCtx context.Context,
	client *gcpcommon.Client,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	metadata *gcpcommon.Metadata,
	projectID string,
) error {
	enabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "pubsub.googleapis.com")
	if err != nil {
		return fmt.Errorf("check Pub/Sub API: %w", err)
	}
	if !enabled {
		return fmt.Errorf(
			"Pub/Sub API is not enabled in project %s. Enable it at https://console.cloud.google.com/apis/library/pubsub.googleapis.com?project=%s",
			projectID,
			projectID,
		)
	}

	enabled, err = gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "artifactregistry.googleapis.com")
	if err != nil {
		return fmt.Errorf("check Artifact Registry API: %w", err)
	}
	if !enabled {
		return fmt.Errorf(
			"Artifact Registry API is not enabled in project %s. Enable it at https://console.cloud.google.com/apis/library/artifactregistry.googleapis.com?project=%s",
			projectID,
			projectID,
		)
	}

	sanitized := sanitizeID(integration.ID().String())

	arSecret, err := g.artifactPushSecret(integration)
	if err != nil {
		return fmt.Errorf("generate artifact push secret: %w", err)
	}
	arSubscriptionID := "sp-ar-sub-" + sanitized
	arPushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/artifact-push-events?token=%s", webhooksBaseURL, integration.ID(), arSecret)

	if err := gcppubsub.CreateTopic(reqCtx, client, projectID, ArtifactPushTopicID); err != nil {
		return fmt.Errorf("create Artifact Registry gcr topic: %w", err)
	}
	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, arSubscriptionID, ArtifactPushTopicID, arPushEndpoint); err != nil {
		return fmt.Errorf("create Artifact Registry push subscription: %w", err)
	}
	metadata.ArtifactPushSubscription = arSubscriptionID

	caEnabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "containeranalysis.googleapis.com")
	if err != nil {
		return fmt.Errorf("check Container Analysis API: %w", err)
	}
	if !caEnabled {
		return nil
	}

	caSubscriptionID := "sp-ca-sub-" + sanitized
	if err := g.createContainerAnalysisSubscription(reqCtx, client, projectID, integration, webhooksBaseURL, caSubscriptionID); err != nil {
		return err
	}
	metadata.ContainerAnalysisSubscription = caSubscriptionID

	return nil
}

func (g *GCP) createContainerAnalysisSubscription(
	reqCtx context.Context,
	client *gcpcommon.Client,
	projectID string,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	subscriptionID string,
) error {
	caSecret, err := g.containerAnalysisSecret(integration)
	if err != nil {
		return fmt.Errorf("generate container analysis secret: %w", err)
	}
	caPushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/artifact-analysis-events?token=%s", webhooksBaseURL, integration.ID(), caSecret)

	if err := gcppubsub.CreateTopic(reqCtx, client, projectID, ContainerAnalysisTopicID); err != nil {
		return fmt.Errorf("create Container Analysis topic: %w", err)
	}
	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, subscriptionID, ContainerAnalysisTopicID, caPushEndpoint); err != nil {
		return fmt.Errorf("create Container Analysis push subscription: %w", err)
	}

	return nil
}

func (g *GCP) cloudBuildSecret(integration core.IntegrationContext) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, s := range secrets {
		if s.Name == CloudBuildSecretName {
			return string(s.Value), nil
		}
	}

	secret, err := crypto.Base64String(32)
	if err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}

	if err := integration.SetSecret(CloudBuildSecretName, []byte(secret)); err != nil {
		return "", fmt.Errorf("store cloud build secret: %w", err)
	}
	return secret, nil
}

func (g *GCP) artifactPushSecret(integration core.IntegrationContext) (string, error) {
	return g.getOrCreateSecret(integration, ArtifactPushSecretName)
}

func (g *GCP) containerAnalysisSecret(integration core.IntegrationContext) (string, error) {
	return g.getOrCreateSecret(integration, ContainerAnalysisSecretName)
}

func (g *GCP) getOrCreateSecret(integration core.IntegrationContext, secretName string) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, s := range secrets {
		if s.Name == secretName {
			return string(s.Value), nil
		}
	}

	secret, err := crypto.Base64String(32)
	if err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}

	if err := integration.SetSecret(secretName, []byte(secret)); err != nil {
		return "", fmt.Errorf("store secret %s: %w", secretName, err)
	}
	return secret, nil
}

func (g *GCP) eventsSecret(integration core.IntegrationContext) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, s := range secrets {
		if s.Name == PubSubSecretName {
			return string(s.Value), nil
		}
	}

	secret, err := crypto.Base64String(32)
	if err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}

	if err := integration.SetSecret(PubSubSecretName, []byte(secret)); err != nil {
		return "", fmt.Errorf("store events secret: %w", err)
	}
	return secret, nil
}

func sanitizeID(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		}
	}
	result := b.String()
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}

func (g *GCP) Cleanup(ctx core.IntegrationCleanupContext) error {
	var m gcpcommon.Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &m); err != nil || m.ProjectID == "" {
		return nil
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("failed to create GCP client for cleanup: %v", err)
		return nil
	}

	reqCtx := context.Background()
	if m.PubSubSubscription != "" {
		if err := gcppubsub.DeleteSubscription(reqCtx, client, m.ProjectID, m.PubSubSubscription); err != nil {
			if !gcpcommon.IsNotFoundError(err) {
				ctx.Logger.Warnf("failed to delete Pub/Sub subscription %s: %v", m.PubSubSubscription, err)
			}
		}
	}
	if m.PubSubTopic != "" {
		if err := gcppubsub.DeleteTopic(reqCtx, client, m.ProjectID, m.PubSubTopic); err != nil {
			if !gcpcommon.IsNotFoundError(err) {
				ctx.Logger.Warnf("failed to delete Pub/Sub topic %s: %v", m.PubSubTopic, err)
			}
		}
	}
	if m.CloudBuildSubscription != "" {
		if err := gcppubsub.DeleteSubscription(reqCtx, client, m.ProjectID, m.CloudBuildSubscription); err != nil {
			if !gcpcommon.IsNotFoundError(err) {
				ctx.Logger.Warnf("failed to delete Cloud Build subscription %s: %v", m.CloudBuildSubscription, err)
			}
		}
	}
	if m.ArtifactPushSubscription != "" {
		if err := gcppubsub.DeleteSubscription(reqCtx, client, m.ProjectID, m.ArtifactPushSubscription); err != nil {
			if !gcpcommon.IsNotFoundError(err) {
				ctx.Logger.Warnf("failed to delete Artifact Registry push subscription %s: %v", m.ArtifactPushSubscription, err)
			}
		}
	}
	if m.ContainerAnalysisSubscription != "" {
		if err := gcppubsub.DeleteSubscription(reqCtx, client, m.ProjectID, m.ContainerAnalysisSubscription); err != nil {
			if !gcpcommon.IsNotFoundError(err) {
				ctx.Logger.Warnf("failed to delete Container Analysis subscription %s: %v", m.ContainerAnalysisSubscription, err)
			}
		}
	}

	return nil
}

func (g *GCP) Actions() []core.Action {
	return []core.Action{
		{Name: gcpcommon.ActionNameEnsureCloudBuild},
		{Name: gcpcommon.ActionNameEnsureArtifactRegistry},
		{Name: gcpcommon.ActionNameEnsurePubSubOnMessage},
	}
}

func (g *GCP) HandleAction(ctx core.IntegrationActionContext) error {
	switch ctx.Name {
	case gcpcommon.ActionNameEnsureCloudBuild:
		return g.handleEnsureCloudBuild(ctx)
	case gcpcommon.ActionNameEnsureArtifactRegistry:
		return g.handleEnsureArtifactRegistry(ctx)
	case gcpcommon.ActionNameEnsurePubSubOnMessage:
		return g.handleEnsurePubSubOnMessage(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (g *GCP) handleEnsurePubSubOnMessage(ctx core.IntegrationActionContext) error {
	var params struct {
		Topic      string `mapstructure:"topic"`
		GCPSubName string `mapstructure:"gcpSubName"`
	}
	if err := mapstructure.Decode(ctx.Parameters, &params); err != nil {
		return fmt.Errorf("failed to decode action params: %w", err)
	}
	if params.Topic == "" || params.GCPSubName == "" {
		return fmt.Errorf("topic and gcpSubName are required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	projectID := client.ProjectID()
	secret, err := g.eventsSecret(ctx.Integration)
	if err != nil {
		return fmt.Errorf("get events secret: %w", err)
	}

	reqCtx := context.Background()

	// Delete existing subscription (handles topic changes and idempotency)
	_ = gcppubsub.DeleteSubscription(reqCtx, client, projectID, params.GCPSubName)

	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/pubsub-events?token=%s&gcpSubName=%s",
		ctx.WebhooksBaseURL, ctx.Integration.ID(), secret, params.GCPSubName)

	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, params.GCPSubName, params.Topic, pushEndpoint); err != nil {
		return fmt.Errorf("create push subscription on topic %q: %w", params.Topic, err)
	}

	ctx.Logger.Infof("Created Pub/Sub push subscription %s on topic %s", params.GCPSubName, params.Topic)
	return nil
}

func (g *GCP) handleEnsureCloudBuild(ctx core.IntegrationActionContext) error {
	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	var metadata gcpcommon.Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if err := g.ensureCloudBuildSetup(context.Background(), client, ctx.Integration, ctx.WebhooksBaseURL, &metadata); err != nil {
		return err
	}

	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (g *GCP) handleEnsureArtifactRegistry(ctx core.IntegrationActionContext) error {
	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	var metadata gcpcommon.Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if err := g.ensureArtifactRegistrySetup(context.Background(), client, ctx.Integration, ctx.WebhooksBaseURL, &metadata); err != nil {
		return err
	}

	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (g *GCP) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}
	reqCtx := context.Background()

	p := ctx.Parameters

	switch resourceType {
	case cloudfunctions.ResourceTypeLocation, cloudfunctions.ResourceTypeFunction:
		projectID := p["projectId"]
		if projectID == "" {
			projectID = client.ProjectID()
		}
		cfEnabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "cloudfunctions.googleapis.com")
		if err != nil {
			return nil, fmt.Errorf("failed to check Cloud Functions API status: %w", err)
		}
		crEnabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "run.googleapis.com")
		if err != nil {
			return nil, fmt.Errorf("failed to check Cloud Run API status: %w", err)
		}
		if !cfEnabled && !crEnabled {
			return nil, fmt.Errorf("Neither Cloud Functions nor Cloud Run API is enabled in project %s", projectID)
		}
		if resourceType == cloudfunctions.ResourceTypeLocation {
			return cloudfunctions.ListLocationResources(reqCtx, client, p["projectId"])
		}
		return cloudfunctions.ListFunctionResources(reqCtx, client, p["projectId"], p["location"])
	case compute.ResourceTypeRegion:
		return compute.ListRegionResources(reqCtx, client)
	case compute.ResourceTypeZone:
		return compute.ListZoneResources(reqCtx, client, p["region"])
	case compute.ResourceTypeMachineFamily:
		return compute.ListMachineFamilyResources(reqCtx, client, p["zone"])
	case compute.ResourceTypeMachineType:
		return compute.ListMachineTypeResources(reqCtx, client, p["zone"], p["machineFamily"])
	case compute.ResourceTypePublicImages:
		return compute.ListPublicImageResources(reqCtx, client, p["project"])
	case compute.ResourceTypeCustomImages:
		return compute.ListCustomImageResources(reqCtx, client, p["project"])
	case compute.ResourceTypeSnapshots:
		return compute.ListSnapshotResources(reqCtx, client, p["project"])
	case compute.ResourceTypeDisks:
		return compute.ListDiskResources(reqCtx, client, p["project"], p["zone"])
	case compute.ResourceTypeDiskTypes:
		return compute.ListDiskTypeResources(reqCtx, client, p["project"], p["zone"], p["bootDiskOnly"] == "true")
	case compute.ResourceTypeSnapshotSchedules:
		return compute.ListSnapshotScheduleResources(reqCtx, client, p["project"], p["region"])
	case compute.ResourceTypeNetwork:
		return compute.ListNetworkResources(reqCtx, client, p["project"])
	case compute.ResourceTypeSubnetwork:
		return compute.ListSubnetworkResources(reqCtx, client, p["project"], p["region"])
	case compute.ResourceTypeAddress:
		return compute.ListAddressResources(reqCtx, client, p["project"], p["region"])
	case compute.ResourceTypeFirewall:
		return compute.ListFirewallResources(reqCtx, client, p["project"])
	case cloudbuild.ResourceTypeTrigger:
		return cloudbuild.ListTriggerResources(reqCtx, client, p["projectId"])
	case cloudbuild.ResourceTypeBuild:
		return cloudbuild.ListBuildResources(reqCtx, client, p["projectId"])
	case cloudbuild.ResourceTypeLocation:
		return cloudbuild.ListLocationResources(reqCtx, client, p["projectId"])
	case cloudbuild.ResourceTypeConnection:
		return cloudbuild.ListConnectionResources(reqCtx, client, p["projectId"], p["location"])
	case cloudbuild.ResourceTypeRepository:
		return cloudbuild.ListRepositoryResources(reqCtx, client, p["connection"])
	case cloudbuild.ResourceTypeBranch:
		return cloudbuild.ListBranchResources(reqCtx, client, p["repository"])
	case cloudbuild.ResourceTypeTag:
		return cloudbuild.ListTagResources(reqCtx, client, p["repository"])
	case artifactregistry.ResourceTypeLocation:
		return artifactregistry.ListLocationResources(reqCtx, client, p["projectId"])
	case artifactregistry.ResourceTypeRepository:
		return artifactregistry.ListRepositoryResources(reqCtx, client, p["projectId"], p["location"])
	case artifactregistry.ResourceTypePackage:
		return artifactregistry.ListPackageResources(reqCtx, client, p["projectId"], p["location"], p["repository"])
	case artifactregistry.ResourceTypeVersion:
		return artifactregistry.ListVersionResources(reqCtx, client, p["projectId"], p["location"], p["repository"], p["package"])
	case gcppubsub.ResourceTypeTopic:
		return gcppubsub.ListTopicResources(reqCtx, client)
	case gcppubsub.ResourceTypeSubscription:
		return gcppubsub.ListSubscriptionResources(reqCtx, client, p["topic"])
	default:
		return nil, nil
	}
}

func (g *GCP) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		g.handleEvent(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/cloud-build-events") {
		g.handleCloudBuildEvent(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/artifact-push-events") {
		g.handleArtifactPushEvent(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/artifact-analysis-events") {
		g.handleArtifactAnalysisEvent(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/pubsub-events") {
		g.handlePubSubEvent(ctx)
		return
	}

	ctx.Response.WriteHeader(http.StatusNotFound)
}

// AuditLogEvent is the normalized event structure extracted from a Cloud Logging
// audit log entry, used both for subscription pattern matching and as the message
// payload delivered to triggers via OnIntegrationMessage.
type AuditLogEvent struct {
	ServiceName  string `json:"serviceName" mapstructure:"serviceName"`
	MethodName   string `json:"methodName" mapstructure:"methodName"`
	ResourceName string `json:"resourceName" mapstructure:"resourceName"`
	LogName      string `json:"logName" mapstructure:"logName"`
	Timestamp    string `json:"timestamp" mapstructure:"timestamp"`
	InsertID     string `json:"insertId" mapstructure:"insertId"`
	Data         any    `json:"data" mapstructure:"data"`
}

// AuditLogEventPattern is the subscription pattern used to match incoming events
// against trigger subscriptions. Only non-empty fields are matched.
type AuditLogEventPattern struct {
	ServiceName string `json:"serviceName" mapstructure:"serviceName"`
	MethodName  string `json:"methodName" mapstructure:"methodName"`
}

type pubsubPushMessage struct {
	Message struct {
		Data        string            `json:"data"`
		MessageID   string            `json:"messageId"`
		PublishTime string            `json:"publishTime"`
		Attributes  map[string]string `json:"attributes"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

type logEntryProtoPayload struct {
	ServiceName  string `json:"serviceName"`
	MethodName   string `json:"methodName"`
	ResourceName string `json:"resourceName"`
}

type logEntry struct {
	ProtoPayload logEntryProtoPayload `json:"protoPayload"`
	LogName      string               `json:"logName"`
	Timestamp    string               `json:"timestamp"`
	InsertID     string               `json:"insertId"`
}

func (g *GCP) handleEvent(ctx core.HTTPRequestContext) {
	token := ctx.Request.URL.Query().Get("token")
	if token == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var secret string
	for _, s := range secrets {
		if s.Name == PubSubSecretName {
			secret = string(s.Value)
			break
		}
	}

	if token != secret {
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var pushMsg pubsubPushMessage
	if err := json.Unmarshal(body, &pushMsg); err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	decoded, err := base64Decode(pushMsg.Message.Data)
	if err != nil {
		ctx.Logger.Warnf("failed to decode Pub/Sub message data: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	var entry logEntry
	if err := json.Unmarshal(decoded, &entry); err != nil {
		ctx.Logger.Warnf("failed to parse log entry: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	var rawData map[string]any
	_ = json.Unmarshal(decoded, &rawData)

	event := AuditLogEvent{
		ServiceName:  entry.ProtoPayload.ServiceName,
		MethodName:   strings.TrimSpace(entry.ProtoPayload.MethodName),
		ResourceName: entry.ProtoPayload.ResourceName,
		LogName:      entry.LogName,
		Timestamp:    entry.Timestamp,
		InsertID:     entry.InsertID,
		Data:         rawData,
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if !g.subscriptionApplies(subscription, event) {
			continue
		}

		if err := subscription.SendMessage(event); err != nil {
			ctx.Logger.Errorf("error sending message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (g *GCP) subscriptionApplies(subscription core.IntegrationSubscriptionContext, event AuditLogEvent) bool {
	var pattern AuditLogEventPattern
	if err := mapstructure.Decode(subscription.Configuration(), &pattern); err != nil {
		return false
	}

	if pattern.ServiceName != "" && pattern.ServiceName != event.ServiceName {
		return false
	}

	if pattern.MethodName != "" && pattern.MethodName != event.MethodName {
		return false
	}

	return true
}

func (g *GCP) handleCloudBuildEvent(ctx core.HTTPRequestContext) {
	token := ctx.Request.URL.Query().Get("token")
	if token == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var secret string
	for _, s := range secrets {
		if s.Name == CloudBuildSecretName {
			secret = string(s.Value)
			break
		}
	}

	if token != secret {
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var pushMsg pubsubPushMessage
	if err := json.Unmarshal(body, &pushMsg); err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	decoded, err := base64Decode(pushMsg.Message.Data)
	if err != nil {
		ctx.Logger.Warnf("failed to decode Cloud Build Pub/Sub message data: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	var build map[string]any
	if err := json.Unmarshal(decoded, &build); err != nil {
		ctx.Logger.Warnf("failed to parse Cloud Build notification: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if !g.cloudBuildSubscriptionApplies(subscription) {
			continue
		}

		if err := subscription.SendMessage(build); err != nil {
			ctx.Logger.Errorf("error sending cloud build message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (g *GCP) cloudBuildSubscriptionApplies(subscription core.IntegrationSubscriptionContext) bool {
	var pattern struct {
		Type string `mapstructure:"type"`
	}
	if err := mapstructure.Decode(subscription.Configuration(), &pattern); err != nil {
		return false
	}
	return pattern.Type == cloudbuild.SubscriptionType
}

func (g *GCP) handleArtifactPushEvent(ctx core.HTTPRequestContext) {
	token := ctx.Request.URL.Query().Get("token")
	if token == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var secret string
	for _, s := range secrets {
		if s.Name == ArtifactPushSecretName {
			secret = string(s.Value)
			break
		}
	}

	if token != secret {
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var pushMsg pubsubPushMessage
	if err := json.Unmarshal(body, &pushMsg); err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	decoded, err := base64Decode(pushMsg.Message.Data)
	if err != nil {
		ctx.Logger.Warnf("failed to decode Artifact Registry Pub/Sub message data: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	var event map[string]any
	if err := json.Unmarshal(decoded, &event); err != nil {
		ctx.Logger.Warnf("failed to parse Artifact Registry push event: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if !g.artifactPushSubscriptionApplies(subscription) {
			continue
		}
		if err := subscription.SendMessage(event); err != nil {
			ctx.Logger.Errorf("error sending artifact push message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (g *GCP) artifactPushSubscriptionApplies(subscription core.IntegrationSubscriptionContext) bool {
	var pattern struct {
		Type string `mapstructure:"type"`
	}
	if err := mapstructure.Decode(subscription.Configuration(), &pattern); err != nil {
		return false
	}
	return pattern.Type == artifactregistry.ArtifactPushSubscriptionType
}

func (g *GCP) handleArtifactAnalysisEvent(ctx core.HTTPRequestContext) {
	token := ctx.Request.URL.Query().Get("token")
	if token == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var secret string
	for _, s := range secrets {
		if s.Name == ContainerAnalysisSecretName {
			secret = string(s.Value)
			break
		}
	}

	if token != secret {
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var pushMsg pubsubPushMessage
	if err := json.Unmarshal(body, &pushMsg); err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	decoded, err := base64Decode(pushMsg.Message.Data)
	if err != nil {
		ctx.Logger.Warnf("failed to decode Container Analysis Pub/Sub message data: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	var occurrence map[string]any
	if err := json.Unmarshal(decoded, &occurrence); err != nil {
		ctx.Logger.Warnf("failed to parse Container Analysis occurrence: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if !g.containerAnalysisSubscriptionApplies(subscription) {
			continue
		}
		if err := subscription.SendMessage(occurrence); err != nil {
			ctx.Logger.Errorf("error sending container analysis message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (g *GCP) containerAnalysisSubscriptionApplies(subscription core.IntegrationSubscriptionContext) bool {
	var pattern struct {
		Type string `mapstructure:"type"`
	}
	if err := mapstructure.Decode(subscription.Configuration(), &pattern); err != nil {
		return false
	}
	return pattern.Type == artifactregistry.ArtifactAnalysisSubscriptionType
}

func (g *GCP) handlePubSubEvent(ctx core.HTTPRequestContext) {
	token := ctx.Request.URL.Query().Get("token")
	if token == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var secret string
	for _, s := range secrets {
		if s.Name == PubSubSecretName {
			secret = string(s.Value)
			break
		}
	}

	if token != secret {
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	gcpSubName := ctx.Request.URL.Query().Get("gcpSubName")
	if gcpSubName == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var pushMsg pubsubPushMessage
	if err := json.Unmarshal(body, &pushMsg); err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	decoded, err := base64Decode(pushMsg.Message.Data)
	if err != nil {
		ctx.Logger.Warnf("failed to decode Pub/Sub user message data: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	var msgData any
	if err := json.Unmarshal(decoded, &msgData); err != nil {
		// Non-JSON payloads: deliver as raw string
		msgData = string(decoded)
	}

	message := map[string]any{
		"messageId":   pushMsg.Message.MessageID,
		"publishTime": pushMsg.Message.PublishTime,
		"data":        msgData,
		"attributes":  pushMsg.Message.Attributes,
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if !g.pubsubOnMessageSubscriptionApplies(subscription, gcpSubName) {
			continue
		}
		if err := subscription.SendMessage(message); err != nil {
			ctx.Logger.Errorf("error sending pub/sub message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (g *GCP) pubsubOnMessageSubscriptionApplies(subscription core.IntegrationSubscriptionContext, gcpSubName string) bool {
	var pattern struct {
		Type       string `mapstructure:"type"`
		GCPSubName string `mapstructure:"gcpSubName"`
	}
	if err := mapstructure.Decode(subscription.Configuration(), &pattern); err != nil {
		return false
	}
	return pattern.Type == gcppubsub.OnMessageSubscriptionType && pattern.GCPSubName == gcpSubName
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
