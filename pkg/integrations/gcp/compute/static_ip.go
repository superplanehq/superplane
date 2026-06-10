package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

// ResourceTypeStaticIP lists regional EXTERNAL reserved IP addresses (static
// IPs). Unlike ResourceTypeAddress, which the Create VM component uses and keys
// by the IP literal, this type keys each option by its selfLink so the Delete
// and Manage components can derive the region and address name without a
// separate lookup.
const ResourceTypeStaticIP = "staticIP"

// addressGetResp is the subset of a Compute Engine regional address resource we
// read back after creating, deleting, or resolving a static IP.
type addressGetResp struct {
	Name        string   `json:"name"`
	Address     string   `json:"address"`
	Region      string   `json:"region"`
	SelfLink    string   `json:"selfLink"`
	Status      string   `json:"status"`
	AddressType string   `json:"addressType"`
	NetworkTier string   `json:"networkTier"`
	Users       []string `json:"users"`
}

// parseAddressPath extracts (project, region, name) from a value of the form
// `regions/<region>/addresses/<name>` (relative path) or a full GCE selfLink
// URL containing `projects/<project>/regions/<region>/addresses/<name>`. The
// project segment is optional — relative paths from the dropdown have no
// project, but selfLinks do, and the caller must verify it matches the
// integration's bound project before issuing a mutation.
func parseAddressPath(value string) (project, region, name string, err error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return "", "", "", fmt.Errorf("static IP is required")
	}
	if idx := strings.Index(s, "projects/"); idx >= 0 {
		rest := s[idx+len("projects/"):]
		if slash := strings.Index(rest, "/"); slash > 0 {
			project = rest[:slash]
		}
	}
	idx := strings.Index(s, "regions/")
	if idx < 0 {
		return "", "", "", fmt.Errorf("static IP %q must be a path like regions/<region>/addresses/<name> or a GCE selfLink URL", value)
	}
	rest := s[idx+len("regions/"):]
	slash := strings.Index(rest, "/")
	if slash <= 0 {
		return "", "", "", fmt.Errorf("static IP %q is missing a region segment", value)
	}
	region = rest[:slash]
	after := rest[slash+1:]
	const prefix = "addresses/"
	if !strings.HasPrefix(after, prefix) {
		return "", "", "", fmt.Errorf("static IP %q is missing an addresses/ segment", value)
	}
	name = after[len(prefix):]
	if q := strings.IndexAny(name, "/?#"); q >= 0 {
		name = name[:q]
	}
	if region == "" || name == "" {
		return "", "", "", fmt.Errorf("static IP %q is missing a region or name", value)
	}
	return project, region, name, nil
}

// GetAddress reads a single regional address resource.
func GetAddress(ctx context.Context, client Client, project, region, name string) ([]byte, error) {
	if project == "" {
		project = client.ProjectID()
	}
	path := fmt.Sprintf("projects/%s/regions/%s/addresses/%s", project, region, name)
	return client.Get(ctx, path)
}

// WaitForRegionOperation polls a regional operation until it reaches DONE,
// mirroring WaitForZoneOperation for regional resources such as addresses.
func WaitForRegionOperation(ctx context.Context, client Client, project, region, operationName string) error {
	path := fmt.Sprintf("projects/%s/regions/%s/operations/%s", project, region, operationName)
	deadline := time.Now().Add(defaultOperationWaitTimeout)
	ticker := time.NewTicker(operationPollInterval)
	defer ticker.Stop()
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for operation %s", operationName)
		}
		body, err := client.Get(ctx, path)
		if err != nil {
			return err
		}
		var op zoneOperationResp
		if err := json.Unmarshal(body, &op); err != nil {
			return fmt.Errorf("parse operation response: %w", err)
		}
		switch op.Status {
		case opStatusDone:
			if op.Error != nil && len(op.Error.Errors) > 0 {
				msg := op.Error.Errors[0].Message
				if msg == "" {
					msg = op.Error.Errors[0].Code
				}
				return fmt.Errorf("operation failed: %s", msg)
			}
			return nil
		case opStatusPending, opStatusRunning:
		default:
			return fmt.Errorf("unexpected operation status: %s", op.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

type addressesScopedList struct {
	Addresses []*addressItem `json:"addresses"`
}

type addressesAggregatedListResp struct {
	Items         map[string]*addressesScopedList `json:"items"`
	NextPageToken string                          `json:"nextPageToken"`
}

// ListStaticIPResources lists every EXTERNAL reserved IP in the project across
// all regions using the aggregatedList endpoint, keyed by selfLink so callers
// can derive the region and name. Aggregating across regions lets the dropdown
// behave like the VM Instance picker — no region selector required. Internal
// addresses are skipped because static IPs attached to VMs are always external.
func ListStaticIPResources(ctx context.Context, c Client, project, instance string) ([]core.IntegrationResource, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		project = c.ProjectID()
	}

	// When an instance is supplied (the Manage Static IP attach flow), only list
	// IPs in that VM's region — a regional static IP can attach only to a VM in
	// the same region. Expressions can't be resolved at list time, so we fall
	// back to listing every region.
	regionFilter := ""
	if instance = strings.TrimSpace(instance); instance != "" && !strings.Contains(instance, "{{") {
		if _, zone, _, err := parseInstancePath(instance); err == nil {
			regionFilter = deriveRegionFromZone(zone)
		}
	}

	path := fmt.Sprintf("projects/%s/aggregated/addresses", project)

	var out []core.IntegrationResource
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp addressesAggregatedListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse addresses aggregated response: %w", err)
		}
		for _, scoped := range resp.Items {
			if scoped == nil {
				continue
			}
			for _, a := range scoped.Addresses {
				if a == nil || a.AddressType != AddressTypeExternal || a.SelfLink == "" {
					continue
				}
				region := lastSegment(a.Region)
				if regionFilter != "" && region != regionFilter {
					continue
				}
				out = append(out, core.IntegrationResource{
					Type: ResourceTypeStaticIP,
					Name: staticIPLabel(a.Name, a.Address, region),
					ID:   a.SelfLink,
				})
			}
		}
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return out, nil
}

// staticIPLabel renders a dropdown label like "web-ip (34.1.2.3, us-central1)"
// so users can tell static IPs apart without a separate region field.
func staticIPLabel(name, address, region string) string {
	switch {
	case address != "" && region != "":
		return fmt.Sprintf("%s (%s, %s)", name, address, region)
	case address != "":
		return fmt.Sprintf("%s (%s)", name, address)
	case region != "":
		return fmt.Sprintf("%s (%s)", name, region)
	default:
		return name
	}
}
