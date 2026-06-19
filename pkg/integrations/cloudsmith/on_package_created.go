package cloudsmith

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// packageCreatedEvent is the Cloudsmith event fired when a new package is
// created (uploaded) in a repository.
const packageCreatedEvent = "package.created"

type OnPackageCreated struct{}

type OnPackageCreatedConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

type OnPackageCreatedMetadata struct {
	Repository *RepositoryRef `json:"repository" mapstructure:"repository"`
	WebhookURL string         `json:"webhookUrl" mapstructure:"webhookUrl"`
	WebhookID  string         `json:"webhookId" mapstructure:"webhookId"`
}

// PackageCreatedEvent is the payload emitted to downstream nodes for each newly
// created package.
type PackageCreatedEvent struct {
	Event      string `json:"event"`
	Namespace  string `json:"namespace"`
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	SlugPerm   string `json:"slug_perm"`
	Format     string `json:"format"`
	License    string `json:"license"`
	Uploader   string `json:"uploader"`
	UploadedAt string `json:"uploaded_at"`
	Status     string `json:"status"`
}

func (t *OnPackageCreated) Name() string {
	return "cloudsmith.onPackageCreated"
}

func (t *OnPackageCreated) Label() string {
	return "On Package Created"
}

func (t *OnPackageCreated) Description() string {
	return "Trigger a workflow when a new package is created (uploaded) in a Cloudsmith repository"
}

func (t *OnPackageCreated) Documentation() string {
	return `The On Package Created trigger starts a workflow whenever a new package is uploaded to the selected repository.

## Use Cases

- **Ingestion pipelines**: React to new artifacts as they land — promote, tag, or notify
- **Auditing**: Record who uploaded which package and when
- **Fan-out**: Kick off downstream checks (e.g. fetch repository details) for each new package

## Configuration

- **Repository**: The repository to watch, in the form ` + "`owner/repository`" + ` (required)

## Webhook Setup

This trigger provisions a Cloudsmith webhook automatically: on setup it registers SuperPlane's webhook URL on the selected repository for the ` + "`package.created`" + ` event, and removes it when the trigger is deleted. The Cloudsmith service account needs the **Admin** privilege on the repository for this. Each delivery is signed (HMAC-SHA1) with a per-node secret and verified on receipt, so forged or unsigned requests are rejected.

## Output

Emits the new package's details: **namespace**, **repository**, **name**, **version**, **slug_perm**, **format**, **license**, **uploader**, **uploaded_at**, and **status**.`
}

func (t *OnPackageCreated) Icon() string {
	return "package-plus"
}

func (t *OnPackageCreated) Color() string {
	return "blue"
}

func (t *OnPackageCreated) ExampleData() map[string]any {
	return onPackageCreatedExampleData()
}

func (t *OnPackageCreated) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository to watch for new packages",
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

func (t *OnPackageCreated) Setup(ctx core.TriggerContext) error {
	metadata := OnPackageCreatedMetadata{}
	_ = mapstructure.Decode(ctx.Metadata.Get(), &metadata)

	config := OnPackageCreatedConfiguration{}
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
		// targets our URL; otherwise recreate so the trigger self-heals.
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
			if _, updErr := client.UpdateWebhook(owner, slug, metadata.WebhookID, metadata.WebhookURL, string(secret), []string{packageCreatedEvent}); updErr != nil {
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

	webhook, err := client.CreateWebhook(owner, slug, webhookURL, string(secret), []string{packageCreatedEvent})
	if err != nil {
		return fmt.Errorf("failed to create Cloudsmith webhook: %w", err)
	}

	return ctx.Metadata.Set(OnPackageCreatedMetadata{
		Repository: &RepositoryRef{Namespace: owner, Slug: slug},
		WebhookURL: webhookURL,
		WebhookID:  webhook.SlugPerm,
	})
}

func (t *OnPackageCreated) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnPackageCreated) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnPackageCreated) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
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

	metadata := OnPackageCreatedMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	pkg, err := parsePackageFromWebhook(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing webhook body: %w", err)
	}

	if metadata.Repository != nil && pkg.Namespace != "" &&
		(pkg.Namespace != metadata.Repository.Namespace || pkg.Repository != metadata.Repository.Slug) {
		ctx.Logger.Infof("Ignoring package.created event for %s/%s", pkg.Namespace, pkg.Repository)
		return http.StatusOK, nil, nil
	}

	event := PackageCreatedEvent{
		Event:      packageCreatedEvent,
		Namespace:  pkg.Namespace,
		Repository: pkg.Repository,
		Name:       pkg.Name,
		Version:    pkg.Version,
		SlugPerm:   pkg.SlugPerm,
		Format:     pkg.Format,
		License:    pkg.License,
		Uploader:   pkg.Uploader,
		UploadedAt: pkg.UploadedAt,
		Status:     pkg.StatusStr,
	}

	if err := ctx.Events.Emit("cloudsmith.package.created", event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnPackageCreated) Cleanup(ctx core.TriggerContext) error {
	metadata := OnPackageCreatedMetadata{}
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
