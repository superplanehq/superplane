package cloudsmith

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// complianceWebhookEvent is the Cloudsmith event fired once a package has been
// processed and its license/policy/governance compliance has been evaluated.
const complianceWebhookEvent = "package.synced"

// signatureHeader is the header Cloudsmith sets on each delivery, formatted as
// "sha1=<hex>" where the digest is HMAC-SHA1 of the request body keyed by the
// webhook's signature_key.
const signatureHeader = "X-Cloudsmith-Signature"

type OnComplianceCheckCompleted struct{}

type OnComplianceCheckCompletedConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

type ComplianceRepositoryMetadata struct {
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Slug      string `json:"slug" mapstructure:"slug"`
}

type OnComplianceCheckCompletedMetadata struct {
	Repository *ComplianceRepositoryMetadata `json:"repository" mapstructure:"repository"`
	WebhookURL string                        `json:"webhookUrl" mapstructure:"webhookUrl"`
	WebhookID  string                        `json:"webhookId" mapstructure:"webhookId"`
}

// ComplianceCheckEvent is the payload emitted to downstream nodes for each
// processed package.
type ComplianceCheckEvent struct {
	Event          string `json:"event"`
	Namespace      string `json:"namespace"`
	Repository     string `json:"repository"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	SlugPerm       string `json:"slug_perm"`
	License        string `json:"license"`
	SPDXLicense    string `json:"spdx_license"`
	OSIApproved    bool   `json:"osi_approved"`
	PolicyViolated bool   `json:"policy_violated"`
	IsQuarantined  bool   `json:"is_quarantined"`
	Status         string `json:"status"`
}

func (t *OnComplianceCheckCompleted) Name() string {
	return "cloudsmith.onComplianceCheckCompleted"
}

func (t *OnComplianceCheckCompleted) Label() string {
	return "On Compliance Check Completed"
}

func (t *OnComplianceCheckCompleted) Description() string {
	return "Trigger a workflow when a Cloudsmith package finishes processing and its compliance is evaluated"
}

func (t *OnComplianceCheckCompleted) Documentation() string {
	return `The On Compliance Check Completed trigger starts a workflow whenever a package in the selected repository finishes processing — the point at which Cloudsmith has evaluated its license, policy, and governance compliance.

> Vulnerability and security-scan events are handled by a separate component.

## Use Cases

- **Governance gates**: React when a package is processed — block, notify, or audit based on its license / quarantine / policy state
- **License monitoring**: Record the detected license of every newly processed package
- **Quarantine response**: Kick off remediation when a processed package is quarantined

## Configuration

- **Repository**: The repository to watch, in the form ` + "`owner/repository`" + ` (required)

## Webhook Setup

This trigger provisions a Cloudsmith webhook automatically: on setup it registers SuperPlane's webhook URL on the selected repository for the ` + "`package.synced`" + ` event, and removes it when the trigger is deleted. The Cloudsmith service account must have permission to manage webhooks on the repository. Each delivery is signed (HMAC-SHA1) with a per-node secret and verified on receipt, so forged or unsigned requests are rejected.

## Output

Emits the processed package's compliance fields: **namespace**, **repository**, **name**, **version**, **slug_perm**, **license**, **spdx_license**, **osi_approved**, **policy_violated**, **is_quarantined**, and **status**. Use these to filter downstream (for example, only act when ` + "`is_quarantined`" + ` is true).`
}

func (t *OnComplianceCheckCompleted) Icon() string {
	return "shield-check"
}

func (t *OnComplianceCheckCompleted) Color() string {
	return "blue"
}

func (t *OnComplianceCheckCompleted) ExampleData() map[string]any {
	return onComplianceCheckCompletedExampleData()
}

func (t *OnComplianceCheckCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository to watch for compliance checks",
			Placeholder: "Select repository",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (t *OnComplianceCheckCompleted) Setup(ctx core.TriggerContext) error {
	metadata := OnComplianceCheckCompletedMetadata{}
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)

	config := OnComplianceCheckCompletedConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repositoryID := strings.TrimSpace(config.Repository)
	if repositoryID == "" {
		return fmt.Errorf("repository is required")
	}

	owner, slug, err := parseRepositoryID(repositoryID)
	if err != nil {
		return fmt.Errorf("invalid repository %q: %w", repositoryID, err)
	}

	// Already provisioned for this repository.
	if metadata.Repository != nil &&
		metadata.Repository.Namespace == owner &&
		metadata.Repository.Slug == slug &&
		metadata.WebhookID != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// The repository changed since the last setup; remove the stale webhook.
	if metadata.WebhookID != "" && metadata.Repository != nil {
		if delErr := client.DeleteWebhook(metadata.Repository.Namespace, metadata.Repository.Slug, metadata.WebhookID); delErr != nil {
			ctx.Logger.Warnf("failed to remove previous Cloudsmith webhook: %v", delErr)
		}
	}

	webhookURL := metadata.WebhookURL
	if webhookURL == "" {
		webhookURL, err = ctx.Webhook.Setup()
		if err != nil {
			return fmt.Errorf("failed to setup webhook: %w", err)
		}
	}

	// Use the node's webhook secret as Cloudsmith's signature key so deliveries
	// are signed (HMAC-SHA1) and can be verified in HandleWebhook.
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return fmt.Errorf("failed to get webhook secret: %w", err)
	}

	webhook, err := client.CreateWebhook(owner, slug, webhookURL, string(secret), []string{complianceWebhookEvent})
	if err != nil {
		return fmt.Errorf("failed to create Cloudsmith webhook: %w", err)
	}

	return ctx.Metadata.Set(OnComplianceCheckCompletedMetadata{
		Repository: &ComplianceRepositoryMetadata{Namespace: owner, Slug: slug},
		WebhookURL: webhookURL,
		WebhookID:  webhook.SlugPerm,
	})
}

func (t *OnComplianceCheckCompleted) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnComplianceCheckCompleted) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnComplianceCheckCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	// Verify the delivery is genuinely from Cloudsmith using the node's webhook
	// secret (set as the webhook's signature key at setup).
	var secret []byte
	if ctx.Webhook != nil {
		s, err := ctx.Webhook.GetSecret()
		if err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("error getting secret: %w", err)
		}
		secret = s
	}
	if err := verifyCloudsmithSignature(ctx.Headers.Get(signatureHeader), ctx.Body, secret); err != nil {
		return http.StatusForbidden, nil, fmt.Errorf("invalid signature: %w", err)
	}

	metadata := OnComplianceCheckCompletedMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	pkg, err := parseCompliancePackage(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing webhook body: %w", err)
	}

	// The webhook is already repository-scoped; this is a defensive guard.
	if metadata.Repository != nil && pkg.Namespace != "" &&
		(pkg.Namespace != metadata.Repository.Namespace || pkg.Repository != metadata.Repository.Slug) {
		ctx.Logger.Infof("Ignoring compliance event for %s/%s", pkg.Namespace, pkg.Repository)
		return http.StatusOK, nil, nil
	}

	event := ComplianceCheckEvent{
		Event:          complianceWebhookEvent,
		Namespace:      pkg.Namespace,
		Repository:     pkg.Repository,
		Name:           pkg.Name,
		Version:        pkg.Version,
		SlugPerm:       pkg.SlugPerm,
		License:        pkg.License,
		SPDXLicense:    pkg.SPDXLicense,
		OSIApproved:    pkg.OSIApproved,
		PolicyViolated: pkg.PolicyViolated,
		IsQuarantined:  pkg.IsQuarantined,
		Status:         pkg.Status,
	}

	if err := ctx.Events.Emit("cloudsmith.package.complianceChecked", event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnComplianceCheckCompleted) Cleanup(ctx core.TriggerContext) error {
	metadata := OnComplianceCheckCompletedMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil
	}
	if metadata.WebhookID == "" || metadata.Repository == nil {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil
	}

	if err := client.DeleteWebhook(metadata.Repository.Namespace, metadata.Repository.Slug, metadata.WebhookID); err != nil {
		ctx.Logger.Warnf("failed to delete Cloudsmith webhook during cleanup: %v", err)
	}
	return nil
}

// verifyCloudsmithSignature checks the X-Cloudsmith-Signature header (formatted
// "sha1=<hex>") against HMAC-SHA1 of the body keyed by secret. When no secret is
// configured, verification is skipped (mirrors the other integrations).
func verifyCloudsmithSignature(signature string, body, secret []byte) error {
	if len(secret) == 0 {
		return nil
	}
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	mac := hmac.New(sha1.New, secret)
	mac.Write(body)
	expected := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// parseCompliancePackage extracts the package from a JSON-object webhook body.
// Cloudsmith delivers the package under a "data" key; we fall back to a
// top-level object so the trigger is resilient to payload-shape differences.
func parseCompliancePackage(body []byte) (*Package, error) {
	var envelope struct {
		Event string  `json:"event"`
		Data  Package `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && (envelope.Data.SlugPerm != "" || envelope.Data.Name != "") {
		return &envelope.Data, nil
	}

	var pkg Package
	if err := json.Unmarshal(body, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
