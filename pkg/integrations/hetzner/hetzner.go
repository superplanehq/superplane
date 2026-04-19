package hetzner

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("hetzner", &Hetzner{})
}

type Hetzner struct{}

type Configuration struct {
	APIToken      string `json:"apiToken" mapstructure:"apiToken"`
	S3AccessKeyId string `json:"s3AccessKeyId" mapstructure:"s3AccessKeyId"`
	S3SecretKey   string `json:"s3SecretAccessKey" mapstructure:"s3SecretAccessKey"`
	S3Region      string `json:"s3Region" mapstructure:"s3Region"`
}

func (h *Hetzner) Name() string {
	return "hetzner"
}

func (h *Hetzner) Label() string {
	return "Hetzner Cloud"
}

func (h *Hetzner) Icon() string {
	return "hetzner"
}

func (h *Hetzner) Description() string {
	return "Create and delete Hetzner Cloud servers/load balancers, manage snapshots, and interact with Hetzner Object Storage via the S3-compatible API"
}

func (h *Hetzner) Instructions() string {
	return `
**API Token:** Create a token in [Hetzner Cloud Console](https://console.hetzner.cloud/) → Project → Security → API Tokens. Use **Read & Write** scope.

**Object Storage (optional):** To use S3 components, go to **Object Storage** in the Hetzner Cloud Console and create S3 credentials (Access Key + Secret Key). Set the region to match your bucket location (e.g. fsn1 or nbg1).
`
}

func (h *Hetzner) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Hetzner Cloud API token with Read & Write access",
		},
		{
			Name:        "s3AccessKeyId",
			Label:       "S3 Access Key ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Object Storage S3 Access Key ID (required for S3 components)",
		},
		{
			Name:        "s3SecretAccessKey",
			Label:       "S3 Secret Access Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Object Storage S3 Secret Access Key (required for S3 components)",
		},
		{
			Name:        "s3Region",
			Label:       "S3 Region",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Object Storage region (required for S3 components)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Falkenstein (fsn1)", Value: "fsn1"},
						{Label: "Nuremberg (nbg1)", Value: "nbg1"},
					},
				},
			},
		},
	}
}

func (h *Hetzner) Components() []core.Component {
	return []core.Component{
		&CreateServer{},
		&CreateSnapshot{},
		&DeleteSnapshot{},
		&DeleteServer{},
		&CreateLoadBalancer{},
		&DeleteLoadBalancer{},
		&CreateBucket{},
		&DeleteBucket{},
		&UploadObject{},
		&DownloadObject{},
		&DeleteObject{},
		&ListObjects{},
		&PresignedURL{},
	}
}

func (h *Hetzner) Triggers() []core.Trigger {
	return nil
}

func (h *Hetzner) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (h *Hetzner) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.APIToken) == "" {
		return fmt.Errorf("apiToken is required")
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Hetzner credentials: %w", err)
	}

	// Verify S3 credentials if any are provided — all three must be set together.
	hasS3Key := strings.TrimSpace(config.S3AccessKeyId) != ""
	hasS3Secret := strings.TrimSpace(config.S3SecretKey) != ""
	hasS3Region := strings.TrimSpace(config.S3Region) != ""
	if hasS3Key || hasS3Secret || hasS3Region {
		if !hasS3Key || !hasS3Secret || !hasS3Region {
			return fmt.Errorf("s3AccessKeyId, s3SecretAccessKey, and s3Region must all be provided together")
		}
		s3Client, err := NewHetznerS3Client(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create S3 client: %w", err)
		}
		if _, err := s3Client.ListBuckets(); err != nil {
			return fmt.Errorf("failed to verify S3 credentials: %w", err)
		}
	}

	ctx.Integration.Ready()
	return nil
}

func (h *Hetzner) HandleRequest(ctx core.HTTPRequestContext) {}

func (h *Hetzner) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	switch resourceType {
	case "server":
		servers, err := client.ListServers()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(servers))
		for _, s := range servers {
			id := s.ID
			name := s.Name
			if name == "" {
				name = id
			}
			resources = append(resources, core.IntegrationResource{Type: "server", Name: name, ID: id})
		}
		return resources, nil
	case "load_balancer":
		loadBalancers, err := client.ListLoadBalancers()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(loadBalancers))
		for _, lb := range loadBalancers {
			id := lb.ID
			name := lb.Name
			if name == "" {
				name = id
			}
			resources = append(resources, core.IntegrationResource{Type: "load_balancer", Name: name, ID: id})
		}
		return resources, nil
	case "server_type":
		types, err := client.ListServerTypes()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(types))
		for _, t := range types {
			id := t.Name
			if id == "" {
				id = fmt.Sprintf("%d", t.ID)
			}
			displayName := t.ServerTypeDisplayName()
			if displayName == "" {
				displayName = id
			}
			resources = append(resources, core.IntegrationResource{Type: "server_type", Name: displayName, ID: id})
		}
		return resources, nil
	case "load_balancer_type":
		types, err := client.ListLoadBalancerTypes()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(types))
		for _, t := range types {
			id := t.Name
			if id == "" {
				id = fmt.Sprintf("%d", t.Id)
			}
			resources = append(resources, core.IntegrationResource{Type: "load_balancer_type", Name: t.Name, ID: id})
		}

		return resources, nil
	case "image":
		images, err := client.ListImages()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(images))
		for _, img := range images {
			id := fmt.Sprintf("%d", img.ID)
			displayName := strings.TrimSpace(img.Name)
			if displayName == "" {
				displayName = strings.TrimSpace(img.Description)
			}
			if displayName == "" {
				displayName = id
			}
			if img.Type != "" {
				displayName = fmt.Sprintf("%s (%s)", displayName, img.Type)
			}
			resources = append(resources, core.IntegrationResource{Type: "image", Name: displayName, ID: id})
		}
		return resources, nil
	case "snapshot_image":
		images, err := client.ListImages()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(images))
		for _, img := range images {
			if strings.TrimSpace(img.Type) != "snapshot" {
				continue
			}
			id := fmt.Sprintf("%d", img.ID)
			displayName := strings.TrimSpace(img.Description)
			if displayName == "" {
				displayName = strings.TrimSpace(img.Name)
			}
			if displayName == "" {
				displayName = id
			}
			displayName = fmt.Sprintf("%s (snapshot)", displayName)
			resources = append(resources, core.IntegrationResource{Type: "snapshot_image", Name: displayName, ID: id})
		}
		return resources, nil
	case "location":
		locations, err := client.ListLocations()
		if err != nil {
			return nil, err
		}
		if serverType := ctx.Parameters["serverType"]; serverType != "" {
			allowedNames, err := client.ServerTypeLocationNames(serverType)
			if err == nil && len(allowedNames) > 0 {
				allowed := make(map[string]bool)
				for _, n := range allowedNames {
					allowed[n] = true
				}
				filtered := locations[:0]
				for _, loc := range locations {
					if allowed[loc.Name] {
						filtered = append(filtered, loc)
					}
				}
				locations = filtered
			}
		}
		resources := make([]core.IntegrationResource, 0, len(locations))
		for _, loc := range locations {
			id := loc.Name
			if id == "" {
				id = fmt.Sprintf("%d", loc.ID)
			}
			displayName := loc.LocationDisplayName()
			if displayName == "" {
				displayName = id
			}
			resources = append(resources, core.IntegrationResource{Type: "location", Name: displayName, ID: id})
		}
		return resources, nil
	case "firewall":
		firewalls, err := client.ListFirewalls()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(firewalls))
		for _, firewall := range firewalls {
			id := fmt.Sprintf("%d", firewall.ID)
			name := strings.TrimSpace(firewall.Name)
			if name == "" {
				name = id
			}
			resources = append(resources, core.IntegrationResource{Type: "firewall", Name: name, ID: id})
		}
		return resources, nil
	case "load_balancing_algorithm":
		return []core.IntegrationResource{
			{
				Type: "load_balancing_algorithm",
				Name: "Round Robin",
				ID:   "round_robin",
			},
			{
				Type: "load_balancing_algorithm",
				Name: "Least connections",
				ID:   "least_connections",
			},
		}, nil
	case "bucket":
		s3Client, err := NewHetznerS3Client(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}
		buckets, err := s3Client.ListBuckets()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(buckets))
		for _, b := range buckets {
			resources = append(resources, core.IntegrationResource{Type: "bucket", Name: b.Name, ID: b.Name})
		}
		return resources, nil
	default:
		return nil, nil
	}
}

func (h *Hetzner) Actions() []core.Action {
	return nil
}

func (h *Hetzner) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
