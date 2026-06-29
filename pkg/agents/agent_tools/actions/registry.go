package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	componentregistry "github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
)

// Dependencies are backend services shared by canvas actions.
type Dependencies struct {
	Encryptor      crypto.Encryptor
	Registry       *componentregistry.Registry
	GitProvider    gitprovider.Provider
	WebhookBaseURL string
	AuthService    authorization.Authorization
	UsageService   usage.Service
}

// Action executes one superplane_app action value.
type Action interface {
	Name() string
	Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error)
}

// Registry dispatches superplane_app action calls by action name.
type Registry struct {
	actions map[string]Action
	names   []string
}

// NewDefaultRegistry creates the standard superplane_app action registry.
func NewDefaultRegistry(deps Dependencies) *Registry {
	return NewRegistry(
		newAccessAction(deps),
		newReadAction(deps),
		newReadRuntimeAction(deps),
		newListFilesAction(deps),
		newReadFileAction(deps),
		createDraftAction{},
		writeFileAction{},
		deleteFileAction{},
		newCommitFilesAction(deps),
		newUpdateDraftAction(deps),
		listIntegrationsAction{},
		newListResourcesAction(deps),
	)
}

// NewRegistry creates a registry from explicit actions.
func NewRegistry(actions ...Action) *Registry {
	byName := make(map[string]Action, len(actions))
	names := make([]string, 0, len(actions))

	for _, action := range actions {
		if action == nil {
			panic("superplane canvas action is nil")
		}

		name := strings.TrimSpace(action.Name())
		if name == "" {
			panic("superplane canvas action name is required")
		}
		if _, exists := byName[name]; exists {
			panic(fmt.Sprintf("superplane canvas action %q already registered", name))
		}

		byName[name] = action
		names = append(names, name)
	}

	return &Registry{actions: byName, names: names}
}

// Names returns the registered action names in schema enum order.
func (r *Registry) Names() []string {
	return append([]string(nil), r.names...)
}

// Execute dispatches the typed action input to the matching action.
func (r *Registry) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	name := strings.TrimSpace(input.Action)
	action, ok := r.actions[name]
	if !ok {
		return nil, fmt.Errorf("unsupported action %q", input.Action)
	}
	return action.Execute(ctx, session, input)
}
