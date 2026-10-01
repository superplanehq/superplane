package datadog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	intakeMonitorNamePrefix       = "SuperPlane "
	intakeMonitorMention          = "@webhook-" + IntegrationWebhookName
	intakeMonitorTagKey           = "superplane_integration"
	errorTrackingMonitorType      = "error-tracking alert"
	monitorPageSize               = 100
	MonitorsWriteForbiddenMessage = "Datadog refused the request. The application key needs monitors_write."
)

type Monitor struct {
	ID      int64          `json:"id,omitempty"`
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Query   string         `json:"query"`
	Message string         `json:"message"`
	Tags    []string       `json:"tags,omitempty"`
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
	var monitors []Monitor
	var previousFirstID int64
	seenPage := false
	for page := 0; ; page++ {
		endpoint := fmt.Sprintf(
			"%s/api/v1/monitor/search?query=%s&page=%d&per_page=%d",
			c.BaseURL,
			url.QueryEscape(query),
			page,
			monitorPageSize,
		)
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
		batch := response.Monitors
		if len(batch) == 0 {
			return monitors, nil
		}
		if seenPage && batch[0].ID == previousFirstID {
			return nil, fmt.Errorf("datadog monitor search repeated page %d", page)
		}
		seenPage = true
		previousFirstID = batch[0].ID
		monitors = append(monitors, batch...)
		if len(batch) < monitorPageSize {
			return monitors, nil
		}
	}
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

	integrationID := ctx.Integration.ID().String()
	previousID := metadata.MonitorID
	previousService := metadata.MonitorService
	if err := ensureIntakeMonitor(client, metadata, service, integrationID); err != nil {
		return err
	}

	if previousID == "" || previousService == "" || previousService == service || previousID == metadata.MonitorID {
		return nil
	}

	return deleteOwnedMonitorWhenUnused(ctx, client, previousID, previousService, integrationID, false)
}

func ensureIntakeMonitor(client *Client, metadata *OnErrorTrackingAlertMetadata, service, integrationID string) error {
	if metadata.MonitorID != "" && metadata.MonitorService == service {
		monitor, owned, err := loadOwnedMonitor(client, metadata.MonitorID, service, integrationID)
		if err != nil {
			return err
		}
		if owned {
			return keepCurrentMonitor(client, metadata, service, integrationID, monitor)
		}
	}

	monitor, err := findIntakeMonitor(client, service, integrationID)
	if err != nil {
		return err
	}
	if monitor != nil {
		return keepCurrentMonitor(client, metadata, service, integrationID, monitor)
	}

	created, err := client.CreateMonitor(desiredIntakeMonitor(service, integrationID))
	if err != nil {
		return err
	}
	metadata.MonitorID = strconv.FormatInt(created.ID, 10)
	metadata.MonitorService = service
	return nil
}

func keepCurrentMonitor(client *Client, metadata *OnErrorTrackingAlertMetadata, service, integrationID string, monitor *Monitor) error {
	if monitorMatches(*monitor, service) {
		metadata.MonitorID = strconv.FormatInt(monitor.ID, 10)
		metadata.MonitorService = service
		return nil
	}

	updated, err := client.UpdateMonitor(monitor.ID, desiredIntakeMonitor(service, integrationID))
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

func findIntakeMonitor(client *Client, service, integrationID string) (*Monitor, error) {
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
		if monitor != nil && monitorOwnedBy(*monitor, integrationID, service) {
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
	if err := deleteOwnedMonitorWhenUnused(ctx, client, monitorID, ownedService, ctx.Integration.ID().String(), includeCurrentNode); err != nil {
		return err
	}
	metadata.MonitorID = ""
	metadata.MonitorService = ""
	return nil
}

// ReleaseRetiredIntakeMonitor deletes the monitor stored on an intake that
// was already removed. The intake is no longer in the active count.
func ReleaseRetiredIntakeMonitor(ctx core.TriggerContext) error {
	metadata := OnErrorTrackingAlertMetadata{}
	if ctx.Metadata != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
			return fmt.Errorf("failed to decode metadata: %w", err)
		}
	}

	config := OnErrorTrackingAlertConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return monitorPermissionError(releaseOwnedMonitor(ctx, &metadata, config.Service, false))
}

// deleteOwnedMonitorWhenUnused deletes a SuperPlane monitor when no other
// trigger on this integration still uses the service. During setup the
// current node already has the new service, so a count of zero means the
// old service is unused. During cleanup the current node still has the
// service, so pass includeCurrentNode and delete when the count is one
// or zero. A monitor without the SuperPlane owner tag is left in place.
func deleteOwnedMonitorWhenUnused(ctx core.TriggerContext, client *Client, monitorID, service, integrationID string, includeCurrentNode bool) error {
	monitor, owned, err := loadOwnedMonitor(client, monitorID, service, integrationID)
	if err != nil || !owned {
		return err
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

	return client.DeleteMonitor(monitor.ID)
}

func loadOwnedMonitor(client *Client, monitorID, service, integrationID string) (*Monitor, bool, error) {
	id, err := strconv.ParseInt(monitorID, 10, 64)
	if err != nil {
		return nil, false, nil
	}

	monitor, err := loadMonitor(client, id)
	if err != nil || monitor == nil {
		return nil, false, err
	}
	if !monitorOwnedBy(*monitor, integrationID, service) {
		return nil, false, nil
	}
	return monitor, true, nil
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

func desiredIntakeMonitor(service, integrationID string) Monitor {
	monitor := Monitor{
		Name:    intakeMonitorName(service),
		Type:    errorTrackingMonitorType,
		Query:   intakeMonitorQuery(service),
		Message: generatedIntakeMonitorMessage(),
		Options: MonitorOptions{
			GroupbySimpleMonitor: false,
			NewHostDelay:         0,
			Thresholds:           MonitorThresholds{Critical: 0},
		},
	}
	if tag := intakeOwnerTag(integrationID); tag != "" {
		monitor.Tags = []string{tag}
	}
	return monitor
}

func generatedIntakeMonitorMessage() string {
	return "A new error tracking issue was detected.\n\n" + intakeMonitorMention
}

func intakeOwnerTag(integrationID string) string {
	integrationID = strings.TrimSpace(integrationID)
	if integrationID == "" {
		return ""
	}
	return intakeMonitorTagKey + ":" + integrationID
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

func deleteIntakeMonitors(client *Client, integrationID string) error {
	if strings.TrimSpace(integrationID) == "" {
		return fmt.Errorf("integration is required to delete datadog intake monitors")
	}

	deleted := map[int64]struct{}{}
	if err := deleteMatchingMonitors(client, "tag:"+intakeOwnerTag(integrationID), deleted, func(monitor Monitor) bool {
		return monitorOwnedBy(monitor, integrationID, "")
	}); err != nil {
		return err
	}

	return deleteMatchingMonitors(client, "title:SuperPlane", deleted, exactGeneratedMonitor)
}

func deleteMatchingMonitors(client *Client, query string, deleted map[int64]struct{}, owned func(Monitor) bool) error {
	monitors, err := client.SearchMonitors(query)
	if err != nil {
		return err
	}

	for _, candidate := range monitors {
		if candidate.ID == 0 {
			continue
		}
		if _, alreadyDeleted := deleted[candidate.ID]; alreadyDeleted {
			continue
		}

		monitor, err := loadMonitor(client, candidate.ID)
		if err != nil {
			return err
		}
		if monitor == nil || !owned(*monitor) {
			continue
		}
		if err := client.DeleteMonitor(monitor.ID); err != nil {
			return err
		}
		deleted[monitor.ID] = struct{}{}
	}
	return nil
}

func monitorOwnedBy(monitor Monitor, integrationID, service string) bool {
	if tag := intakeOwnerTag(integrationID); tag != "" && slices.Contains(monitor.Tags, tag) {
		return true
	}
	if strings.TrimSpace(service) == "" {
		return false
	}
	return exactGeneratedMonitorForService(monitor, service)
}

func exactGeneratedMonitor(monitor Monitor) bool {
	service := strings.TrimPrefix(monitor.Name, intakeMonitorNamePrefix)
	if service == "" || service == monitor.Name {
		return false
	}
	return exactGeneratedMonitorForService(monitor, service)
}

func exactGeneratedMonitorForService(monitor Monitor, service string) bool {
	return monitor.Name == intakeMonitorName(service) &&
		monitor.Type == errorTrackingMonitorType &&
		monitor.Query == intakeMonitorQuery(service) &&
		monitor.Message == generatedIntakeMonitorMessage()
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
