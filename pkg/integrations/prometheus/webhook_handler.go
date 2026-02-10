package prometheus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type PrometheusWebhookHandler struct{}

type webhookRuntimeConfiguration struct {
	AuthType    string `json:"authType"`
	BearerToken string `json:"bearerToken,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
}

func (h *PrometheusWebhookHandler) CompareConfig(a any, b any) (bool, error) {
	return true, nil
}

func (h *PrometheusWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	// External Alertmanager provisioning is manual, but we cache webhook auth settings
	// in the internal webhook secret for runtime request validation.
	config, err := readWebhookRuntimeConfiguration(ctx.Integration)
	if err != nil {
		return nil, err
	}

	secret, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to encode webhook auth config: %w", err)
	}

	if err := ctx.Webhook.SetSecret(secret); err != nil {
		return nil, fmt.Errorf("failed to store webhook auth config: %w", err)
	}

	return nil, nil
}

func (h *PrometheusWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

func (h *PrometheusWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func readWebhookRuntimeConfiguration(integration core.IntegrationContext) (webhookRuntimeConfiguration, error) {
	authType, err := requiredConfig(integration, "webhookAuthType")
	if err != nil {
		return webhookRuntimeConfiguration{}, err
	}
	authType = sanitizeWebhookAuthTypeFromSetup(authType)

	config := webhookRuntimeConfiguration{AuthType: authType}
	switch authType {
	case AuthTypeNone:
		return config, nil
	case AuthTypeBearer:
		bearerToken, err := requiredConfig(integration, "webhookBearerToken")
		if err != nil {
			return webhookRuntimeConfiguration{}, fmt.Errorf("webhookBearerToken is required when webhookAuthType is bearer")
		}
		config.BearerToken = bearerToken
		return config, nil
	case AuthTypeBasic:
		username, err := requiredConfig(integration, "webhookUsername")
		if err != nil {
			return webhookRuntimeConfiguration{}, fmt.Errorf("webhookUsername is required when webhookAuthType is basic")
		}
		password, err := requiredConfig(integration, "webhookPassword")
		if err != nil {
			return webhookRuntimeConfiguration{}, fmt.Errorf("webhookPassword is required when webhookAuthType is basic")
		}
		config.Username = username
		config.Password = password
		return config, nil
	default:
		return webhookRuntimeConfiguration{}, fmt.Errorf("invalid webhookAuthType %q", authType)
	}
}

func getWebhookRuntimeConfiguration(webhook core.NodeWebhookContext) (webhookRuntimeConfiguration, error) {
	secret, err := webhook.GetSecret()
	if err != nil {
		return webhookRuntimeConfiguration{}, fmt.Errorf("failed to read webhook secret: %w", err)
	}

	if len(secret) == 0 {
		return webhookRuntimeConfiguration{AuthType: AuthTypeNone}, nil
	}

	config := webhookRuntimeConfiguration{}
	if err := json.Unmarshal(secret, &config); err != nil {
		return webhookRuntimeConfiguration{}, fmt.Errorf("invalid webhook auth configuration")
	}

	if config.AuthType == "" {
		config.AuthType = AuthTypeNone
	}

	return config, nil
}

func sanitizeWebhookAuthTypeFromSetup(authType string) string {
	if strings.EqualFold(authType, AuthTypeNone) {
		return AuthTypeNone
	}

	if strings.EqualFold(authType, AuthTypeBasic) {
		return AuthTypeBasic
	}

	if strings.EqualFold(authType, AuthTypeBearer) {
		return AuthTypeBearer
	}

	return authType
}
