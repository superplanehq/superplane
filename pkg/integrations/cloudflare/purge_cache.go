package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PurgeCachePayloadType = "cloudflare.cache.purged"

var allowedPurgeModes = []string{"everything", "files", "tags", "hosts", "prefixes"}

type PurgeCache struct{}

type PurgeCacheSpec struct {
	Zone     string   `json:"zone"`
	Mode     string   `json:"mode"`
	Files    []string `json:"files"`
	Tags     []string `json:"tags"`
	Hosts    []string `json:"hosts"`
	Prefixes []string `json:"prefixes"`
}

func (c *PurgeCache) Name() string {
	return "cloudflare.purgeCache"
}

func (c *PurgeCache) Label() string {
	return "Purge Cache"
}

func (c *PurgeCache) Description() string {
	return "Purge cached content from the Cloudflare CDN for a zone"
}

func (c *PurgeCache) Documentation() string {
	return `The Purge Cache component clears cached content from the Cloudflare CDN.

## Use Cases

- **Deployments**: Immediately serve fresh content after a release without waiting for TTL expiry
- **Hotfixes**: Force CDN edge nodes to re-fetch updated assets after a critical fix
- **Preview environments**: Clear cache for specific preview subdomain URLs

## Configuration

- **Zone**: The Cloudflare zone whose cache to purge
- **Mode**:
  - *Everything*: Purge all cached content in the zone (equivalent to "Purge Everything" in the dashboard). Use with care — this can spike origin traffic.
  - *Files*: Purge one or more specific URLs. Supports both plain URLs and URLs with custom cache headers.
  - *Tags*: Purge by Cache-Tag header values (Enterprise plan only).
  - *Hosts*: Purge all files cached under one or more hostnames (Enterprise plan only).
  - *Prefixes*: Purge all cached assets whose URL starts with one of the provided prefixes.

## Output

Emits the Cloudflare purge result ID, zone ID, zone name (when known from integration metadata), and the purge scope (mode + items).`
}

func (c *PurgeCache) Icon() string {
	return "zap"
}

func (c *PurgeCache) Color() string {
	return "orange"
}

func (c *PurgeCache) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PurgeCache) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "zone",
			Label:       "Zone",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Cloudflare zone whose cache to purge",
			Placeholder: "Select a zone",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "zone",
				},
			},
		},
		{
			Name:     "mode",
			Label:    "Mode",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "files",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Files (specific URLs)", Value: "files"},
						{Label: "Everything (purge all)", Value: "everything"},
						{Label: "Tags (cache tags, Enterprise)", Value: "tags"},
						{Label: "Hosts (hostnames, Enterprise)", Value: "hosts"},
						{Label: "Prefixes (URL prefixes)", Value: "prefixes"},
					},
				},
			},
		},
		{
			Name:        "files",
			Label:       "URLs",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "URLs to purge from the cache",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"files"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{"files"}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "URL",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Cache Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Cache-Tag header values to purge (Enterprise plan required)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"tags"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{"tags"}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "hosts",
			Label:       "Hostnames",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Hostnames to purge all cached files for (Enterprise plan required)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"hosts"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{"hosts"}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Hostname",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "prefixes",
			Label:       "Prefixes",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "URL prefixes to purge from the cache",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"prefixes"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{"prefixes"}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Prefix",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *PurgeCache) Setup(ctx core.SetupContext) error {
	spec := PurgeCacheSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	return validatePurgeCacheSpec(spec)
}

func validatePurgeCacheSpec(spec PurgeCacheSpec) error {
	if strings.TrimSpace(spec.Zone) == "" {
		return errors.New("zone is required")
	}

	if strings.TrimSpace(spec.Mode) == "" {
		return errors.New("mode is required")
	}

	mode := strings.TrimSpace(spec.Mode)
	found := false
	for _, allowed := range allowedPurgeModes {
		if mode == allowed {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("mode must be one of %s", strings.Join(allowedPurgeModes, ", "))
	}

	switch mode {
	case "files":
		if len(spec.Files) == 0 {
			return errors.New("at least one URL is required when mode is files")
		}
	case "tags":
		if len(spec.Tags) == 0 {
			return errors.New("at least one tag is required when mode is tags")
		}
	case "hosts":
		if len(spec.Hosts) == 0 {
			return errors.New("at least one hostname is required when mode is hosts")
		}
	case "prefixes":
		if len(spec.Prefixes) == 0 {
			return errors.New("at least one prefix is required when mode is prefixes")
		}
	}

	return nil
}

func (c *PurgeCache) Execute(ctx core.ExecutionContext) error {
	spec := PurgeCacheSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if err := validatePurgeCacheSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	zoneID := resolveZoneID(spec.Zone, ctx.Integration)

	req := buildPurgeCacheRequest(spec)

	result, err := client.PurgeCache(zoneID, req)
	if err != nil {
		return fmt.Errorf("failed to purge cache: %w", err)
	}

	payload := map[string]any{
		"zoneId": zoneID,
		"id":     result.ID,
		"mode":   spec.Mode,
	}
	if zoneName := resolveZoneName(zoneID, ctx.Integration); zoneName != "" {
		payload["zoneName"] = zoneName
	}

	switch spec.Mode {
	case "files":
		payload["files"] = spec.Files
	case "tags":
		payload["tags"] = spec.Tags
	case "hosts":
		payload["hosts"] = spec.Hosts
	case "prefixes":
		payload["prefixes"] = spec.Prefixes
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PurgeCachePayloadType, []any{payload})
}

func buildPurgeCacheRequest(spec PurgeCacheSpec) PurgeCacheRequest {
	switch spec.Mode {
	case "everything":
		return PurgeCacheRequest{PurgeEverything: true}
	case "files":
		return PurgeCacheRequest{Files: spec.Files}
	case "tags":
		return PurgeCacheRequest{Tags: spec.Tags}
	case "hosts":
		return PurgeCacheRequest{Hosts: spec.Hosts}
	case "prefixes":
		return PurgeCacheRequest{Prefixes: spec.Prefixes}
	default:
		return PurgeCacheRequest{}
	}
}

func (c *PurgeCache) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PurgeCache) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PurgeCache) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *PurgeCache) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *PurgeCache) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *PurgeCache) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
