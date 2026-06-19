package cloudsmith

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// securityScanEvent is the Cloudsmith event fired when a package's security
// (vulnerability) scan completes.
const securityScanEvent = "package.security_scanned"

type OnSecurityScanCompleted struct{}

type OnSecurityScanCompletedConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnSecurityScanCompletedMetadata struct {
	Repository *RepositoryRef `json:"repository" mapstructure:"repository"`
	WebhookURL string         `json:"webhookUrl" mapstructure:"webhookUrl"`
	WebhookID  string         `json:"webhookId" mapstructure:"webhookId"`
}

// SecurityScanEvent is the payload emitted to downstream nodes when a package's
// security scan completes.
type SecurityScanEvent struct {
	Event                       string `json:"event"`
	Namespace                   string `json:"namespace"`
	Repository                  string `json:"repository"`
	Name                        string `json:"name"`
	Version                     string `json:"version"`
	SlugPerm                    string `json:"slug_perm"`
	Format                      string `json:"format"`
	SecurityScanStatus          string `json:"security_scan_status"`
	VulnerabilityScanResultsURL string `json:"vulnerability_scan_results_url"`
	HasVulnerabilities          bool   `json:"has_vulnerabilities"`
	MaxSeverity                 string `json:"max_severity"`
	NumVulnerabilities          int    `json:"num_vulnerabilities"`
}

func (t *OnSecurityScanCompleted) Name() string {
	return "cloudsmith.onSecurityScanCompleted"
}

func (t *OnSecurityScanCompleted) Label() string {
	return "On Security Scan Completed"
}

func (t *OnSecurityScanCompleted) Description() string {
	return "Trigger a workflow when a Cloudsmith package's security (vulnerability) scan completes"
}

func (t *OnSecurityScanCompleted) Documentation() string {
	return `The On Security Scan Completed trigger starts a workflow whenever Cloudsmith finishes scanning a package in the selected repository for vulnerabilities.

## Use Cases

- **Block vulnerable packages**: Quarantine or reject a package when its scan finds High/Critical vulnerabilities
- **Security alerts**: Notify a channel when vulnerabilities are detected
- **Audit**: Record the scan outcome for every package

## Configuration

- **Repository**: The repository to watch, in the form ` + "`owner/repository`" + ` (required)

## Webhook Setup

This trigger provisions a Cloudsmith webhook automatically: on setup it registers SuperPlane's webhook URL on the selected repository for the ` + "`package.security_scanned`" + ` event, and removes it when the trigger is deleted. The Cloudsmith service account needs the **Admin** privilege on the repository for this. Each delivery is signed (HMAC-SHA1) with a per-node secret and verified on receipt, so forged or unsigned requests are rejected.

## Output

Emits the package's identity (**namespace**, **repository**, **name**, **version**, **slug_perm**, **format**) and the scan results: **security_scan_status**, **has_vulnerabilities**, **max_severity**, **num_vulnerabilities**, and **vulnerability_scan_results_url**. Because this fires when the scan *completes*, the vulnerability fields are settled — filter downstream, e.g. only act when ` + "`max_severity`" + ` is High/Critical.`
}

func (t *OnSecurityScanCompleted) Icon() string {
	return "shield-alert"
}

func (t *OnSecurityScanCompleted) Color() string {
	return "blue"
}

func (t *OnSecurityScanCompleted) ExampleData() map[string]any {
	return onSecurityScanCompletedExampleData()
}

func (t *OnSecurityScanCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository to watch for completed security scans",
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

func (t *OnSecurityScanCompleted) Setup(ctx core.TriggerContext) error {
	metadata := OnSecurityScanCompletedMetadata{}
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)

	config := OnSecurityScanCompletedConfiguration{}
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	sameRepo := metadata.Repository != nil &&
		metadata.Repository.Namespace == owner &&
		metadata.Repository.Slug == slug

	if sameRepo && metadata.WebhookID != "" && metadata.WebhookURL != "" {
		// Already provisioned: only skip when the remote webhook still exists and
		// targets our URL; otherwise recreate it so the trigger self-heals.
		existing, getErr := client.GetWebhook(owner, slug, metadata.WebhookID)
		if getErr == nil && existing.TargetURL == metadata.WebhookURL {
			// Re-assert the webhook's desired state: the signing key (Cloudsmith
			// only sets it on write and never returns it for comparison), plus the
			// subscribed event and active flag, in case it was disabled or edited
			// out of band at Cloudsmith.
			secret, secErr := ctx.Webhook.GetSecret()
			if secErr != nil {
				return fmt.Errorf("failed to get webhook secret: %w", secErr)
			}
			if _, updErr := client.UpdateWebhook(owner, slug, metadata.WebhookID, metadata.WebhookURL, string(secret), []string{securityScanEvent}); updErr != nil {
				return fmt.Errorf("failed to refresh Cloudsmith webhook: %w", updErr)
			}
			return nil
		}
		if getErr == nil {
			if delErr := client.DeleteWebhook(owner, slug, metadata.WebhookID); delErr != nil {
				ctx.Logger.Warnf("failed to remove stale Cloudsmith webhook: %v", delErr)
			}
		}
	} else if metadata.WebhookID != "" && metadata.Repository != nil {
		// The repository changed since the last setup; remove the old webhook.
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

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return fmt.Errorf("failed to get webhook secret: %w", err)
	}

	webhook, err := client.CreateWebhook(owner, slug, webhookURL, string(secret), []string{securityScanEvent})
	if err != nil {
		return fmt.Errorf("failed to create Cloudsmith webhook: %w", err)
	}

	return ctx.Metadata.Set(OnSecurityScanCompletedMetadata{
		Repository: &RepositoryRef{Namespace: owner, Slug: slug},
		WebhookURL: webhookURL,
		WebhookID:  webhook.SlugPerm,
	})
}

func (t *OnSecurityScanCompleted) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnSecurityScanCompleted) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnSecurityScanCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
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

	metadata := OnSecurityScanCompletedMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	pkg, scan, err := parseSecurityScanWebhook(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing webhook body: %w", err)
	}

	if metadata.Repository != nil && pkg.Namespace != "" &&
		(pkg.Namespace != metadata.Repository.Namespace || pkg.Repository != metadata.Repository.Slug) {
		ctx.Logger.Infof("Ignoring security scan event for %s/%s", pkg.Namespace, pkg.Repository)
		return http.StatusOK, nil, nil
	}

	event := SecurityScanEvent{
		Event:                       securityScanEvent,
		Namespace:                   pkg.Namespace,
		Repository:                  pkg.Repository,
		Name:                        pkg.Name,
		Version:                     pkg.Version,
		SlugPerm:                    pkg.SlugPerm,
		Format:                      pkg.Format,
		SecurityScanStatus:          pkg.SecurityScanStatus,
		VulnerabilityScanResultsURL: pkg.VulnerabilityResultsURL,
		HasVulnerabilities:          scan.HasVulnerabilities,
		MaxSeverity:                 scan.MaxSeverity,
		NumVulnerabilities:          scan.NumVulnerabilities,
	}

	if err := ctx.Events.Emit("cloudsmith.package.securityScanned", event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnSecurityScanCompleted) Cleanup(ctx core.TriggerContext) error {
	metadata := OnSecurityScanCompletedMetadata{}
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

// parseSecurityScanWebhook extracts the package (from "data") and the
// vulnerability scan summary (from "context.vulnerability_scan_results") of a
// package.security_scanned webhook body.
func parseSecurityScanWebhook(body []byte) (*Package, VulnerabilityScan, error) {
	var envelope struct {
		Data    Package `json:"data"`
		Context struct {
			VulnerabilityScanResults VulnerabilityScan `json:"vulnerability_scan_results"`
		} `json:"context"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, VulnerabilityScan{}, err
	}

	pkg := envelope.Data
	if pkg.SlugPerm == "" && pkg.Name == "" {
		// Fall back to a top-level package object.
		_ = json.Unmarshal(body, &pkg)
	}
	return &pkg, envelope.Context.VulnerabilityScanResults, nil
}
