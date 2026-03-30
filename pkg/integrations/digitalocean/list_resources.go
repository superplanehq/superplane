package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

func (d *DigitalOcean) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "droplet":
		return listDroplets(ctx)
	case "region":
		return listRegions(ctx)
	case "size":
		return listSizes(ctx)
	case "image":
		return listImages(ctx)
	case "snapshot":
		return listSnapshots(ctx)
	case "domain":
		return listDomains(ctx)
	case "dns_record":
		return listDNSRecords(ctx)
	case "load_balancer":
		return listLoadBalancers(ctx)
	case "reserved_ip":
		return listReservedIPs(ctx)
	case "ssh_key":
		return listSSHKeys(ctx)
	case "vpc":
		return listVPCs(ctx)
	case "alert_policy":
		return listAlertPolicies(ctx)
	case "spaces_bucket":
		return listSpacesBuckets(ctx)
	case "app":
		return listApps(ctx)
	case "gradientai_model_provider":
		return listGradientAIModelProviders(ctx)
	case "gradientai_model":
		return listGradientAIModels(ctx)
	case "gradientai_workspace":
		return listGradientAIWorkspaces(ctx)
	case "gradientai_knowledge_base":
		return listGradientAIKnowledgeBases(ctx)
	case "gradientai_guardrail":
		return listGradientAIGuardrails(ctx)
	case "gradientai_agent":
		return listGradientAIAgents(ctx)
	case "gradientai_region":
		return listGradientAIRegions(ctx)
	case "do_project":
		return listDOProjects(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listDroplets(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	droplets, err := client.ListDroplets()
	if err != nil {
		return nil, fmt.Errorf("error listing droplets: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(droplets))
	for _, droplet := range droplets {
		resources = append(resources, core.IntegrationResource{
			Type: "droplet",
			Name: droplet.Name,
			ID:   fmt.Sprintf("%d", droplet.ID),
		})
	}

	return resources, nil
}

func listRegions(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	regions, err := client.ListRegions()
	if err != nil {
		return nil, fmt.Errorf("error listing regions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(regions))
	for _, region := range regions {
		if region.Available {
			resources = append(resources, core.IntegrationResource{
				Type: "region",
				Name: region.Name,
				ID:   region.Slug,
			})
		}
	}

	return resources, nil
}

func listSizes(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	sizes, err := client.ListSizes()
	if err != nil {
		return nil, fmt.Errorf("error listing sizes: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(sizes))
	for _, size := range sizes {
		if size.Available {
			resources = append(resources, core.IntegrationResource{
				Type: "size",
				Name: size.Slug,
				ID:   size.Slug,
			})
		}
	}

	return resources, nil
}

func listImages(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	images, err := client.ListImages("distribution")
	if err != nil {
		return nil, fmt.Errorf("error listing images: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(images))
	for _, image := range images {
		name := image.Name
		if image.Distribution != "" {
			name = fmt.Sprintf("%s %s", image.Distribution, image.Name)
		}

		resources = append(resources, core.IntegrationResource{
			Type: "image",
			Name: name,
			ID:   image.Slug,
		})
	}

	return resources, nil
}

func listSnapshots(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	snapshots, err := client.ListSnapshots()
	if err != nil {
		return nil, fmt.Errorf("error listing snapshots: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(snapshots))
	for _, snapshot := range snapshots {
		resources = append(resources, core.IntegrationResource{
			Type: "snapshot",
			Name: snapshot.Name,
			ID:   snapshot.ID.String(),
		})
	}

	return resources, nil
}

func listDomains(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	domains, err := client.ListDomains()
	if err != nil {
		return nil, fmt.Errorf("error listing domains: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(domains))
	for _, domain := range domains {
		resources = append(resources, core.IntegrationResource{
			Type: "domain",
			Name: domain.Name,
			ID:   domain.Name,
		})
	}
	return resources, nil
}

func listDNSRecords(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	domain := ctx.Parameters["domain"]
	if domain == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	records, err := client.ListDNSRecords(domain)
	if err != nil {
		return nil, fmt.Errorf("error listing DNS records: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(records))
	for _, record := range records {
		resources = append(resources, core.IntegrationResource{
			Type: "dns_record",
			Name: fmt.Sprintf("%s (%s)", record.Name, record.Type),
			ID:   fmt.Sprintf("%d", record.ID),
		})
	}
	return resources, nil
}

func listLoadBalancers(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	loadBalancers, err := client.ListLoadBalancers()
	if err != nil {
		return nil, fmt.Errorf("error listing load balancers: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(loadBalancers))
	for _, lb := range loadBalancers {
		resources = append(resources, core.IntegrationResource{
			Type: "load_balancer",
			Name: lb.Name,
			ID:   lb.ID,
		})
	}

	return resources, nil
}

func listReservedIPs(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	reservedIPs, err := client.ListReservedIPs()
	if err != nil {
		return nil, fmt.Errorf("error listing reserved IPs: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(reservedIPs))
	for _, ip := range reservedIPs {
		resources = append(resources, core.IntegrationResource{
			Type: "reserved_ip",
			Name: ip.IP,
			ID:   ip.IP,
		})
	}

	return resources, nil
}

func listSSHKeys(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	keys, err := client.ListSSHKeys()
	if err != nil {
		return nil, fmt.Errorf("error listing SSH keys: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(keys))
	for _, key := range keys {
		resources = append(resources, core.IntegrationResource{
			Type: "ssh_key",
			Name: key.Name,
			ID:   key.Fingerprint,
		})
	}

	return resources, nil
}

func listAlertPolicies(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	policies, err := client.ListAlertPolicies()
	if err != nil {
		return nil, fmt.Errorf("error listing alert policies: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(policies))
	for _, policy := range policies {
		resources = append(resources, core.IntegrationResource{
			Type: "alert_policy",
			Name: policy.Description,
			ID:   policy.UUID,
		})
	}

	return resources, nil
}

var allSpacesRegions = []string{
	"nyc1",
	"nyc2",
	"nyc3",
	"sfo2",
	"sfo3",
	"ams3",
	"sgp1",
	"fra1",
	"blr1",
	"syd1",
	"lon1",
	"tor1",
	"atl1",
	"ric1",
}

func listSpacesBuckets(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	resources := make([]core.IntegrationResource, 0)
	var firstErr error
	successCount := 0

	for _, region := range allSpacesRegions {
		client, err := NewSpacesClient(ctx.HTTP, ctx.Integration, region)
		if err != nil {
			return nil, fmt.Errorf("failed to create spaces client: %w", err)
		}

		buckets, err := client.ListBuckets()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("region %s: %w", region, err)
			}
			continue
		}

		successCount++
		for _, name := range buckets {
			resources = append(resources, core.IntegrationResource{
				Type: "spaces_bucket",
				Name: fmt.Sprintf("%s (%s)", name, region),
				ID:   fmt.Sprintf("%s/%s", region, name),
			})
		}
	}

	// If no region succeeded and we have an error, it's likely a credentials issue
	if successCount == 0 && firstErr != nil {
		return nil, fmt.Errorf("error listing spaces buckets: %w", firstErr)
	}

	return resources, nil
}

func listVPCs(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	vpcs, err := client.ListVPCs()
	if err != nil {
		return nil, fmt.Errorf("error listing VPCs: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(vpcs))
	for _, vpc := range vpcs {
		resources = append(resources, core.IntegrationResource{
			Type: "vpc",
			Name: vpc.Name,
			ID:   vpc.ID,
		})
	}

	return resources, nil
}

func listGradientAIModelProviders(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Provider groups are top-level models with no parent_uuid.
	providers, err := client.ListGradientAIModelProviders()
	if err != nil {
		return nil, fmt.Errorf("error listing model providers: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(providers))
	for _, p := range providers {
		resources = append(resources, core.IntegrationResource{
			Type: "gradientai_model_provider",
			Name: p.Name,
			ID:   p.UUID,
		})
	}

	return resources, nil
}

func listGradientAIModels(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	models, err := client.ListGradientAIModels()
	if err != nil {
		return nil, fmt.Errorf("error listing models: %w", err)
	}

	// Filter by provider key (lowercase first word of model name) when a provider
	// has been selected. The provider key matches what ListGradientAIModelProviders
	// stores as the UUID of each provider entry.
	// Return empty when no provider is selected so the model dropdown stays disabled.
	providerKey := strings.ToLower(ctx.Parameters["provider"])
	if providerKey == "" {
		return []core.IntegrationResource{}, nil
	}

	resources := make([]core.IntegrationResource, 0, len(models))
	for _, m := range models {
		if providerKey != "" && strings.ToLower(gradientAIProviderName(m.Name)) != providerKey {
			continue
		}
		if !modelSupportsAgents(m) {
			continue
		}
		resources = append(resources, core.IntegrationResource{
			Type: "gradientai_model",
			Name: m.Name,
			ID:   m.UUID,
		})
	}

	return resources, nil
}

func listGradientAIWorkspaces(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	workspaces, err := client.ListGradientAIWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("error listing workspaces: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(workspaces))
	for _, w := range workspaces {
		resources = append(resources, core.IntegrationResource{
			Type: "gradientai_workspace",
			Name: w.Name,
			ID:   w.UUID,
		})
	}

	return resources, nil
}

func listGradientAIKnowledgeBases(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	kbs, err := client.ListGradientAIKnowledgeBases()
	if err != nil {
		if isGradientAINotFound(err) {
			return []core.IntegrationResource{}, nil
		}
		return nil, fmt.Errorf("error listing knowledge bases: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(kbs))
	for _, kb := range kbs {
		resources = append(resources, core.IntegrationResource{
			Type: "gradientai_knowledge_base",
			Name: kb.Name,
			ID:   kb.UUID,
		})
	}

	return resources, nil
}

func listGradientAIGuardrails(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	guardrails, err := client.ListGradientAIGuardrails()
	if err != nil {
		if isGradientAINotFound(err) {
			return []core.IntegrationResource{}, nil
		}
		return nil, fmt.Errorf("error listing guardrails: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(guardrails))
	for _, g := range guardrails {
		resources = append(resources, core.IntegrationResource{
			Type: "gradientai_guardrail",
			Name: g.Name,
			ID:   g.UUID,
		})
	}

	return resources, nil
}

// isGradientAINotFound returns true when the DO API responded with 404 or 422.
// Some GradientAI sub-resources (e.g. guardrails) may not be available on all
// accounts or regions; we treat those as empty rather than a hard error so that
// optional dropdowns show "No resources available" instead of a connection error.
func isGradientAINotFound(err error) bool {
	var apiErr *DOAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound ||
			apiErr.StatusCode == http.StatusUnprocessableEntity
	}
	return false
}

func listGradientAIAgents(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	agents, err := client.ListGradientAIAgents()
	if err != nil {
		return nil, fmt.Errorf("error listing agents: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(agents))
	for _, a := range agents {
		resources = append(resources, core.IntegrationResource{
			Type: "gradientai_agent",
			Name: a.Name,
			ID:   a.UUID,
		})
	}

	return resources, nil
}

func listDOProjects(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	projects, err := client.ListDOProjects()
	if err != nil {
		return nil, fmt.Errorf("error listing projects: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, p := range projects {
		name := p.Name
		if p.IsDefault {
			name = fmt.Sprintf("%s (default)", p.Name)
		}
		resources = append(resources, core.IntegrationResource{
			Type: "do_project",
			Name: name,
			ID:   p.ID,
		})
	}

	return resources, nil
}

func listGradientAIRegions(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	regions, err := client.ListGradientAIDatacenterRegions()
	if err != nil {
		return nil, fmt.Errorf("error listing regions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(regions))
	for _, r := range regions {
		resources = append(resources, core.IntegrationResource{
			Type: "gradientai_region",
			Name: r.Region,
			ID:   r.Region,
		})
	}

	return resources, nil
}

func listApps(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	apps, err := client.ListApps()
	if err != nil {
		return nil, fmt.Errorf("error listing apps: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(apps))
	for _, app := range apps {
		name := app.ID
		if app.Spec != nil {
			name = app.Spec.Name
		}

		resources = append(resources, core.IntegrationResource{
			Type: "app",
			Name: name,
			ID:   app.ID,
		})
	}

	return resources, nil
}
