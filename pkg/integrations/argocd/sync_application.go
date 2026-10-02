package argocd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const SyncApplicationPayloadType = "argocd.application.sync"

type SyncApplication struct{}

type SyncApplicationConfiguration struct {
	Project              string   `json:"project" mapstructure:"project"`
	Application          string   `json:"application" mapstructure:"application"`
	ApplicationNamespace string   `json:"applicationNamespace" mapstructure:"applicationNamespace"`
	Revision             string   `json:"revision" mapstructure:"revision"`
	Revisions            []string `json:"revisions" mapstructure:"revisions"`
	Prune                bool     `json:"prune" mapstructure:"prune"`
	DryRun               bool     `json:"dryRun" mapstructure:"dryRun"`
	Force                bool     `json:"force" mapstructure:"force"`
	Strategy             string   `json:"strategy" mapstructure:"strategy"`
}

type SyncApplicationRequest struct {
	AppNamespace string       `json:"appNamespace,omitempty"`
	DryRun       bool         `json:"dryRun"`
	Name         string       `json:"name"`
	Project      string       `json:"project"`
	Prune        bool         `json:"prune"`
	Revision     string       `json:"revision,omitempty"`
	Revisions    []string     `json:"revisions,omitempty"`
	Strategy     SyncStrategy `json:"strategy"`
}

type SyncStrategy struct {
	Apply *SyncStrategyApply `json:"apply,omitempty"`
	Hook  *SyncStrategyHook  `json:"hook,omitempty"`
}

type SyncStrategyApply struct {
	Force bool `json:"force"`
}

type SyncStrategyHook struct {
	SyncStrategyApply *SyncStrategyApply `json:"syncStrategyApply"`
}

func (s *SyncApplication) Name() string  { return "argocd.syncApplication" }
func (s *SyncApplication) Label() string { return "Sync Application" }
func (s *SyncApplication) Description() string {
	return "Start a synchronization for an Argo CD application"
}
func (s *SyncApplication) Documentation() string {
	return "The Sync Application component starts an Argo CD application synchronization and emits the returned application state."
}
func (s *SyncApplication) Icon() string  { return "kubernetes" }
func (s *SyncApplication) Color() string { return "gray" }
func (s *SyncApplication) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (s *SyncApplication) ExampleOutput() map[string]any {
	return (&GetApplication{}).ExampleOutput()
}
func (s *SyncApplication) Hooks() []core.Hook                      { return []core.Hook{} }
func (s *SyncApplication) HandleHook(core.ActionHookContext) error { return nil }
func (s *SyncApplication) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (s *SyncApplication) Cancel(core.ExecutionContext) error { return nil }
func (s *SyncApplication) Cleanup(core.SetupContext) error    { return nil }

func (s *SyncApplication) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "project", Label: "Project", Type: configuration.FieldTypeString, Required: true, Description: "Argo CD project that contains the application"},
		{Name: "application", Label: "Application", Type: configuration.FieldTypeString, Required: true, Description: "Argo CD application name"},
		{Name: "applicationNamespace", Label: "Application Namespace", Type: configuration.FieldTypeString, Togglable: true, Description: "Application namespace when Argo CD allows applications in any namespace"},
		{Name: "revision", Label: "Revision", Type: configuration.FieldTypeString, Togglable: true, Description: "Revision to synchronize for a single-source application"},
		{Name: "revisions", Label: "Source Revisions", Type: configuration.FieldTypeList, Togglable: true, Description: "Revisions to synchronize for each application source", TypeOptions: &configuration.TypeOptions{List: &configuration.ListTypeOptions{ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString}}}},
		{Name: "prune", Label: "Prune", Type: configuration.FieldTypeBool, Default: false, Description: "Remove resources that are no longer defined"},
		{Name: "dryRun", Label: "Dry Run", Type: configuration.FieldTypeBool, Default: false, Description: "Preview the synchronization without changing resources"},
		{Name: "force", Label: "Force Apply", Type: configuration.FieldTypeBool, Default: false, Description: "Force apply resources during synchronization"},
		{Name: "strategy", Label: "Strategy", Type: configuration.FieldTypeSelect, Default: "apply", Description: "Synchronization strategy", TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{{Label: "Apply", Value: "apply"}, {Label: "Hook", Value: "hook"}}}}},
	}
}

func (s *SyncApplication) Setup(ctx core.SetupContext) error {
	_, err := decodeSyncApplicationConfiguration(ctx.Configuration)
	return err
}

func (s *SyncApplication) Execute(ctx core.ExecutionContext) error {
	cfg, err := decodeSyncApplicationConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	request := SyncApplicationRequest{
		AppNamespace: cfg.ApplicationNamespace, DryRun: cfg.DryRun, Name: cfg.Application,
		Project: cfg.Project, Prune: cfg.Prune, Revision: cfg.Revision, Revisions: cfg.Revisions,
		Strategy: syncStrategy(cfg.Strategy, cfg.Force),
	}
	application, err := client.SyncApplication(request)
	if err != nil {
		return err
	}
	if strings.TrimSpace(application.Metadata.Name) == "" {
		return fmt.Errorf("Argo CD response missing application metadata.name")
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, SyncApplicationPayloadType, []any{applicationOutputFrom(application, cfg.Project)})
}

func decodeSyncApplicationConfiguration(value any) (SyncApplicationConfiguration, error) {
	cfg := SyncApplicationConfiguration{Strategy: "apply"}
	if err := mapstructure.Decode(value, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to decode configuration: %w", err)
	}
	cfg.Project, cfg.Application, cfg.ApplicationNamespace, cfg.Revision, cfg.Strategy = strings.TrimSpace(cfg.Project), strings.TrimSpace(cfg.Application), strings.TrimSpace(cfg.ApplicationNamespace), strings.TrimSpace(cfg.Revision), strings.TrimSpace(cfg.Strategy)
	if cfg.Project == "" {
		return cfg, fmt.Errorf("project is required")
	}
	if cfg.Application == "" {
		return cfg, fmt.Errorf("application is required")
	}
	if cfg.Revision != "" && len(cfg.Revisions) > 0 {
		return cfg, fmt.Errorf("revision and revisions cannot both be set")
	}
	if cfg.Strategy == "" {
		cfg.Strategy = "apply"
	}
	if cfg.Strategy != "apply" && cfg.Strategy != "hook" {
		return cfg, fmt.Errorf("strategy must be apply or hook")
	}
	for i := range cfg.Revisions {
		cfg.Revisions[i] = strings.TrimSpace(cfg.Revisions[i])
		if cfg.Revisions[i] == "" {
			return cfg, fmt.Errorf("revisions cannot contain an empty value")
		}
	}
	return cfg, nil
}

func syncStrategy(strategy string, force bool) SyncStrategy {
	apply := &SyncStrategyApply{Force: force}
	if strategy == "hook" {
		return SyncStrategy{Hook: &SyncStrategyHook{SyncStrategyApply: apply}}
	}
	return SyncStrategy{Apply: apply}
}
