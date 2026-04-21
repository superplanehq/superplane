package oci

import (
	"crypto/subtle"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/oci/compute"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler(&OCI{}, &OCIWebhookHandler{})
}

type Configuration struct {
	TenancyOCID  string `mapstructure:"tenancyOcid"`
	UserOCID     string `mapstructure:"userOcid"`
	Fingerprint  string `mapstructure:"fingerprint"`
	Region       string `mapstructure:"region"`
	PrivateKey   string `mapstructure:"privateKey"`
	WebhookToken string `mapstructure:"webhookToken"`
}

type OCI struct{}

func (o *OCI) Name() string { return "oci" }
func (o *OCI) Label() string { return "Oracle Cloud Infrastructure" }
func (o *OCI) Description() string { return "Connect with Oracle Cloud Infrastructure (OCI)" }
func (o *OCI) Icon() string { return "oci" }
func (o *OCI) Instructions() string {
	return "Enter your OCI credentials including Tenancy OCID, User OCID, Fingerprint, Region, and Private Key."
}

func (o *OCI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "tenancyOcid",
			Label:       "Tenancy OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of your OCI tenancy",
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
			Description: "The fingerprint of the public key",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "e.g. us-phoenix-1",
		},
		{
			Name:        "privateKey",
			Label:       "Private Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "The private key associated with the public key fingerprint",
		},
		{
			Name:        "webhookToken",
			Label:       "Webhook Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "A secret token to verify incoming OCI events",
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

	metadata := map[string]any{
		"tenancyOcid": config.TenancyOCID,
		"userOcid":    config.UserOCID,
		"fingerprint": config.Fingerprint,
		"region":      config.Region,
	}
	ctx.Integration.SetMetadata(metadata)

	ctx.Integration.Ready()
	return nil
}

func (o *OCI) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

type OCIWebhookHandler struct{}

func (h *OCIWebhookHandler) HandleRequest(ctx core.HTTPRequestContext) {
	tokenRaw, err := ctx.Integration.GetConfig("webhookToken")
	if err != nil || len(tokenRaw) == 0 {
		ctx.Logger.Errorf("OCI webhook: failed to retrieve webhook token from config")
		ctx.Response.WriteHeader(500)
		return
	}
	webhookToken := string(tokenRaw)

	receivedToken := ctx.Request.Header.Get("X-OCI-Token")

	if receivedToken == "" || subtle.ConstantTimeCompare([]byte(receivedToken), []byte(webhookToken)) != 1 {
		ctx.Logger.Warnf("OCI webhook: unauthorized request")
		ctx.Response.WriteHeader(401)
		return
	}

	ctx.Integration.Subscribe(nil)
	ctx.Response.WriteHeader(200)
}

func (h *OCIWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}
