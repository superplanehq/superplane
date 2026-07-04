package cloudsql

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// ResourceTypeInstance lists the Cloud SQL instances in the project.
	ResourceTypeInstance = "cloudsqlInstance"
	// ResourceTypeDatabase lists the databases in a selected instance.
	ResourceTypeDatabase = "cloudsqlDatabase"
)

// ResourceTypeRegion lists the regions where Cloud SQL is available, and
// ResourceTypeTier lists the predefined machine tiers (optionally for a region).
// Both are derived from the tiers.list endpoint.
const (
	ResourceTypeRegion = "cloudsqlRegion"
	ResourceTypeTier   = "cloudsqlTier"
)

// ListInstanceResources lists Cloud SQL instances for the instance dropdown.
func ListInstanceResources(ctx context.Context, client Client) ([]core.IntegrationResource, error) {
	instances, err := ListInstances(ctx, client, client.ProjectID())
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(instances))
	for _, inst := range instances {
		label := inst.Name
		if inst.DatabaseVersion != "" {
			label = fmt.Sprintf("%s (%s)", inst.Name, inst.DatabaseVersion)
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeInstance, Name: label, ID: inst.Name})
	}
	return out, nil
}

// ListDatabaseResources lists the databases in the selected instance for the
// database dropdown.
func ListDatabaseResources(ctx context.Context, client Client, instance string) ([]core.IntegrationResource, error) {
	if instance == "" {
		return nil, nil
	}
	databases, err := ListDatabases(ctx, client, client.ProjectID(), instance)
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(databases))
	for _, db := range databases {
		out = append(out, core.IntegrationResource{Type: ResourceTypeDatabase, Name: db.Name, ID: db.Name})
	}
	return out, nil
}

// ListRegionResources lists the regions Cloud SQL offers, derived from the union
// of the regions each available tier is offered in.
func ListRegionResources(ctx context.Context, client Client) ([]core.IntegrationResource, error) {
	tiers, err := ListTiers(ctx, client, client.ProjectID())
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	regions := make([]string, 0)
	for _, t := range tiers {
		for _, r := range t.Region {
			if _, ok := seen[r]; ok {
				continue
			}
			seen[r] = struct{}{}
			regions = append(regions, r)
		}
	}
	sort.Strings(regions)
	out := make([]core.IntegrationResource, 0, len(regions))
	for _, r := range regions {
		out = append(out, core.IntegrationResource{Type: ResourceTypeRegion, Name: r, ID: r})
	}
	return out, nil
}

// ListTierResources lists the predefined machine tiers available in the selected
// region. It returns an empty list until a region is chosen, mirroring the
// region-dependent pickers in the Compute integration.
func ListTierResources(ctx context.Context, client Client, region string) ([]core.IntegrationResource, error) {
	region = strings.TrimSpace(region)
	if region == "" {
		return []core.IntegrationResource{}, nil
	}
	tiers, err := ListTiers(ctx, client, client.ProjectID())
	if err != nil {
		return nil, err
	}
	out := make([]core.IntegrationResource, 0, len(tiers))
	for _, t := range tiers {
		if !slices.Contains(t.Region, region) {
			continue
		}
		out = append(out, core.IntegrationResource{Type: ResourceTypeTier, Name: tierLabel(t), ID: t.Tier})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// tierLabel renders a tier with its memory, e.g. "db-custom-2-7680 (7.5 GB RAM)".
func tierLabel(t Tier) string {
	if t.RAM > 0 {
		return fmt.Sprintf("%s (%.1f GB RAM)", t.Tier, float64(t.RAM)/(1<<30))
	}
	return t.Tier
}
