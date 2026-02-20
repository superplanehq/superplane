package registry

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	runtimeTS "github.com/superplanehq/superplane/pkg/runtime/typescript"
)

type typeScriptRuntimeComponent struct {
	definition runtimeTS.ComponentDefinition
	binary     string
	timeout    time.Duration
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
	request := runtimeTS.ComponentExecutionRequest{
		Operation: runtimeTS.OperationComponentSetup,
		Component: c.definition.Name,
		Context: runtimeTS.ComponentExecutionInput{
			ExecutionID:    "",
			WorkflowID:     "",
			OrganizationID: "",
			NodeID:         "",
			SourceNodeID:   "",
			Configuration:  ctx.Configuration,
			Data:           nil,
		},
	}

	if ctx.Metadata != nil {
		if metadata, ok := ctx.Metadata.Get().(map[string]any); ok {
			request.Context.Metadata = metadata
		}
	}

	response, err := runtimeTS.ExecuteComponentEntrypoint(c.binary, c.timeout, c.definition.Entrypoint, request)
	if err != nil {
		return err
	}

	if ctx.Metadata != nil && response.Metadata != nil {
		if err := ctx.Metadata.Set(response.Metadata); err != nil {
			return err
		}
	}

	if response.Outcome == runtimeTS.OutcomeFail {
		message := response.Error
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
	request := runtimeTS.ComponentExecutionRequest{
		Operation: runtimeTS.OperationComponentExecute,
		Component: c.definition.Name,
		Context: runtimeTS.ComponentExecutionInput{
			ExecutionID:    ctx.ID.String(),
			WorkflowID:     ctx.WorkflowID,
			OrganizationID: ctx.OrganizationID,
			NodeID:         ctx.NodeID,
			SourceNodeID:   ctx.SourceNodeID,
			Configuration:  ctx.Configuration,
			Data:           ctx.Data,
		},
	}

	if metadata, ok := ctx.Metadata.Get().(map[string]any); ok {
		request.Context.Metadata = metadata
	}
	if metadata, ok := ctx.NodeMetadata.Get().(map[string]any); ok {
		request.Context.NodeMetadata = metadata
	}

	response, err := runtimeTS.ExecuteComponentEntrypoint(c.binary, c.timeout, c.definition.Entrypoint, request)
	if err != nil {
		return err
	}

	if response.Metadata != nil {
		if err := ctx.Metadata.Set(response.Metadata); err != nil {
			return err
		}
	}
	if response.NodeMetadata != nil {
		if err := ctx.NodeMetadata.Set(response.NodeMetadata); err != nil {
			return err
		}
	}
	for _, kv := range response.KVs {
		if err := ctx.ExecutionState.SetKV(kv.Key, kv.Value); err != nil {
			return err
		}
	}

	switch response.Outcome {
	case runtimeTS.OutcomePass:
		if len(response.Outputs) == 0 {
			return ctx.ExecutionState.Pass()
		}
		for _, output := range response.Outputs {
			if err := ctx.ExecutionState.Emit(output.Channel, output.PayloadType, []any{output.Payload}); err != nil {
				return err
			}
		}
		return nil
	case runtimeTS.OutcomeFail:
		reason := response.ErrorReason
		if reason == "" {
			reason = "error"
		}
		message := response.Error
		if message == "" {
			message = "TypeScript component failed"
		}
		return ctx.ExecutionState.Fail(reason, message)
	case runtimeTS.OutcomeNoop:
		return nil
	default:
		return fmt.Errorf("unsupported TypeScript component outcome: %s", response.Outcome)
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

	binary := strings.TrimSpace(os.Getenv("DENO_BINARY"))
	if binary == "" {
		binary = runtimeTS.DefaultDenoBinary
	}

	timeout := runtimeTS.DefaultDenoExecutionTimeout
	timeoutValue := strings.TrimSpace(os.Getenv("DENO_EXECUTION_TIMEOUT"))
	if timeoutValue != "" {
		if parsed, err := time.ParseDuration(timeoutValue); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	for _, definition := range definitions {
		if _, exists := r.Components[definition.Name]; exists {
			return fmt.Errorf("typescript component %s conflicts with existing registered component", definition.Name)
		}

		r.Components[definition.Name] = NewPanicableComponent(&typeScriptRuntimeComponent{
			definition: definition,
			binary:     binary,
			timeout:    timeout,
		})
	}

	return nil
}
