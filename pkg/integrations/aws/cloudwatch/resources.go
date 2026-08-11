package cloudwatch

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListAlarms(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	client, err := clientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	alarms, err := client.ListAlarms()
	if err != nil {
		return nil, fmt.Errorf("failed to list CloudWatch alarms: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(alarms))
	for _, alarm := range alarms {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: alarm.AlarmName,
			ID:   alarm.AlarmName,
		})
	}

	return resources, nil
}

func ListNamespaces(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	client, err := clientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	metrics, err := client.ListMetrics("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to list CloudWatch namespaces: %w", err)
	}

	namespaces := distinct(metrics, func(metric Metric) string { return metric.Namespace })
	resources := make([]core.IntegrationResource, 0, len(namespaces))
	for _, namespace := range namespaces {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: namespace,
			ID:   namespace,
		})
	}

	return resources, nil
}

func ListMetricNames(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	client, err := clientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	namespace := strings.TrimSpace(ctx.Parameters["namespace"])
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	metrics, err := client.ListMetrics(namespace, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list CloudWatch metrics: %w", err)
	}

	names := distinct(metrics, func(metric Metric) string { return metric.MetricName })
	resources := make([]core.IntegrationResource, 0, len(names))
	for _, name := range names {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   name,
		})
	}

	return resources, nil
}

// ListMetricDimensions lists the dimension sets a metric is published with.
// A metric is identified by its namespace, name and full dimension set, so each
// resource carries the whole set, JSON-encoded.
func ListMetricDimensions(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	client, err := clientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	namespace := strings.TrimSpace(ctx.Parameters["namespace"])
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	metricName := strings.TrimSpace(ctx.Parameters["metricName"])
	if metricName == "" {
		return nil, fmt.Errorf("metric name is required")
	}

	metrics, err := client.ListMetrics(namespace, metricName)
	if err != nil {
		return nil, fmt.Errorf("failed to list dimensions for metric %s: %w", metricName, err)
	}

	seen := map[string]bool{}
	resources := []core.IntegrationResource{}
	for _, metric := range metrics {
		if len(metric.Dimensions) == 0 {
			continue
		}

		id, err := EncodeDimensions(metric.Dimensions)
		if err != nil || seen[id] {
			continue
		}

		seen[id] = true
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: FormatDimensions(metric.Dimensions),
			ID:   id,
		})
	}

	sort.Slice(resources, func(i, j int) bool { return resources[i].Name < resources[j].Name })
	return resources, nil
}

// EncodeDimensions serializes a dimension set into the value stored in the
// component configuration. JSON keeps values with separators unambiguous.
func EncodeDimensions(dimensions []Dimension) (string, error) {
	encoded, err := json.Marshal(dimensions)
	if err != nil {
		return "", fmt.Errorf("failed to encode dimensions: %w", err)
	}

	return string(encoded), nil
}

func DecodeDimensions(value string) ([]Dimension, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	dimensions := []Dimension{}
	if err := json.Unmarshal([]byte(value), &dimensions); err != nil {
		return nil, fmt.Errorf("invalid dimensions: %w", err)
	}

	return dimensions, nil
}

func FormatDimensions(dimensions []Dimension) string {
	parts := make([]string, 0, len(dimensions))
	for _, dimension := range dimensions {
		parts = append(parts, fmt.Sprintf("%s=%s", dimension.Name, dimension.Value))
	}

	return strings.Join(parts, ", ")
}

func clientFromContext(ctx core.ListResourcesContext) (*Client, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	return NewClient(ctx.HTTP, creds, region), nil
}

func distinct(metrics []Metric, key func(Metric) string) []string {
	seen := map[string]bool{}
	values := []string{}
	for _, metric := range metrics {
		value := strings.TrimSpace(key(metric))
		if value == "" || seen[value] {
			continue
		}

		seen[value] = true
		values = append(values, value)
	}

	sort.Strings(values)
	return values
}
