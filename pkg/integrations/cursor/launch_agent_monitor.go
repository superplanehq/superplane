package cursor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// HandleWebhook processes incoming updates from Cursor
func (c *LaunchAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	signature := ctx.Headers.Get(LaunchAgentWebhookSignatureHeader)
	if signature == "" {
		return http.StatusUnauthorized, nil, fmt.Errorf("missing signature header")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error getting webhook secret: %w", err)
	}

	if !verifyWebhookSignature(ctx.Body, signature, string(secret)) {
		return http.StatusUnauthorized, nil, fmt.Errorf("invalid webhook signature")
	}

	// 2. Parse payload
	var payload launchAgentWebhookPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("invalid json body: %w", err)
	}

	if payload.ID == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("id missing from webhook payload")
	}

	// 3. Correlate Webhook to Execution
	executionCtx, err := ctx.FindExecutionByKV("agent_id", payload.ID)
	if err != nil {
		// Execution not found (likely old or deleted), ack to stop retries
		return http.StatusOK, nil, nil
	}

	metadata := LaunchAgentExecutionMetadata{}
	if err := mapstructure.Decode(executionCtx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	// 4. Idempotency Check
	if metadata.Agent != nil && isTerminalStatus(metadata.Agent.Status) {
		return http.StatusOK, nil, nil
	}

	// 5. Update State
	executionCtx.Logger.Infof("Received webhook for Agent %s: %s", payload.ID, payload.Status)
	mergeWebhookPayloadIntoMetadata(&metadata, payload)

	if isTerminalStatus(payload.Status) &&
		(metadata.Agent.Summary == "" || metadata.Target == nil || metadata.Target.PrURL == "") &&
		executionCtx.HTTP != nil &&
		executionCtx.Integration != nil {
		client, err := NewClient(executionCtx.HTTP, executionCtx.Integration)
		if err == nil && client.LaunchAgentKey != "" {
			agentStatus, err := client.GetAgentStatus(payload.ID)
			if err == nil {
				mergeAgentResponseIntoMetadata(&metadata, agentStatus)
			} else {
				executionCtx.Logger.WithError(err).Warnf("Failed to refresh terminal Cursor Agent %s after webhook", payload.ID)
			}
		}
	}

	if err := executionCtx.Metadata.Set(metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to set metadata: %w", err)
	}

	// 6. Complete Workflow if finished
	if isTerminalStatus(payload.Status) {
		prURL := ""
		branchName := ""
		if metadata.Target != nil {
			prURL = metadata.Target.PrURL
			branchName = metadata.Target.BranchName
		}
		outputPayload := buildOutputPayload(metadata.Agent.Status, metadata.Agent.ID, prURL, metadata.Agent.Summary, branchName)
		if err := executionCtx.ExecutionState.Emit(LaunchAgentDefaultChannel, LaunchAgentPayloadType, []any{outputPayload}); err != nil {
			return http.StatusInternalServerError, nil, err
		}
	}

	return http.StatusOK, nil, nil
}

func (c *LaunchAgent) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: "poll",
			Type: core.HookTypeInternal,
		},
	}
}

func (c *LaunchAgent) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *LaunchAgent) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := LaunchAgentExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Agent == nil || metadata.Agent.ID == "" || isTerminalStatus(metadata.Agent.Status) {
		return nil
	}

	// Retrieve polling parameters
	pollAttempt := 1
	pollErrors := 0
	if attempt, ok := ctx.Parameters["attempt"].(float64); ok {
		pollAttempt = int(attempt)
	}
	if errors, ok := ctx.Parameters["errors"].(float64); ok {
		pollErrors = int(errors)
	}

	// Check Max Attempts
	if pollAttempt > LaunchAgentMaxPollAttempts {
		ctx.Logger.Errorf("Agent %s exceeded maximum poll attempts. Failing.", metadata.Agent.ID)
		branchName := ""
		if metadata.Target != nil {
			branchName = metadata.Target.BranchName
		}
		outputPayload := buildOutputPayload("timeout", metadata.Agent.ID, "", "Polling timed out", branchName)
		return ctx.ExecutionState.Emit(LaunchAgentDefaultChannel, LaunchAgentPayloadType, []any{outputPayload})
	}

	// Perform API Check
	ctx.Logger.Infof("Polling Agent %s (attempt %d/%d)...", metadata.Agent.ID, pollAttempt, LaunchAgentMaxPollAttempts)
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return c.scheduleNextPoll(ctx, pollAttempt+1, pollErrors)
	}

	agentStatus, err := client.GetAgentStatus(metadata.Agent.ID)
	if err != nil {
		pollErrors++
		if pollErrors >= LaunchAgentMaxPollErrors {
			ctx.Logger.Errorf("Agent %s exceeded max poll errors. Failing.", metadata.Agent.ID)
			branchName := ""
			if metadata.Target != nil {
				branchName = metadata.Target.BranchName
			}
			outputPayload := buildOutputPayload("error", metadata.Agent.ID, "", "Polling failed repeatedly", branchName)
			return ctx.ExecutionState.Emit(LaunchAgentDefaultChannel, LaunchAgentPayloadType, []any{outputPayload})
		}
		return c.scheduleNextPoll(ctx, pollAttempt+1, pollErrors)
	}

	// Update Metadata
	pollErrors = 0
	mergeAgentResponseIntoMetadata(&metadata, agentStatus)
	_ = ctx.Metadata.Set(metadata) // Best effort save

	// Check for Completion
	if isTerminalStatus(metadata.Agent.Status) {
		prURL := ""
		branchName := ""
		if metadata.Target != nil {
			prURL = metadata.Target.PrURL
			branchName = metadata.Target.BranchName
		}
		outputPayload := buildOutputPayload(metadata.Agent.Status, metadata.Agent.ID, prURL, metadata.Agent.Summary, branchName)
		return ctx.ExecutionState.Emit(LaunchAgentDefaultChannel, LaunchAgentPayloadType, []any{outputPayload})
	}

	return c.scheduleNextPoll(ctx, pollAttempt+1, pollErrors)
}

func (c *LaunchAgent) scheduleNextPoll(ctx core.ActionHookContext, nextAttempt, errors int) error {
	interval := LaunchAgentInitialPollInterval * time.Duration(1<<uint(min(nextAttempt-1, 8)))
	if interval > LaunchAgentMaxPollInterval {
		interval = LaunchAgentMaxPollInterval
	}
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": nextAttempt, "errors": errors}, interval)
}

func (c *LaunchAgent) Cancel(ctx core.ExecutionContext) error {
	metadata := LaunchAgentExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil
	}
	if metadata.Agent == nil || metadata.Agent.ID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil
	}

	if err := client.CancelAgent(metadata.Agent.ID); err != nil {
		ctx.Logger.Warnf("Failed to cancel Cursor Agent %s: %v", metadata.Agent.ID, err)
	} else {
		ctx.Logger.Infof("Cancelled Cursor Agent %s", metadata.Agent.ID)
	}
	return nil
}

func (c *LaunchAgent) Cleanup(ctx core.SetupContext) error { return nil }
