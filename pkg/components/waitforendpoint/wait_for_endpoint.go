package waitforendpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ReadyOutputChannel   = "ready"
	TimeoutOutputChannel = "timeout"

	ReadyPayloadType   = "endpoint.ready"
	TimeoutPayloadType = "endpoint.timeout"

	checkEndpointHook = "checkEndpoint"

	defaultExpectedStatus       = "2xx"
	defaultIntervalSeconds      = 10
	defaultTimeoutSeconds       = 300
	defaultRequestTimeout       = 10
	defaultMaxResponseBytes     = 64 * 1024
	maxIntervalSeconds          = 300
	maxTimeoutSeconds           = 3600
	maxRequestTimeoutSeconds    = 30
	minMaxResponseBytes         = 1024
	maxMaxResponseBytes         = 256 * 1024
	responseSizeErrorMessage    = "response exceeds configured maximum size of %d bytes"
	invalidConditionErrorFormat = "condition must evaluate to a boolean, got %T"
)

func init() {
	registry.RegisterAction("waitForEndpoint", &WaitForEndpoint{})
}

type WaitForEndpoint struct {
	now func() time.Time
}

type Spec struct {
	URL                   string             `json:"url"`
	Method                string             `json:"method"`
	ExpectedStatus        *string            `json:"expectedStatus,omitempty"`
	Condition             *string            `json:"condition,omitempty"`
	IntervalSeconds       *int               `json:"intervalSeconds,omitempty"`
	TimeoutSeconds        *int               `json:"timeoutSeconds,omitempty"`
	RequestTimeoutSeconds *int               `json:"requestTimeoutSeconds,omitempty"`
	MaxResponseBytes      *int               `json:"maxResponseBytes,omitempty"`
	Headers               *[]Header          `json:"headers,omitempty"`
	Authorization         *AuthorizationSpec `json:"authorization,omitempty" mapstructure:"authorization"`
}

type Metadata struct {
	StartedAt     string `json:"startedAt" mapstructure:"startedAt"`
	DeadlineAt    string `json:"deadlineAt" mapstructure:"deadlineAt"`
	Attempts      int    `json:"attempts" mapstructure:"attempts"`
	LastStatus    *int   `json:"lastStatus,omitempty" mapstructure:"lastStatus"`
	LastBody      any    `json:"lastBody,omitempty" mapstructure:"lastBody"`
	LastError     string `json:"lastError,omitempty" mapstructure:"lastError"`
	NextAttemptAt string `json:"nextAttemptAt,omitempty" mapstructure:"nextAttemptAt"`
	ResolvedURL   string `json:"resolvedUrl,omitempty" mapstructure:"resolvedUrl"`
}

type attemptResult struct {
	Status      *int
	Body        any
	LastError   string
	ResolvedURL string
	Ready       bool
}

type responseTooLargeError struct {
	maxBytes int
}

func (e *responseTooLargeError) Error() string {
	return fmt.Sprintf(responseSizeErrorMessage, e.maxBytes)
}

func (s Spec) method() string {
	if s.Method == "" {
		return http.MethodGet
	}
	return strings.ToUpper(s.Method)
}

func (s Spec) expectedStatus() string {
	if s.ExpectedStatus == nil {
		return defaultExpectedStatus
	}
	return *s.ExpectedStatus
}

func (s Spec) interval() time.Duration {
	if s.IntervalSeconds == nil {
		return defaultIntervalSeconds * time.Second
	}
	return time.Duration(*s.IntervalSeconds) * time.Second
}

func (s Spec) timeout() time.Duration {
	if s.TimeoutSeconds == nil {
		return defaultTimeoutSeconds * time.Second
	}
	return time.Duration(*s.TimeoutSeconds) * time.Second
}

func (s Spec) requestTimeout() time.Duration {
	if s.RequestTimeoutSeconds == nil {
		return defaultRequestTimeout * time.Second
	}
	return time.Duration(*s.RequestTimeoutSeconds) * time.Second
}

func (s Spec) maxResponseBytes() int {
	if s.MaxResponseBytes == nil {
		return defaultMaxResponseBytes
	}
	return *s.MaxResponseBytes
}

func (c *WaitForEndpoint) Name() string {
	return "waitForEndpoint"
}

func (c *WaitForEndpoint) Label() string {
	return "Wait for Endpoint"
}

func (c *WaitForEndpoint) Description() string {
	return "Poll an HTTP endpoint until it is ready or the deadline expires"
}

func (c *WaitForEndpoint) Documentation() string {
	return `Wait for Endpoint checks a service after deployment and continues the workflow only when it is ready.

The first check happens immediately. If the endpoint is not ready, SuperPlane schedules durable checks without keeping a worker occupied.

## Readiness

By default, any 2xx response is ready. You can configure exact statuses such as ` + "`200,204`" + ` and an optional condition such as ` + "`body.status == \"ready\"`" + `.

## Outcomes

- **Ready**: the status and optional condition matched.
- **Timeout**: the deadline expired. Connect this channel to rollback, notification, or diagnostics nodes.

Network failures are retried until timeout. Invalid configuration, blocked URLs, missing secrets, oversized responses, and invalid conditions fail the execution.

## Local testing

Start SuperPlane locally and add this component after a Start trigger. Use ` + "`https://httpbin.org/status/200`" + ` for a simple Ready-path check.

To test a service running on your machine, enable private network access under Installation Settings and use an address reachable from the app container, such as ` + "`http://host.docker.internal:8080/ready`" + `. Connect Ready and Timeout to separate downstream nodes so both outcomes are visible.`
}

func (c *WaitForEndpoint) Icon() string {
	return "activity"
}

func (c *WaitForEndpoint) Color() string {
	return "green"
}

func (c *WaitForEndpoint) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ReadyOutputChannel, Label: "Ready", Description: "The endpoint satisfied the readiness criteria"},
		{Name: TimeoutOutputChannel, Label: "Timeout", Description: "The readiness deadline expired"},
	}
}

func (c *WaitForEndpoint) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://service.example.com/ready",
			Description: "Endpoint to check. Supports expressions.",
		},
		{
			Name:     "method",
			Label:    "Method",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  http.MethodGet,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: http.MethodGet, Value: http.MethodGet},
						{Label: http.MethodHead, Value: http.MethodHead},
					},
				},
			},
		},
		{
			Name:        "expectedStatus",
			Label:       "Expected status",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     defaultExpectedStatus,
			Placeholder: "2xx or 200,204",
			Description: "Comma-separated status codes or status classes.",
		},
		{
			Name:        "condition",
			Label:       "Readiness condition",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Togglable:   true,
			Placeholder: `body.status == "ready"`,
			Description: "Optional boolean expression evaluated with status, headers, and body.",
		},
		{
			Name:        "intervalSeconds",
			Label:       "Check interval (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultIntervalSeconds,
			Description: "Seconds between readiness checks.",
			TypeOptions: numberOptions(1, maxIntervalSeconds),
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Overall timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultTimeoutSeconds,
			Description: "Maximum total time to wait for readiness.",
			TypeOptions: numberOptions(1, maxTimeoutSeconds),
		},
		{
			Name:        "requestTimeoutSeconds",
			Label:       "Request timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultRequestTimeout,
			Description: "Maximum duration of each individual request.",
			TypeOptions: numberOptions(1, maxRequestTimeoutSeconds),
		},
		{
			Name:        "maxResponseBytes",
			Label:       "Maximum response size (bytes)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultMaxResponseBytes,
			Description: "Maximum response body retained for condition evaluation and output.",
			TypeOptions: numberOptions(minMaxResponseBytes, maxMaxResponseBytes),
		},
		headersField(),
		AuthorizationField(),
	}
}

func (c *WaitForEndpoint) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	return validateSpec(spec)
}

func (c *WaitForEndpoint) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *WaitForEndpoint) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	startedAt := c.currentTime()
	deadline := startedAt.Add(spec.timeout())
	metadata := Metadata{
		StartedAt:  startedAt.Format(time.RFC3339Nano),
		DeadlineAt: deadline.Format(time.RFC3339Nano),
	}

	return c.checkAndContinue(ctx.HTTP, ctx.Secrets, ctx.Metadata, ctx.ExecutionState, ctx.Requests, spec, metadata, deadline)
}

func (c *WaitForEndpoint) Hooks() []core.Hook {
	return []core.Hook{
		{Name: checkEndpointHook, Type: core.HookTypeInternal},
	}
}

func (c *WaitForEndpoint) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != checkEndpointHook {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	var metadata Metadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode endpoint metadata: %w", err)
	}

	deadline, err := time.Parse(time.RFC3339Nano, metadata.DeadlineAt)
	if err != nil {
		return fmt.Errorf("invalid endpoint deadline: %w", err)
	}

	if !c.currentTime().Before(deadline) {
		return c.emitTimeout(ctx.Metadata, ctx.ExecutionState, spec, metadata)
	}

	return c.checkAndContinue(ctx.HTTP, ctx.Secrets, ctx.Metadata, ctx.ExecutionState, ctx.Requests, spec, metadata, deadline)
}

func (c *WaitForEndpoint) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *WaitForEndpoint) Cancel(core.ExecutionContext) error {
	return nil
}

func (c *WaitForEndpoint) Cleanup(core.SetupContext) error {
	return nil
}

func (c *WaitForEndpoint) checkAndContinue(
	httpCtx core.HTTPContext,
	secrets core.SecretsContext,
	metadataCtx core.MetadataWriter,
	executionState core.ExecutionStateContext,
	requests core.RequestContext,
	spec Spec,
	metadata Metadata,
	deadline time.Time,
) error {
	now := c.currentTime()
	if !now.Before(deadline) {
		return c.emitTimeout(metadataCtx, executionState, spec, metadata)
	}

	attemptTimeout := spec.requestTimeout()
	if remaining := deadline.Sub(now); remaining < attemptTimeout {
		attemptTimeout = remaining
	}

	result, err := c.performAttempt(httpCtx, secrets, spec, attemptTimeout)
	if err != nil {
		return err
	}

	metadata.Attempts++
	metadata.LastStatus = result.Status
	metadata.LastBody = result.Body
	metadata.LastError = result.LastError
	metadata.ResolvedURL = result.ResolvedURL
	metadata.NextAttemptAt = ""

	if result.Ready {
		if err := metadataCtx.Set(metadata); err != nil {
			return fmt.Errorf("failed to set endpoint metadata: %w", err)
		}
		return c.emitReady(executionState, spec, metadata)
	}

	now = c.currentTime()
	if !now.Before(deadline) {
		return c.emitTimeout(metadataCtx, executionState, spec, metadata)
	}

	delay := spec.interval()
	if remaining := deadline.Sub(now); remaining < delay {
		delay = remaining
	}
	if delay < time.Second {
		delay = time.Second
	}
	metadata.NextAttemptAt = now.Add(delay).Format(time.RFC3339Nano)

	if err := metadataCtx.Set(metadata); err != nil {
		return fmt.Errorf("failed to set endpoint metadata: %w", err)
	}

	return requests.ScheduleActionCall(checkEndpointHook, map[string]any{}, delay)
}

func (c *WaitForEndpoint) performAttempt(
	httpCtx core.HTTPContext,
	secrets core.SecretsContext,
	spec Spec,
	timeout time.Duration,
) (attemptResult, error) {
	requestContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestContext, spec.method(), spec.URL, nil)
	if err != nil {
		return attemptResult{}, fmt.Errorf("failed to create endpoint request: %w", err)
	}

	if spec.Headers != nil {
		for _, header := range *spec.Headers {
			request.Header.Set(header.Name, header.Value)
		}
	}

	sensitiveHeader, err := ApplyAuthorization(secrets, spec.Authorization, request)
	if err != nil {
		return attemptResult{}, err
	}
	request = registry.WithSensitiveHeaders(request, sensitiveHeader)

	response, err := httpCtx.Do(request)
	if err != nil {
		var policyErr *registry.HTTPPolicyError
		if errors.As(err, &policyErr) {
			return attemptResult{}, err
		}
		var responseTooLargeErr *registry.HTTPResponseTooLargeError
		if errors.As(err, &responseTooLargeErr) {
			return attemptResult{}, err
		}
		if errors.Is(requestContext.Err(), context.DeadlineExceeded) {
			return attemptResult{LastError: fmt.Sprintf("request timed out after %s", timeout)}, nil
		}
		return attemptResult{LastError: err.Error()}, nil
	}
	defer response.Body.Close()

	body, err := readResponseBody(response.Body, spec.maxResponseBytes())
	if err != nil {
		var tooLargeErr *responseTooLargeError
		if errors.As(err, &tooLargeErr) {
			return attemptResult{}, err
		}
		if errors.Is(requestContext.Err(), context.DeadlineExceeded) {
			return attemptResult{LastError: fmt.Sprintf("request timed out after %s", timeout)}, nil
		}
		return attemptResult{LastError: err.Error()}, nil
	}

	parsedBody := parseBody(body)
	status := response.StatusCode
	result := attemptResult{
		Status:      &status,
		Body:        parsedBody,
		ResolvedURL: spec.URL,
	}
	if response.Request != nil && response.Request.URL != nil {
		result.ResolvedURL = response.Request.URL.String()
	}

	if !MatchesStatus(response.StatusCode, spec.expectedStatus()) {
		return result, nil
	}

	if spec.Condition != nil && strings.TrimSpace(*spec.Condition) != "" {
		ready, err := evaluateCondition(*spec.Condition, response.StatusCode, response.Header, parsedBody)
		if err != nil {
			return attemptResult{}, err
		}
		result.Ready = ready
		return result, nil
	}

	result.Ready = true
	return result, nil
}

func (c *WaitForEndpoint) emitReady(executionState core.ExecutionStateContext, spec Spec, metadata Metadata) error {
	finishedAt := c.currentTime()
	payload := map[string]any{
		"url":         spec.URL,
		"resolvedUrl": metadata.ResolvedURL,
		"status":      metadata.LastStatus,
		"body":        metadata.LastBody,
		"attempts":    metadata.Attempts,
		"startedAt":   metadata.StartedAt,
		"finishedAt":  finishedAt.Format(time.RFC3339Nano),
		"elapsedMs":   elapsedMilliseconds(metadata.StartedAt, finishedAt),
	}
	fitPayloadBody(ReadyPayloadType, payload, "body")
	return executionState.Emit(
		ReadyOutputChannel,
		ReadyPayloadType,
		[]any{payload},
	)
}

func (c *WaitForEndpoint) emitTimeout(
	metadataCtx core.MetadataWriter,
	executionState core.ExecutionStateContext,
	spec Spec,
	metadata Metadata,
) error {
	metadata.NextAttemptAt = ""
	if err := metadataCtx.Set(metadata); err != nil {
		return fmt.Errorf("failed to set endpoint metadata: %w", err)
	}

	finishedAt := c.currentTime()
	payload := map[string]any{
		"url":        spec.URL,
		"attempts":   metadata.Attempts,
		"startedAt":  metadata.StartedAt,
		"finishedAt": finishedAt.Format(time.RFC3339Nano),
		"elapsedMs":  elapsedMilliseconds(metadata.StartedAt, finishedAt),
		"lastStatus": metadata.LastStatus,
		"lastBody":   metadata.LastBody,
		"lastError":  metadata.LastError,
		"reason":     "deadline_exceeded",
	}
	fitPayloadBody(TimeoutPayloadType, payload, "lastBody")
	return executionState.Emit(
		TimeoutOutputChannel,
		TimeoutPayloadType,
		[]any{payload},
	)
}

func fitPayloadBody(payloadType string, payload map[string]any, bodyKey string) {
	maxPayloadSize := config.MaxPayloadSize()
	if payloadFits(payloadType, payload, maxPayloadSize) {
		return
	}

	body, err := json.Marshal(payload[bodyKey])
	if err != nil {
		payload[bodyKey] = "[response body omitted: cannot serialize]"
		return
	}

	const suffix = "...[truncated]"
	low, high := 0, len(body)
	for low < high {
		mid := (low + high + 1) / 2
		payload[bodyKey] = string(body[:mid]) + suffix
		if payloadFits(payloadType, payload, maxPayloadSize) {
			low = mid
		} else {
			high = mid - 1
		}
	}
	payload[bodyKey] = string(body[:low]) + suffix
	if !payloadFits(payloadType, payload, maxPayloadSize) {
		payload[bodyKey] = "[response body omitted: event payload limit exceeded]"
	}
}

func payloadFits(payloadType string, payload map[string]any, maxPayloadSize int) bool {
	event := map[string]any{
		"type":      payloadType,
		"timestamp": time.Now(),
		"data":      payload,
	}
	data, err := json.Marshal(event)
	return err == nil && len(data) <= maxPayloadSize
}

func (c *WaitForEndpoint) currentTime() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func decodeSpec(configurationValue any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(configurationValue, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode endpoint configuration: %w", err)
	}
	return spec, nil
}

func validateSpec(spec Spec) error {
	if spec.URL == "" {
		return fmt.Errorf("url is required")
	}

	parsedURL, err := url.Parse(spec.URL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("url must use http or https")
	}
	if parsedURL.Hostname() == "" {
		return fmt.Errorf("url must have a host")
	}

	if spec.method() != http.MethodGet && spec.method() != http.MethodHead {
		return fmt.Errorf("method must be GET or HEAD")
	}
	if err := ValidateStatusMatcher(spec.expectedStatus()); err != nil {
		return err
	}
	if err := ValidateAuthorization(spec.Authorization); err != nil {
		return err
	}

	if interval := int(spec.interval().Seconds()); interval < 1 || interval > maxIntervalSeconds {
		return fmt.Errorf("interval seconds must be between 1 and %d", maxIntervalSeconds)
	}
	if timeout := int(spec.timeout().Seconds()); timeout < 1 || timeout > maxTimeoutSeconds {
		return fmt.Errorf("timeout seconds must be between 1 and %d", maxTimeoutSeconds)
	}
	if requestTimeout := int(spec.requestTimeout().Seconds()); requestTimeout < 1 || requestTimeout > maxRequestTimeoutSeconds {
		return fmt.Errorf("request timeout seconds must be between 1 and %d", maxRequestTimeoutSeconds)
	}
	if maxBytes := spec.maxResponseBytes(); maxBytes < minMaxResponseBytes || maxBytes > maxMaxResponseBytes {
		return fmt.Errorf("maximum response bytes must be between %d and %d", minMaxResponseBytes, maxMaxResponseBytes)
	}

	if spec.Condition != nil && strings.TrimSpace(*spec.Condition) != "" {
		if _, err := expr.Compile(*spec.Condition, expr.AllowUndefinedVariables(), expr.AsBool()); err != nil {
			return fmt.Errorf("invalid readiness condition: %w", err)
		}
	}

	return nil
}

func evaluateCondition(condition string, status int, headers http.Header, body any) (bool, error) {
	program, err := expr.Compile(condition, expr.AllowUndefinedVariables())
	if err != nil {
		return false, fmt.Errorf("invalid readiness condition: %w", err)
	}

	value, err := expr.Run(program, map[string]any{
		"status":  status,
		"headers": map[string][]string(headers),
		"body":    body,
	})
	if err != nil {
		return false, fmt.Errorf("failed to evaluate readiness condition: %w", err)
	}

	ready, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf(invalidConditionErrorFormat, value)
	}
	return ready, nil
}

func readResponseBody(body io.Reader, maxBytes int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read endpoint response: %w", err)
	}
	if len(data) > maxBytes {
		return nil, &responseTooLargeError{maxBytes: maxBytes}
	}
	return data, nil
}

func parseBody(data []byte) any {
	if len(data) == 0 {
		return nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err == nil {
		return value
	}
	return string(data)
}

func elapsedMilliseconds(startedAt string, finishedAt time.Time) int64 {
	start, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return 0
	}
	return finishedAt.Sub(start).Milliseconds()
}

func numberOptions(minimum, maximum int) *configuration.TypeOptions {
	return &configuration.TypeOptions{
		Number: &configuration.NumberTypeOptions{
			Min: &minimum,
			Max: &maximum,
		},
	}
}

func headersField() configuration.Field {
	return configuration.Field{
		Name:        "headers",
		Label:       "Headers",
		Type:        configuration.FieldTypeList,
		Required:    false,
		Togglable:   true,
		Description: "Headers included with every readiness request.",
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: "Header",
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeObject,
					Schema: []configuration.Field{
						{
							Name:     "name",
							Label:    "Name",
							Type:     configuration.FieldTypeString,
							Required: true,
						},
						{
							Name:     "value",
							Label:    "Value",
							Type:     configuration.FieldTypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}
