package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// CreateMonitorPayloadType is the emitted execution payload type (dash0-style: integration.resource.operation).
const CreateMonitorPayloadType = "cloudflare.monitor.created"

const (
	defaultMonitorIntervalSeconds = 60
	defaultMonitorTimeoutSeconds  = 5
	defaultMonitorRetries         = 0
	minMonitorIntervalSeconds     = 10
	maxMonitorIntervalSeconds     = 86400
	minMonitorTimeoutSeconds      = 1
	maxMonitorRetries             = 5
)

var (
	allowedMonitorTypes   = []string{"http", "https", "tcp", "udp_icmp", "icmp_ping", "smtp"}
	httpMonitorTypes      = []string{"http", "https"}
	portMonitorTypes      = []string{"http", "https", "tcp", "udp_icmp", "smtp"}
	allowedMonitorMethods = []string{"GET", "HEAD"}
)

type CreateMonitor struct{}

type MonitorHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CreateMonitorSpec struct {
	Type        string                     `json:"type"`
	Description string                     `json:"description"`
	Path        string                     `json:"path"`
	Port        *int                       `json:"port"`
	Advanced    *CreateMonitorAdvancedSpec `json:"advanced"`
	Pool        string                     `json:"pool"`
	// Flat timing/threshold fields from legacy workflows or alongside partial `advanced`
	// objects when nested pointers are unset.
	Interval        *int `json:"interval,omitempty"`
	Timeout         *int `json:"timeout,omitempty"`
	Retries         *int `json:"retries,omitempty"`
	ConsecutiveUp   *int `json:"consecutiveUp,omitempty"`
	ConsecutiveDown *int `json:"consecutiveDown,omitempty"`
}

type CreateMonitorAdvancedSpec struct {
	Method          string          `json:"method"`
	ExpectedCodes   string          `json:"expectedCodes"`
	ExpectedBody    string          `json:"expectedBody"`
	Headers         []MonitorHeader `json:"headers"`
	FollowRedirects *bool           `json:"followRedirects"`
	AllowInsecure   *bool           `json:"allowInsecure"`
	ProbeZone       string          `json:"probeZone"`
	Interval        *int            `json:"interval"`
	Timeout         *int            `json:"timeout"`
	Retries         *int            `json:"retries"`
	ConsecutiveUp   *int            `json:"consecutiveUp"`
	ConsecutiveDown *int            `json:"consecutiveDown"`
}

func (c *CreateMonitor) Name() string {
	return "cloudflare.createMonitor"
}

func (c *CreateMonitor) Label() string {
	return "Create Monitor"
}

func (c *CreateMonitor) Description() string {
	return "Create a Cloudflare load balancing health monitor"
}

func (c *CreateMonitor) Documentation() string {
	return `The Create Monitor component creates a Cloudflare Load Balancing health monitor.

## Use Cases

- **Health checks**: Monitor HTTP, HTTPS, TCP, ICMP, UDP, or SMTP endpoints
- **Failover**: Attach the new monitor to a pool so unhealthy origins are removed from load balancer rotation
- **Release safety**: Create monitor definitions as part of load balancing setup automation

## Configuration

- **Name / Type / Path / Port**: Basic monitor settings. Path is required for HTTP and HTTPS monitors. Port is required for HTTP, HTTPS, TCP, UDP ICMP, and SMTP monitors.
- **Advanced Health Check Settings**: Optional method, expected response, headers, redirects, TLS, probe zone, **interval** and **timeout** (seconds; omit both to let Cloudflare pick plan defaults), **retries**, and consecutive thresholds. Plan minimum interval: Pro 60s, Business 15s, Enterprise 10s.
- **Pool**: Optional pool to attach the created monitor to immediately.

## Output

Emits the created monitor and, when configured, the attached pool.`
}

func (c *CreateMonitor) Icon() string {
	return "activity"
}

func (c *CreateMonitor) Color() string {
	return "orange"
}

func (c *CreateMonitor) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateMonitor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "type",
			Label:    "Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "http",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "HTTP", Value: "http"},
						{Label: "HTTPS", Value: "https"},
						{Label: "TCP", Value: "tcp"},
						{Label: "UDP ICMP", Value: "udp_icmp"},
						{Label: "ICMP Ping", Value: "icmp_ping"},
						{Label: "SMTP", Value: "smtp"},
					},
				},
			},
		},
		{
			Name:        "description",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Human-readable monitor name",
			Placeholder: "Login page monitor",
		},
		{
			Name:        "path",
			Label:       "Path",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Endpoint path to check",
			Default:     "/",
			Placeholder: "/health",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: httpMonitorTypes},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: httpMonitorTypes},
			},
		},
		{
			Name:        "port",
			Label:       "Port",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     80,
			Description: "Port for checks. Required for HTTP, HTTPS, TCP, UDP ICMP, and SMTP.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
					Max: func() *int { max := 65535; return &max }(),
				},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "type", Values: portMonitorTypes},
			},
		},
		{
			Name:        "advanced",
			Label:       "Advanced Health Check Settings",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Optional method, response validation, headers, redirect, TLS, and health threshold settings",
			Default: map[string]any{
				"method":          "GET",
				"expectedCodes":   "2xx",
				"followRedirects": true,
				"allowInsecure":   false,
				"interval":        defaultMonitorIntervalSeconds,
				"timeout":         defaultMonitorTimeoutSeconds,
				"retries":         defaultMonitorRetries,
			},
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: createMonitorAdvancedFields(),
				},
			},
		},
		{
			Name:        "pool",
			Label:       "Attach to Pool",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Optional pool to attach this monitor to after creation",
			Placeholder: "Select a pool",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "pool",
				},
			},
		},
	}
}

func createMonitorAdvancedFields() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "method",
			Label:       "Method",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "GET",
			Description: "HTTP method used for the health check",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET", Value: "GET"},
						{Label: "HEAD", Value: "HEAD"},
					},
				},
			},
		},
		{
			Name:        "expectedCodes",
			Label:       "Expected Codes",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "2xx",
			Description: "Expected HTTP response code or code range",
			Placeholder: "2xx",
		},
		{
			Name:        "expectedBody",
			Label:       "Expected Body",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Case-insensitive response body substring expected by Cloudflare",
		},
		{
			Name:        "headers",
			Label:       "Headers",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "HTTP request headers sent by the monitor",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Header",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "name", Label: "Name", Type: configuration.FieldTypeString, Required: true},
							{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true},
						},
					},
				},
			},
		},
		{
			Name:        "followRedirects",
			Label:       "Follow Redirects",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Whether Cloudflare should follow origin redirects",
		},
		{
			Name:        "allowInsecure",
			Label:       "Allow Insecure HTTPS",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Do not validate the origin certificate for HTTPS checks",
		},
		{
			Name:        "probeZone",
			Label:       "Probe Zone",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Zone to emulate while probing",
			Placeholder: "example.com",
		},
		{
			Name:        "interval",
			Label:       "Interval (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultMonitorIntervalSeconds,
			Description: "Seconds between health checks. Cloudflare minimums by plan: Pro 60s, Business 15s, Enterprise 10s. Values outside your plan often return a vague API error.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := minMonitorIntervalSeconds; return &min }(),
					Max: func() *int { max := maxMonitorIntervalSeconds; return &max }(),
				},
			},
		},
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultMonitorTimeoutSeconds,
			Description: "Seconds before marking a check as failed. Must be less than the interval.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := minMonitorTimeoutSeconds; return &min }(),
				},
			},
		},
		{
			Name:        "retries",
			Label:       "Retries",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultMonitorRetries,
			Description: "Immediate retries after a timeout before marking the origin unhealthy",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 0; return &min }(),
					Max: func() *int { max := maxMonitorRetries; return &max }(),
				},
			},
		},
		{
			Name:        "consecutiveUp",
			Label:       "Consecutive Up",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Passes needed before marking an origin healthy",
			TypeOptions: &configuration.TypeOptions{Number: &configuration.NumberTypeOptions{Min: func() *int { min := 0; return &min }()}},
		},
		{
			Name:        "consecutiveDown",
			Label:       "Consecutive Down",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Failures needed before marking an origin unhealthy",
			TypeOptions: &configuration.TypeOptions{Number: &configuration.NumberTypeOptions{Min: func() *int { min := 0; return &min }()}},
		},
	}
}

func decodeCreateMonitorSpec(configuration any) (CreateMonitorSpec, error) {
	spec := CreateMonitorSpec{}
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		TagName:          "json",
		Result:           &spec,
	})
	if err != nil {
		return spec, fmt.Errorf("error configuring decoder: %w", err)
	}

	if err := dec.Decode(configuration); err != nil {
		return spec, fmt.Errorf("error decoding configuration: %w", err)
	}

	return spec, nil
}

func augmentLoadBalancerMonitorCreateError(err error) error {
	var apiErr *CloudflareAPIError
	if !errors.As(err, &apiErr) {
		return err
	}

	for _, e := range apiErr.Errors {
		msg := strings.ToLower(e.Message)
		if strings.Contains(msg, "interval") && strings.Contains(msg, "not in range") {
			return fmt.Errorf(
				"%w — Cloudflare applies plan-specific minimum intervals for monitors (typically 60s on Pro, 15s on Business, 10s on Enterprise); "+
					"intervals below your plan minimum produce misleading \"not in range\" errors. "+
					"If timing fields are omitted, Cloudflare applies account defaults. "+
					"Confirm Load Balancing is enabled and the integration Account ID matches your balancer account",
				err,
			)
		}
	}

	return err
}

func (c *CreateMonitor) Setup(ctx core.SetupContext) error {
	spec, err := decodeCreateMonitorSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateCreateMonitorSpec(spec); err != nil {
		return err
	}

	poolID := strings.TrimSpace(spec.Pool)
	if poolID == "" {
		return nil
	}

	accountID, err := accountIDForIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	return resolvePoolMetadata(ctx, accountID, poolID)
}

func validateCreateMonitorSpec(spec CreateMonitorSpec) error {
	monitorType := normalizeMonitorType(spec.Type)
	if monitorType == "" {
		return errors.New("type is required")
	}
	if !slices.Contains(allowedMonitorTypes, monitorType) {
		return fmt.Errorf("type must be one of %s", strings.Join(allowedMonitorTypes, ", "))
	}

	if strings.TrimSpace(spec.Description) == "" {
		return errors.New("name is required")
	}

	if slices.Contains(httpMonitorTypes, monitorType) {
		if strings.TrimSpace(spec.Path) == "" {
			return errors.New("path is required for HTTP and HTTPS monitors")
		}
		method := normalizeMonitorMethod(effectiveMonitorAdvanced(spec).Method)
		if method != "" && !slices.Contains(allowedMonitorMethods, method) {
			return fmt.Errorf("method must be one of %s", strings.Join(allowedMonitorMethods, ", "))
		}
	}

	if slices.Contains(portMonitorTypes, monitorType) && spec.Port == nil {
		return fmt.Errorf("port is required for %s monitors", monitorType)
	}

	if spec.Port != nil && (*spec.Port < 1 || *spec.Port > 65535) {
		return errors.New("port must be between 1 and 65535")
	}

	advanced := effectiveMonitorAdvanced(spec)
	fields := []struct {
		name  string
		value *int
		min   int
	}{
		{name: "consecutiveUp", value: advanced.ConsecutiveUp, min: 0},
		{name: "consecutiveDown", value: advanced.ConsecutiveDown, min: 0},
	}

	for _, field := range fields {
		if field.value != nil && *field.value < field.min {
			return fmt.Errorf("%s must be >= %d", field.name, field.min)
		}
	}

	return validateMonitorTiming(effectiveMonitorAdvanced(spec))
}

func (c *CreateMonitor) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCreateMonitorSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateCreateMonitorSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	accountID, err := accountIDForIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	monitor, err := client.CreateMonitor(accountID, createMonitorRequest(spec))
	if err != nil {
		return fmt.Errorf("failed to create monitor: %w", augmentLoadBalancerMonitorCreateError(err))
	}

	payload := map[string]any{
		"accountId": accountID,
		"monitor":   monitor,
		"monitorId": monitor.ID,
	}

	if strings.TrimSpace(spec.Pool) != "" {
		pool, err := client.PatchPoolMonitor(accountID, spec.Pool, monitor.ID)
		if err != nil {
			return fmt.Errorf("failed to attach monitor to pool: %w", err)
		}
		payload["pool"] = pool
		payload["poolId"] = spec.Pool
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateMonitorPayloadType, []any{payload})
}

func createMonitorRequest(spec CreateMonitorSpec) CreateMonitorRequest {
	monitorType := normalizeMonitorType(spec.Type)
	advanced := effectiveMonitorAdvanced(spec)
	req := CreateMonitorRequest{
		Type:            monitorType,
		Description:     strings.TrimSpace(spec.Description),
		Interval:        advanced.Interval,
		Timeout:         advanced.Timeout,
		Retries:         advanced.Retries,
		ConsecutiveUp:   advanced.ConsecutiveUp,
		ConsecutiveDown: advanced.ConsecutiveDown,
	}

	if slices.Contains(portMonitorTypes, monitorType) {
		req.Port = spec.Port
	}

	if slices.Contains(httpMonitorTypes, monitorType) {
		req.Method = normalizeMonitorMethod(advanced.Method)
		if req.Method == "" {
			req.Method = "GET"
		}
		req.Path = strings.TrimSpace(spec.Path)
		if req.Path == "" {
			req.Path = "/"
		}
		req.ExpectedCodes = strings.TrimSpace(advanced.ExpectedCodes)
		if req.ExpectedCodes == "" {
			req.ExpectedCodes = "2xx"
		}
		req.ExpectedBody = strings.TrimSpace(advanced.ExpectedBody)
		req.FollowRedirects = advanced.FollowRedirects
		if req.FollowRedirects == nil {
			followRedirects := true
			req.FollowRedirects = &followRedirects
		}
		req.ProbeZone = strings.TrimSpace(advanced.ProbeZone)
		req.Header = monitorHeadersToMap(advanced.Headers)
	}

	if monitorType == "https" {
		req.AllowInsecure = advanced.AllowInsecure
	}

	return req
}

func effectiveMonitorAdvanced(spec CreateMonitorSpec) CreateMonitorAdvancedSpec {
	var adv CreateMonitorAdvancedSpec
	if spec.Advanced != nil {
		adv = *spec.Advanced
	}
	if adv.Interval == nil {
		adv.Interval = spec.Interval
	}
	if adv.Timeout == nil {
		adv.Timeout = spec.Timeout
	}
	if adv.Retries == nil {
		adv.Retries = spec.Retries
	}
	if adv.ConsecutiveUp == nil {
		adv.ConsecutiveUp = spec.ConsecutiveUp
	}
	if adv.ConsecutiveDown == nil {
		adv.ConsecutiveDown = spec.ConsecutiveDown
	}
	return adv
}

func resolvedMonitorTiming(advanced CreateMonitorAdvancedSpec) (interval int, timeout int, retries int) {
	interval = defaultMonitorIntervalSeconds
	if advanced.Interval != nil {
		interval = *advanced.Interval
	}

	timeout = defaultMonitorTimeoutSeconds
	if advanced.Timeout != nil {
		timeout = *advanced.Timeout
	}

	retries = defaultMonitorRetries
	if advanced.Retries != nil {
		retries = *advanced.Retries
	}

	return interval, timeout, retries
}

// resolvedMonitorTimingForUpdate overlays advanced timing fields onto the current monitor (or
// create-style defaults when current is nil), matching mergeMonitorUpdate behavior.
func resolvedMonitorTimingForUpdate(advanced CreateMonitorAdvancedSpec, current *Monitor) (interval int, timeout int, retries int) {
	interval = defaultMonitorIntervalSeconds
	timeout = defaultMonitorTimeoutSeconds
	retries = defaultMonitorRetries
	if current != nil {
		if current.Interval != nil {
			interval = *current.Interval
		}
		if current.Timeout != nil {
			timeout = *current.Timeout
		}
		if current.Retries != nil {
			retries = *current.Retries
		}
	}
	if advanced.Interval != nil {
		interval = *advanced.Interval
	}
	if advanced.Timeout != nil {
		timeout = *advanced.Timeout
	}
	if advanced.Retries != nil {
		retries = *advanced.Retries
	}
	return interval, timeout, retries
}

func validateMonitorTimingFields(advanced CreateMonitorAdvancedSpec) error {
	if advanced.Interval != nil {
		if *advanced.Interval < minMonitorIntervalSeconds {
			return fmt.Errorf("interval must be at least %d seconds", minMonitorIntervalSeconds)
		}
		if *advanced.Interval > maxMonitorIntervalSeconds {
			return fmt.Errorf("interval must be at most %d seconds", maxMonitorIntervalSeconds)
		}
	}

	if advanced.Timeout != nil && *advanced.Timeout < minMonitorTimeoutSeconds {
		return fmt.Errorf("timeout must be at least %d second(s)", minMonitorTimeoutSeconds)
	}

	if advanced.Retries != nil && (*advanced.Retries < 0 || *advanced.Retries > maxMonitorRetries) {
		return fmt.Errorf("retries must be between 0 and %d", maxMonitorRetries)
	}

	return nil
}

func validateMonitorTimeoutLessThanInterval(interval, timeout int) error {
	if timeout >= interval {
		return fmt.Errorf("timeout (%ds) must be less than interval (%ds)", timeout, interval)
	}
	return nil
}

func validateMonitorTiming(advanced CreateMonitorAdvancedSpec) error {
	if err := validateMonitorTimingFields(advanced); err != nil {
		return err
	}

	interval, timeout, _ := resolvedMonitorTiming(advanced)
	return validateMonitorTimeoutLessThanInterval(interval, timeout)
}

// validateMonitorTimingForUpdate validates advanced timing for monitor updates. When current is nil
// (e.g. setup without integration context), the interval/timeout relationship is only checked if both
// are explicitly set in advanced, since unresolved fields will keep the existing monitor values.
func validateMonitorTimingForUpdate(advanced CreateMonitorAdvancedSpec, current *Monitor) error {
	if err := validateMonitorTimingFields(advanced); err != nil {
		return err
	}

	if current == nil {
		if advanced.Interval != nil && advanced.Timeout != nil {
			return validateMonitorTimeoutLessThanInterval(*advanced.Interval, *advanced.Timeout)
		}
		return nil
	}

	interval, timeout, _ := resolvedMonitorTimingForUpdate(advanced, current)
	return validateMonitorTimeoutLessThanInterval(interval, timeout)
}

func monitorHeadersToMap(headers []MonitorHeader) map[string][]string {
	result := map[string][]string{}
	for _, header := range headers {
		name := strings.TrimSpace(header.Name)
		value := strings.TrimSpace(header.Value)
		if name == "" || value == "" {
			continue
		}
		result[name] = append(result[name], value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeMonitorType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeMonitorMethod(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func (c *CreateMonitor) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateMonitor) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateMonitor) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateMonitor) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateMonitor) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateMonitor) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
