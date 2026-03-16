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
