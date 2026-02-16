package gcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	createvm "github.com/superplanehq/superplane/pkg/integrations/gcp/create_vm"
	onvmcreate "github.com/superplanehq/superplane/pkg/integrations/gcp/on_vm_created"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	pathPrefixMachineTypes = "/gcp/machine-types/"
	pathPrefixImageFamily  = "/gcp/images/family/"
)

func init() {
	registry.RegisterIntegration("gcp", &GCP{})
	createvm.SetClientFactory(func(ctx core.ExecutionContext) (createvm.Client, error) {
		return NewClient(ctx.HTTP, ctx.Integration)
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

type Metadata struct {
	ProjectID            string `json:"projectId"`
	ClientEmail          string `json:"clientEmail"`
	AuthMethod           string `json:"authMethod"`
	AccessTokenExpiresAt string `json:"accessTokenExpiresAt"`
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

Choose **Service Account Key** (paste JSON) or **Workload Identity Federation** (keyless, using this SuperPlane instance as OIDC issuer).

### Service Account Key

1. In [Google Cloud Console](https://console.cloud.google.com/) → **IAM & Admin** → **Service Accounts**
2. Open a service account → **Keys** → **Add Key** → **Create new key** → **JSON**
3. Download the JSON and paste its **entire contents** below.

### Workload Identity Federation (keyless)

1. Create a [Workload Identity Pool](https://cloud.google.com/iam/docs/workload-identity-federation) with an **OIDC provider** in GCP.
2. Set **Issuer** to **this SuperPlane instance URL** (must serve /.well-known/openid-configuration and /.well-known/jwks.json over HTTPS and be reachable by Google; otherwise use Service Account Key).
3. Set **Audience** to the pool provider resource name.
4. Configure [attribute mapping](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-providers#mapping) so the federated identity can impersonate a GCP service account with the roles your workflows need.
5. Below, choose **Workload Identity Federation** and enter the **pool provider resource name** and **Project ID**.

> **Note**: Use a dedicated service account (or WIF mapping) with only the IAM roles your workflows need.`
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
		&createvm.CreateVM{},
	}
}

func (g *GCP) Triggers() []core.Trigger {
	return []core.Trigger{
		&onvmcreate.OnVMCreated{},
	}
}

func (g *GCP) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	connectionMethod := strings.TrimSpace(config.ConnectionMethod)
	if connectionMethod == "" {
		connectionMethod = ConnectionMethodServiceAccountKey
	}

	if connectionMethod == ConnectionMethodWIF {
		return g.syncWIF(ctx, config)
	}
	return g.syncServiceAccountKey(ctx, config)
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

	if err := ctx.Integration.SetSecret(SecretNameAccessToken, []byte(accessToken)); err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}

	expiresAt := time.Now().Add(expiresIn)
	refreshAfter := expiresIn / 2
	if refreshAfter < time.Minute {
		refreshAfter = time.Minute
	}

	metadata := Metadata{
		ProjectID:            projectID,
		ClientEmail:          "",
		AuthMethod:           AuthMethodWIF,
		AccessTokenExpiresAt: expiresAt.Format(time.RFC3339),
	}
	ctx.Integration.SetMetadata(metadata)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
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
	// Sensitive config is stored encrypted; get decrypted key via GetConfig.
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
	metadata.AuthMethod = AuthMethodServiceAccountKey

	if err := ctx.Integration.SetSecret(SecretNameServiceAccountKey, keyJSON); err != nil {
		return fmt.Errorf("failed to store service account key: %w", err)
	}

	ctx.Integration.SetMetadata(metadata)
	client, err := NewClient(ctx.HTTP, ctx.Integration)
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

func validateAndParseServiceAccountKey(keyJSON []byte) (Metadata, error) {
	var raw map[string]any
	if err := json.Unmarshal(keyJSON, &raw); err != nil {
		return Metadata{}, fmt.Errorf("invalid JSON: %w", err)
	}

	for _, k := range RequiredJSONKeys {
		if _, ok := raw[k]; !ok {
			return Metadata{}, fmt.Errorf("missing required field %q in service account key", k)
		}
	}

	projectID, _ := raw["project_id"].(string)
	clientEmail, _ := raw["client_email"].(string)

	return Metadata{
		ProjectID:   strings.TrimSpace(projectID),
		ClientEmail: strings.TrimSpace(clientEmail),
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

func trimmedParam(params map[string]string, key string) string {
	return strings.TrimSpace(params[key])
}

func (g *GCP) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}
	reqCtx := context.Background()

	switch resourceType {
	case createvm.ResourceTypeRegion:
		return createvm.ListRegionResources(reqCtx, client)
	case createvm.ResourceTypeZone:
		return createvm.ListZoneResources(reqCtx, client, ctx.Parameters["region"])
	case createvm.ResourceTypeMachineFamily:
		zone := trimmedParam(ctx.Parameters, "zone")
		if zone == "" {
			return []core.IntegrationResource{}, nil
		}
		return createvm.ListMachineFamilyResources(reqCtx, client, zone)
	case createvm.ResourceTypeMachineType:
		zone := trimmedParam(ctx.Parameters, "zone")
		if zone == "" {
			return []core.IntegrationResource{}, nil
		}
		machineFamily := trimmedParam(ctx.Parameters, "machineFamily")
		return createvm.ListMachineTypeResources(reqCtx, client, zone, machineFamily)
	case createvm.ResourceTypePublicImages:
		return createvm.ListPublicImageResources(reqCtx, client, ctx.Parameters["project"])
	case createvm.ResourceTypeCustomImages:
		return createvm.ListCustomImageResources(reqCtx, client, ctx.Parameters["project"])
	case createvm.ResourceTypeSnapshots:
		return createvm.ListSnapshotResources(reqCtx, client, ctx.Parameters["project"])
	case createvm.ResourceTypeDisks:
		zone := trimmedParam(ctx.Parameters, "zone")
		if zone == "" {
			return []core.IntegrationResource{}, nil
		}
		return createvm.ListDiskResources(reqCtx, client, ctx.Parameters["project"], zone)
	case createvm.ResourceTypeDiskTypes:
		zone := trimmedParam(ctx.Parameters, "zone")
		if zone == "" {
			return []core.IntegrationResource{}, nil
		}
		bootDiskOnly := ctx.Parameters["bootDiskOnly"] == "true"
		return createvm.ListDiskTypeResources(reqCtx, client, ctx.Parameters["project"], zone, bootDiskOnly)
	case createvm.ResourceTypeSnapshotSchedules:
		region := trimmedParam(ctx.Parameters, "region")
		if region == "" {
			return []core.IntegrationResource{}, nil
		}
		return createvm.ListSnapshotScheduleResources(reqCtx, client, ctx.Parameters["project"], region)
	case createvm.ResourceTypeNetwork:
		return createvm.ListNetworkResources(reqCtx, client, ctx.Parameters["project"])
	case createvm.ResourceTypeSubnetwork:
		region := trimmedParam(ctx.Parameters, "region")
		if region == "" {
			return []core.IntegrationResource{}, nil
		}
		return createvm.ListSubnetworkResources(reqCtx, client, ctx.Parameters["project"], region)
	case createvm.ResourceTypeAddress:
		region := trimmedParam(ctx.Parameters, "region")
		if region == "" {
			return []core.IntegrationResource{}, nil
		}
		return createvm.ListAddressResources(reqCtx, client, ctx.Parameters["project"], region)
	case createvm.ResourceTypeFirewall:
		return createvm.ListFirewallResources(reqCtx, client, ctx.Parameters["project"])
	default:
		return nil, nil
	}
}

func (g *GCP) HandleRequest(ctx core.HTTPRequestContext) {
	if ctx.Request.Method != http.MethodGet {
		ctx.Response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimSuffix(ctx.Request.URL.Path, "/")
	if path == "" {
		path = ctx.Request.URL.Path
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		writeJSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	reqCtx := context.Background()

	switch {
	case strings.HasSuffix(path, "/gcp/regions"):
		g.handleListRegions(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/zones"):
		g.handleListZones(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/machine-families"):
		g.handleListMachineFamilies(ctx, client, reqCtx)
	case strings.Contains(path, pathPrefixMachineTypes):
		g.handleGetMachineType(ctx, client, reqCtx, path)
	case strings.HasSuffix(path, "/gcp/machine-types"):
		g.handleListMachineTypes(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/provisioning-models"):
		g.handleProvisioningModels(ctx)
	case strings.Contains(path, pathPrefixImageFamily):
		g.handleGetImageFromFamily(ctx, client, reqCtx, path)
	case strings.HasSuffix(path, "/gcp/public-images"):
		g.handleListPublicImages(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/custom-images"):
		g.handleListCustomImages(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/snapshots"):
		g.handleListSnapshots(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/disks"):
		g.handleListDisks(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/disk-types"):
		g.handleListDiskTypes(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/snapshot-schedules"):
		g.handleListSnapshotSchedules(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/networks"):
		g.handleListNetworks(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/subnetworks"):
		g.handleListSubnetworks(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/addresses"):
		g.handleListAddresses(ctx, client, reqCtx)
	case strings.HasSuffix(path, "/gcp/firewalls"):
		g.handleListFirewalls(ctx, client, reqCtx)
	default:
		ctx.Response.WriteHeader(http.StatusNotFound)
	}
}

func requireQueryParam(ctx core.HTTPRequestContext, name, message string) (string, bool) {
	v := strings.TrimSpace(ctx.Request.URL.Query().Get(name))
	if v == "" {
		if message == "" {
			message = name + " query parameter is required"
		}
		writeJSONError(ctx, http.StatusBadRequest, message)
		return "", false
	}
	return v, true
}

func pathSuffixAfter(path, prefix string) (suffix string, found bool) {
	idx := strings.Index(path, prefix)
	if idx < 0 {
		return "", false
	}
	suffix = path[idx+len(prefix):]
	return suffix, suffix != ""
}

func (g *GCP) handleListRegions(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	list, err := createvm.ListRegions(reqCtx, c)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListZones(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	region := ctx.Request.URL.Query().Get("region")
	list, err := createvm.ListZones(reqCtx, c, region)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListMachineTypes(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	zone, ok := requireQueryParam(ctx, "zone", "")
	if !ok {
		return
	}
	list, err := createvm.ListMachineTypes(reqCtx, c, zone)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleGetMachineType(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context, path string) {
	zone, ok := requireQueryParam(ctx, "zone", "")
	if !ok {
		return
	}
	machineType, found := pathSuffixAfter(path, pathPrefixMachineTypes)
	if !found {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}
	mt, err := createvm.GetMachineType(reqCtx, c, zone, machineType)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, mt)
}

func (g *GCP) handleListMachineFamilies(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	zone, ok := requireQueryParam(ctx, "zone", "")
	if !ok {
		return
	}
	list, err := createvm.ListMachineFamilies(reqCtx, c, zone)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleProvisioningModels(ctx core.HTTPRequestContext) {
	writeJSON(ctx, http.StatusOK, []struct {
		Value string `json:"value"`
	}{
		{Value: string(createvm.ProvisioningStandard)},
		{Value: string(createvm.ProvisioningSpot)},
	})
}

func (g *GCP) handleListPublicImages(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListPublicImages(reqCtx, c, project)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleGetImageFromFamily(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context, path string) {
	project := ctx.Request.URL.Query().Get("project")
	family, found := pathSuffixAfter(path, pathPrefixImageFamily)
	if !found {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}
	img, err := createvm.GetImageFromFamily(reqCtx, c, project, family)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, img)
}

func (g *GCP) handleListCustomImages(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListCustomImages(reqCtx, c, project)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListSnapshots(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListSnapshots(reqCtx, c, project)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListDisks(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	zone, ok := requireQueryParam(ctx, "zone", "")
	if !ok {
		return
	}
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListDisks(reqCtx, c, project, zone)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListDiskTypes(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	zone, ok := requireQueryParam(ctx, "zone", "")
	if !ok {
		return
	}
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListDiskTypes(reqCtx, c, project, zone)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListSnapshotSchedules(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	region, ok := requireQueryParam(ctx, "region", "")
	if !ok {
		return
	}
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListSnapshotSchedules(reqCtx, c, project, region)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListNetworks(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListNetworks(reqCtx, c, project)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListSubnetworks(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	region, ok := requireQueryParam(ctx, "region", "")
	if !ok {
		return
	}
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListSubnetworks(reqCtx, c, project, region)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListAddresses(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	region, ok := requireQueryParam(ctx, "region", "")
	if !ok {
		return
	}
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListAddresses(reqCtx, c, project, region)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func (g *GCP) handleListFirewalls(ctx core.HTTPRequestContext, c createvm.Client, reqCtx context.Context) {
	project := ctx.Request.URL.Query().Get("project")
	list, err := createvm.ListFirewalls(reqCtx, c, project)
	if err != nil {
		writeGCPError(ctx, err)
		return
	}
	writeJSON(ctx, http.StatusOK, list)
}

func writeJSON(ctx core.HTTPRequestContext, status int, data any) {
	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.WriteHeader(status)
	if data == nil {
		return
	}
	if err := json.NewEncoder(ctx.Response).Encode(data); err != nil {
		ctx.Logger.Errorf("encode JSON: %v", err)
	}
}

func writeJSONError(ctx core.HTTPRequestContext, status int, message string) {
	writeJSON(ctx, status, map[string]string{"error": message})
}

func writeGCPError(ctx core.HTTPRequestContext, err error) {
	var apiErr *GCPAPIError
	if errors.As(err, &apiErr) {
		status := apiErr.StatusCode
		message := apiErr.Message
		if status == http.StatusForbidden {
			message = "GCP access denied. Ensure Compute Engine API is enabled and the service account has roles/compute.viewer (or sufficient permissions)."
		}
		if status == http.StatusNotFound {
			message = "Resource not found (e.g. zone or project may be unavailable). " + message
		}
		if status == http.StatusTooManyRequests {
			message = "GCP quota exceeded. Retry with backoff. " + message
		}
		writeJSONError(ctx, status, message)
		return
	}
	writeJSONError(ctx, http.StatusInternalServerError, err.Error())
}
