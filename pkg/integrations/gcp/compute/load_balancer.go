package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// Resource types for the load balancer pickers.
const (
	// ResourceTypeInstanceGroup lists zonal instance groups (the VM backends a
	// regional passthrough Network Load Balancer forwards traffic to).
	ResourceTypeInstanceGroup = "instanceGroup"
	// ResourceTypeForwardingRule lists regional external forwarding rules — the
	// "anchor" that identifies a load balancer for Delete.
	ResourceTypeForwardingRule = "forwardingRule"
)

// loadBalancingSchemeExternal is the scheme for a public (internet-facing)
// regional passthrough Network Load Balancer.
const loadBalancingSchemeExternal = "EXTERNAL"

// healthCheckBody assembles the regional health check request body for the given
// protocol (TCP/HTTP/HTTPS) and port, with conservative defaults.
func healthCheckBody(name, protocol string, port int) map[string]any {
	protocol = strings.ToUpper(strings.TrimSpace(protocol))
	if protocol == "" {
		protocol = "TCP"
	}
	body := map[string]any{
		"name":               name,
		"type":               protocol,
		"checkIntervalSec":   5,
		"timeoutSec":         5,
		"healthyThreshold":   2,
		"unhealthyThreshold": 2,
	}
	switch protocol {
	case "HTTP":
		body["httpHealthCheck"] = map[string]any{"port": port, "requestPath": "/"}
	case "HTTPS":
		body["httpsHealthCheck"] = map[string]any{"port": port, "requestPath": "/"}
	default:
		body["tcpHealthCheck"] = map[string]any{"port": port}
	}
	return body
}

// parseRegionalResource extracts (project, region, name) from a regional compute
// resource reference — a full selfLink URL, or a relative path like
// `regions/<region>/<kind>/<name>`. project is empty for relative paths; callers
// that mutate must verify it against the integration's bound project.
func parseRegionalResource(value, kind string) (project, region, name string, err error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return "", "", "", fmt.Errorf("%s is required", strings.TrimSuffix(kind, "s"))
	}
	if idx := strings.Index(s, "projects/"); idx >= 0 {
		rest := s[idx+len("projects/"):]
		if slash := strings.Index(rest, "/"); slash > 0 {
			project = rest[:slash]
		}
	}
	idx := strings.Index(s, "regions/")
	if idx < 0 {
		return "", "", "", fmt.Errorf("%q must be a path like regions/<region>/%s/<name> or a selfLink URL", value, kind)
	}
	rest := s[idx+len("regions/"):]
	slash := strings.Index(rest, "/")
	if slash <= 0 {
		return "", "", "", fmt.Errorf("%q is missing a region segment", value)
	}
	region = rest[:slash]
	after := rest[slash+1:]
	prefix := kind + "/"
	if !strings.HasPrefix(after, prefix) {
		return "", "", "", fmt.Errorf("%q is missing a %s/ segment", value, kind)
	}
	name = after[len(prefix):]
	if q := strings.IndexAny(name, "/?#"); q >= 0 {
		name = name[:q]
	}
	if region == "" || name == "" {
		return "", "", "", fmt.Errorf("%q is missing a region or name", value)
	}
	return project, region, name, nil
}

// --- Read responses used by the create assembly and delete teardown ---

type forwardingRuleGetResp struct {
	Name           string `json:"name"`
	SelfLink       string `json:"selfLink"`
	Region         string `json:"region"`
	IPAddress      string `json:"IPAddress"`
	IPProtocol     string `json:"IPProtocol"`
	BackendService string `json:"backendService"`
	Target         string `json:"target"`
}

type backendServiceGetResp struct {
	Name         string   `json:"name"`
	SelfLink     string   `json:"selfLink"`
	Region       string   `json:"region"`
	HealthChecks []string `json:"healthChecks"`
}

// --- Instance group picker (aggregated across zones) ---

type instanceGroupItem struct {
	Name     string `json:"name"`
	SelfLink string `json:"selfLink"`
	Zone     string `json:"zone"`
	Size     int    `json:"size"`
}

type instanceGroupsScopedList struct {
	InstanceGroups []*instanceGroupItem `json:"instanceGroups"`
}

type instanceGroupsAggregatedListResp struct {
	Items         map[string]*instanceGroupsScopedList `json:"items"`
	NextPageToken string                               `json:"nextPageToken"`
}

// ListInstanceGroupResources lists every instance group in the project across all
// zones, keyed by selfLink so the backend service can reference it directly.
func ListInstanceGroupResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		project = c.ProjectID()
	}
	path := fmt.Sprintf("projects/%s/aggregated/instanceGroups", project)

	var out []core.IntegrationResource
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp instanceGroupsAggregatedListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse instance groups aggregated response: %w", err)
		}
		for _, scoped := range resp.Items {
			if scoped == nil {
				continue
			}
			for _, g := range scoped.InstanceGroups {
				if g == nil || g.SelfLink == "" {
					continue
				}
				zone := lastSegment(g.Zone)
				out = append(out, core.IntegrationResource{
					Type: ResourceTypeInstanceGroup,
					Name: fmt.Sprintf("%s (%s, %d VMs)", g.Name, zone, g.Size),
					ID:   g.SelfLink,
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

// --- Forwarding rule picker (regional external, for Delete) ---

type forwardingRuleItem struct {
	Name                string `json:"name"`
	SelfLink            string `json:"selfLink"`
	Region              string `json:"region"`
	IPAddress           string `json:"IPAddress"`
	LoadBalancingScheme string `json:"loadBalancingScheme"`
	BackendService      string `json:"backendService"`
}

type forwardingRulesScopedList struct {
	ForwardingRules []*forwardingRuleItem `json:"forwardingRules"`
}

type forwardingRulesAggregatedListResp struct {
	Items         map[string]*forwardingRulesScopedList `json:"items"`
	NextPageToken string                                `json:"nextPageToken"`
}

// ListForwardingRuleResources lists regional EXTERNAL forwarding rules across all
// regions, keyed by selfLink. These are the entry points of passthrough Network
// Load Balancers, so the Delete component targets one of them.
func ListForwardingRuleResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		project = c.ProjectID()
	}
	path := fmt.Sprintf("projects/%s/aggregated/forwardingRules", project)

	var out []core.IntegrationResource
	var pageToken string
	for {
		body, err := c.Get(ctx, withPageToken(path, pageToken))
		if err != nil {
			return nil, err
		}
		var resp forwardingRulesAggregatedListResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse forwarding rules aggregated response: %w", err)
		}
		for _, scoped := range resp.Items {
			if scoped == nil {
				continue
			}
			for _, f := range scoped.ForwardingRules {
				// Regional external rules backed by a backend service only. This
				// excludes global rules (HTTP(S) load balancers) and legacy
				// target-pool NLBs — neither of which this component manages, and
				// which Delete cannot fully tear down.
				if f == nil || f.SelfLink == "" || f.Region == "" || f.LoadBalancingScheme != loadBalancingSchemeExternal || f.BackendService == "" {
					continue
				}
				region := lastSegment(f.Region)
				out = append(out, core.IntegrationResource{
					Type: ResourceTypeForwardingRule,
					Name: staticIPLabel(f.Name, f.IPAddress, region),
					ID:   f.SelfLink,
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
