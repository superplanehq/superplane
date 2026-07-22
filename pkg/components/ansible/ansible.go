package ansible

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	channelSuccess = "success"
	channelFailed  = "failed"

	ModePlaybook = "playbook"
	ModeAdhoc    = "adhoc"

	ModuleDefault = "shell"

	payloadTypePlaybook = "ansible.playbook.executed"
	payloadTypeAdhoc    = "ansible.adhoc.executed"

	defaultInventory = "localhost ansible_connection=local"
	defaultTimeout   = 300
)

// moduleNameRegex allows module names like `shell` and fully-qualified ones
// like `ansible.builtin.copy`.
var moduleNameRegex = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.]*$`)

// extraVarNameRegex matches valid Ansible/YAML variable names.
var extraVarNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func init() {
	registry.RegisterAction("ansible", &Ansible{})
}

// Ansible runs Ansible playbooks and ad-hoc commands. The SuperPlane process
// acts as the Ansible control node and reaches managed hosts via the provided
// inventory.
type Ansible struct {
	// runner is injectable for testing; when nil the real execRunner is used.
	runner ansibleRunner
}

type ExtraVar struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type Spec struct {
	Mode        string     `json:"mode" mapstructure:"mode"`
	Inventory   string     `json:"inventory" mapstructure:"inventory"`
	Playbook    *string    `json:"playbook,omitempty" mapstructure:"playbook"`
	HostPattern *string    `json:"hostPattern,omitempty" mapstructure:"hostPattern"`
	Module      *string    `json:"module,omitempty" mapstructure:"module"`
	ModuleArgs  *string    `json:"moduleArgs,omitempty" mapstructure:"moduleArgs"`
	ExtraVars   []ExtraVar `json:"extraVars,omitempty" mapstructure:"extraVars"`
	Limit       *string    `json:"limit,omitempty" mapstructure:"limit"`
	Become      bool       `json:"become" mapstructure:"become"`
	Verbosity   int        `json:"verbosity" mapstructure:"verbosity"`
	Timeout     int        `json:"timeout" mapstructure:"timeout"`
}

func (a *Ansible) Name() string  { return "ansible" }
func (a *Ansible) Label() string { return "Ansible" }

func (a *Ansible) Description() string {
	return "Run an Ansible playbook or ad-hoc command against an inventory."
}

func (a *Ansible) Documentation() string {
	return `Run Ansible from a SuperPlane workflow. The SuperPlane node acts as the Ansible control node and reaches managed hosts through the inventory you provide.

## Modes

- **Playbook**: Provide playbook YAML inline. It is run with ` + "`ansible-playbook`" + ` and the JSON output callback so the play recap (ok/changed/unreachable/failed per host) is captured.
- **Ad-hoc**: Run a single module against a host pattern, e.g. module ` + "`ping`" + `, or module ` + "`shell`" + ` with arguments ` + "`uptime`" + `.

## Configuration

- **Inventory**: Inline inventory (INI or YAML). Defaults to ` + "`localhost ansible_connection=local`" + ` so it works without any remote hosts.
- **Playbook** (playbook mode): The playbook YAML to run.
- **Host pattern / Module / Module arguments** (ad-hoc mode): The target pattern, module name, and module arguments.
- **Extra variables**: Optional ` + "`name=value`" + ` pairs passed via ` + "`-e`" + `.
- **Limit**: Optional ` + "`--limit`" + ` host pattern.
- **Become**: Run with privilege escalation (` + "`--become`" + `).
- **Verbosity**: 0-4, mapped to ` + "`-v`..`-vvvv`" + `.
- **Timeout (seconds)**: Maximum run time before the execution errors out.

## Output

- **success**: Ansible exited with status 0.
- **failed**: Ansible ran but exited non-zero (e.g. a task failed or a host was unreachable).

If Ansible cannot be run at all (binary missing, timeout, invalid working directory), the run finishes in the **error** state.
`
}

func (a *Ansible) Icon() string  { return "server" }
func (a *Ansible) Color() string { return "red" }

func (a *Ansible) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: channelSuccess, Label: "Success"},
		{Name: channelFailed, Label: "Failed"},
	}
}

func (a *Ansible) Configuration() []configuration.Field {
	playbookOnly := []configuration.VisibilityCondition{{Field: "mode", Values: []string{ModePlaybook}}}
	adhocOnly := []configuration.VisibilityCondition{{Field: "mode", Values: []string{ModeAdhoc}}}

	return []configuration.Field{
		{
			Name:        "mode",
			Label:       "Mode",
			Type:        configuration.FieldTypeSelect,
			Description: "Run a playbook or a single ad-hoc module",
			Required:    true,
			Default:     ModePlaybook,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Playbook", Value: ModePlaybook},
						{Label: "Ad-hoc command", Value: ModeAdhoc},
					},
				},
			},
		},
		{
			Name:        "inventory",
			Label:       "Inventory",
			Type:        configuration.FieldTypeText,
			Description: "Inline Ansible inventory (INI or YAML)",
			Required:    true,
			Default:     defaultInventory,
		},
		{
			Name:                 "playbook",
			Label:                "Playbook",
			Type:                 configuration.FieldTypeText,
			Description:          "Playbook YAML to run",
			Placeholder:          "- hosts: all\n  tasks:\n    - ping:",
			VisibilityConditions: playbookOnly,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "mode", Values: []string{ModePlaybook}}},
		},
		{
			Name:                 "hostPattern",
			Label:                "Host pattern",
			Type:                 configuration.FieldTypeString,
			Description:          "Hosts to target, e.g. all, webservers, or a specific host",
			Placeholder:          "all",
			VisibilityConditions: adhocOnly,
			RequiredConditions:   []configuration.RequiredCondition{{Field: "mode", Values: []string{ModeAdhoc}}},
		},
		{
			Name:                 "module",
			Label:                "Module",
			Type:                 configuration.FieldTypeString,
			Description:          "Ansible module to run, e.g. ping, shell, copy",
			Default:              ModuleDefault,
			VisibilityConditions: adhocOnly,
		},
		{
			Name:                 "moduleArgs",
			Label:                "Module arguments",
			Type:                 configuration.FieldTypeString,
			Description:          "Arguments passed to the module via -a",
			Placeholder:          "uptime",
			VisibilityConditions: adhocOnly,
		},
		{
			Name:        "extraVars",
			Label:       "Extra variables",
			Type:        configuration.FieldTypeList,
			Description: "Variables passed to Ansible via -e",
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Description: "Variable name (letters, numbers, underscore)",
								Required:    true,
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Description: "Variable value",
								Required:    true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeString,
			Description: "Restrict the run to a subset of hosts (--limit)",
			Togglable:   true,
		},
		{
			Name:        "become",
			Label:       "Become (privilege escalation)",
			Type:        configuration.FieldTypeBool,
			Description: "Run with --become (e.g. sudo)",
			Default:     false,
		},
		{
			Name:        "verbosity",
			Label:       "Verbosity",
			Type:        configuration.FieldTypeNumber,
			Description: "Ansible verbosity level (0-4)",
			Togglable:   true,
			Default:     0,
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{Min: intPtr(0), Max: intPtr(4)},
			},
		},
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Description: "Maximum run time before the execution errors out",
			Required:    true,
			Default:     defaultTimeout,
		},
	}
}

func (a *Ansible) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validate(spec)
}

func (a *Ansible) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validate(spec); err != nil {
		return err
	}

	runner := a.runner
	if runner == nil {
		runner = execRunner{}
	}

	runCtx, cancel := context.WithTimeout(context.Background(), time.Duration(spec.Timeout)*time.Second)
	defer cancel()

	result, err := runner.Run(runCtx, spec, ctx.Logger)
	if err != nil {
		// Could not run Ansible at all -> error state.
		return err
	}

	if err := ctx.Metadata.Set(result); err != nil {
		return err
	}

	payloadType := payloadTypePlaybook
	if spec.Mode == ModeAdhoc {
		payloadType = payloadTypeAdhoc
	}

	channel := channelSuccess
	if result.ExitCode != 0 {
		channel = channelFailed
	}

	return ctx.ExecutionState.Emit(channel, payloadType, []any{result})
}

func (a *Ansible) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *Ansible) Hooks() []core.Hook { return []core.Hook{} }

func (a *Ansible) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (a *Ansible) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 404, nil, fmt.Errorf("ansible component does not handle webhooks")
}

func (a *Ansible) Cancel(ctx core.ExecutionContext) error { return nil }

func (a *Ansible) Cleanup(ctx core.SetupContext) error { return nil }

// decodeSpec decodes the raw configuration into a Spec, applying defaults for
// optional fields.
func decodeSpec(raw any) (Spec, error) {
	spec := Spec{}
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return spec, fmt.Errorf("invalid configuration: %w", err)
	}
	if spec.Mode == "" {
		spec.Mode = ModePlaybook
	}
	if spec.Timeout == 0 {
		spec.Timeout = defaultTimeout
	}
	return spec, nil
}

// validate enforces required fields and guards inputs that become argv values
// to prevent them from being interpreted as flags or injecting commands.
func validate(spec Spec) error {
	if spec.Mode != ModePlaybook && spec.Mode != ModeAdhoc {
		return fmt.Errorf("invalid mode: %s", spec.Mode)
	}
	if strings.TrimSpace(spec.Inventory) == "" {
		return fmt.Errorf("inventory is required")
	}
	if spec.Timeout < 1 {
		return fmt.Errorf("timeout must be at least 1 second")
	}
	if spec.Verbosity < 0 || spec.Verbosity > 4 {
		return fmt.Errorf("verbosity must be between 0 and 4")
	}

	switch spec.Mode {
	case ModePlaybook:
		if spec.Playbook == nil || strings.TrimSpace(*spec.Playbook) == "" {
			return fmt.Errorf("playbook is required in playbook mode")
		}
	case ModeAdhoc:
		if spec.HostPattern == nil || strings.TrimSpace(*spec.HostPattern) == "" {
			return fmt.Errorf("host pattern is required in ad-hoc mode")
		}
		if err := validateArgValue("host pattern", *spec.HostPattern); err != nil {
			return err
		}
		if spec.Module != nil && *spec.Module != "" && !moduleNameRegex.MatchString(*spec.Module) {
			return fmt.Errorf("invalid module name: %s", *spec.Module)
		}
	}

	if spec.Limit != nil && *spec.Limit != "" {
		if err := validateArgValue("limit", *spec.Limit); err != nil {
			return err
		}
	}

	for _, v := range spec.ExtraVars {
		if v.Name == "" {
			return fmt.Errorf("extra variable name is required")
		}
		if !extraVarNameRegex.MatchString(v.Name) {
			return fmt.Errorf("invalid extra variable name: %s", v.Name)
		}
	}

	return nil
}

// validateArgValue rejects values that would be misread as a command-line flag
// (positional argv values must not start with "-") or that contain newlines.
func validateArgValue(label, value string) error {
	if strings.HasPrefix(value, "-") {
		return fmt.Errorf("%s must not start with '-'", label)
	}
	if strings.ContainsAny(value, "\n\r") {
		return fmt.Errorf("%s must not contain newlines", label)
	}
	return nil
}

func intPtr(v int) *int { return &v }
