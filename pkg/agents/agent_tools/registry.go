package agenttools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	componentregistry "github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
)

// Dependencies are shared by registered tools that need backend services to
// execute provider custom tool calls.
type Dependencies struct {
	Encryptor         crypto.Encryptor
	ComponentRegistry *componentregistry.Registry
	GitProvider       gitprovider.Provider
	WebhookBaseURL    string
	AuthService       authorization.Authorization
	UsageService      usage.Service
}

// Definition is the provider-facing metadata for a managed-agent custom tool.
type Definition interface {
	Name() string
	Description() string
	InputSchema() agents.CustomToolInputSchema
}

// Result is the backend payload returned by a tool after a typed call.
type Result struct {
	Payload any
}

// AgentTool is implemented by concrete managed-agent tools. Each tool owns a
// typed input contract and does not need to know about provider event payloads.
type AgentTool[T any] interface {
	Definition
	Call(ctx context.Context, session agents.AgentSessionContext, input T) (Result, error)
}

type tool interface {
	Definition
	call(ctx context.Context, call toolCall) (Result, error)
}

type toolCall struct {
	Session agents.AgentSessionContext
	Input   json.RawMessage
}

type factory func(Dependencies) tool

const toolContractRevision = "agent-tools-v1.1"

var registeredTools = struct {
	sync.RWMutex
	factories map[string]factory
}{
	factories: map[string]factory{},
}

// Register adds a managed-agent custom tool factory to the global registry.
// Tool packages call Register from init, following the integration registry
// pattern used elsewhere in SuperPlane.
func Register[T any](name string, factory func(Dependencies) AgentTool[T]) {
	if name == "" {
		panic("agent tool name is required")
	}
	if factory == nil {
		panic(fmt.Sprintf("agent tool %q factory is required", name))
	}

	registeredTools.Lock()
	defer registeredTools.Unlock()
	if _, exists := registeredTools.factories[name]; exists {
		panic(fmt.Sprintf("agent tool %q already registered", name))
	}
	registeredTools.factories[name] = func(deps Dependencies) tool {
		return typedTool[T]{tool: factory(deps)}
	}
}

type Registry struct {
	tools map[string]tool
}

// NewRegistry instantiates all registered managed-agent custom tools.
func NewRegistry(deps Dependencies) *Registry {
	factories := registeredToolFactories()
	tools := make(map[string]tool, len(factories))
	for name, factory := range factories {
		tool := factory(deps)
		if tool == nil {
			panic(fmt.Sprintf("agent tool %q factory returned nil", name))
		}
		if tool.Name() != name {
			panic(fmt.Sprintf("agent tool %q registered with mismatched name %q", name, tool.Name()))
		}
		tools[name] = tool
	}
	return &Registry{tools: tools}
}

// DefaultDefinitions returns the provider-facing metadata for all registered
// tools without requiring worker-only runtime dependencies.
func DefaultDefinitions() []Definition {
	return NewRegistry(Dependencies{}).Definitions()
}

// DefinitionMaps returns Anthropic-compatible custom tool definitions.
func DefinitionMaps() []map[string]any {
	definitions := DefaultDefinitions()
	tools := make([]map[string]any, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, map[string]any{
			"type":         "custom",
			"name":         definition.Name(),
			"description":  definition.Description(),
			"input_schema": definition.InputSchema().Map(),
		})
	}
	return tools
}

// SchemaRevision identifies the provider-facing custom tool contract. The
// hash changes when tool names, descriptions, or input schemas change; bump
// toolContractRevision for behavior changes that keep the JSON schema stable.
func SchemaRevision() string {
	payload := struct {
		Contract string           `json:"contract"`
		Tools    []map[string]any `json:"tools"`
	}{
		Contract: toolContractRevision,
		Tools:    DefinitionMaps(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("agent tool schema revision: %v", err))
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%s:%x", toolContractRevision, sum[:])
}

// Definitions returns the registered custom tool definitions in stable name
// order.
func (r *Registry) Definitions() []Definition {
	names := r.sortedToolNames()
	definitions := make([]Definition, 0, len(names))
	for _, name := range names {
		definitions = append(definitions, r.tools[name])
	}
	return definitions
}

// ExecuteCustomTool dispatches one provider custom tool invocation to the
// matching registered backend implementation.
func (r *Registry) ExecuteCustomTool(ctx context.Context, session agents.AgentSessionContext, toolUse agents.CustomToolUse) agents.CustomToolResult {
	tool, ok := r.tools[toolUse.Name]
	if !ok {
		return customToolError(toolUse.ID, fmt.Sprintf("unsupported custom tool %q", toolUse.Name))
	}

	result, err := tool.call(ctx, toolCall{
		Session: session,
		Input:   json.RawMessage(toolUse.Input),
	})
	if err != nil {
		return customToolError(toolUse.ID, err.Error())
	}

	content, err := json.Marshal(result.Payload)
	if err != nil {
		return customToolError(toolUse.ID, fmt.Sprintf("encode result: %v", err))
	}

	return agents.CustomToolResult{
		CustomToolUseID: toolUse.ID,
		Content:         string(content),
	}
}

func (r *Registry) sortedToolNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func registeredToolFactories() map[string]factory {
	registeredTools.RLock()
	defer registeredTools.RUnlock()

	factories := make(map[string]factory, len(registeredTools.factories))
	for name, factory := range registeredTools.factories {
		factories[name] = factory
	}
	return factories
}

type typedTool[T any] struct {
	tool AgentTool[T]
}

func (t typedTool[T]) Name() string {
	return t.tool.Name()
}

func (t typedTool[T]) Description() string {
	return t.tool.Description()
}

func (t typedTool[T]) InputSchema() agents.CustomToolInputSchema {
	return t.tool.InputSchema()
}

func (t typedTool[T]) call(ctx context.Context, call toolCall) (Result, error) {
	var input T
	if len(bytes.TrimSpace(call.Input)) != 0 {
		if err := json.Unmarshal(call.Input, &input); err != nil {
			return Result{}, fmt.Errorf("invalid input: %w", err)
		}
	}
	return t.tool.Call(ctx, call.Session, input)
}
