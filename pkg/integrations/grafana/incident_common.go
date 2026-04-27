package grafana

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

var grafanaIncidentSeverities = []core.IntegrationResource{
	{Type: resourceTypeIncidentSeverity, Name: "Pending", ID: "pending"},
	{Type: resourceTypeIncidentSeverity, Name: "Critical", ID: "critical"},
	{Type: resourceTypeIncidentSeverity, Name: "Major", ID: "major"},
	{Type: resourceTypeIncidentSeverity, Name: "Minor", ID: "minor"},
}

func incidentResourceField(name, label, description string) configuration.Field {
	return configuration.Field{
		Name:        name,
		Label:       label,
		Type:        configuration.FieldTypeIntegrationResource,
		Required:    true,
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type: resourceTypeIncident,
			},
		},
	}
}

func decodeIncidentSpec[T any](config any) (T, error) {
	var spec T
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return spec, fmt.Errorf("error creating decoder: %v", err)
	}
	if err := decoder.Decode(config); err != nil {
		return spec, fmt.Errorf("error decoding configuration: %v", err)
	}
	return spec, nil
}

func validateIncidentSeverity(severity string, required bool) error {
	severity = strings.TrimSpace(severity)
	if severity == "" {
		if required {
			return errors.New("severity is required")
		}
		return nil
	}

	if isTemplateExpression(severity) {
		return nil
	}

	for _, resource := range grafanaIncidentSeverities {
		if severity == resource.ID {
			return nil
		}
	}

	return fmt.Errorf("severity must be one of: pending, critical, major, minor")
}

func grafanaIncidentSeverityResources() []core.IntegrationResource {
	resources := make([]core.IntegrationResource, len(grafanaIncidentSeverities))
	copy(resources, grafanaIncidentSeverities)
	return resources
}

func validateIncidentSeverityPointer(severity *string) error {
	if severity == nil {
		return nil
	}
	if strings.TrimSpace(*severity) == "" {
		return errors.New("severity must not be empty when provided")
	}
	return validateIncidentSeverity(*severity, false)
}

func validateIncidentRequired(incident string) error {
	if strings.TrimSpace(incident) == "" {
		return errors.New("incident is required")
	}
	return nil
}

func validateUpdateIncidentSpec(spec UpdateIncidentSpec) error {
	if err := validateIncidentRequired(spec.Incident); err != nil {
		return err
	}

	hasUpdate := false
	if spec.Title != nil {
		if strings.TrimSpace(*spec.Title) == "" {
			return errors.New("title must not be empty when provided")
		}
		hasUpdate = true
	}
	if spec.Severity != nil {
		if err := validateIncidentSeverityPointer(spec.Severity); err != nil {
			return err
		}
		hasUpdate = true
	}
	if len(spec.Labels) > 0 {
		if len(incidentLabelsFromStrings(spec.Labels)) == 0 {
			return errors.New("labels must include at least one non-empty label when provided")
		}
		hasUpdate = true
	}
	if spec.IsDrill != nil {
		hasUpdate = true
	}
	if !hasUpdate {
		return errors.New("at least one incident field must be provided")
	}
	return nil
}

func validateAddIncidentActivitySpec(spec AddIncidentActivitySpec) error {
	if err := validateIncidentRequired(spec.Incident); err != nil {
		return err
	}
	if strings.TrimSpace(spec.Body) == "" {
		return errors.New("body is required")
	}
	return nil
}
