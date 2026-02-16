package dash0

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

type CreateSyntheticCheck struct{}

type CreateSyntheticCheckSpec struct {
	Name            string           `json:"name"`
	URL             string           `json:"url"`
	Method          string           `json:"method"`
	Dataset         string           `json:"dataset"`
	Locations       []string         `json:"locations"`
	Interval        string           `json:"interval"`
	Assertions      *[]AssertionSpec `json:"assertions,omitempty"`
	Headers         *[]Header        `json:"headers,omitempty"`
	Body            *string          `json:"body,omitempty"`
	Strategy        *string          `json:"strategy,omitempty"`
	Retries         *RetrySpec       `json:"retries,omitempty" mapstructure:"retries"`
	FollowRedirects *string          `json:"followRedirects,omitempty"`
	AllowInsecure   *string          `json:"allowInsecure,omitempty"`
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type RetrySpec struct {
	Attempts int    `json:"attempts" mapstructure:"attempts"`
	Delay    string `json:"delay"    mapstructure:"delay"`
}

type AssertionSpec struct {
	Kind     string `json:"kind"     mapstructure:"kind"`
	Severity string `json:"severity" mapstructure:"severity"`

	// status_code fields
	StatusCodeOperator string `json:"statusCodeOperator,omitempty" mapstructure:"statusCodeOperator"`
	StatusCodeValue    string `json:"statusCodeValue,omitempty"    mapstructure:"statusCodeValue"`

	// timing fields
	TimingType     string `json:"timingType,omitempty"     mapstructure:"timingType"`
	TimingOperator string `json:"timingOperator,omitempty" mapstructure:"timingOperator"`
	TimingValue    string `json:"timingValue,omitempty"    mapstructure:"timingValue"`

	// error_type fields
	ErrorTypeValue string `json:"errorTypeValue,omitempty" mapstructure:"errorTypeValue"`

	// ssl_certificate_validity fields
	SSLOperator string `json:"sslOperator,omitempty" mapstructure:"sslOperator"`
	SSLDays     string `json:"sslDays,omitempty"     mapstructure:"sslDays"`

	// response_header fields
	HeaderName     string `json:"headerName,omitempty"     mapstructure:"headerName"`
	HeaderOperator string `json:"headerOperator,omitempty" mapstructure:"headerOperator"`
	HeaderValue    string `json:"headerValue,omitempty"    mapstructure:"headerValue"`

	// json_body fields
	JSONPath     string `json:"jsonPath,omitempty"     mapstructure:"jsonPath"`
	JSONOperator string `json:"jsonOperator,omitempty" mapstructure:"jsonOperator"`
	JSONValue    string `json:"jsonValue,omitempty"    mapstructure:"jsonValue"`

	// text_body fields
	TextOperator string `json:"textOperator,omitempty" mapstructure:"textOperator"`
	TextValue    string `json:"textValue,omitempty"    mapstructure:"textValue"`
}

func (c *CreateSyntheticCheck) Name() string {
	return "dash0.createSyntheticCheck"
}

func (c *CreateSyntheticCheck) Label() string {
	return "Create Synthetic Check"
}

func (c *CreateSyntheticCheck) Description() string {
	return "Create an HTTP synthetic check in Dash0 to monitor endpoint availability and performance"
}

func (c *CreateSyntheticCheck) Documentation() string {
	return `The Create Synthetic Check component creates an HTTP synthetic check in Dash0 to monitor the availability and performance of your endpoints.

## Use Cases

- **Uptime monitoring**: Create checks to monitor API endpoints and websites
- **Performance validation**: Set response time thresholds to catch regressions
- **Deployment verification**: Create synthetic checks after deployments to verify availability
- **Multi-region monitoring**: Monitor endpoints from multiple global locations

## Configuration

### Essential
- **Name**: Display name of the synthetic check
- **URL**: Target URL to monitor
- **Method**: HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD)
- **Locations**: Probe locations (Frankfurt, Oregon, North Virginia, London, Brussels, Melbourne)
- **Interval**: How often the check runs (e.g. 30s, 1m, 5m, 1h, 2d)

### Assertions
Each assertion has a kind, severity (critical or degraded), and kind-specific parameters:
- **Status Code**: Validate the HTTP response status code
- **Timing**: Set thresholds for response, request, SSL, connection, DNS, or total time
- **Error Type**: Detect specific error types (DNS, connection, SSL, timeout)
- **SSL Certificate Validity**: Enforce minimum days until certificate expiration
- **Response Header**: Validate presence or value of a specific response header
- **JSON Body**: Validate JSON response fields using JSONPath expressions
- **Text Body**: Match plain-text response content

### Optional
- **Headers**: Custom HTTP request headers
- **Body**: Request body (for POST/PUT/PATCH)
- **Strategy**: Execution strategy (all locations or round-robin)
- **Retries**: Retry attempts and delay between retries
- **Follow Redirects**: Whether to follow HTTP redirects
- **Allow Insecure**: Skip TLS certificate validation

## Output

Returns the created synthetic check details from the Dash0 API, including the check ID and full configuration.`
}

func (c *CreateSyntheticCheck) Icon() string {
	return "activity"
}

func (c *CreateSyntheticCheck) Color() string {
	return "green"
}

func (c *CreateSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Display name of the synthetic check",
			Placeholder: "Login API health check",
		},
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Target URL to monitor",
			Placeholder: "https://api.example.com/health",
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset to create the synthetic check in",
		},
		{
			Name:     "method",
			Label:    "HTTP Method",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "get",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET", Value: "get"},
						{Label: "POST", Value: "post"},
						{Label: "PUT", Value: "put"},
						{Label: "PATCH", Value: "patch"},
						{Label: "DELETE", Value: "delete"},
						{Label: "HEAD", Value: "head"},
					},
				},
			},
		},
		{
			Name:     "locations",
			Label:    "Locations",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"de-frankfurt"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Frankfurt (DE)", Value: "de-frankfurt"},
						{Label: "Oregon (US)", Value: "us-oregon"},
						{Label: "North Virginia (US)", Value: "us-north-virginia"},
						{Label: "London (UK)", Value: "uk-london"},
						{Label: "Brussels (BE)", Value: "be-brussels"},
						{Label: "Melbourne (AU)", Value: "au-melbourne"},
					},
				},
			},
			Description: "Locations to run the synthetic check from",
		},
		{
			Name:        "interval",
			Label:       "Interval",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "1m",
			Description: "How often the check runs (e.g. 30s, 1m, 5m, 1h, 2d)",
			Placeholder: "1m",
		},
		{
			Name:        "assertions",
			Label:       "Assertions",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Conditions the synthetic check must satisfy. Failed assertions mark the check as critical or degraded.",
			Default:     `[{"kind":"status_code","severity":"critical","statusCodeOperator":"is","statusCodeValue":"200"},{"kind":"timing","severity":"critical","timingType":"response","timingOperator":"lte","timingValue":"5000ms"},{"kind":"timing","severity":"degraded","timingType":"response","timingOperator":"lte","timingValue":"2000ms"}]`,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assertion",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: assertionFieldSchema(),
					},
				},
			},
		},
		{
			Name:      "headers",
			Label:     "Headers",
			Type:      configuration.FieldTypeList,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Header",
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeObject,
					Schema: []configuration.Field{
						{
							Name:               "name",
							Label:              "Name",
							Type:               configuration.FieldTypeString,
							Required:           true,
							DisallowExpression: true,
							Placeholder:        "Content-Type",
						},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "application/json",
							},
						},
					},
				},
			},
			Description: "Custom HTTP request headers",
		},
		{
			Name:        "body",
			Label:       "Request Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Request body payload",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"post", "put", "patch"}},
			},
		},
		{
			Name:      "strategy",
			Label:     "Execution Strategy",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			Default:   "all_locations",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "All locations", Value: "all_locations"},
						{Label: "Round-robin", Value: "round_robin"},
					},
				},
			},
			Description: "How checks are distributed across locations",
		},
		{
			Name:        "retries",
			Label:       "Retries",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Default:     map[string]any{"attempts": 3, "delay": "1s"},
			Description: "Retry configuration for failed checks",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "attempts",
							Label:       "Attempts",
							Type:        configuration.FieldTypeNumber,
							Required:    true,
							Default:     "3",
							Description: "Number of retry attempts on failure",
						},
						{
							Name:        "delay",
							Label:       "Delay",
							Type:        configuration.FieldTypeString,
							Required:    true,
							Default:     "1s",
							Description: "Delay between retries (e.g. 1s, 2s, 5s)",
							Placeholder: "1s",
						},
					},
				},
			},
		},
		{
			Name:      "followRedirects",
			Label:     "Follow Redirects",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			Default:   "follow",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Follow", Value: "follow"},
						{Label: "Do not follow", Value: "do_not_follow"},
					},
				},
			},
			Description: "Whether to follow HTTP redirects",
		},
		{
			Name:      "allowInsecure",
			Label:     "Allow Insecure TLS",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			Default:   "false",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "No", Value: "false"},
						{Label: "Yes", Value: "true"},
					},
				},
			},
			Description: "Skip TLS certificate validation (useful for staging environments)",
		},
	}
}

func (c *CreateSyntheticCheck) Setup(ctx core.SetupContext) error {
	spec := CreateSyntheticCheckSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.URL == "" {
		return errors.New("url is required")
	}

	if !strings.HasPrefix(spec.URL, "http://") && !strings.HasPrefix(spec.URL, "https://") {
		return errors.New("url must start with http:// or https://")
	}

	if len(spec.Locations) == 0 {
		return errors.New("at least one location is required")
	}

	return nil
}

func (c *CreateSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	spec := CreateSyntheticCheckSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	dataset := spec.Dataset
	if dataset == "" {
		dataset = "default"
	}

	request := c.buildRequest(spec)

	data, err := client.CreateSyntheticCheck(request, dataset)
	if err != nil {
		return fmt.Errorf("failed to create synthetic check: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.syntheticCheck.created",
		[]any{data},
	)
}

func (c *CreateSyntheticCheck) buildRequest(spec CreateSyntheticCheckSpec) SyntheticCheckRequest {
	method := spec.Method
	if method == "" {
		method = "get"
	}

	redirects := "follow"
	if spec.FollowRedirects != nil {
		redirects = *spec.FollowRedirects
	}

	allowInsecure := false
	if spec.AllowInsecure != nil && *spec.AllowInsecure == "true" {
		allowInsecure = true
	}

	strategy := "all_locations"
	if spec.Strategy != nil {
		strategy = *spec.Strategy
	}

	interval := spec.Interval
	if interval == "" {
		interval = "1m"
	}

	retryAttempts := 3
	retryDelay := "1s"
	if spec.Retries != nil {
		if spec.Retries.Attempts > 0 {
			retryAttempts = spec.Retries.Attempts
		}
		if spec.Retries.Delay != "" {
			retryDelay = spec.Retries.Delay
		}
	}

	// Build headers
	headers := make([]SyntheticCheckHeader, 0)
	if spec.Headers != nil {
		for _, h := range *spec.Headers {
			headers = append(headers, SyntheticCheckHeader{
				Name:  h.Name,
				Value: h.Value,
			})
		}
	}

	// Build assertions
	assertions := c.buildAssertions(spec)

	return SyntheticCheckRequest{
		Kind: "Dash0SyntheticCheck",
		Metadata: SyntheticCheckMetadata{
			Name:   strings.ReplaceAll(strings.ToLower(spec.Name), " ", "-"),
			Labels: map[string]any{},
		},
		Spec: SyntheticCheckTopLevelSpec{
			Enabled: true,
			Schedule: SyntheticCheckSchedule{
				Interval:  interval,
				Locations: spec.Locations,
				Strategy:  strategy,
			},
			Plugin: SyntheticCheckPlugin{
				Display: SyntheticCheckDisplay{
					Name: spec.Name,
				},
				Kind: "http",
				Spec: SyntheticCheckPluginSpec{
					Request: SyntheticCheckHTTPRequest{
						Method:          method,
						URL:             spec.URL,
						Headers:         headers,
						QueryParameters: make([]any, 0),
						Body:            spec.Body,
						Redirects:       redirects,
						TLS: SyntheticCheckTLS{
							AllowInsecure: allowInsecure,
						},
						Tracing: SyntheticCheckTracing{
							AddTracingHeaders: true,
						},
					},
					Assertions: assertions,
					Retries: SyntheticCheckRetries{
						Kind: "fixed",
						Spec: SyntheticCheckRetriesSpec{
							Attempts: retryAttempts,
							Delay:    retryDelay,
						},
					},
				},
			},
		},
	}
}

func assertionFieldSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "kind",
			Label:    "Kind",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "status_code",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Status Code", Value: "status_code"},
						{Label: "Timing", Value: "timing"},
						{Label: "Error Type", Value: "error_type"},
						{Label: "SSL Certificate Validity", Value: "ssl_certificate_validity"},
						{Label: "Response Header", Value: "response_header"},
						{Label: "JSON Body", Value: "json_body"},
						{Label: "Text Body", Value: "text_body"},
					},
				},
			},
		},
		{
			Name:     "severity",
			Label:    "Severity",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "critical",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Critical", Value: "critical"},
						{Label: "Degraded", Value: "degraded"},
					},
				},
			},
		},

		// status_code fields
		{
			Name:    "statusCodeOperator",
			Label:   "Operator",
			Type:    configuration.FieldTypeSelect,
			Default: "is",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Is", Value: "is"},
						{Label: "Is not", Value: "is_not"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"status_code"}},
			},
		},
		{
			Name:        "statusCodeValue",
			Label:       "Status Code",
			Type:        configuration.FieldTypeString,
			Placeholder: "200",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"status_code"}},
			},
		},

		// timing fields
		{
			Name:    "timingType",
			Label:   "Timing Phase",
			Type:    configuration.FieldTypeSelect,
			Default: "response",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Response", Value: "response"},
						{Label: "Request", Value: "request"},
						{Label: "SSL", Value: "ssl"},
						{Label: "Connection", Value: "connection"},
						{Label: "DNS", Value: "dns"},
						{Label: "Total", Value: "total"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"timing"}},
			},
		},
		{
			Name:    "timingOperator",
			Label:   "Operator",
			Type:    configuration.FieldTypeSelect,
			Default: "lte",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "<=", Value: "lte"},
						{Label: ">=", Value: "gte"},
						{Label: "<", Value: "lt"},
						{Label: ">", Value: "gt"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"timing"}},
			},
		},
		{
			Name:        "timingValue",
			Label:       "Duration",
			Type:        configuration.FieldTypeString,
			Placeholder: "5000ms",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"timing"}},
			},
		},

		// error_type fields
		{
			Name:  "errorTypeValue",
			Label: "Error Type",
			Type:  configuration.FieldTypeSelect,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "DNS", Value: "dns"},
						{Label: "Connection", Value: "connection"},
						{Label: "SSL", Value: "ssl"},
						{Label: "Timeout", Value: "timeout"},
						{Label: "Unknown", Value: "unknown"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"error_type"}},
			},
		},

		// ssl_certificate_validity fields
		{
			Name:    "sslOperator",
			Label:   "Operator",
			Type:    configuration.FieldTypeSelect,
			Default: "gte",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: ">=", Value: "gte"},
						{Label: ">", Value: "gt"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"ssl_certificate_validity"}},
			},
		},
		{
			Name:        "sslDays",
			Label:       "Minimum Days Valid",
			Type:        configuration.FieldTypeString,
			Placeholder: "30",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"ssl_certificate_validity"}},
			},
		},

		// response_header fields
		{
			Name:        "headerName",
			Label:       "Header Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "x-auth-token",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"response_header"}},
			},
		},
		{
			Name:    "headerOperator",
			Label:   "Operator",
			Type:    configuration.FieldTypeSelect,
			Default: "is",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Is", Value: "is"},
						{Label: "Is not", Value: "is_not"},
						{Label: "Contains", Value: "contains"},
						{Label: "Does not contain", Value: "not_contains"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"response_header"}},
			},
		},
		{
			Name:        "headerValue",
			Label:       "Header Value",
			Type:        configuration.FieldTypeString,
			Placeholder: "Bearer example-token",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"response_header"}},
			},
		},

		// json_body fields
		{
			Name:        "jsonPath",
			Label:       "JSONPath Expression",
			Type:        configuration.FieldTypeString,
			Placeholder: "$.status",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"json_body"}},
			},
		},
		{
			Name:    "jsonOperator",
			Label:   "Operator",
			Type:    configuration.FieldTypeSelect,
			Default: "is",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Is", Value: "is"},
						{Label: "Is not", Value: "is_not"},
						{Label: "Contains", Value: "contains"},
						{Label: "Does not contain", Value: "not_contains"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"json_body"}},
			},
		},
		{
			Name:        "jsonValue",
			Label:       "Expected Value",
			Type:        configuration.FieldTypeString,
			Placeholder: "200",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"json_body"}},
			},
		},

		// text_body fields
		{
			Name:    "textOperator",
			Label:   "Operator",
			Type:    configuration.FieldTypeSelect,
			Default: "is",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Is", Value: "is"},
						{Label: "Is not", Value: "is_not"},
						{Label: "Contains", Value: "contains"},
						{Label: "Does not contain", Value: "not_contains"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"text_body"}},
			},
		},
		{
			Name:        "textValue",
			Label:       "Expected Text",
			Type:        configuration.FieldTypeString,
			Placeholder: "OK",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"text_body"}},
			},
		},
	}
}

func (c *CreateSyntheticCheck) buildAssertions(spec CreateSyntheticCheckSpec) SyntheticCheckAssertions {
	criticalAssertions := make([]SyntheticCheckAssertion, 0)
	degradedAssertions := make([]SyntheticCheckAssertion, 0)

	if spec.Assertions == nil {
		return SyntheticCheckAssertions{
			CriticalAssertions: criticalAssertions,
			DegradedAssertions: degradedAssertions,
		}
	}

	for _, a := range *spec.Assertions {
		assertion := c.buildSingleAssertion(a)
		if assertion == nil {
			continue
		}

		switch a.Severity {
		case "degraded":
			degradedAssertions = append(degradedAssertions, *assertion)
		default:
			criticalAssertions = append(criticalAssertions, *assertion)
		}
	}

	return SyntheticCheckAssertions{
		CriticalAssertions: criticalAssertions,
		DegradedAssertions: degradedAssertions,
	}
}

func (c *CreateSyntheticCheck) buildSingleAssertion(a AssertionSpec) *SyntheticCheckAssertion {
	switch a.Kind {
	case "status_code":
		return &SyntheticCheckAssertion{
			Kind: "status_code",
			Spec: map[string]any{
				"operator": a.StatusCodeOperator,
				"value":    a.StatusCodeValue,
			},
		}

	case "timing":
		return &SyntheticCheckAssertion{
			Kind: "timing",
			Spec: map[string]any{
				"type":     a.TimingType,
				"operator": a.TimingOperator,
				"value":    a.TimingValue,
			},
		}

	case "error_type":
		return &SyntheticCheckAssertion{
			Kind: "error_type",
			Spec: map[string]any{
				"value": a.ErrorTypeValue,
			},
		}

	case "ssl_certificate_validity":
		return &SyntheticCheckAssertion{
			Kind: "ssl_certificate_validity",
			Spec: map[string]any{
				"operator": a.SSLOperator,
				"value":    a.SSLDays,
			},
		}

	case "response_header":
		return &SyntheticCheckAssertion{
			Kind: "response_header",
			Spec: map[string]any{
				"name":     a.HeaderName,
				"operator": a.HeaderOperator,
				"value":    a.HeaderValue,
			},
		}

	case "json_body":
		return &SyntheticCheckAssertion{
			Kind: "json_body",
			Spec: map[string]any{
				"expression": a.JSONPath,
				"operator":   a.JSONOperator,
				"value":      a.JSONValue,
			},
		}

	case "text_body":
		return &SyntheticCheckAssertion{
			Kind: "text_body",
			Spec: map[string]any{
				"operator": a.TextOperator,
				"value":    a.TextValue,
			},
		}
	}

	return nil
}

func (c *CreateSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateSyntheticCheck) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateSyntheticCheck) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
