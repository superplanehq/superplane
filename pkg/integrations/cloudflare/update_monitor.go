package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UpdateMonitorPayloadType = "cloudflare.monitor.updated"

type UpdateMonitor struct{}

type UpdateMonitorSpec struct {
	Monitor     string                     `json:"monitor"`
	Type        string                     `json:"type"`
	Description string                     `json:"description"`
	Path        *string                    `json:"path,omitempty"`
	Port        *int                       `json:"port,omitempty"`
	Advanced    *CreateMonitorAdvancedSpec `json:"advanced,omitempty"`
}

func (c *UpdateMonitor) Name() string {
	return "cloudflare.updateMonitor"
}

func (c *UpdateMonitor) Label() string {
	return "Update Monitor"
}

func (c *UpdateMonitor) Description() string {
	return "Update a Cloudflare load balancing health monitor's configuration"
}

func (c *UpdateMonitor) Documentation() string {
	return `The Update Monitor component modifies an existing Cloudflare Load Balancing health monitor.

## Use Cases

- **Adjust thresholds**: Change health check intervals, timeouts, or consecutive up/down counts
- **Change protocol**: Switch a monitor from HTTP to HTTPS or TCP
- **Update endpoint**: Modify the path or port being checked
- **Tune headers**: Add or remove request headers sent during health checks

## Configuration

- **Monitor**: The health monitor to update (required)
- **Type / Description / Path / Port**: Override basic monitor settings. Only toggled fields are applied.
- **Advanced Health Check Settings**: Override method, expected response, headers, redirect, TLS, and timing thresholds.

## Output

Returns the updated monitor configuration after the change is applied.`
}

func (c *UpdateMonitor) Icon() string {
	return "activity"
}

func (c *UpdateMonitor) Color() string {
	return "orange"
}

func (c *UpdateMonitor) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateMonitor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "monitor",
			Label:       "Monitor",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The load balancing health monitor to update",
			Placeholder: "Select a monitor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "monitor",
				},
			},
		},
		{
			Name:        "description",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New human-readable name for the monitor",
			Placeholder: "Login page monitor",
		},
		{
			Name:      "type",
			Label:     "Type",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
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
			Name:        "path",
			Label:       "Path",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New endpoint path to check (HTTP and HTTPS only)",
			Placeholder: "/health",
		},
		{
			Name:        "port",
			Label:       "Port",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "New port for health checks",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
					Max: func() *int { max := 65535; return &max }(),
				},
			},
		},
		{
			Name:        "advanced",
			Label:       "Advanced Health Check Settings",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Override method, response validation, headers, redirect, TLS, and health threshold settings",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: createMonitorAdvancedFields(),
				},
			},
		},
	}
}

func (c *UpdateMonitor) Setup(ctx core.SetupContext) error {
	spec := UpdateMonitorSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	monitorID := strings.TrimSpace(spec.Monitor)
	if monitorID == "" {
		return errors.New("monitor is required")
	}

	if spec.Type != "" {
		monitorType := normalizeMonitorType(spec.Type)
		if !isAllowedMonitorType(monitorType) {
			return fmt.Errorf("type must be one of %s", strings.Join(allowedMonitorTypes, ", "))
		}
	}

	if spec.Port != nil && (*spec.Port < 1 || *spec.Port > 65535) {
		return errors.New("port must be between 1 and 65535")
	}

	var preloaded *Monitor
	if spec.Advanced != nil {
		if ctx.HTTP != nil && ctx.Integration != nil && !strings.Contains(monitorID, "{{") {
			client, err := NewClient(ctx.HTTP, ctx.Integration)
			if err != nil {
				return fmt.Errorf("failed to create client for monitor validation: %w", err)
			}
			accountID, err := accountIDForIntegration(ctx.Integration)
			if err != nil {
				return err
			}
			preloaded, err = client.GetMonitor(accountID, monitorID)
			if err != nil {
				return fmt.Errorf("failed to fetch monitor for validation: %w", err)
			}
		}
		if err := validateMonitorTimingForUpdate(*spec.Advanced, preloaded); err != nil {
			return err
		}
	}

	return resolveMonitorMetadata(ctx, monitorID, preloaded)
}

func isAllowedMonitorType(t string) bool {
	for _, allowed := range allowedMonitorTypes {
		if t == allowed {
			return true
		}
	}
	return false
}

func (c *UpdateMonitor) Execute(ctx core.ExecutionContext) error {
	spec := UpdateMonitorSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	monitorID := strings.TrimSpace(spec.Monitor)
	if monitorID == "" {
		return errors.New("monitor is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	accountID, err := accountIDForIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	current, err := client.GetMonitor(accountID, monitorID)
	if err != nil {
		return fmt.Errorf("failed to fetch current monitor: %w", err)
	}

	req := mergeMonitorUpdate(current, spec)

	monitor, err := client.UpdateMonitor(accountID, monitorID, req)
	if err != nil {
		return fmt.Errorf("failed to update monitor: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		UpdateMonitorPayloadType,
		[]any{map[string]any{
			"accountId": accountID,
			"monitor":   monitor,
			"monitorId": monitorID,
		}},
	)
}

// mergeMonitorUpdate overlays user-specified fields onto the current monitor state.
// Cloudflare's monitor update endpoint is PUT (full replacement), so we fetch first.
func mergeMonitorUpdate(current *Monitor, spec UpdateMonitorSpec) CreateMonitorRequest {
	req := CreateMonitorRequest{
		Type:            current.Type,
		Description:     current.Description,
		Method:          current.Method,
		Path:            current.Path,
		ExpectedCodes:   current.ExpectedCodes,
		ExpectedBody:    current.ExpectedBody,
		Header:          current.Header,
		FollowRedirects: current.FollowRedirects,
		AllowInsecure:   current.AllowInsecure,
		ProbeZone:       current.ProbeZone,
		Interval:        current.Interval,
		Timeout:         current.Timeout,
		Retries:         current.Retries,
		ConsecutiveUp:   current.ConsecutiveUp,
		ConsecutiveDown: current.ConsecutiveDown,
		Port:            current.Port,
	}

	if spec.Type != "" {
		req.Type = normalizeMonitorType(spec.Type)
	}
	if spec.Description != "" {
		req.Description = strings.TrimSpace(spec.Description)
	}
	if spec.Path != nil {
		req.Path = strings.TrimSpace(*spec.Path)
	}
	if spec.Port != nil {
		req.Port = spec.Port
	}

	if spec.Advanced != nil {
		adv := spec.Advanced
		if adv.Method != "" {
			req.Method = normalizeMonitorMethod(adv.Method)
		}
		if adv.ExpectedCodes != "" {
			req.ExpectedCodes = strings.TrimSpace(adv.ExpectedCodes)
		}
		if adv.ExpectedBody != "" {
			req.ExpectedBody = strings.TrimSpace(adv.ExpectedBody)
		}
		if len(adv.Headers) > 0 {
			req.Header = monitorHeadersToMap(adv.Headers)
		}
		if adv.FollowRedirects != nil {
			req.FollowRedirects = adv.FollowRedirects
		}
		if adv.AllowInsecure != nil {
			req.AllowInsecure = adv.AllowInsecure
		}
		if adv.ProbeZone != "" {
			req.ProbeZone = strings.TrimSpace(adv.ProbeZone)
		}
		if adv.Interval != nil {
			req.Interval = adv.Interval
		}
		if adv.Timeout != nil {
			req.Timeout = adv.Timeout
		}
		if adv.Retries != nil {
			req.Retries = adv.Retries
		}
		if adv.ConsecutiveUp != nil {
			req.ConsecutiveUp = adv.ConsecutiveUp
		}
		if adv.ConsecutiveDown != nil {
			req.ConsecutiveDown = adv.ConsecutiveDown
		}
	}

	return req
}

func (c *UpdateMonitor) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateMonitor) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateMonitor) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateMonitor) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateMonitor) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateMonitor) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
