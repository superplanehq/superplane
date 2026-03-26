package digitalocean

import (
	"fmt"

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

		for _, name := range buckets {
			resources = append(resources, core.IntegrationResource{
				Type: "spaces_bucket",
				Name: fmt.Sprintf("%s (%s)", name, region),
				ID:   fmt.Sprintf("%s/%s", region, name),
			})
		}
	}

	if firstErr != nil {
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
