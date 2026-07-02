package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// Enabled-state select values for Update Firewall Rule. A select option must not
// use an empty string value (the frontend's Radix-based select throws on empty
// item values), so an explicit "no change" sentinel is used instead.
const (
	FirewallEnabledNoChange = "NO_CHANGE"
	FirewallEnabledEnabled  = "ENABLED"
	FirewallEnabledDisabled = "DISABLED"

	// FirewallTargetingNoChange is the "leave this side's targeting untouched"
	// sentinel for the targetType / sourceFilterType selects. It shares the same
	// wire value as FirewallEnabledNoChange because empty select option values are
	// forbidden by the frontend Radix select.
	FirewallTargetingNoChange = "NO_CHANGE"
)

type UpdateFirewall struct{}

type UpdateFirewallSpec struct {
	Firewall                    string             `mapstructure:"firewall"`
	EnabledState                string             `mapstructure:"enabledState"`
	Priority                    *int               `mapstructure:"priority"`
	ProtocolsAndPorts           string             `mapstructure:"protocolsAndPorts"`
	Rules                       []FirewallRuleSpec `mapstructure:"rules"`
	Ranges                      []string           `mapstructure:"ranges"`
	TargetType                  string             `mapstructure:"targetType"`
	TargetTags                  []string           `mapstructure:"targetTags"`
	TargetServiceAccounts       []string           `mapstructure:"targetServiceAccounts"`
	TargetServiceAccountsCustom []string           `mapstructure:"targetServiceAccountsCustom"`
	SourceFilterType            string             `mapstructure:"sourceFilterType"`
	SourceTags                  []string           `mapstructure:"sourceTags"`
	SourceServiceAccounts       []string           `mapstructure:"sourceServiceAccounts"`
	SourceServiceAccountsCustom []string           `mapstructure:"sourceServiceAccountsCustom"`
	Logging                     string             `mapstructure:"logging"`
	LogMetadata                 string             `mapstructure:"logMetadata"`
	Description                 *string            `mapstructure:"description"`
}

func (u *UpdateFirewall) Name() string {
	return "gcp.compute.updateFirewallRule"
}

func (u *UpdateFirewall) Label() string {
	return "Compute • Update Firewall Rule"
}

func (u *UpdateFirewall) Description() string {
	return "Update a VPC firewall rule: its protocols and ports, ranges, priority, targets and source filters, description, or enabled state"
}

func (u *UpdateFirewall) Documentation() string {
	return `The Update Firewall Rule component changes an existing VPC firewall rule. **Toggle on only the fields you want to change; everything left off is untouched.** Enabled state, protocols and ports, targets, source filter, and logs are dropdowns you toggle on and then pick a value.

## Use Cases

- **Adjust access**: Change the allowed/denied protocols and ports
- **Widen or narrow scope**: Update the source/destination CIDR ranges or the targets
- **Re-prioritize**: Change the rule's priority
- **Pause a rule**: Disable a rule without deleting it, then re-enable it later

## Configuration

- **Firewall rule**: The firewall rule to update (required)
- **Enabled state** (toggle): enable or disable the rule
- **Priority** (toggle): new priority (0-65535); lower numbers take precedence
- **Protocols and ports** (toggle): "Specified protocols and ports" (replace with a list) or "All protocols and ports" (match everything). The rule keeps its existing action (allow or deny)
- **Ranges** (toggle): Replace the rule's CIDR ranges. Independent of the **Source filter**; applied as source ranges for INGRESS rules and destination ranges for EGRESS rules (the rule's direction is fixed). ` + "`sourceRanges`" + ` may legitimately coexist with source tags **or** source service accounts, and editing the source filter does **not** touch ranges.
- **Targets** (toggle): "All instances in the network", "Specified target tags", or "Specified service accounts". Choosing tags or service accounts **clears the other automatically** — a rule cannot use both. A specified tags/service-accounts selection with an empty list is rejected (it would silently broaden the rule to all instances).
- **Source filter (INGRESS only, toggle)**: "IP ranges only" (clears source tags/service accounts), "Source tags", or "Source service accounts". Toggling this on for an EGRESS rule is rejected.
- **Logs** (toggle): turn Firewall Rules Logging on or off (with optional metadata)
- **Description** (toggle): Replace the rule's description

> A firewall rule filters by **network tags** or by **service accounts**, never both. Switching a rule from one to the other is a single dropdown choice — the component auto-clears the opposite side. The component still rejects an update whose **result** would mix tags and service accounts across the Targets and Source filter sides.

## Output

Emits the updated firewall rule: name, selfLink, network, direction, priority, action, the allowed/denied protocols, source/destination ranges, target/source tags, target/source service accounts, disabled, logging state, creationTimestamp, and a console link.

## Important Notes

- A rule's **network** and **direction** are fixed at creation and cannot be changed; this component cannot switch an allow rule to a deny rule.
- You must change at least one field.
- Requires the ` + "`roles/compute.securityAdmin`" + ` IAM role (or ` + "`roles/compute.admin`" + `).
- The **service-account dropdowns** additionally require ` + "`iam.serviceAccounts.list`" + ` (e.g. ` + "`roles/iam.serviceAccountViewer`" + `); without it, use the custom field to enter emails directly. Cross-project and non-existent service accounts aren't validated by GCP, so prefer the dropdown.
- The component waits for the underlying global operation to complete before emitting.`
}

func (u *UpdateFirewall) Icon() string {
	return "shield"
}

func (u *UpdateFirewall) Color() string {
	return "blue"
}

func (u *UpdateFirewall) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateFirewall) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "firewall",
			Label:       "Firewall rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The firewall rule to update.",
			Placeholder: "Select firewall rule",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeFirewall},
			},
		},
		{
			Name:        "enabledState",
			Label:       "Enabled state",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Default:     FirewallEnabledEnabled,
			Description: "Toggle on to enable or disable the rule; leave off to keep its current state.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Enabled", Value: FirewallEnabledEnabled},
				{Label: "Disabled", Value: FirewallEnabledDisabled},
			}}},
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "New rule priority (0-65535). Lower numbers take precedence.",
		},
		{
			Name:        "protocolsAndPorts",
			Label:       "Protocols and ports",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Default:     FirewallProtocolsSpecified,
			Description: "Toggle on to change the rule's protocols/ports; leave off to keep them. Replace them with a specified list, or match all protocols and ports.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Specified protocols and ports", Value: FirewallProtocolsSpecified},
				{Label: "All protocols and ports", Value: FirewallProtocolsAll},
			}}},
		},
		{
			Name:        "rules",
			Label:       "Protocols & ports",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Replace the rule's protocols/ports. Leave ports empty to match all ports; use protocol \"all\" to match every protocol. The rule keeps its existing allow/deny action.",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "protocolsAndPorts", Values: []string{FirewallProtocolsSpecified}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "protocolsAndPorts", Values: []string{FirewallProtocolsSpecified}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Protocol",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "protocol",
								Label:       "Protocol",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Default:     "tcp",
								Description: "IP protocol (e.g. tcp, udp, icmp, or all).",
								Placeholder: "e.g. tcp",
							},
							{
								Name:        "ports",
								Label:       "Ports",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Description: "Comma-separated ports or ranges (e.g. 80, 443, 8080-8090). Leave empty for all ports.",
								Placeholder: "e.g. 80, 443",
							},
						},
					},
				},
			},
		},
		{
			Name:        "ranges",
			Label:       "Source / destination ranges",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Replace the rule's CIDR ranges. Applied as source ranges for INGRESS rules and destination ranges for EGRESS rules.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Range",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name:        "targetType",
			Label:       "Targets",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Default:     FirewallTargetAll,
			Description: "Toggle on to change which instances the rule applies to; leave off to keep the current targets. Switching to tags or service accounts clears the other automatically (a rule cannot use both).",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "All instances in the network", Value: FirewallTargetAll},
				{Label: "Specified target tags", Value: FirewallFilterTags},
				{Label: "Specified service accounts", Value: FirewallFilterServiceAccounts},
			}}},
		},
		{
			Name:        "targetTags",
			Label:       "Target tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Replace the rule's target tags (the VMs it applies to).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Tag",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "targetType", Values: []string{FirewallFilterTags}},
			},
		},
		{
			Name:        "targetServiceAccounts",
			Label:       "Target service accounts",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Replace the rule's target service accounts. Cannot be combined with network tags.",
			Placeholder: "Select service accounts",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeServiceAccount, Multi: true},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "targetType", Values: []string{FirewallFilterServiceAccounts}},
			},
		},
		{
			Name:        "targetServiceAccountsCustom",
			Label:       "Target service accounts (custom / cross-project)",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Additional target service-account emails not shown in the dropdown (e.g. from another project). Merged with the selections above.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Service account email",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "targetType", Values: []string{FirewallFilterServiceAccounts}},
			},
		},
		{
			Name:        "sourceFilterType",
			Label:       "Source filter",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Default:     FirewallSourceRanges,
			Description: "Toggle on to change how incoming traffic is matched (INGRESS rules only); leave off to keep source tags/service accounts as-is. \"IP ranges only\" clears source tags/service accounts (CIDR ranges are edited in the Source / destination ranges field above). Toggling this on for an EGRESS rule is rejected.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "IP ranges only", Value: FirewallSourceRanges},
				{Label: "Source tags", Value: FirewallFilterTags},
				{Label: "Source service accounts", Value: FirewallFilterServiceAccounts},
			}}},
		},
		{
			Name:        "sourceTags",
			Label:       "Source tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Replace the rule's source tags (INGRESS rules only).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Tag",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceFilterType", Values: []string{FirewallFilterTags}},
			},
		},
		{
			Name:        "sourceServiceAccounts",
			Label:       "Source service accounts",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Replace the rule's source service accounts (INGRESS rules only). Cannot be combined with network tags.",
			Placeholder: "Select service accounts",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeServiceAccount, Multi: true},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceFilterType", Values: []string{FirewallFilterServiceAccounts}},
			},
		},
		{
			Name:        "sourceServiceAccountsCustom",
			Label:       "Source service accounts (custom / cross-project)",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Additional source service-account emails not shown in the dropdown (e.g. from another project). Merged with the selections above. INGRESS rules only.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Service account email",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceFilterType", Values: []string{FirewallFilterServiceAccounts}},
			},
		},
		{
			Name:        "logging",
			Label:       "Logs",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Default:     FirewallEnabledEnabled,
			Description: "Toggle on to turn Firewall Rules Logging on or off; leave off to keep the current setting.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Enabled", Value: FirewallEnabledEnabled},
				{Label: "Disabled", Value: FirewallEnabledDisabled},
			}}},
		},
		{
			Name:        "logMetadata",
			Label:       "Log metadata",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     FirewallLogMetadataIncludeAll,
			Description: "Whether firewall logs include metadata. Only applies when logging is being enabled.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Include all metadata", Value: FirewallLogMetadataIncludeAll},
				{Label: "Exclude all metadata", Value: FirewallLogMetadataExcludeAll},
			}}},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "logging", Values: []string{FirewallEnabledEnabled}},
			},
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Replace the rule's description.",
			Placeholder: "Firewall rule description",
		},
	}
}

func (u *UpdateFirewall) Setup(ctx core.SetupContext) error {
	spec := UpdateFirewallSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Firewall) == "" {
		return errors.New("firewall rule is required")
	}
	if err := validateFirewallEnabledState(spec.EnabledState); err != nil {
		return err
	}
	if err := validateFirewallEnabledState(spec.Logging); err != nil {
		return err
	}
	if _, err := normalizeFirewallLogMetadata(spec.LogMetadata); err != nil {
		return err
	}
	if err := validateFirewallPriority(spec.Priority); err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(spec.ProtocolsAndPorts), FirewallProtocolsSpecified) {
		if _, err := buildFirewallRules(spec.Rules); err != nil {
			return err
		}
	}
	if err := validateFirewallTargetType(spec.TargetType); err != nil {
		return err
	}
	if err := validateFirewallSourceFilterType(spec.SourceFilterType); err != nil {
		return err
	}
	mergedTargetServiceAccounts := mergeDedup(trimList(spec.TargetServiceAccounts), trimList(spec.TargetServiceAccountsCustom))
	mergedSourceServiceAccounts := mergeDedup(trimList(spec.SourceServiceAccounts), trimList(spec.SourceServiceAccountsCustom))
	// Source filters only apply to INGRESS; the rule's direction is unknown at
	// Setup (no GetFirewall), so assume INGRESS here. EGRESS misuse is rejected at
	// Execute once the current rule's direction is known.
	effTT, effTSA, effST, effSSA := resolveFirewallTargeting(
		spec.TargetType, spec.SourceFilterType, true,
		trimList(spec.TargetTags), mergedTargetServiceAccounts,
		trimList(spec.SourceTags), mergedSourceServiceAccounts,
	)
	if err := validateServiceAccountEmails(mergeDedup(effSSA, effTSA)); err != nil {
		return err
	}
	if err := validateFirewallTargetsAndSources(effST, effTT, effSSA, effTSA); err != nil {
		return err
	}
	if err := validateFirewallFilterSelections(spec.TargetType, spec.SourceFilterType, true, effTT, effTSA, effST, effSSA); err != nil {
		return err
	}
	return resolveFirewallNodeMetadata(ctx, spec.Firewall)
}

// validateFirewallTargetType validates the "Targets" select value.
func validateFirewallTargetType(targetType string) error {
	switch strings.TrimSpace(targetType) {
	case "", FirewallTargetingNoChange, FirewallTargetAll, FirewallFilterTags, FirewallFilterServiceAccounts:
		return nil
	default:
		return fmt.Errorf("invalid target type %q", targetType)
	}
}

// validateFirewallSourceFilterType validates the "Source filter" select value.
func validateFirewallSourceFilterType(sourceFilterType string) error {
	switch strings.TrimSpace(sourceFilterType) {
	case "", FirewallTargetingNoChange, FirewallSourceRanges, FirewallFilterTags, FirewallFilterServiceAccounts:
		return nil
	default:
		return fmt.Errorf("invalid source filter type %q", sourceFilterType)
	}
}

func validateFirewallEnabledState(state string) error {
	switch strings.TrimSpace(state) {
	case "", FirewallEnabledNoChange, FirewallEnabledEnabled, FirewallEnabledDisabled:
		return nil
	default:
		return fmt.Errorf("invalid enabled state %q", state)
	}
}

func (u *UpdateFirewall) Execute(ctx core.ExecutionContext) error {
	spec := UpdateFirewallSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if err := validateFirewallEnabledState(spec.EnabledState); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := validateFirewallEnabledState(spec.Logging); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := validateFirewallTargetType(spec.TargetType); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := validateFirewallSourceFilterType(spec.SourceFilterType); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	urlProject, name, err := parseFirewallPath(spec.Firewall)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}
	project := client.ProjectID()
	if urlProject != "" && urlProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"firewall rule belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			urlProject, project,
		))
	}

	callCtx := context.Background()

	// Read the current rule to learn its direction (which CIDR field ranges map
	// to) and its action (whether protocols patch `allowed` or `denied`).
	body, err := GetFirewall(callCtx, client, project, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read firewall rule: %v", err))
	}
	var current firewallGetResp
	if err := json.Unmarshal(body, &current); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse firewall rule: %v", err))
	}

	patch := map[string]any{}

	egress := strings.EqualFold(current.Direction, FirewallDirectionEgress)

	mergedTargetServiceAccounts := mergeDedup(trimList(spec.TargetServiceAccounts), trimList(spec.TargetServiceAccountsCustom))
	mergedSourceServiceAccounts := mergeDedup(trimList(spec.SourceServiceAccounts), trimList(spec.SourceServiceAccountsCustom))

	// ---- Targets ---- (applies to both INGRESS and EGRESS). The dropdown picks
	// one kind; the opposite field is cleared so the patch/merge can't leave a
	// stale filter (a rule cannot mix network tags and service accounts).
	var writtenTargetSAs []string
	switch strings.TrimSpace(spec.TargetType) {
	case "", FirewallTargetingNoChange:
		// leave both target fields untouched
	case FirewallTargetAll:
		patch["targetTags"] = []string{}
		patch["targetServiceAccounts"] = []string{}
	case FirewallFilterTags:
		tags := trimList(spec.TargetTags)
		if len(tags) == 0 {
			return ctx.ExecutionState.Fail("error", `select at least one target tag, or choose "All instances in the network"`)
		}
		patch["targetTags"] = tags
		patch["targetServiceAccounts"] = []string{}
	case FirewallFilterServiceAccounts:
		if len(mergedTargetServiceAccounts) == 0 {
			return ctx.ExecutionState.Fail("error", `select at least one target service account, or choose "All instances in the network"`)
		}
		writtenTargetSAs = mergedTargetServiceAccounts
		patch["targetServiceAccounts"] = mergedTargetServiceAccounts
		patch["targetTags"] = []string{}
	default:
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid target type %q", spec.TargetType))
	}

	// ---- Source filter ---- (INGRESS only).
	var writtenSourceSAs []string
	if sel := strings.TrimSpace(spec.SourceFilterType); sel != "" && sel != FirewallTargetingNoChange {
		if egress {
			return ctx.ExecutionState.Fail("error", "source filters apply only to INGRESS firewall rules; this rule is EGRESS")
		}
		switch sel {
		case FirewallSourceRanges:
			// "IP ranges only": drop tag/SA matching. The ranges field is independent.
			patch["sourceTags"] = []string{}
			patch["sourceServiceAccounts"] = []string{}
		case FirewallFilterTags:
			tags := trimList(spec.SourceTags)
			if len(tags) == 0 {
				return ctx.ExecutionState.Fail("error", `select at least one source tag, or choose "IP ranges only"`)
			}
			patch["sourceTags"] = tags
			patch["sourceServiceAccounts"] = []string{}
		case FirewallFilterServiceAccounts:
			if len(mergedSourceServiceAccounts) == 0 {
				return ctx.ExecutionState.Fail("error", `select at least one source service account, or choose "IP ranges only"`)
			}
			writtenSourceSAs = mergedSourceServiceAccounts
			patch["sourceServiceAccounts"] = mergedSourceServiceAccounts
			patch["sourceTags"] = []string{}
		default:
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid source filter type %q", spec.SourceFilterType))
		}
	}

	// SA email format — validate only the lists actually written to the patch.
	if err := validateServiceAccountEmails(mergeDedup(writtenTargetSAs, writtenSourceSAs)); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	// Validate the rule's *resulting* targeting won't mix network tags and service
	// accounts. firewalls.patch is a merge: a targeting field this update doesn't
	// set keeps its current value, so derive each resulting list from the patch
	// when written, else fall back to the current rule.
	resolved := func(key string, cur []string) []string {
		if v, ok := patch[key]; ok {
			return v.([]string) // only this code writes these keys; always []string
		}
		return cur
	}
	if err := validateFirewallTargetsAndSources(
		resolved("sourceTags", current.SourceTags),
		resolved("targetTags", current.TargetTags),
		resolved("sourceServiceAccounts", current.SourceServiceAccounts),
		resolved("targetServiceAccounts", current.TargetServiceAccounts),
	); err != nil {
		return ctx.ExecutionState.Fail("error",
			fmt.Sprintf("%v — set both Targets and Source filter to the same kind (tags or service accounts)", err))
	}

	switch strings.TrimSpace(spec.EnabledState) {
	case FirewallEnabledEnabled:
		patch["disabled"] = false
	case FirewallEnabledDisabled:
		patch["disabled"] = true
	}

	if err := validateFirewallPriority(spec.Priority); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if spec.Priority != nil {
		patch["priority"] = *spec.Priority
	}

	// Protocols & ports: "all" sends the match-everything rule; "specified" builds
	// from the list; "no change" leaves them untouched. A rule keeps its existing
	// allow/deny action — patch whichever array it already uses.
	switch strings.TrimSpace(spec.ProtocolsAndPorts) {
	case FirewallProtocolsAll:
		if len(current.Denied) > 0 {
			patch["denied"] = allProtocolsRule()
		} else {
			patch["allowed"] = allProtocolsRule()
		}
	case FirewallProtocolsSpecified:
		rules, err := buildFirewallRules(spec.Rules)
		if err != nil {
			return ctx.ExecutionState.Fail("error", err.Error())
		}
		if len(current.Denied) > 0 {
			patch["denied"] = rules
		} else {
			patch["allowed"] = rules
		}
	}

	if ranges := trimList(spec.Ranges); len(ranges) > 0 {
		if strings.EqualFold(current.Direction, FirewallDirectionEgress) {
			patch["destinationRanges"] = ranges
		} else {
			patch["sourceRanges"] = ranges
		}
	}

	switch strings.TrimSpace(spec.Logging) {
	case FirewallEnabledEnabled:
		metadata, err := normalizeFirewallLogMetadata(spec.LogMetadata)
		if err != nil {
			return ctx.ExecutionState.Fail("error", err.Error())
		}
		patch["logConfig"] = map[string]any{"enable": true, "metadata": metadata}
	case FirewallEnabledDisabled:
		patch["logConfig"] = map[string]any{"enable": false}
	}

	if spec.Description != nil {
		patch["description"] = strings.TrimSpace(*spec.Description)
	}

	if len(patch) == 0 {
		return ctx.ExecutionState.Fail("error", "nothing to update: change at least one field")
	}
	// Include the resource name in the body; firewalls.patch expects it to match
	// the rule being patched.
	patch["name"] = name

	respBody, err := client.Patch(callCtx, fmt.Sprintf("projects/%s/global/firewalls/%s", project, name), patch)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to update firewall rule: %v", err))
	}
	opName, err := operationNameFromResponse(respBody, "update firewall rule")
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := WaitForGlobalOperation(callCtx, client, project, opName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error waiting for update firewall rule operation: %v", err))
	}

	updated, err := GetFirewall(callCtx, client, project, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read firewall rule after update: %v", err))
	}
	payload, err := FirewallPayloadFromGetResponse(updated, project)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse updated firewall rule: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.firewallRule.updated",
		[]any{payload},
	)
}

func (u *UpdateFirewall) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateFirewall) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateFirewall) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateFirewall) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (u *UpdateFirewall) Hooks() []core.Hook {
	return []core.Hook{}
}

func (u *UpdateFirewall) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
