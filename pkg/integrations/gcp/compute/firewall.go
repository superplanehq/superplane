package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// Firewall rule direction and action constants used by the firewall rule
// components. A firewall rule either allows or denies traffic in one direction.
const (
	FirewallDirectionIngress = "INGRESS"
	FirewallDirectionEgress  = "EGRESS"

	FirewallActionAllow = "allow"
	FirewallActionDeny  = "deny"

	// Firewall Rules Logging metadata options (logConfig.metadata).
	FirewallLogMetadataIncludeAll = "INCLUDE_ALL_METADATA"
	FirewallLogMetadataExcludeAll = "EXCLUDE_ALL_METADATA"

	// Target type — which instances a rule applies to. Mirrors the GCP Console
	// "Targets" dropdown: all instances, instances with target tags, or instances
	// running as target service accounts (mutually exclusive).
	FirewallTargetAll = "all"
	// Source filter type (INGRESS) — how incoming traffic is matched. Mirrors the
	// Console "Source filter" dropdown: IP ranges, source tags, or source service
	// accounts (mutually exclusive).
	FirewallSourceRanges = "ranges"
	// Shared tag/service-account selectors for both target type and source filter.
	FirewallFilterTags            = "tags"
	FirewallFilterServiceAccounts = "serviceAccounts"

	// Protocols & ports mode — mirrors the GCP Console radio: match a specific
	// list of protocols/ports, or all of them.
	FirewallProtocolsSpecified = "specified"
	FirewallProtocolsAll       = "all"
)

// allProtocolsRule is the single allowed/denied entry that matches every
// protocol and port (the "all protocols and ports" mode). The Compute API
// represents "all" as a protocol named "all" with no ports.
func allProtocolsRule() []map[string]any {
	return []map[string]any{{"IPProtocol": "all"}}
}

// FirewallRuleSpec is one protocol/ports entry of a firewall rule, configured as
// a list item. It maps to a single element of the Compute Engine `allowed` or
// `denied` arrays.
type FirewallRuleSpec struct {
	Protocol string `mapstructure:"protocol"`
	Ports    string `mapstructure:"ports"`
}

// FirewallNodeMetadata is persisted on the node so the collapsed UI can show the
// targeted firewall rule name. Update/Delete Firewall Rule share it.
type FirewallNodeMetadata struct {
	FirewallName string `json:"firewallName" mapstructure:"firewallName"`
}

// firewallRule is one entry of the allowed/denied arrays of a firewall rule.
type firewallRule struct {
	IPProtocol string   `json:"IPProtocol"`
	Ports      []string `json:"ports,omitempty"`
}

// firewallLogConfig mirrors a firewall rule's logConfig (Firewall Rules Logging).
type firewallLogConfig struct {
	Enable   bool   `json:"enable"`
	Metadata string `json:"metadata,omitempty"`
}

type firewallGetResp struct {
	Name                  string             `json:"name"`
	SelfLink              string             `json:"selfLink"`
	Network               string             `json:"network"`
	Direction             string             `json:"direction"`
	Priority              int64              `json:"priority"`
	Description           string             `json:"description"`
	Disabled              bool               `json:"disabled"`
	Allowed               []firewallRule     `json:"allowed"`
	Denied                []firewallRule     `json:"denied"`
	SourceRanges          []string           `json:"sourceRanges"`
	DestinationRanges     []string           `json:"destinationRanges"`
	SourceTags            []string           `json:"sourceTags"`
	TargetTags            []string           `json:"targetTags"`
	SourceServiceAccounts []string           `json:"sourceServiceAccounts"`
	TargetServiceAccounts []string           `json:"targetServiceAccounts"`
	LogConfig             *firewallLogConfig `json:"logConfig"`
	CreationTimestamp     string             `json:"creationTimestamp"`
}

// parseFirewallPath extracts (project, name) from a firewall rule value. Compute
// Engine firewall rules are global resources, so the accepted forms are:
//   - a full selfLink URL containing projects/<project>/global/firewalls/<name>
//   - a relative path global/firewalls/<name> or projects/<project>/global/firewalls/<name>
//   - a bare firewall rule name (no slash), in which case project is empty
//
// The project segment is optional — relative dropdown values and bare names
// carry no project, but chained selfLinks do, and the caller must verify it
// matches the integration's bound project before issuing a mutating call.
func parseFirewallPath(value string) (project, name string, err error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return "", "", errors.New("firewall rule is required")
	}

	if idx := strings.Index(s, "projects/"); idx >= 0 {
		rest := s[idx+len("projects/"):]
		if slash := strings.Index(rest, "/"); slash > 0 {
			project = rest[:slash]
		}
	}

	const marker = "global/firewalls/"
	if idx := strings.Index(s, marker); idx >= 0 {
		name = s[idx+len(marker):]
	} else if !strings.Contains(s, "/") {
		name = s
	} else {
		return "", "", fmt.Errorf("firewall rule %q must be a name or a path like global/firewalls/<name> or a GCE selfLink URL", value)
	}

	if q := strings.IndexAny(name, "/?#"); q >= 0 {
		name = name[:q]
	}
	if name == "" {
		return "", "", fmt.Errorf("firewall rule %q is missing a name", value)
	}
	return project, name, nil
}

// GetFirewall reads a global firewall rule by name.
func GetFirewall(ctx context.Context, client Client, project, name string) ([]byte, error) {
	if project == "" {
		project = client.ProjectID()
	}
	path := fmt.Sprintf("projects/%s/global/firewalls/%s", project, name)
	return client.Get(ctx, path)
}

// firewallConsoleURL builds the Google Cloud Console URL for a VPC firewall
// rule's details page, giving the user a one-click way to inspect the rule.
func firewallConsoleURL(project, name string) string {
	return fmt.Sprintf("https://console.cloud.google.com/networking/firewalls/details/%s?project=%s", name, project)
}

// FirewallPayloadFromGetResponse converts a firewalls.get response body into the
// flat payload emitted by the firewall rule components.
func FirewallPayloadFromGetResponse(body []byte, project string) (map[string]any, error) {
	var fw firewallGetResp
	if err := json.Unmarshal(body, &fw); err != nil {
		return nil, fmt.Errorf("parse firewall rule response: %w", err)
	}

	payload := map[string]any{
		"name":              fw.Name,
		"selfLink":          fw.SelfLink,
		"network":           lastSegment(fw.Network),
		"direction":         fw.Direction,
		"priority":          fw.Priority,
		"disabled":          fw.Disabled,
		"creationTimestamp": fw.CreationTimestamp,
		"link":              firewallConsoleURL(project, fw.Name),
	}
	if fw.Description != "" {
		payload["description"] = fw.Description
	}
	// A firewall rule is either an allow rule or a deny rule. Surface the action
	// and the matching protocol/ports list under the GCP-native key.
	if len(fw.Allowed) > 0 {
		payload["action"] = "ALLOW"
		payload["allowed"] = fw.Allowed
	} else if len(fw.Denied) > 0 {
		payload["action"] = "DENY"
		payload["denied"] = fw.Denied
	}
	if len(fw.SourceRanges) > 0 {
		payload["sourceRanges"] = fw.SourceRanges
	}
	if len(fw.DestinationRanges) > 0 {
		payload["destinationRanges"] = fw.DestinationRanges
	}
	if len(fw.SourceTags) > 0 {
		payload["sourceTags"] = fw.SourceTags
	}
	if len(fw.TargetTags) > 0 {
		payload["targetTags"] = fw.TargetTags
	}
	if len(fw.SourceServiceAccounts) > 0 {
		payload["sourceServiceAccounts"] = fw.SourceServiceAccounts
	}
	if len(fw.TargetServiceAccounts) > 0 {
		payload["targetServiceAccounts"] = fw.TargetServiceAccounts
	}
	if fw.LogConfig != nil {
		payload["loggingEnabled"] = fw.LogConfig.Enable
		if fw.LogConfig.Enable && fw.LogConfig.Metadata != "" {
			payload["logMetadata"] = fw.LogConfig.Metadata
		}
	}
	return payload, nil
}

// validateFirewallPriority checks a configured priority is within the Compute
// Engine firewall range. A nil priority means "unset" (the API applies its own
// default of 1000), so it is accepted.
func validateFirewallPriority(priority *int) error {
	if priority == nil {
		return nil
	}
	if *priority < 0 || *priority > 65535 {
		return fmt.Errorf("invalid priority %d: must be between 0 and 65535", *priority)
	}
	return nil
}

// normalizeFirewallLogMetadata validates the Firewall Rules Logging metadata
// option, defaulting to INCLUDE_ALL_METADATA.
func normalizeFirewallLogMetadata(metadata string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(metadata)) {
	case "", FirewallLogMetadataIncludeAll:
		return FirewallLogMetadataIncludeAll, nil
	case FirewallLogMetadataExcludeAll:
		return FirewallLogMetadataExcludeAll, nil
	default:
		return "", fmt.Errorf("invalid log metadata %q: must be INCLUDE_ALL_METADATA or EXCLUDE_ALL_METADATA", metadata)
	}
}

// resolveFirewallTargeting keeps only the tag/service-account lists that match
// the selected target type and source-filter type. The "Targets" and "Source
// filter" dropdowns make the choice mutually exclusive in the form, so this
// ensures the dropdown selection — not a stale hidden input — decides what the
// rule targets. The service-account lists passed in should already be merged
// (dropdown selections + custom entries); source filters only apply to INGRESS.
func resolveFirewallTargeting(
	targetType, sourceFilterType string, ingress bool,
	targetTags, targetServiceAccounts, sourceTags, sourceServiceAccounts []string,
) (effTargetTags, effTargetSAs, effSourceTags, effSourceSAs []string) {
	switch strings.TrimSpace(targetType) {
	case FirewallFilterTags:
		effTargetTags = targetTags
	case FirewallFilterServiceAccounts:
		effTargetSAs = targetServiceAccounts
	}
	if ingress {
		switch strings.TrimSpace(sourceFilterType) {
		case FirewallFilterTags:
			effSourceTags = sourceTags
		case FirewallFilterServiceAccounts:
			effSourceSAs = sourceServiceAccounts
		}
	}
	return
}

// validateFirewallTargetsAndSources enforces a Compute Engine constraint: a
// single firewall rule filters by network tags OR by service accounts, never a
// mix of the two. Catching it here gives a clearer error than the API's.
func validateFirewallTargetsAndSources(sourceTags, targetTags, sourceServiceAccounts, targetServiceAccounts []string) error {
	usesTags := len(sourceTags) > 0 || len(targetTags) > 0
	usesServiceAccounts := len(sourceServiceAccounts) > 0 || len(targetServiceAccounts) > 0
	if usesTags && usesServiceAccounts {
		return errors.New("a firewall rule cannot combine network tags and service accounts; use one or the other for both source and target filters")
	}
	return nil
}

// normalizeFirewallDirection maps a configured direction to the value the
// Compute API expects, defaulting to INGRESS.
func normalizeFirewallDirection(direction string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(direction)) {
	case "", FirewallDirectionIngress:
		return FirewallDirectionIngress, nil
	case FirewallDirectionEgress:
		return FirewallDirectionEgress, nil
	default:
		return "", fmt.Errorf("invalid direction %q: must be INGRESS or EGRESS", direction)
	}
}

// buildFirewallRules converts the configured protocol/ports list into the
// Compute Engine allowed/denied array shape. Entries with an empty protocol are
// skipped; ports are split on commas. An empty ports list means "all ports" for
// that protocol, which the API represents by omitting the ports field.
func buildFirewallRules(specs []FirewallRuleSpec) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(specs))
	for _, s := range specs {
		protocol := strings.ToLower(strings.TrimSpace(s.Protocol))
		if protocol == "" {
			continue
		}
		entry := map[string]any{"IPProtocol": protocol}
		if ports := splitFirewallPorts(s.Ports); len(ports) > 0 {
			// "all" is not a real protocol with ports; reject ports on it so the
			// API does not return a confusing error.
			if protocol == "all" {
				return nil, fmt.Errorf("protocol %q cannot specify ports", protocol)
			}
			entry["ports"] = ports
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil, errors.New("at least one protocol is required")
	}
	return out, nil
}

// splitFirewallPorts splits a comma-separated ports string into individual port
// or port-range entries (e.g. "80, 443, 8080-8090"), trimming blanks.
func splitFirewallPorts(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// mergeDedup concatenates already-trimmed string lists, dropping duplicates and
// preserving order. Used to combine the service-account dropdown selections
// with any custom (cross-project) entries.
func mergeDedup(lists ...[]string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, list := range lists {
		for _, v := range list {
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// trimList trims and drops empty entries from a list of strings (CIDR ranges,
// tags, etc.).
func trimList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// resolveFirewallNodeMetadata stores the targeted firewall rule name on the node
// so the collapsed UI can display something meaningful. Update/Delete share it.
func resolveFirewallNodeMetadata(ctx core.SetupContext, firewallValue string) error {
	if strings.Contains(firewallValue, "{{") {
		return ctx.Metadata.Set(FirewallNodeMetadata{FirewallName: firewallValue})
	}
	_, name, err := parseFirewallPath(firewallValue)
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(FirewallNodeMetadata{FirewallName: name})
}
