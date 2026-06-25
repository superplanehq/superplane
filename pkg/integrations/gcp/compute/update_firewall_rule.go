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
)

type UpdateFirewall struct{}

type UpdateFirewallSpec struct {
	Firewall                    string             `mapstructure:"firewall"`
	EnabledState                string             `mapstructure:"enabledState"`
	Priority                    *int               `mapstructure:"priority"`
	Rules                       []FirewallRuleSpec `mapstructure:"rules"`
	Ranges                      []string           `mapstructure:"ranges"`
	TargetTags                  *[]string          `mapstructure:"targetTags"`
	SourceTags                  *[]string          `mapstructure:"sourceTags"`
	TargetServiceAccounts       *[]string          `mapstructure:"targetServiceAccounts"`
	SourceServiceAccounts       *[]string          `mapstructure:"sourceServiceAccounts"`
	TargetServiceAccountsCustom *[]string          `mapstructure:"targetServiceAccountsCustom"`
	SourceServiceAccountsCustom *[]string          `mapstructure:"sourceServiceAccountsCustom"`
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
	return "Update a VPC firewall rule: its protocols and ports, ranges, priority, target tags, description, or enabled state"
}

func (u *UpdateFirewall) Documentation() string {
	return `The Update Firewall Rule component changes an existing VPC firewall rule. Toggle on only the fields you want to change; everything else is left untouched.

## Use Cases

- **Adjust access**: Change the allowed/denied protocols and ports
- **Widen or narrow scope**: Update the source/destination CIDR ranges or target tags
- **Re-prioritize**: Change the rule's priority
- **Pause a rule**: Disable a rule without deleting it, then re-enable it later

## Configuration

- **Firewall rule**: The firewall rule to update (required)
- **Enabled state**: Leave unchanged, enable, or disable the rule
- **Priority**: New priority (0-65535); lower numbers take precedence
- **Protocols & ports**: Replace the rule's protocols/ports. The rule keeps its existing action (allow or deny)
- **Ranges**: Replace the rule's CIDR ranges. Applied as source ranges for INGRESS rules and destination ranges for EGRESS rules (the rule's direction is fixed)
- **Target tags / Target service accounts**: Replace the VMs the rule applies to
- **Source tags / Source service accounts**: Replace the rule's source filters (INGRESS rules only)
- **Logs**: Leave Firewall Rules Logging unchanged, or turn it on/off (with optional metadata)
- **Description**: Replace the rule's description

> A firewall rule filters by **network tags** or by **service accounts**, never both — the component rejects an update that would mix them.

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
			Default:     FirewallEnabledNoChange,
			Description: "Leave the rule's enabled state unchanged, or enable/disable it.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "No change", Value: FirewallEnabledNoChange},
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
			Name:        "rules",
			Label:       "Protocols & ports",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Replace the rule's protocols/ports. Leave ports empty to match all ports; use protocol \"all\" to match every protocol. The rule keeps its existing allow/deny action.",
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
			Name:        "targetTags",
			Label:       "Target tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Replace the rule's target tags (the VMs it applies to).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Tag",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name:        "targetServiceAccounts",
			Label:       "Target service accounts",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Replace the rule's target service accounts. Cannot be combined with network tags.",
			Placeholder: "Select service accounts",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeServiceAccount, Multi: true},
			},
		},
		{
			Name:        "targetServiceAccountsCustom",
			Label:       "Target service accounts (custom / cross-project)",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Additional target service-account emails not shown in the dropdown (e.g. from another project). Merged with the selections above.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Service account email",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name:        "sourceTags",
			Label:       "Source tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Replace the rule's source tags (INGRESS rules only).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Tag",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name:        "sourceServiceAccounts",
			Label:       "Source service accounts",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Replace the rule's source service accounts (INGRESS rules only). Cannot be combined with network tags.",
			Placeholder: "Select service accounts",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeServiceAccount, Multi: true},
			},
		},
		{
			Name:        "sourceServiceAccountsCustom",
			Label:       "Source service accounts (custom / cross-project)",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Additional source service-account emails not shown in the dropdown (e.g. from another project). Merged with the selections above. INGRESS rules only.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Service account email",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
		},
		{
			Name:        "logging",
			Label:       "Logs",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     FirewallEnabledNoChange,
			Description: "Leave Firewall Rules Logging unchanged, or turn it on/off.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "No change", Value: FirewallEnabledNoChange},
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
	if len(spec.Rules) > 0 {
		if _, err := buildFirewallRules(spec.Rules); err != nil {
			return err
		}
	}
	sourceServiceAccounts := mergeDedup(providedList(spec.SourceServiceAccounts), providedList(spec.SourceServiceAccountsCustom))
	targetServiceAccounts := mergeDedup(providedList(spec.TargetServiceAccounts), providedList(spec.TargetServiceAccountsCustom))
	if err := validateServiceAccountEmails(mergeDedup(sourceServiceAccounts, targetServiceAccounts)); err != nil {
		return err
	}
	if err := validateFirewallTargetsAndSources(
		providedList(spec.SourceTags), providedList(spec.TargetTags),
		sourceServiceAccounts, targetServiceAccounts,
	); err != nil {
		return err
	}
	return resolveFirewallNodeMetadata(ctx, spec.Firewall)
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
	mergedSourceServiceAccounts := mergeDedup(providedList(spec.SourceServiceAccounts), providedList(spec.SourceServiceAccountsCustom))
	mergedTargetServiceAccounts := mergeDedup(providedList(spec.TargetServiceAccounts), providedList(spec.TargetServiceAccountsCustom))
	if err := validateServiceAccountEmails(mergeDedup(mergedSourceServiceAccounts, mergedTargetServiceAccounts)); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := validateFirewallTargetsAndSources(
		providedList(spec.SourceTags), providedList(spec.TargetTags),
		mergedSourceServiceAccounts, mergedTargetServiceAccounts,
	); err != nil {
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

	if len(spec.Rules) > 0 {
		rules, err := buildFirewallRules(spec.Rules)
		if err != nil {
			return ctx.ExecutionState.Fail("error", err.Error())
		}
		// A rule keeps its action; patch whichever array it already uses. Default
		// to `allowed` for a rule with neither populated (shouldn't happen).
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

	// targetTags is sent whenever the field is toggled on (non-nil), even when
	// empty: an empty list clears the rule's tags so it applies to every VM in
	// the network. A nil pointer means the field is toggled off — leave it alone.
	if spec.TargetTags != nil {
		patch["targetTags"] = trimList(*spec.TargetTags)
	}
	// Target service accounts are patched when either the dropdown or the custom
	// field is toggled on; the value is the merge of both.
	if spec.TargetServiceAccounts != nil || spec.TargetServiceAccountsCustom != nil {
		patch["targetServiceAccounts"] = mergedTargetServiceAccounts
	}
	// Source tags / service accounts only exist on INGRESS rules; reject them on
	// an EGRESS rule rather than letting the API return a less obvious error.
	if spec.SourceTags != nil {
		if strings.EqualFold(current.Direction, FirewallDirectionEgress) {
			return ctx.ExecutionState.Fail("error", "source tags apply only to INGRESS firewall rules")
		}
		patch["sourceTags"] = trimList(*spec.SourceTags)
	}
	if spec.SourceServiceAccounts != nil || spec.SourceServiceAccountsCustom != nil {
		if strings.EqualFold(current.Direction, FirewallDirectionEgress) {
			return ctx.ExecutionState.Fail("error", "source service accounts apply only to INGRESS firewall rules")
		}
		patch["sourceServiceAccounts"] = mergedSourceServiceAccounts
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
