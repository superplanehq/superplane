package oci

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/oci/common"
	"github.com/superplanehq/superplane/pkg/integrations/oci/compute"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	compute.SetClientFactory(func(ctx core.ExecutionContext) (compute.Client, error) {
		return common.NewComputeClientWrapper(ctx.Integration)
	})
	registry.RegisterIntegrationWithWebhookHandler("oci", &OCI{}, &OCI{})
}

type OCI struct{}

type Configuration struct {
	TenancyOCID string `json:"tenancyOcid" mapstructure:"tenancyOcid"`
	UserOCID    string `json:"userOcid" mapstructure:"userOcid"`
	Fingerprint string `json:"fingerprint" mapstructure:"fingerprint"`
	Region      string `json:"region" mapstructure:"region"`
	PrivateKey  string `json:"privateKey" mapstructure:"privateKey"`
}

func (o *OCI) Name() string {
	return "oci"
}

func (o *OCI) Label() string {
	return "Oracle Cloud"
}

func (o *OCI) Icon() string {
	return "oci"
}

func (o *OCI) Description() string {
	return "Manage Oracle Cloud Infrastructure (OCI) resources in your workflows"
}

func (o *OCI) Instructions() string {
	return `## Authentication
1. Go to the OCI Console.
2. Under **Identity & Security**, go to **Users**.
3. Select your user, then click **API Keys**.
4. Click **Add API Key**, download the private key, and copy the configuration snippet.
5. Paste the required OCIDs, fingerprint, region, and private key content below.`
}

func (o *OCI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tenancyOcid",
			Label:       "Tenancy OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of your tenancy",
		},
		{
			Name:        "userOcid",
			Label:       "User OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the user",
		},
		{
			Name:        "fingerprint",
			Label:       "Fingerprint",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The fingerprint of the API key",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Home region (e.g. us-ashburn-1)",
		},
		{
			Name:        "privateKey",
			Label:       "Private Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "The full content of your private key file (.pem)",
		},
	}
}

func (o *OCI) Components() []core.Component {
	return []core.Component{
		&compute.CreateInstance{},
		&compute.UpdateInstance{},
		&compute.GetInstance{},
		&compute.ManageInstancePower{},
		&compute.DeleteInstance{},
	}
}

func (o *OCI) Triggers() []core.Trigger {
	return []core.Trigger{
		&compute.OnInstanceCreated{},
		&compute.OnInstanceStateChange{},
	}
}

func (o *OCI) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (o *OCI) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OCI) ListResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (o *OCI) HandleRequest(ctx core.HTTPRequestContext) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(500)
		return
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		// If not JSON, we still try to wrap it or just log error
		ctx.Logger.Errorf("failed to unmarshal OCI webhook body: %v", err)
		ctx.Response.WriteHeader(400)
		return
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Response.WriteHeader(500)
		return
	}

	for _, sub := range subscriptions {
		err := sub.SendMessage(data)
		if err != nil {
			ctx.Logger.Errorf("failed to send message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(200)
}
