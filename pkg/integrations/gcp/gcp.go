package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/compute"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("gcp", &GCP{}, &WebhookHandler{})
	compute.SetClientFactory(func(ctx core.ExecutionContext) (compute.Client, error) {
		return gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	})
}

type GCP struct{}

const (
	ConnectionMethodServiceAccountKey = "serviceAccountKey"
	ConnectionMethodWIF               = "workloadIdentityFederation"
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

> Grant only the IAM roles your workflows need.`
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
	}
}

func (g *GCP) Triggers() []core.Trigger {
	return []core.Trigger{
		&compute.OnVMInstance{},
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

	ctx.Integration.Ready()
	return nil
}

func validateAndParseServiceAccountKey(keyJSON []byte) (gcpcommon.Metadata, error) {
	var raw map[string]any
	if err := json.Unmarshal(keyJSON, &raw); err != nil {
		return gcpcommon.Metadata{}, fmt.Errorf("invalid JSON: %w", err)
	}

	projectID := ""
	clientEmail := ""

	if projectID, ok := raw["project_id"].(string); ok {
		projectID = strings.TrimSpace(projectID)
	}

	if clientEmail, ok := raw["client_email"].(string); ok {
		clientEmail = strings.TrimSpace(clientEmail)
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

func (g *GCP) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (g *GCP) Actions() []core.Action {
	return nil
}

func (g *GCP) HandleAction(ctx core.IntegrationActionContext) error {
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
	default:
		return nil, nil
	}
}

func (g *GCP) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(http.StatusNotFound)
}
