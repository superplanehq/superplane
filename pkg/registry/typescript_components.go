package registry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/runtime/runner"
	runtimeTS "github.com/superplanehq/superplane/pkg/runtime/typescript"
)

type typeScriptRuntimeComponent struct {
	definition runtimeTS.ComponentDefinition
	runner     runner.Client
	timeout    time.Duration
	version    string
}

func (c *typeScriptRuntimeComponent) Name() string {
	return c.definition.Name
}

func (c *typeScriptRuntimeComponent) Label() string {
	return c.definition.Manifest.Label
}

func (c *typeScriptRuntimeComponent) Description() string {
	return c.definition.Manifest.Description
}

func (c *typeScriptRuntimeComponent) Documentation() string {
	return c.definition.Manifest.Documentation
}

func (c *typeScriptRuntimeComponent) Icon() string {
	return c.definition.Manifest.Icon
}

func (c *typeScriptRuntimeComponent) Color() string {
	return c.definition.Manifest.Color
}

func (c *typeScriptRuntimeComponent) ExampleOutput() map[string]any {
	return c.definition.Manifest.ExampleOutput
}

func (c *typeScriptRuntimeComponent) OutputChannels(_ any) []core.OutputChannel {
	return c.definition.Manifest.OutputChannels
}

func (c *typeScriptRuntimeComponent) Configuration() []configuration.Field {
	return c.definition.Manifest.Configuration
}

func (c *typeScriptRuntimeComponent) Setup(ctx core.SetupContext) error {
	input := runtimeTS.ComponentExecutionInput{
		ExecutionID:    "",
		WorkflowID:     "",
		OrganizationID: "",
		NodeID:         "",
		SourceNodeID:   "",
		Configuration:  ctx.Configuration,
		Data:           nil,
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
		return typeScriptRunnerResponseError(response, "TypeScript component setup failed")
	}

	var runtimeResponse runtimeTS.ComponentExecutionResponse
	if err := decodeTypeScriptRunnerOutput(response.Output, &runtimeResponse); err != nil {
		return err
	}

	if ctx.Metadata != nil && runtimeResponse.Metadata != nil {
		if err := ctx.Metadata.Set(runtimeResponse.Metadata); err != nil {
			return err
		}
	}

	if runtimeResponse.Outcome == runtimeTS.OutcomeFail {
		message := runtimeResponse.Error
		if message == "" {
			message = "TypeScript component setup failed"
		}
		return fmt.Errorf("%s", message)
	}

	return nil
}

func (c *typeScriptRuntimeComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *typeScriptRuntimeComponent) Execute(ctx core.ExecutionContext) error {
	input := runtimeTS.ComponentExecutionInput{
		ExecutionID:    ctx.ID.String(),
		WorkflowID:     ctx.WorkflowID,
		OrganizationID: ctx.OrganizationID,
		NodeID:         ctx.NodeID,
		SourceNodeID:   ctx.SourceNodeID,
		Configuration:  ctx.Configuration,
		Data:           ctx.Data,
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
		return typeScriptRunnerResponseError(response, "TypeScript component execution failed")
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
			message = "TypeScript component failed"
		}
		return ctx.ExecutionState.Fail(reason, message)
	case runtimeTS.OutcomeNoop:
		return nil
	default:
		return fmt.Errorf("unsupported TypeScript component outcome: %s", runtimeResponse.Outcome)
	}
}

func (c *typeScriptRuntimeComponent) Actions() []core.Action {
	return []core.Action{}
}

func (c *typeScriptRuntimeComponent) HandleAction(_ core.ActionContext) error {
	return fmt.Errorf("component %s does not support actions", c.definition.Name)
}

func (c *typeScriptRuntimeComponent) HandleWebhook(_ core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *typeScriptRuntimeComponent) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *typeScriptRuntimeComponent) Cleanup(_ core.SetupContext) error {
	return nil
}

func (r *Registry) registerTypeScriptComponentsFromEnv() error {
	definitions, err := runtimeTS.DiscoverComponentsFromEnv()
	if err != nil {
		return err
	}

	runnerClient, cfg, err := newTypeScriptRunner()
	if err != nil {
		return err
	}

	for _, definition := range definitions {
		if _, exists := r.Components[definition.Name]; exists {
			return fmt.Errorf("typescript component %s conflicts with existing registered component", definition.Name)
		}

		r.Components[definition.Name] = NewPanicableComponent(&typeScriptRuntimeComponent{
			definition: definition,
			runner:     runnerClient,
			timeout:    cfg.Timeout,
			version:    cfg.Version,
		})
	}

	return nil
}
