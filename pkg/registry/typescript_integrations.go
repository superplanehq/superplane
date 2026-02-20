package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/runtime/runner"
	runtimeTS "github.com/superplanehq/superplane/pkg/runtime/typescript"
)

type typeScriptRuntimeIntegration struct {
	definition runtimeTS.IntegrationDefinition
	runner     runner.Client
	timeout    time.Duration
	version    string
}

func (i *typeScriptRuntimeIntegration) Name() string {
	return i.definition.Name
}

func (i *typeScriptRuntimeIntegration) Label() string {
	return i.definition.Manifest.Label
}

func (i *typeScriptRuntimeIntegration) Icon() string {
	return i.definition.Manifest.Icon
}

func (i *typeScriptRuntimeIntegration) Description() string {
	return i.definition.Manifest.Description
}

func (i *typeScriptRuntimeIntegration) Instructions() string {
	return i.definition.Manifest.Instructions
}

func (i *typeScriptRuntimeIntegration) Configuration() []configuration.Field {
	return i.definition.Manifest.Configuration
}

func (i *typeScriptRuntimeIntegration) Components() []core.Component {
	components := make([]core.Component, 0, len(i.definition.Components))
	for _, component := range i.definition.Components {
		components = append(components, &typeScriptRuntimeIntegrationComponent{
			definition:               component,
			integrationConfiguration: i.definition.Manifest.Configuration,
			runner:                   i.runner,
			timeout:                  i.timeout,
			version:                  i.version,
		})
	}

	return components
}

func (i *typeScriptRuntimeIntegration) Triggers() []core.Trigger {
	triggers := make([]core.Trigger, 0, len(i.definition.Triggers))
	for _, trigger := range i.definition.Triggers {
		triggers = append(triggers, &typeScriptRuntimeIntegrationTrigger{
			definition: trigger,
			runner:     i.runner,
			timeout:    i.timeout,
			version:    i.version,
		})
	}

	return triggers
}

func (i *typeScriptRuntimeIntegration) Sync(ctx core.SyncContext) error {
	input := runtimeTS.IntegrationRuntimeContext{
		Configuration:   resolveIntegrationConfiguration(i.definition.Manifest.Configuration, ctx.Integration),
		OrganizationID:  ctx.OrganizationID,
		BaseURL:         ctx.BaseURL,
		WebhooksBaseURL: ctx.WebhooksBaseURL,
	}

	if metadata, ok := ctx.Integration.GetMetadata().(map[string]any); ok {
		input.Metadata = metadata
	}

	runnerCtx, cancel := withTimeout(i.timeout)
	defer cancel()

	response, err := i.runner.SyncIntegration(
		runnerCtx,
		i.definition.Name,
		newTypeScriptRunnerRequest(
			i.version,
			i.timeout,
			runner.RuntimeContext{
				OrganizationID: ctx.OrganizationID,
			},
			input,
		),
	)
	if err != nil {
		return err
	}
	if !response.OK {
		return typeScriptRunnerResponseError(response, "TypeScript integration sync failed")
	}

	var runtimeResponse runtimeTS.IntegrationRuntimeResponse
	if err := decodeTypeScriptRunnerOutput(response.Output, &runtimeResponse); err != nil {
		return err
	}

	applyIntegrationRuntimeResponse(ctx.Integration, &runtimeResponse)
	if runtimeResponse.Outcome == runtimeTS.OutcomeFail {
		return runtimeResponseError(&runtimeResponse, "TypeScript integration sync failed")
	}

	ctx.Integration.Ready()
	return nil
}

func (i *typeScriptRuntimeIntegration) Cleanup(ctx core.IntegrationCleanupContext) error {
	input := runtimeTS.IntegrationRuntimeContext{
		Configuration:  resolveIntegrationConfiguration(i.definition.Manifest.Configuration, ctx.Integration),
		OrganizationID: ctx.OrganizationID,
		BaseURL:        ctx.BaseURL,
	}
	if metadata, ok := ctx.Integration.GetMetadata().(map[string]any); ok {
		input.Metadata = metadata
	}

	runnerCtx, cancel := withTimeout(i.timeout)
	defer cancel()

	response, err := i.runner.CleanupIntegration(
		runnerCtx,
		i.definition.Name,
		newTypeScriptRunnerRequest(
			i.version,
			i.timeout,
			runner.RuntimeContext{
				OrganizationID: ctx.OrganizationID,
			},
			input,
		),
	)
	if err != nil {
		return err
	}
	if !response.OK {
		return typeScriptRunnerResponseError(response, "TypeScript integration cleanup failed")
	}

	return nil
}

func (i *typeScriptRuntimeIntegration) Actions() []core.Action {
	return []core.Action{}
}

func (i *typeScriptRuntimeIntegration) HandleAction(ctx core.IntegrationActionContext) error {
	_ = ctx
	return nil
}

func (i *typeScriptRuntimeIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	_ = resourceType
	_ = ctx
	return []core.IntegrationResource{}, nil
}

func (i *typeScriptRuntimeIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	_ = i
	_, _ = io.Copy(io.Discard, ctx.Request.Body)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

type typeScriptRuntimeIntegrationComponent struct {
	definition               runtimeTS.IntegrationComponentDefinition
	integrationConfiguration []configuration.Field
	runner                   runner.Client
	timeout                  time.Duration
	version                  string
}

func (c *typeScriptRuntimeIntegrationComponent) Name() string {
	return c.definition.Name
}

func (c *typeScriptRuntimeIntegrationComponent) Label() string {
	return c.definition.Manifest.Label
}

func (c *typeScriptRuntimeIntegrationComponent) Description() string {
	return c.definition.Manifest.Description
}

func (c *typeScriptRuntimeIntegrationComponent) Documentation() string {
	return c.definition.Manifest.Documentation
}

func (c *typeScriptRuntimeIntegrationComponent) Icon() string {
	return c.definition.Manifest.Icon
}

func (c *typeScriptRuntimeIntegrationComponent) Color() string {
	return c.definition.Manifest.Color
}

func (c *typeScriptRuntimeIntegrationComponent) ExampleOutput() map[string]any {
	return c.definition.Manifest.ExampleOutput
}

func (c *typeScriptRuntimeIntegrationComponent) OutputChannels(_ any) []core.OutputChannel {
	return c.definition.Manifest.OutputChannels
}

func (c *typeScriptRuntimeIntegrationComponent) Configuration() []configuration.Field {
	return c.definition.Manifest.Configuration
}

func (c *typeScriptRuntimeIntegrationComponent) Setup(ctx core.SetupContext) error {
	input := runtimeTS.ComponentExecutionInput{
		Configuration:            ctx.Configuration,
		IntegrationConfiguration: resolveIntegrationConfiguration(c.integrationConfiguration, ctx.Integration),
	}

	if ctx.Metadata != nil {
		if metadata, ok := ctx.Metadata.Get().(map[string]any); ok {
			input.Metadata = metadata
		}
	}

	runnerCtx, cancel := withTimeout(c.timeout)
	defer cancel()

	response, err := c.runner.SetupComponent(
		runnerCtx,
		c.definition.Name,
		newTypeScriptRunnerRequest(c.version, c.timeout, runner.RuntimeContext{}, input),
	)
	if err != nil {
		return err
	}
	if !response.OK {
		return typeScriptRunnerResponseError(response, "TypeScript integration component setup failed")
	}

	var runtimeResponse runtimeTS.ComponentExecutionResponse
	if err := decodeTypeScriptRunnerOutput(response.Output, &runtimeResponse); err != nil {
		return err
	}

	if runtimeResponse.Outcome == runtimeTS.OutcomeFail {
		message := runtimeResponse.Error
		if message == "" {
			message = "TypeScript integration component setup failed"
		}
		return fmt.Errorf("%s", message)
	}

	if ctx.Metadata != nil && runtimeResponse.Metadata != nil {
		if err := ctx.Metadata.Set(runtimeResponse.Metadata); err != nil {
			return err
		}
	}

	return nil
}

func (c *typeScriptRuntimeIntegrationComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *typeScriptRuntimeIntegrationComponent) Execute(ctx core.ExecutionContext) error {
	input := runtimeTS.ComponentExecutionInput{
		ExecutionID:              ctx.ID.String(),
		WorkflowID:               ctx.WorkflowID,
		OrganizationID:           ctx.OrganizationID,
		NodeID:                   ctx.NodeID,
		SourceNodeID:             ctx.SourceNodeID,
		Configuration:            ctx.Configuration,
		IntegrationConfiguration: resolveIntegrationConfiguration(c.integrationConfiguration, ctx.Integration),
		Data:                     ctx.Data,
	}

	if metadata, ok := ctx.Metadata.Get().(map[string]any); ok {
		input.Metadata = metadata
	}
	if metadata, ok := ctx.NodeMetadata.Get().(map[string]any); ok {
		input.NodeMetadata = metadata
	}

	runnerCtx, cancel := withTimeout(c.timeout)
	defer cancel()

	response, err := c.runner.ExecuteComponent(
		runnerCtx,
		c.definition.Name,
		newTypeScriptRunnerRequest(
			c.version,
			c.timeout,
			runner.RuntimeContext{
				OrganizationID: ctx.OrganizationID,
				NodeID:         ctx.NodeID,
			},
			input,
		),
	)
	if err != nil {
		return err
	}
	if !response.OK {
		return typeScriptRunnerResponseError(response, "TypeScript integration component execution failed")
	}

	var runtimeResponse runtimeTS.ComponentExecutionResponse
	if err := decodeTypeScriptRunnerOutput(response.Output, &runtimeResponse); err != nil {
		return err
	}

	if runtimeResponse.Metadata != nil {
		if err := ctx.Metadata.Set(runtimeResponse.Metadata); err != nil {
			return err
		}
	}
	if runtimeResponse.NodeMetadata != nil {
		if err := ctx.NodeMetadata.Set(runtimeResponse.NodeMetadata); err != nil {
			return err
		}
	}
	for _, kv := range runtimeResponse.KVs {
		if err := ctx.ExecutionState.SetKV(kv.Key, kv.Value); err != nil {
			return err
		}
	}

	switch runtimeResponse.Outcome {
	case runtimeTS.OutcomePass:
		if len(runtimeResponse.Outputs) == 0 {
			return ctx.ExecutionState.Pass()
		}
		for _, output := range runtimeResponse.Outputs {
			if err := ctx.ExecutionState.Emit(output.Channel, output.PayloadType, []any{output.Payload}); err != nil {
				return err
			}
		}
		return nil
	case runtimeTS.OutcomeFail:
		reason := runtimeResponse.ErrorReason
		if reason == "" {
			reason = "error"
		}
		message := runtimeResponse.Error
		if message == "" {
			message = "TypeScript integration component failed"
		}
		return ctx.ExecutionState.Fail(reason, message)
	case runtimeTS.OutcomeNoop:
		return nil
	default:
		return fmt.Errorf("unsupported TypeScript integration component outcome: %s", runtimeResponse.Outcome)
	}
}

func (c *typeScriptRuntimeIntegrationComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *typeScriptRuntimeIntegrationComponent) HandleAction(_ core.ActionContext) error {
	return fmt.Errorf("component %s does not support actions", c.definition.Name)
}

func (c *typeScriptRuntimeIntegrationComponent) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *typeScriptRuntimeIntegrationComponent) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *typeScriptRuntimeIntegrationComponent) Cleanup(_ core.SetupContext) error {
	return nil
}

type typeScriptRuntimeIntegrationTrigger struct {
	definition runtimeTS.IntegrationTriggerDefinition
	runner     runner.Client
	timeout    time.Duration
	version    string
}

func (t *typeScriptRuntimeIntegrationTrigger) Name() string {
	return t.definition.Name
}

func (t *typeScriptRuntimeIntegrationTrigger) Label() string {
	return t.definition.Manifest.Label
}

func (t *typeScriptRuntimeIntegrationTrigger) Description() string {
	return t.definition.Manifest.Description
}

func (t *typeScriptRuntimeIntegrationTrigger) Documentation() string {
	return t.definition.Manifest.Documentation
}

func (t *typeScriptRuntimeIntegrationTrigger) Icon() string {
	return t.definition.Manifest.Icon
}

func (t *typeScriptRuntimeIntegrationTrigger) Color() string {
	return t.definition.Manifest.Color
}

func (t *typeScriptRuntimeIntegrationTrigger) ExampleData() map[string]any {
	return t.definition.Manifest.ExampleData
}

func (t *typeScriptRuntimeIntegrationTrigger) Configuration() []configuration.Field {
	return t.definition.Manifest.Configuration
}

func (t *typeScriptRuntimeIntegrationTrigger) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *typeScriptRuntimeIntegrationTrigger) Setup(ctx core.TriggerContext) error {
	input := map[string]any{
		"configuration": ctx.Configuration,
	}

	if ctx.Metadata != nil {
		if metadata, ok := ctx.Metadata.Get().(map[string]any); ok {
			input["metadata"] = metadata
		}
	}

	runnerCtx, cancel := withTimeout(t.timeout)
	defer cancel()

	response, err := t.runner.SetupTrigger(
		runnerCtx,
		t.definition.Name,
		newTypeScriptRunnerRequest(
			t.version,
			t.timeout,
			runner.RuntimeContext{},
			input,
		),
	)
	if err != nil {
		return err
	}
	if !response.OK {
		return typeScriptRunnerResponseError(response, fmt.Sprintf("TypeScript trigger setup failed for %s", t.definition.Name))
	}

	return nil
}

func (t *typeScriptRuntimeIntegrationTrigger) Actions() []core.Action {
	return []core.Action{}
}

func (t *typeScriptRuntimeIntegrationTrigger) HandleAction(_ core.TriggerActionContext) (map[string]any, error) {
	return nil, fmt.Errorf("typescript integration trigger actions not implemented yet for %s", t.definition.Name)
}

func (t *typeScriptRuntimeIntegrationTrigger) Cleanup(_ core.TriggerContext) error {
	return nil
}

func (r *Registry) registerTypeScriptIntegrationsFromEnv() error {
	definitions, err := runtimeTS.DiscoverIntegrationsFromEnv()
	if err != nil {
		return err
	}

	runnerClient, cfg, err := newTypeScriptRunner()
	if err != nil {
		return err
	}

	for _, definition := range definitions {
		if _, exists := r.Integrations[definition.Name]; exists {
			return fmt.Errorf("typescript integration %s conflicts with existing registered integration", definition.Name)
		}

		r.Integrations[definition.Name] = NewPanicableIntegration(&typeScriptRuntimeIntegration{
			definition: definition,
			runner:     runnerClient,
			timeout:    cfg.Timeout,
			version:    cfg.Version,
		})
	}

	return nil
}

func resolveIntegrationConfiguration(fields []configuration.Field, integration core.IntegrationContext) map[string]any {
	if integration == nil || len(fields) == 0 {
		return nil
	}

	values := map[string]any{}
	for _, field := range fields {
		fieldName := strings.TrimSpace(field.Name)
		if fieldName == "" {
			continue
		}

		raw, err := integration.GetConfig(fieldName)
		if err != nil || len(raw) == 0 {
			continue
		}

		values[fieldName] = decodeIntegrationConfigValue(field.Type, raw)
	}

	if len(values) == 0 {
		return nil
	}

	return values
}

func decodeIntegrationConfigValue(fieldType string, raw []byte) any {
	switch fieldType {
	case configuration.FieldTypeBool:
		var value bool
		if err := json.Unmarshal(raw, &value); err == nil {
			return value
		}
	case configuration.FieldTypeNumber:
		var value float64
		if err := json.Unmarshal(raw, &value); err == nil {
			return value
		}
	}

	return string(raw)
}

func applyIntegrationRuntimeResponse(integration core.IntegrationContext, response *runtimeTS.IntegrationRuntimeResponse) {
	if integration == nil || response == nil {
		return
	}

	if response.Metadata != nil {
		integration.SetMetadata(response.Metadata)
	}

	switch response.State {
	case "ready":
		integration.Ready()
	case "error":
		integration.Error(response.StateDescription)
	}
}

func runtimeResponseError(response *runtimeTS.IntegrationRuntimeResponse, fallback string) error {
	if response == nil {
		return fmt.Errorf("%s", fallback)
	}

	message := strings.TrimSpace(response.Error)
	if message == "" {
		message = fallback
	}

	return fmt.Errorf("%s", message)
}
