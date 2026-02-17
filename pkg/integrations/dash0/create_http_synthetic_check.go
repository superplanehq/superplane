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

type CreateHTTPSyntheticCheck struct{}

type CreateHTTPSyntheticCheckSpec struct {
	Name       string           `mapstructure:"name"`
	Dataset    string           `mapstructure:"dataset"`
	Request    RequestSpec      `mapstructure:"request"`
	Schedule   ScheduleSpec     `mapstructure:"schedule"`
	Assertions *[]AssertionSpec `mapstructure:"assertions"`
	Retries    *RetrySpec       `mapstructure:"retries"`
}

type RequestSpec struct {
	URL           string    `mapstructure:"url"`
	Method        string    `mapstructure:"method"`
	Redirects     string    `mapstructure:"redirects"`
	AllowInsecure string    `mapstructure:"allowInsecure"`
	Headers       *[]Header `mapstructure:"headers"`
	Body          *string   `mapstructure:"body"`
}

type ScheduleSpec struct {
	Strategy  string   `mapstructure:"strategy"`
	Locations []string `mapstructure:"locations"`
	Interval  string   `mapstructure:"interval"`
}

type Header struct {
	Name  string `mapstructure:"name"`
	Value string `mapstructure:"value"`
}

type RetrySpec struct {
	Attempts int    `mapstructure:"attempts"`
	Delay    string `mapstructure:"delay"`
}

type AssertionSpec struct {
	Kind     string `mapstructure:"kind"`
	Severity string `mapstructure:"severity"`

	// Shared fields reused across assertion kinds
	Operator   string `mapstructure:"operator"`   // status_code, timing, ssl, response_header, json_body, text_body
	Value      string `mapstructure:"value"`      // status_code, timing, error_type, ssl, response_header, json_body, text_body
	Type       string `mapstructure:"type"`       // timing (phase: response, request, ssl, connection, dns, total)
	Name       string `mapstructure:"name"`       // response_header (header name)
	Expression string `mapstructure:"expression"` // json_body (JSONPath expression)
}

func (c *CreateHTTPSyntheticCheck) Name() string {
	return "dash0.createHttpSyntheticCheck"
}

func (c *CreateHTTPSyntheticCheck) Label() string {
	return "Create HTTP Synthetic Check"
}

func (c *CreateHTTPSyntheticCheck) Description() string {
	return "Create an HTTP synthetic check in Dash0 to monitor endpoint availability and performance"
}

func (c *CreateHTTPSyntheticCheck) Documentation() string {
	return `The Create Synthetic Check component creates an HTTP synthetic check in Dash0 to monitor the availability and performance of your endpoints.

## Use Cases

- **Uptime monitoring**: Create checks to monitor API endpoints and websites
- **Performance validation**: Set response time thresholds to catch regressions
- **Deployment verification**: Create synthetic checks after deployments to verify availability
- **Multi-region monitoring**: Monitor endpoints from multiple global locations

## Configuration

### Name & Dataset
- **Name**: Display name of the synthetic check
- **Dataset**: The Dash0 dataset to create the check in (defaults to "default")

### Request
- **URL**: Target URL to monitor
- **Method**: HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD)
- **Redirects**: Whether to follow HTTP redirects
- **Allow Insecure**: Skip TLS certificate validation (useful for staging environments)
- **Headers**: Custom HTTP request headers
- **Body**: Request body payload (for POST/PUT/PATCH)

### Schedule
- **Interval**: How often the check runs (e.g. 30s, 1m, 5m, 1h, 2d)
- **Locations**: Probe locations (Frankfurt, Oregon, North Virginia, London, Brussels, Melbourne)
- **Strategy**: Execution strategy (all locations or round-robin)

### Assertions
Each assertion has a kind, severity (critical or degraded), and kind-specific parameters:
- **Status Code**: Validate the HTTP response status code
- **Timing**: Set thresholds for response, request, SSL, connection, DNS, or total time
- **Error Type**: Detect specific error types (DNS, connection, SSL, timeout)
- **SSL Certificate Validity**: Enforce minimum days until certificate expiration
- **Response Header**: Validate presence or value of a specific response header
- **JSON Body**: Validate JSON response fields using JSONPath expressions
- **Text Body**: Match plain-text response content

### Retries
- **Attempts**: Number of retry attempts on failure
- **Delay**: Delay between retries (e.g. 1s, 2s, 5s)

## Output

Returns the created synthetic check details from the Dash0 API, including the check ID and full configuration.`
}

func (c *CreateHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (c *CreateHTTPSyntheticCheck) Color() string {
	return "green"
}

func (c *CreateHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateHTTPSyntheticCheck) Configuration() []configuration.Field {
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
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset to create the synthetic check in",
		},
		{
			Name:        "request",
			Label:       "Request",
			Type:        configuration.FieldTypeObject,
			Required:    true,
			Description: "HTTP request configuration",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "url",
							Label:       "URL",
							Type:        configuration.FieldTypeString,
							Required:    true,
							Description: "Target URL to monitor",
							Placeholder: "https://api.example.com/health",
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
						Name:    "redirects",
						Label:   "Redirects",
						Type:    configuration.FieldTypeSelect,
						Default: "follow",
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
						Name:    "allowInsecure",
						Label:   "Allow Insecure TLS",
						Type:    configuration.FieldTypeSelect,
						Default: "false",
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
					},
				},
			},
		},
		{
			Name:        "schedule",
			Label:       "Schedule",
			Type:        configuration.FieldTypeObject,
			Required:    true,
			Description: "Schedule configuration for the synthetic check",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
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
					},
				},
			},
		},
		{
			Name:        "assertions",
			Label:       "Assertions",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Conditions the synthetic check must satisfy. Failed assertions mark the check as critical or degraded.",
			Default: []map[string]any{
				{"kind": "status_code", "severity": "critical", "operator": "is", "value": "200"},
				{"kind": "timing", "severity": "critical", "type": "response", "operator": "lte", "value": "5000ms"},
				{"kind": "timing", "severity": "degraded", "type": "response", "operator": "lte", "value": "2000ms"},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assertion",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: AssertionFieldSchema(),
					},
				},
			},
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
	}
}

func (c *CreateHTTPSyntheticCheck) Setup(ctx core.SetupContext) error {
	spec := CreateHTTPSyntheticCheckSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.Request.URL == "" {
		return errors.New("url is required")
	}

	if !strings.HasPrefix(spec.Request.URL, "http://") && !strings.HasPrefix(spec.Request.URL, "https://") {
		return errors.New("url must start with http:// or https://")
	}

	if len(spec.Schedule.Locations) == 0 {
		return errors.New("at least one location is required")
	}

	return nil
}

func (c *CreateHTTPSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	spec := CreateHTTPSyntheticCheckSpec{}
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

func (c *CreateHTTPSyntheticCheck) buildRequest(spec CreateHTTPSyntheticCheckSpec) SyntheticCheckRequest {
	return BuildSyntheticCheckRequest(spec.Name, spec.Request, spec.Schedule, BuildSyntheticCheckAssertions(spec.Assertions), spec.Retries)
}

// BuildSyntheticCheckRequest builds the API request payload from spec fields (shared by create and update components).
func BuildSyntheticCheckRequest(name string, req RequestSpec, sched ScheduleSpec, assertions SyntheticCheckAssertions, retries *RetrySpec) SyntheticCheckRequest {
	method := req.Method
	if method == "" {
		method = "get"
	}

	redirects := req.Redirects
	if redirects == "" {
		redirects = "follow"
	}

	strategy := sched.Strategy
	if strategy == "" {
		strategy = "all_locations"
	}

	interval := sched.Interval
	if interval == "" {
		interval = "1m"
	}

	retryAttempts := 3
	retryDelay := "1s"
	if retries != nil {
		if retries.Attempts > 0 {
			retryAttempts = retries.Attempts
		}
		if retries.Delay != "" {
			retryDelay = retries.Delay
		}
	}

	headers := make([]SyntheticCheckHeader, 0)
	if req.Headers != nil {
		for _, h := range *req.Headers {
			headers = append(headers, SyntheticCheckHeader{
				Name:  h.Name,
				Value: h.Value,
			})
		}
	}

	return SyntheticCheckRequest{
		Kind: "Dash0SyntheticCheck",
		Metadata: SyntheticCheckMetadata{
			Name:   strings.ReplaceAll(strings.ToLower(name), " ", "-"),
			Labels: map[string]any{},
		},
		Spec: SyntheticCheckTopLevelSpec{
			Enabled: true,
			Schedule: SyntheticCheckSchedule{
				Interval:  interval,
				Locations: sched.Locations,
				Strategy:  strategy,
			},
			Plugin: SyntheticCheckPlugin{
				Display: SyntheticCheckDisplay{
					Name: name,
				},
				Kind: "http",
				Spec: SyntheticCheckPluginSpec{
					Request: SyntheticCheckHTTPRequest{
						Method:          method,
						URL:             req.URL,
						Headers:         headers,
						QueryParameters: make([]any, 0),
						Body:            req.Body,
						Redirects:       redirects,
						TLS: SyntheticCheckTLS{
							AllowInsecure: req.AllowInsecure == "true",
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

// BuildSyntheticCheckAssertions builds the API assertions payload from spec (shared by create and update components).
func BuildSyntheticCheckAssertions(assertions *[]AssertionSpec) SyntheticCheckAssertions {
	criticalAssertions := make([]SyntheticCheckAssertion, 0)
	degradedAssertions := make([]SyntheticCheckAssertion, 0)

	if assertions == nil {
		return SyntheticCheckAssertions{
			CriticalAssertions: criticalAssertions,
			DegradedAssertions: degradedAssertions,
		}
	}

	for _, a := range *assertions {
		assertion := buildSingleAssertion(a)
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

func buildSingleAssertion(a AssertionSpec) *SyntheticCheckAssertion {
	spec := map[string]any{}

	if a.Operator != "" {
		spec["operator"] = a.Operator
	}
	if a.Value != "" {
		spec["value"] = a.Value
	}
	if a.Type != "" {
		spec["type"] = a.Type
	}
	if a.Name != "" {
		spec["name"] = a.Name
	}
	if a.Expression != "" {
		spec["expression"] = a.Expression
	}

	return &SyntheticCheckAssertion{
		Kind: a.Kind,
		Spec: spec,
	}
}

// AssertionFieldSchema returns the configuration fields for a single assertion (used by create and update components).
func AssertionFieldSchema() []configuration.Field {
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

		// type - used by timing (phase selector)
		{
			Name:    "type",
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

		// name - used by response_header (header name)
		{
			Name:        "name",
			Label:       "Header Name",
			Type:        configuration.FieldTypeString,
			Placeholder: "x-auth-token",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"response_header"}},
			},
		},

		// expression - used by json_body (JSONPath)
		{
			Name:        "expression",
			Label:       "JSONPath Expression",
			Type:        configuration.FieldTypeString,
			Placeholder: "$.status",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"json_body"}},
			},
		},

		// operator - shared across most assertion kinds
		{
			Name:    "operator",
			Label:   "Operator",
			Type:    configuration.FieldTypeSelect,
			Default: "is",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Is", Value: "is"},
						{Label: "Is not", Value: "is_not"},
						{Label: "<=", Value: "lte"},
						{Label: ">=", Value: "gte"},
						{Label: "<", Value: "lt"},
						{Label: ">", Value: "gt"},
						{Label: "Contains", Value: "contains"},
						{Label: "Does not contain", Value: "not_contains"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"status_code", "timing", "ssl_certificate_validity", "response_header", "json_body", "text_body"}},
			},
		},

		// value - shared across most assertion kinds
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeString,
			Placeholder: "200",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "kind", Values: []string{"status_code", "timing", "error_type", "ssl_certificate_validity", "response_header", "json_body", "text_body"}},
			},
		},
	}
}

func (c *CreateHTTPSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateHTTPSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateHTTPSyntheticCheck) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateHTTPSyntheticCheck) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateHTTPSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateHTTPSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
