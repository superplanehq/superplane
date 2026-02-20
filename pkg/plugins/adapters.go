package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

// PluginComponentAdapter implements core.Component using manifest data for metadata
// and JSON-RPC delegation to the Plugin Host for execution methods.
type PluginComponentAdapter struct {
	contribution ComponentContribution
	pluginName   string
	manager      *Manager
}

func NewPluginComponentAdapter(contribution ComponentContribution, pluginName string, manager *Manager) *PluginComponentAdapter {
	return &PluginComponentAdapter{
		contribution: contribution,
		pluginName:   pluginName,
		manager:      manager,
	}
}

func (a *PluginComponentAdapter) Name() string        { return a.contribution.Name }
func (a *PluginComponentAdapter) Label() string       { return a.contribution.Label }
func (a *PluginComponentAdapter) Description() string { return a.contribution.Description }
func (a *PluginComponentAdapter) Icon() string        { return a.contribution.Icon }
func (a *PluginComponentAdapter) Color() string       { return a.contribution.Color }

func (a *PluginComponentAdapter) Documentation() string {
	return a.contribution.Documentation
}

func (a *PluginComponentAdapter) ExampleOutput() map[string]any {
	if a.contribution.ExampleOutput != nil {
		return a.contribution.ExampleOutput
	}
	return map[string]any{}
}

func (a *PluginComponentAdapter) Configuration() []configuration.Field {
	if a.contribution.Configuration != nil {
		return a.contribution.Configuration
	}
	return []configuration.Field{}
}

func (a *PluginComponentAdapter) OutputChannels(config any) []core.OutputChannel {
	if len(a.contribution.OutputChannels) == 0 {
		return []core.OutputChannel{core.DefaultOutputChannel}
	}

	channels := make([]core.OutputChannel, len(a.contribution.OutputChannels))
	for i, ch := range a.contribution.OutputChannels {
		channels[i] = ch.ToCoreOutputChannel()
	}
	return channels
}

func (a *PluginComponentAdapter) Actions() []core.Action {
	return []core.Action{}
}

func (a *PluginComponentAdapter) Setup(ctx core.SetupContext) error {
	_, err := a.manager.CallPluginWithContext("component/setup", a.pluginName, map[string]any{
		"component": a.contribution.Name,
		"context": map[string]any{
			"configuration": ctx.Configuration,
		},
	}, &CallContext{
		Webhook:  ctx.Webhook,
		HTTP:     ctx.HTTP,
		Metadata: ctx.Metadata,
		Secrets:  ctx.Secrets,
	})
	return err
}

func (a *PluginComponentAdapter) Execute(ctx core.ExecutionContext) error {
	result, err := a.manager.CallPluginWithContext("component/execute", a.pluginName, map[string]any{
		"component": a.contribution.Name,
		"context": map[string]any{
			"id":             ctx.ID.String(),
			"workflowId":     ctx.WorkflowID,
			"organizationId": ctx.OrganizationID,
			"nodeId":         ctx.NodeID,
			"sourceNodeId":   ctx.SourceNodeID,
			"baseUrl":        ctx.BaseURL,
			"input":          ctx.Data,
			"configuration":  ctx.Configuration,
		},
	}, &CallContext{
		HTTP:     ctx.HTTP,
		Metadata: ctx.Metadata,
		Secrets:  ctx.Secrets,
	})

	if err != nil {
		return err
	}

	return applyComponentResult(result, ctx)
}

func (a *PluginComponentAdapter) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *PluginComponentAdapter) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("plugin component %s does not support actions", a.contribution.Name)
}

func (a *PluginComponentAdapter) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (a *PluginComponentAdapter) Cancel(ctx core.ExecutionContext) error {
	_, err := a.manager.CallPluginWithContext("component/cancel", a.pluginName, map[string]any{
		"component": a.contribution.Name,
		"context": map[string]any{
			"id":            ctx.ID.String(),
			"workflowId":    ctx.WorkflowID,
			"nodeId":        ctx.NodeID,
			"configuration": ctx.Configuration,
		},
	}, nil)
	return err
}

func (a *PluginComponentAdapter) Cleanup(ctx core.SetupContext) error {
	_, err := a.manager.CallPluginWithContext("component/cleanup", a.pluginName, map[string]any{
		"component": a.contribution.Name,
		"context": map[string]any{
			"configuration": ctx.Configuration,
		},
	}, &CallContext{
		Webhook:  ctx.Webhook,
		HTTP:     ctx.HTTP,
		Metadata: ctx.Metadata,
		Secrets:  ctx.Secrets,
	})
	return err
}

func applyComponentResult(result *RPCResult, ctx core.ExecutionContext) error {
	if result == nil {
		return nil
	}

	switch result.Action {
	case "emit":
		payloads := []any{result.Data}
		return ctx.ExecutionState.Emit(result.Channel, result.PayloadType, payloads)
	case "pass":
		return ctx.ExecutionState.Pass()
	case "fail":
		return ctx.ExecutionState.Fail(result.Reason, result.Message)
	case "setKV":
		return ctx.ExecutionState.SetKV(result.Key, result.Value)
	}

	return nil
}

// PluginIntegrationAdapter implements core.Integration, delegating lifecycle
// methods (Sync, HandleRequest, Cleanup) to the Plugin Host via RPC.
type PluginIntegrationAdapter struct {
	meta       IntegrationManifest
	pluginName string
	manager    *Manager
	components []core.Component
	triggers   []core.Trigger
}

func NewPluginIntegrationAdapter(meta IntegrationManifest, pluginName string, manager *Manager, components []core.Component, triggers []core.Trigger) *PluginIntegrationAdapter {
	return &PluginIntegrationAdapter{
		meta:       meta,
		pluginName: pluginName,
		manager:    manager,
		components: components,
		triggers:   triggers,
	}
}

func (a *PluginIntegrationAdapter) Name() string                 { return a.meta.Name }
func (a *PluginIntegrationAdapter) Label() string                { return a.meta.Label }
func (a *PluginIntegrationAdapter) Icon() string                 { return a.meta.Icon }
func (a *PluginIntegrationAdapter) Description() string          { return a.meta.Description }
func (a *PluginIntegrationAdapter) Instructions() string         { return "" }
func (a *PluginIntegrationAdapter) Components() []core.Component { return a.components }
func (a *PluginIntegrationAdapter) Triggers() []core.Trigger     { return a.triggers }
func (a *PluginIntegrationAdapter) Actions() []core.Action       { return []core.Action{} }

func (a *PluginIntegrationAdapter) Configuration() []configuration.Field {
	return a.meta.Configuration
}

func (a *PluginIntegrationAdapter) Sync(ctx core.SyncContext) error {
	_, err := a.manager.CallPluginWithContext("integration/sync", a.pluginName, map[string]any{
		"integration": a.meta.Name,
		"context": map[string]any{
			"configuration":   ctx.Configuration,
			"baseUrl":         ctx.BaseURL,
			"webhooksBaseUrl": ctx.WebhooksBaseURL,
			"organizationId":  ctx.OrganizationID,
		},
	}, &CallContext{
		Integration: ctx.Integration,
		HTTP:        ctx.HTTP,
	})
	return err
}

func (a *PluginIntegrationAdapter) HandleRequest(ctx core.HTTPRequestContext) {
	body, _ := readRequestBody(ctx.Request)
	query := make(map[string]string)
	for k, v := range ctx.Request.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	headers := make(map[string]string)
	for k, v := range ctx.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	raw, err := a.manager.CallPluginRaw("integration/handleRequest", a.pluginName, map[string]any{
		"integration": a.meta.Name,
		"context": map[string]any{
			"request": map[string]any{
				"method":  ctx.Request.Method,
				"path":    ctx.Request.URL.Path,
				"query":   query,
				"headers": headers,
				"body":    string(body),
			},
			"organizationId":  ctx.OrganizationID,
			"baseUrl":         ctx.BaseURL,
			"webhooksBaseUrl": ctx.WebhooksBaseURL,
		},
	}, &CallContext{
		Integration: ctx.Integration,
		HTTP:        ctx.HTTP,
	})

	if err != nil {
		ctx.Logger.Errorf("plugin handleRequest error: %v", err)
		http.Error(ctx.Response, "internal server error", http.StatusInternalServerError)
		return
	}

	if raw == nil {
		return
	}

	var resp struct {
		Action  string `json:"action"`
		URL     string `json:"url"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		ctx.Logger.Errorf("plugin handleRequest: failed to parse response: %v", err)
		return
	}

	switch resp.Action {
	case "redirect":
		http.Redirect(ctx.Response, ctx.Request, resp.URL, http.StatusSeeOther)
	case "json":
		ctx.Response.WriteHeader(resp.Status)
	case "error":
		status := resp.Status
		if status == 0 {
			status = http.StatusInternalServerError
		}
		http.Error(ctx.Response, resp.Message, status)
	}
}

func (a *PluginIntegrationAdapter) Cleanup(ctx core.IntegrationCleanupContext) error {
	_, err := a.manager.CallPluginWithContext("integration/cleanup", a.pluginName, map[string]any{
		"integration": a.meta.Name,
		"context": map[string]any{
			"configuration":  ctx.Configuration,
			"baseUrl":        ctx.BaseURL,
			"organizationId": ctx.OrganizationID,
		},
	}, &CallContext{
		Integration: ctx.Integration,
		HTTP:        ctx.HTTP,
	})
	return err
}

func (a *PluginIntegrationAdapter) HandleAction(ctx core.IntegrationActionContext) error {
	return fmt.Errorf("plugin integration %s does not support actions", a.meta.Name)
}

func (a *PluginIntegrationAdapter) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return nil, nil
}

// PluginWebhookHandler implements core.WebhookHandler by delegating to the Plugin Host.
type PluginWebhookHandler struct {
	integrationName string
	pluginName      string
	manager         *Manager
}

func NewPluginWebhookHandler(integrationName, pluginName string, manager *Manager) *PluginWebhookHandler {
	return &PluginWebhookHandler{
		integrationName: integrationName,
		pluginName:      pluginName,
		manager:         manager,
	}
}

func (h *PluginWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("getting webhook secret: %w", err)
	}

	result, err := h.manager.CallPluginWithContext("webhookHandler/setup", h.pluginName, map[string]any{
		"integration": h.integrationName,
		"context": map[string]any{
			"webhookUrl":    ctx.Webhook.GetURL(),
			"webhookSecret": string(secret),
			"configuration": ctx.Webhook.GetConfiguration(),
		},
	}, &CallContext{
		Integration: ctx.Integration,
		HTTP:        ctx.HTTP,
		WebhookCtx:  ctx.Webhook,
	})

	if err != nil {
		return nil, err
	}

	if result != nil && result.Data != nil {
		return result.Data, nil
	}

	return nil, nil
}

func (h *PluginWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	_, err := h.manager.CallPluginWithContext("webhookHandler/cleanup", h.pluginName, map[string]any{
		"integration": h.integrationName,
		"context": map[string]any{
			"webhookMetadata": ctx.Webhook.GetMetadata(),
			"configuration":   ctx.Webhook.GetConfiguration(),
		},
	}, &CallContext{
		Integration: ctx.Integration,
		HTTP:        ctx.HTTP,
		WebhookCtx:  ctx.Webhook,
	})
	return err
}

func (h *PluginWebhookHandler) CompareConfig(a, b any) (bool, error) {
	result, err := h.manager.CallPluginWithContext("webhookHandler/compareConfig", h.pluginName, map[string]any{
		"integration": h.integrationName,
		"context": map[string]any{
			"a": a,
			"b": b,
		},
	}, nil)

	if err != nil {
		return false, err
	}

	if result != nil && result.Data != nil {
		if v, ok := result.Data.(bool); ok {
			return v, nil
		}
	}

	return false, nil
}

func (h *PluginWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

// PluginTriggerAdapter implements core.Trigger using manifest data for metadata
// and JSON-RPC delegation to the Plugin Host for execution methods.
type PluginTriggerAdapter struct {
	contribution TriggerContribution
	pluginName   string
	manager      *Manager
}

func NewPluginTriggerAdapter(contribution TriggerContribution, pluginName string, manager *Manager) *PluginTriggerAdapter {
	return &PluginTriggerAdapter{
		contribution: contribution,
		pluginName:   pluginName,
		manager:      manager,
	}
}

func (a *PluginTriggerAdapter) Name() string        { return a.contribution.Name }
func (a *PluginTriggerAdapter) Label() string       { return a.contribution.Label }
func (a *PluginTriggerAdapter) Description() string { return a.contribution.Description }
func (a *PluginTriggerAdapter) Icon() string        { return a.contribution.Icon }
func (a *PluginTriggerAdapter) Color() string       { return a.contribution.Color }

func (a *PluginTriggerAdapter) Documentation() string {
	return a.contribution.Documentation
}

func (a *PluginTriggerAdapter) ExampleData() map[string]any {
	if a.contribution.ExampleData != nil {
		return a.contribution.ExampleData
	}
	return map[string]any{}
}

func (a *PluginTriggerAdapter) Configuration() []configuration.Field {
	if a.contribution.Configuration != nil {
		return a.contribution.Configuration
	}
	return []configuration.Field{}
}

func (a *PluginTriggerAdapter) Actions() []core.Action {
	return []core.Action{}
}

func (a *PluginTriggerAdapter) Setup(ctx core.TriggerContext) error {
	_, err := a.manager.CallPluginWithContext("trigger/setup", a.pluginName, map[string]any{
		"trigger": a.contribution.Name,
		"context": map[string]any{
			"configuration": ctx.Configuration,
		},
	}, &CallContext{
		Webhook:     ctx.Webhook,
		HTTP:        ctx.HTTP,
		Metadata:    ctx.Metadata,
		Events:      ctx.Events,
		Secrets:     ctx.Secrets,
		Integration: ctx.Integration,
	})
	return err
}

func (a *PluginTriggerAdapter) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	_, err := a.manager.CallPluginWithContext("trigger/handleWebhook", a.pluginName, map[string]any{
		"trigger": a.contribution.Name,
		"context": map[string]any{
			"body":          string(ctx.Body),
			"headers":       ctx.Headers,
			"workflowId":    ctx.WorkflowID,
			"nodeId":        ctx.NodeID,
			"configuration": ctx.Configuration,
		},
	}, &CallContext{
		Webhook:  ctx.Webhook,
		HTTP:     ctx.HTTP,
		Metadata: ctx.Metadata,
		Events:   ctx.Events,
	})

	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (a *PluginTriggerAdapter) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("plugin trigger %s does not support actions", a.contribution.Name)
}

func (a *PluginTriggerAdapter) Cleanup(ctx core.TriggerContext) error {
	_, err := a.manager.CallPluginWithContext("trigger/cleanup", a.pluginName, map[string]any{
		"trigger": a.contribution.Name,
		"context": map[string]any{
			"configuration": ctx.Configuration,
		},
	}, &CallContext{
		Webhook:     ctx.Webhook,
		HTTP:        ctx.HTTP,
		Metadata:    ctx.Metadata,
		Events:      ctx.Events,
		Secrets:     ctx.Secrets,
		Integration: ctx.Integration,
	})
	return err
}
