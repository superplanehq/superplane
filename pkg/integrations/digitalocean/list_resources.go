package digitalocean

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

func (d *DigitalOcean) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "droplet":
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
				Type: resourceType,
				Name: droplet.Name,
				ID:   fmt.Sprintf("%d", droplet.ID),
			})
		}
		return resources, nil

	case "region":
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
					Type: resourceType,
					Name: region.Name,
					ID:   region.Slug,
				})
			}
		}
		return resources, nil

	case "size":
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
					Type: resourceType,
					Name: size.Slug,
					ID:   size.Slug,
				})
			}
		}
		return resources, nil

	case "image":
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
				Type: resourceType,
				Name: name,
				ID:   image.Slug,
			})
		}
		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}
