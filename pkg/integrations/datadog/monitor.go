package datadog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	intakeMonitorNamePrefix       = "SuperPlane "
	intakeMonitorMention          = "@webhook-" + IntegrationWebhookName
	errorTrackingMonitorType      = "error-tracking alert"
	MonitorsWriteForbiddenMessage = "Datadog refused the request. The application key needs monitors_write."
)

type Monitor struct {
	ID      int64          `json:"id,omitempty"`
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Query   string         `json:"query"`
	Message string         `json:"message"`
	Options MonitorOptions `json:"options"`
}

type MonitorOptions struct {
	GroupbySimpleMonitor bool              `json:"groupby_simple_monitor"`
	NewHostDelay         int               `json:"new_host_delay"`
	Thresholds           MonitorThresholds `json:"thresholds"`
}

type MonitorThresholds struct {
	Critical float64 `json:"critical"`
}

func (c *Client) GetMonitor(id int64) (*Monitor, error) {
	responseBody, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/monitor/%d", c.BaseURL, id), nil)
	if err != nil {
		return nil, err
	}

	var monitor Monitor
	if err := json.Unmarshal(responseBody, &monitor); err != nil {
		return nil, fmt.Errorf("error unmarshaling monitor: %w", err)
	}
	return &monitor, nil
}

func (c *Client) CreateMonitor(monitor Monitor) (*Monitor, error) {
	body, err := json.Marshal(monitor)
	if err != nil {
		return nil, fmt.Errorf("error marshaling monitor: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/monitor", c.BaseURL), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var created Monitor
	if err := json.Unmarshal(responseBody, &created); err != nil {
		return nil, fmt.Errorf("error unmarshaling monitor: %w", err)
	}
	return &created, nil
}

func (c *Client) UpdateMonitor(id int64, monitor Monitor) (*Monitor, error) {
	body, err := json.Marshal(monitor)
	if err != nil {
		return nil, fmt.Errorf("error marshaling monitor: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/monitor/%d", c.BaseURL, id), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var updated Monitor
	if err := json.Unmarshal(responseBody, &updated); err != nil {
		return nil, fmt.Errorf("error unmarshaling monitor: %w", err)
	}
	return &updated, nil
}

func (c *Client) DeleteMonitor(id int64) error {
	_, err := c.execRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/monitor/%d", c.BaseURL, id), nil)
	if isDatadogNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) SearchMonitors(query string) ([]Monitor, error) {
	endpoint := fmt.Sprintf("%s/api/v1/monitor/search?query=%s", c.BaseURL, url.QueryEscape(query))
	responseBody, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Monitors []Monitor `json:"monitors"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling monitor search: %w", err)
	}
	return response.Monitors, nil
}

func (c *Client) ListMonitors() ([]Monitor, error) {
	var monitors []Monitor
	for page := 0; page < 20; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/monitor?page=%d&page_size=100", c.BaseURL, page)
		responseBody, err := c.execRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}

		var batch []Monitor
		if err := json.Unmarshal(responseBody, &batch); err != nil {
			return nil, fmt.Errorf("error unmarshaling monitors: %w", err)
		}
		monitors = append(monitors, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return monitors, nil
}

func reconcileIntakeMonitor(ctx core.TriggerContext, metadata *OnErrorTrackingAlertMetadata, service string) error {
	err := reconcileIntakeMonitorBody(ctx, metadata, service)
	if err != nil {
		return monitorPermissionError(err)
	}
	return nil
}

func reconcileIntakeMonitorBody(ctx core.TriggerContext, metadata *OnErrorTrackingAlertMetadata, service string) error {
	service = strings.TrimSpace(service)
	if service == "" {
		return releaseOwnedMonitor(ctx, metadata, "", false)
	}
	if ctx.HTTP == nil || ctx.Integration == nil {
		return fmt.Errorf("integration is required to update the Datadog monitor")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	previousID := metadata.MonitorID
	previousService := metadata.MonitorService
	if err := ensureIntakeMonitor(client, metadata, service); err != nil {
		return err
	}

	if previousID == "" || previousService == "" || previousService == service || previousID == metadata.MonitorID {
		return nil
	}

	return deleteMonitorWhenUnused(ctx, client, previousID, previousService, false)
}

func ensureIntakeMonitor(client *Client, metadata *OnErrorTrackingAlertMetadata, service string) error {
	if metadata.MonitorID != "" && metadata.MonitorService == service {
		id, err := strconv.ParseInt(metadata.MonitorID, 10, 64)
		if err == nil {
			monitor, err := client.GetMonitor(id)
			if err == nil {
				return keepCurrentMonitor(client, metadata, service, monitor)
			}
			if !isDatadogNotFound(err) {
				return err
			}
		}
	}

	monitor, err := findIntakeMonitor(client, service)
	if err != nil {
		return err
	}
	if monitor != nil {
		return keepCurrentMonitor(client, metadata, service, monitor)
	}

	created, err := client.CreateMonitor(desiredIntakeMonitor(service))
	if err != nil {
		return err
	}
	metadata.MonitorID = strconv.FormatInt(created.ID, 10)
	metadata.MonitorService = service
	return nil
}

func keepCurrentMonitor(client *Client, metadata *OnErrorTrackingAlertMetadata, service string, monitor *Monitor) error {
	if monitorMatches(*monitor, service) {
		metadata.MonitorID = strconv.FormatInt(monitor.ID, 10)
		metadata.MonitorService = service
		return nil
	}

	updated, err := client.UpdateMonitor(monitor.ID, desiredIntakeMonitor(service))
	if err != nil {
		return err
	}
	id := monitor.ID
	if updated != nil && updated.ID != 0 {
		id = updated.ID
	}
	metadata.MonitorID = strconv.FormatInt(id, 10)
	metadata.MonitorService = service
	return nil
}

func findIntakeMonitor(client *Client, service string) (*Monitor, error) {
	named, err := client.SearchMonitors(fmt.Sprintf("title:\"%s\"", intakeMonitorName(service)))
	if err != nil {
		return nil, err
	}
	for _, candidate := range named {
		if candidate.ID == 0 || candidate.Name != intakeMonitorName(service) {
			continue
		}
		monitor, err := loadMonitor(client, candidate.ID)
		if err != nil {
			return nil, err
		}
		if monitor != nil {
			return monitor, nil
		}
	}

	notified, err := client.SearchMonitors("notification:webhook-superplane")
	if err != nil {
		return nil, err
	}
	for _, candidate := range notified {
		if candidate.ID == 0 {
			continue
		}
		monitor, err := loadMonitor(client, candidate.ID)
		if err != nil {
			return nil, err
		}
		if monitor == nil {
			continue
		}
		if strings.Contains(monitor.Message, intakeMonitorMention) && containsServiceToken(monitor.Query, service) {
			return monitor, nil
		}
	}
	return nil, nil
}

func loadMonitor(client *Client, id int64) (*Monitor, error) {
	monitor, err := client.GetMonitor(id)
	if isDatadogNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return monitor, nil
}

func releaseOwnedMonitor(ctx core.TriggerContext, metadata *OnErrorTrackingAlertMetadata, service string, includeCurrentNode bool) error {
	if metadata.MonitorID == "" {
		metadata.MonitorService = ""
		return nil
	}
	if ctx.HTTP == nil || ctx.Integration == nil {
		return fmt.Errorf("integration is required to update the Datadog monitor")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	ownedService := metadata.MonitorService
	if ownedService == "" {
		ownedService = service
	}
	monitorID := metadata.MonitorID
	if err := deleteMonitorWhenUnused(ctx, client, monitorID, ownedService, includeCurrentNode); err != nil {
		return err
	}
	metadata.MonitorID = ""
	metadata.MonitorService = ""
	return nil
}

// deleteMonitorWhenUnused deletes the monitor when no other trigger on this
// integration still uses the service. During setup the current node already
// has the new service, so a count of zero means the old service is unused.
// During cleanup the current node still has the service, so pass
// includeCurrentNode and delete when the count is one or zero.
func deleteMonitorWhenUnused(ctx core.TriggerContext, client *Client, monitorID string, service string, includeCurrentNode bool) error {
	id, err := strconv.ParseInt(monitorID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid monitor id %q", monitorID)
	}

	count, err := countTriggersForService(ctx, service)
	if err != nil {
		return err
	}

	limit := 0
	if includeCurrentNode {
		limit = 1
	}
	if count > limit {
		return nil
	}

	return client.DeleteMonitor(id)
}

func countTriggersForService(ctx core.TriggerContext, service string) (int, error) {
	if ctx.Integration == nil || strings.TrimSpace(service) == "" {
		return 0, nil
	}

	configurations, err := ctx.Integration.ListNodeConfigurations()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, configuration := range configurations {
		config := OnErrorTrackingAlertConfiguration{}
		if err := mapstructure.Decode(configuration, &config); err != nil {
			continue
		}
		if strings.TrimSpace(config.Service) == service {
			count++
		}
	}
	return count, nil
}

func desiredIntakeMonitor(service string) Monitor {
	return Monitor{
		Name:    intakeMonitorName(service),
		Type:    errorTrackingMonitorType,
		Query:   intakeMonitorQuery(service),
		Message: "A new error tracking issue was detected.\n\n" + intakeMonitorMention,
		Options: MonitorOptions{
			GroupbySimpleMonitor: false,
			NewHostDelay:         0,
			Thresholds:           MonitorThresholds{Critical: 0},
		},
	}
}

func intakeMonitorName(service string) string {
	return intakeMonitorNamePrefix + service
}

func intakeMonitorQuery(service string) string {
	return fmt.Sprintf(
		`error-tracking("service:%s").source("all").new().rollup("count").by("issue.id").last("1d") > 0`,
		escapeMonitorQueryValue(service),
	)
}

func escapeMonitorQueryValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func monitorMatches(monitor Monitor, service string) bool {
	return monitor.Query == intakeMonitorQuery(service) &&
		strings.Contains(monitor.Message, intakeMonitorMention) &&
		!monitor.Options.GroupbySimpleMonitor &&
		monitor.Options.NewHostDelay == 0
}

func deleteIntakeMonitors(client *Client) error {
	monitors, err := client.ListMonitors()
	if err != nil {
		return err
	}

	for _, monitor := range monitors {
		if monitor.ID == 0 || !ownedIntakeMonitor(monitor) {
			continue
		}
		if err := client.DeleteMonitor(monitor.ID); err != nil {
			return err
		}
	}
	return nil
}

func ownedIntakeMonitor(monitor Monitor) bool {
	return strings.HasPrefix(monitor.Name, intakeMonitorNamePrefix) ||
		strings.Contains(monitor.Message, intakeMonitorMention)
}

func monitorPermissionError(err error) error {
	if isDatadogForbidden(err) {
		return errors.New(MonitorsWriteForbiddenMessage)
	}
	return err
}

func isDatadogForbidden(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden
}

func isDatadogNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
