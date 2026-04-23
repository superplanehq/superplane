package contexts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionStateContext struct {
	execution      *models.CanvasNodeExecution
	tx             *gorm.DB
	maxPayloadSize int
	onNewEvents    func([]models.CanvasEvent)
	configBuilder  *NodeConfigurationBuilder
}

func NewExecutionStateContext(
	tx *gorm.DB,
	execution *models.CanvasNodeExecution,
	onNewEvents func([]models.CanvasEvent),
) *ExecutionStateContext {
	return &ExecutionStateContext{
		tx:             tx,
		execution:      execution,
		maxPayloadSize: DefaultMaxPayloadSize,
		onNewEvents:    onNewEvents,
	}
}

//
// SetConfigBuilder wires a NodeConfigurationBuilder so that report
// templates can be resolved against the full expression namespace (root,
// previous, $, memory) when the node finishes. Without this set, the
// report template will be skipped rather than failing the execution.
//
func (s *ExecutionStateContext) SetConfigBuilder(builder *NodeConfigurationBuilder) {
	s.configBuilder = builder
}

func (s *ExecutionStateContext) IsFinished() bool {
	return s.execution.State == models.CanvasNodeExecutionStateFinished
}

func (s *ExecutionStateContext) Pass() error {
	s.resolveReportEntry(nil)

	newEvents, err := s.execution.PassInTransaction(s.tx, map[string][]any{})
	if err != nil {
		return err
	}

	if s.onNewEvents != nil {
		s.onNewEvents(newEvents)
	}

	return nil
}

func (s *ExecutionStateContext) Emit(channel, payloadType string, payloads []any) error {
	outputs := map[string][]any{
		channel: {},
	}

	outputEvents := make([]any, 0, len(payloads))
	for _, payload := range payloads {
		event := map[string]any{
			"type":      payloadType,
			"timestamp": time.Now(),
			"data":      payload,
		}

		outputEvents = append(outputEvents, event)

		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

		if len(data) > s.maxPayloadSize {
			return fmt.Errorf("event payload too large: %d bytes (max %d)", len(data), s.maxPayloadSize)
		}

		outputs[channel] = append(outputs[channel], json.RawMessage(data))
	}

	s.resolveReportEntry(outputEvents)

	newEvents, err := s.execution.PassInTransaction(s.tx, outputs)
	if err != nil {
		return err
	}

	if s.onNewEvents != nil {
		s.onNewEvents(newEvents)
	}

	return nil
}

func (s *ExecutionStateContext) resolveReportEntry(outputEvents []any) {
	config := s.execution.Configuration.Data()
	if config == nil {
		return
	}

	rawTemplate, ok := config["reportTemplate"]
	if !ok || rawTemplate == nil {
		return
	}

	tmpl, ok := rawTemplate.(string)
	if !ok {
		return
	}

	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return
	}

	//
	// Most call sites wire a configBuilder when constructing the execution
	// context (so expressions resolve against the full message chain), but
	// several completion paths -- node_executor, request worker, integration
	// subscriptions, approval actions -- construct an ExecutionStateContext
	// directly without calling SetConfigBuilder. Without a builder we'd
	// silently drop the report template for every downstream execution, so
	// we lazily reconstruct a minimal builder from the execution itself.
	// Expressions still get `previous()` and `root()` because the builder
	// derives that from the previous execution chain on the execution.
	//
	builder := s.configBuilder
	if builder == nil {
		builder = s.buildFallbackConfigBuilder()
	}
	if builder == nil {
		return
	}

	resolved, errs := builder.ResolveReportTemplate(tmpl, outputEvents)
	resolved = strings.TrimSpace(resolved)

	if len(errs) > 0 {
		lines := make([]string, 0, len(errs))
		for _, e := range errs {
			lines = append(lines, fmt.Sprintf("> `%s`", e.Error()))
		}
		resolved += fmt.Sprintf("\n\n> [!CAUTION]\n> Expression errors:\n%s", strings.Join(lines, "\n"))
	}

	if resolved != "" {
		s.execution.ReportEntry = resolved
	}
}

//
// buildFallbackConfigBuilder produces a NodeConfigurationBuilder wired with
// enough context to evaluate report template expressions even when the
// outer execution context didn't call SetConfigBuilder. The builder needs
// the input event plus the chain references (root event, previous
// execution) so that root() / previous() / $ resolve the same way they do
// on the happy path. If the input event can't be loaded we return nil so
// the caller silently skips template resolution (matching prior behavior
// for unavailable builders).
//
func (s *ExecutionStateContext) buildFallbackConfigBuilder() *NodeConfigurationBuilder {
	inputEvent, err := models.FindCanvasEventInTransaction(s.tx, s.execution.EventID)
	if err != nil {
		return nil
	}

	builder := NewNodeConfigurationBuilder(s.tx, s.execution.WorkflowID).
		WithNodeID(s.execution.NodeID).
		WithRootEvent(&s.execution.RootEventID).
		WithInput(map[string]any{inputEvent.NodeID: inputEvent.Data.Data()})

	if s.execution.PreviousExecutionID != nil {
		builder = builder.WithPreviousExecution(s.execution.PreviousExecutionID)
	}

	return builder
}

func (s *ExecutionStateContext) Fail(reason, message string) error {
	err := s.execution.FailInTransaction(s.tx, reason, message)
	return err
}

func (s *ExecutionStateContext) SetKV(key, value string) error {
	return models.CreateNodeExecutionKVInTransaction(s.tx, s.execution.WorkflowID, s.execution.NodeID, s.execution.ID, key, value)
}
