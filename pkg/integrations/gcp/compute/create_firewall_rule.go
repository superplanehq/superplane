package compute

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateFirewall struct{}

type CreateFirewallSpec struct {
	Name                  string             `mapstructure:"name"`
	Network               string             `mapstructure:"network"`
	Direction             string             `mapstructure:"direction"`
	Action                string             `mapstructure:"action"`
	Priority              *int               `mapstructure:"priority"`
	Rules                 []FirewallRuleSpec `mapstructure:"rules"`
	SourceRanges          []string           `mapstructure:"sourceRanges"`
	DestinationRanges     []string           `mapstructure:"destinationRanges"`
	SourceTags            []string           `mapstructure:"sourceTags"`
	TargetTags            []string           `mapstructure:"targetTags"`
	SourceServiceAccounts []string           `mapstructure:"sourceServiceAccounts"`
	TargetServiceAccounts []string           `mapstructure:"targetServiceAccounts"`
	// *Custom hold free-text service-account emails (e.g. cross-project ones not
	// listed by the dropdown). They are merged with the dropdown selections.
	SourceServiceAccountsCustom []string `mapstructure:"sourceServiceAccountsCustom"`
	TargetServiceAccountsCustom []string `mapstructure:"targetServiceAccountsCustom"`
	Description                 string   `mapstructure:"description"`
	Disabled              bool               `mapstructure:"disabled"`
	EnableLogging         bool               `mapstructure:"enableLogging"`
	LogMetadata           string             `mapstructure:"logMetadata"`
}

func (c *CreateFirewall) Name() string {
	return "gcp.compute.createFirewallRule"
}

func (c *CreateFirewall) Label() string {
	return "Compute • Create Firewall Rule"
}

func (c *CreateFirewall) Description() string {
	return "Create a VPC firewall rule that allows or denies traffic to or from your VM instances"
}

func (c *CreateFirewall) Documentation() string {
	return `The Create Firewall Rule component creates a VPC firewall rule that controls traffic to and from the VM instances in a network.

## Use Cases

- **Open a port**: Allow inbound traffic on specific ports (e.g. 80, 443) to your VMs
- **Lock down access**: Deny traffic from specific source ranges
- **Targeted rules**: Apply a rule only to VMs carrying specific network tags

## Configuration

- **Name**: Name for the new firewall rule (lowercase letters, numbers and hyphens; 1–63 chars) (required)
- **Network**: The VPC network the rule applies to (required)
- **Direction**: ` + "`INGRESS`" + ` (incoming, default) or ` + "`EGRESS`" + ` (outgoing)
- **Action**: ` + "`allow`" + ` (default) or ` + "`deny`" + ` the matched traffic
- **Priority**: 0–65535; lower numbers win (default 1000)
- **Protocols & ports**: One or more protocol/ports entries (e.g. ` + "`tcp`" + ` with ports ` + "`80, 443`" + `). Leave ports empty to match all ports for that protocol; use ` + "`all`" + ` to match every protocol
- **Source ranges** (INGRESS): CIDR ranges the rule applies to (default ` + "`0.0.0.0/0`" + `)
- **Destination ranges** (EGRESS): CIDR ranges the rule applies to (default ` + "`0.0.0.0/0`" + `)
- **Target tags**: Optionally limit the rule to VMs with these network tags; leave empty to apply to all VMs in the network
- **Target service accounts**: Optionally limit the rule to VMs running as these service accounts (alternative to target tags). Picked from a dropdown of the project's service accounts; a separate custom field accepts cross-project emails not in the list
- **Source tags / Source service accounts** (INGRESS): Optionally match traffic from VMs with these network tags or service accounts
- **Description**: Optional human-readable description
- **Disabled**: Create the rule in a disabled state
- **Logs**: Turn on Firewall Rules Logging (optionally choosing whether to include metadata)

> A firewall rule filters by **network tags** or by **service accounts**, never both — the component rejects a rule that mixes them.

## Output

Emits the created firewall rule: name, selfLink, network, direction, priority, action, the allowed/denied protocols, source/destination ranges, target/source tags, target/source service accounts, disabled, logging state, creationTimestamp, and a console link.

## Important Notes

- Firewall rules are **global** resources; the network and rule live at the project level.
- Requires the ` + "`roles/compute.securityAdmin`" + ` IAM role (or ` + "`roles/compute.admin`" + `).
- The **service-account dropdowns** additionally require ` + "`iam.serviceAccounts.list`" + ` (e.g. ` + "`roles/iam.serviceAccountViewer`" + `) on the project; without it the dropdown can't list accounts — use the custom field to enter emails directly.
- GCP does **not** verify that a service account exists when creating the rule, so a wrong or non-existent email produces a rule that silently matches nothing. The dropdown avoids this; the custom field is format-checked.
- The component waits for the underlying global operation to complete before emitting.`
}

func (c *CreateFirewall) Icon() string {
	return "shield"
}

func (c *CreateFirewall) Color() string {
	return "blue"
}

func (c *CreateFirewall) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateFirewall) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name for the new firewall rule. Start with a letter; use only a-z, 0-9, and hyphens; 1 to 63 characters.",
			Placeholder: "e.g. allow-http",
		},
		{
			Name:        "network",
			Label:       "Network",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The VPC network the firewall rule applies to.",
			Placeholder: "Select network",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeNetwork},
			},
		},
		{
			Name:        "direction",
			Label:       "Direction",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     FirewallDirectionIngress,
			Description: "Whether the rule applies to incoming (INGRESS) or outgoing (EGRESS) traffic.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Ingress (incoming)", Value: FirewallDirectionIngress},
				{Label: "Egress (outgoing)", Value: FirewallDirectionEgress},
			}}},
		},
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     FirewallActionAllow,
			Description: "Whether to allow or deny matched traffic.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Allow", Value: FirewallActionAllow},
				{Label: "Deny", Value: FirewallActionDeny},
			}}},
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     1000,
			Description: "Rule priority (0-65535). Lower numbers take precedence.",
		},
		{
			Name:        "rules",
			Label:       "Protocols & ports",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Protocols (and optional ports) the rule matches. Leave ports empty to match all ports; use protocol \"all\" to match every protocol.",
			Default:     []map[string]any{{"protocol": "tcp", "ports": ""}},
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
			Name:        "sourceRanges",
			Label:       "Source ranges",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []string{"0.0.0.0/0"},
			Description: "Source CIDR ranges the rule applies to.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Range",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "direction", Values: []string{FirewallDirectionIngress}},
			},
		},
		{
			Name:        "destinationRanges",
			Label:       "Destination ranges",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Default:     []string{"0.0.0.0/0"},
			Description: "Destination CIDR ranges the rule applies to.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Range",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "direction", Values: []string{FirewallDirectionEgress}},
			},
		},
		{
			Name:        "targetTags",
			Label:       "Target tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Limit the rule to VMs with these network tags. Leave empty to apply to all VMs in the network.",
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
			Description: "Limit the rule to VMs running as these service accounts. Cannot be combined with network tags.",
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
			Description: "Apply the rule to traffic from VMs with these network tags (INGRESS only).",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Tag",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "direction", Values: []string{FirewallDirectionIngress}},
			},
		},
		{
			Name:        "sourceServiceAccounts",
			Label:       "Source service accounts",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Apply the rule to traffic from VMs running as these service accounts (INGRESS only). Cannot be combined with network tags.",
			Placeholder: "Select service accounts",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: ResourceTypeServiceAccount, Multi: true},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "direction", Values: []string{FirewallDirectionIngress}},
			},
		},
		{
			Name:        "sourceServiceAccountsCustom",
			Label:       "Source service accounts (custom / cross-project)",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Additional source service-account emails not shown in the dropdown (e.g. from another project). Merged with the selections above. INGRESS only.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:      "Service account email",
					ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "direction", Values: []string{FirewallDirectionIngress}},
			},
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Optional firewall rule description",
		},
		{
			Name:        "disabled",
			Label:       "Disabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Create the rule in a disabled state.",
		},
		{
			Name:        "enableLogging",
			Label:       "Logs",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Turn on Firewall Rules Logging for this rule. Logging can generate a large volume of logs and increase Cloud Logging costs.",
		},
		{
			Name:        "logMetadata",
			Label:       "Log metadata",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     FirewallLogMetadataIncludeAll,
			Description: "Whether firewall logs include metadata. Only applies when logging is enabled.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Include all metadata", Value: FirewallLogMetadataIncludeAll},
				{Label: "Exclude all metadata", Value: FirewallLogMetadataExcludeAll},
			}}},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "enableLogging", Values: []string{"true"}},
			},
		},
	}
}

func (c *CreateFirewall) Setup(ctx core.SetupContext) error {
	spec := CreateFirewallSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	if strings.TrimSpace(spec.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(spec.Network) == "" {
		return errors.New("network is required")
	}
	if _, err := normalizeFirewallDirection(spec.Direction); err != nil {
		return err
	}
	if _, err := normalizeFirewallAction(spec.Action); err != nil {
		return err
	}
	if err := validateFirewallPriority(spec.Priority); err != nil {
		return err
	}
	if _, err := buildFirewallRules(spec.Rules); err != nil {
		return err
	}
	if _, err := normalizeFirewallLogMetadata(spec.LogMetadata); err != nil {
		return err
	}
	sourceServiceAccounts := mergeDedup(trimList(spec.SourceServiceAccounts), trimList(spec.SourceServiceAccountsCustom))
	targetServiceAccounts := mergeDedup(trimList(spec.TargetServiceAccounts), trimList(spec.TargetServiceAccountsCustom))
	if err := validateServiceAccountEmails(mergeDedup(sourceServiceAccounts, targetServiceAccounts)); err != nil {
		return err
	}
	if err := validateFirewallTargetsAndSources(
		trimList(spec.SourceTags), trimList(spec.TargetTags),
		sourceServiceAccounts, targetServiceAccounts,
	); err != nil {
		return err
	}
	return ctx.Metadata.Set(FirewallNodeMetadata{FirewallName: strings.TrimSpace(spec.Name)})
}

func (c *CreateFirewall) Execute(ctx core.ExecutionContext) error {
	spec := CreateFirewallSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return ctx.ExecutionState.Fail("error", "name is required")
	}
	if strings.TrimSpace(spec.Network) == "" {
		return ctx.ExecutionState.Fail("error", "network is required")
	}
	direction, err := normalizeFirewallDirection(spec.Direction)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	action, err := normalizeFirewallAction(spec.Action)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	rules, err := buildFirewallRules(spec.Rules)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := validateFirewallPriority(spec.Priority); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	sourceServiceAccounts := mergeDedup(trimList(spec.SourceServiceAccounts), trimList(spec.SourceServiceAccountsCustom))
	targetServiceAccounts := mergeDedup(trimList(spec.TargetServiceAccounts), trimList(spec.TargetServiceAccountsCustom))
	if err := validateServiceAccountEmails(mergeDedup(sourceServiceAccounts, targetServiceAccounts)); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := validateFirewallTargetsAndSources(
		trimList(spec.SourceTags), trimList(spec.TargetTags),
		sourceServiceAccounts, targetServiceAccounts,
	); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}
	project := client.ProjectID()
	callCtx := context.Background()

	body := map[string]any{
		"name":      name,
		"network":   resolveNetworkURL(project, strings.TrimSpace(spec.Network)),
		"direction": direction,
		"disabled":  spec.Disabled,
	}
	if spec.Priority != nil {
		body["priority"] = *spec.Priority
	}
	if action == FirewallActionAllow {
		body["allowed"] = rules
	} else {
		body["denied"] = rules
	}
	if direction == FirewallDirectionEgress {
		if ranges := trimList(spec.DestinationRanges); len(ranges) > 0 {
			body["destinationRanges"] = ranges
		}
	} else {
		if ranges := trimList(spec.SourceRanges); len(ranges) > 0 {
			body["sourceRanges"] = ranges
		}
		if tags := trimList(spec.SourceTags); len(tags) > 0 {
			body["sourceTags"] = tags
		}
		if len(sourceServiceAccounts) > 0 {
			body["sourceServiceAccounts"] = sourceServiceAccounts
		}
	}
	if tags := trimList(spec.TargetTags); len(tags) > 0 {
		body["targetTags"] = tags
	}
	if len(targetServiceAccounts) > 0 {
		body["targetServiceAccounts"] = targetServiceAccounts
	}
	if desc := strings.TrimSpace(spec.Description); desc != "" {
		body["description"] = desc
	}
	if spec.EnableLogging {
		metadata, err := normalizeFirewallLogMetadata(spec.LogMetadata)
		if err != nil {
			return ctx.ExecutionState.Fail("error", err.Error())
		}
		body["logConfig"] = map[string]any{"enable": true, "metadata": metadata}
	}

	respBody, err := client.Post(callCtx, fmt.Sprintf("projects/%s/global/firewalls", project), body)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create firewall rule: %v", err))
	}
	opName, err := operationNameFromResponse(respBody, "create firewall rule")
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if err := WaitForGlobalOperation(callCtx, client, project, opName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("error waiting for create firewall rule operation: %v", err))
	}

	fwBody, err := GetFirewall(callCtx, client, project, name)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read created firewall rule: %v", err))
	}
	payload, err := FirewallPayloadFromGetResponse(fwBody, project)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse created firewall rule: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.firewallRule.created",
		[]any{payload},
	)
}

// normalizeFirewallAction validates the configured action, defaulting to allow.
func normalizeFirewallAction(action string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "", FirewallActionAllow:
		return FirewallActionAllow, nil
	case FirewallActionDeny:
		return FirewallActionDeny, nil
	default:
		return "", fmt.Errorf("invalid action %q: must be allow or deny", action)
	}
}

func (c *CreateFirewall) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateFirewall) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateFirewall) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateFirewall) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateFirewall) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateFirewall) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
