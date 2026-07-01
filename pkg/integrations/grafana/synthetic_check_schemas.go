package grafana

import "github.com/superplanehq/superplane/pkg/configuration"

// syntheticCheckSharedFields returns configuration fields grouped like dash0.createHttpSyntheticCheck:
// job + labels, Request (object), Schedule (object), Validation (object), Per-Check Alerts (list).
func syntheticCheckSharedFields() []configuration.Field {
	frequencyMin := 1
	timeoutMin := 1
	alertThresholdMin := 1

	return []configuration.Field{
		{
			Name:        "job",
			Label:       "Job",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Display name for the synthetic check",
			Placeholder: "API health check",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional labels added to the synthetic check",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label name",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label value",
							},
						},
					},
				},
			},
		},
		{
			Name:        "request",
			Label:       "Request",
			Type:        configuration.FieldTypeObject,
			Required:    true,
			Description: "HTTP request configuration",
			Default: map[string]any{
				"method":            "GET",
				"noFollowRedirects": false,
			},
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: syntheticCheckRequestObjectSchema(),
				},
			},
		},
		{
			Name:        "schedule",
			Label:       "Schedule",
			Type:        configuration.FieldTypeObject,
			Required:    true,
			Description: "How often the check runs and which probes execute it",
			Default: map[string]any{
				"enabled":   true,
				"frequency": defaultSyntheticCheckFrequencySeconds,
				"timeout":   defaultSyntheticCheckTimeout,
			},
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: syntheticCheckScheduleObjectSchema(&frequencyMin, &timeoutMin),
				},
			},
		},
		{
			Name:        "validation",
			Label:       "Response validation",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Optional rules for SSL, status codes, body, and header matching",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: syntheticCheckValidationObjectSchema(),
				},
			},
		},
		syntheticCheckAlertsField(&alertThresholdMin),
	}
}

// syntheticCheckUpdateSharedFields matches Create, but each mutable section is togglable so users opt in
// to changes; omitted sections keep values loaded from the existing check at execution time.
func syntheticCheckUpdateSharedFields() []configuration.Field {
	base := syntheticCheckSharedFields()
	out := make([]configuration.Field, len(base))
	copy(out, base)
	for i := range out {
		switch out[i].Name {
		case "job", "labels", "request", "schedule":
			out[i].Required = false
			out[i].Togglable = true
		case "alerts":
			out[i].Togglable = true
		}
	}
	return out
}

func syntheticCheckRequestObjectSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "target",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Target URL to monitor",
			Placeholder: "https://api.example.com/health",
		},
		{
			Name:        "method",
			Label:       "HTTP Method",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "GET",
			Description: "HTTP request method",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET", Value: "GET"},
						{Label: "POST", Value: "POST"},
						{Label: "PUT", Value: "PUT"},
						{Label: "DELETE", Value: "DELETE"},
						{Label: "HEAD", Value: "HEAD"},
						{Label: "PATCH", Value: "PATCH"},
						{Label: "OPTIONS", Value: "OPTIONS"},
					},
				},
			},
		},
		{
			Name:        "headers",
			Label:       "Headers",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional HTTP request headers",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Header",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Header name",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Header value",
							},
						},
					},
				},
			},
		},
		{
			Name:        "body",
			Label:       "Request Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Optional HTTP request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH", "DELETE"}},
			},
		},
		{
			Name:        "noFollowRedirects",
			Label:       "Do Not Follow Redirects",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Disable automatic redirect following",
		},
		{
			Name:        "basicAuth",
			Label:       "Basic Auth",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Optional HTTP basic authentication credentials",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "username",
							Label:       "Username",
							Type:        configuration.FieldTypeString,
							Required:    true,
							Description: "Basic auth username",
						},
						{
							Name:        "password",
							Label:       "Password",
							Type:        configuration.FieldTypeString,
							Required:    true,
							Sensitive:   true,
							Description: "Basic auth password",
						},
					},
				},
			},
		},
		{
			Name:        "bearerToken",
			Label:       "Bearer Token",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Sensitive:   true,
			Description: "Optional bearer token sent with the HTTP request",
		},
	}
}

func syntheticCheckScheduleObjectSchema(frequencyMin, timeoutMin *int) []configuration.Field {
	return []configuration.Field{
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Whether the check should run immediately after creation or update",
		},
		{
			Name:        "frequency",
			Label:       "Frequency (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     defaultSyntheticCheckFrequencySeconds,
			Description: "How often the check should run, in seconds. Existing workflows that still store milliseconds remain supported.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: frequencyMin,
				},
			},
		},
		{
			Name:        "timeout",
			Label:       "Timeout (ms)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     defaultSyntheticCheckTimeout,
			Description: "Request timeout in milliseconds",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: timeoutMin,
				},
			},
		},
		{
			Name:        "probes",
			Label:       "Locations",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Synthetic monitoring probes (locations) that should run the check",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  resourceTypeSyntheticProbe,
					Multi: true,
				},
			},
		},
	}
}

func syntheticCheckValidationObjectSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "failIfSSL",
			Label:       "Fail If SSL Present",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Fail the check if the target responds over SSL/TLS",
		},
		{
			Name:        "failIfNotSSL",
			Label:       "Fail If SSL Missing",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Fail the check if the target does not respond over SSL/TLS",
		},
		{
			Name:        "validStatusCodes",
			Label:       "Valid Status Codes",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional list of accepted HTTP status codes",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Status Code",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeNumber,
					},
				},
			},
		},
		{
			Name:        "failIfBodyMatchesRegexp",
			Label:       "Fail If Body Matches Regex",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional body regexes that should fail the check when matched",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Regex",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "failIfBodyNotMatchesRegexp",
			Label:       "Fail If Body Does Not Match Regex",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional body regexes that must match",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Regex",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "failIfHeaderMatchesRegexp",
			Label:       "Fail If Header Matches Regex",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional header regex matchers that should fail the check when matched",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Header Matcher",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "header",
								Label:       "Header",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Response header name to inspect",
							},
							{
								Name:        "regexp",
								Label:       "Regex",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Regex that should fail the check when matched",
							},
							{
								Name:        "allowMissing",
								Label:       "Allow Missing Header",
								Type:        configuration.FieldTypeBool,
								Required:    false,
								Default:     false,
								Description: "Do not fail when the response header is missing",
							},
						},
					},
				},
			},
		},
	}
}

func syntheticCheckAlertsField(alertThresholdMin *int) configuration.Field {
	return configuration.Field{
		Name:        "alerts",
		Label:       "Per-Check Alerts",
		Type:        configuration.FieldTypeList,
		Required:    false,
		Description: "Optional per-check alerts configured for the synthetic check",
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: "Alert",
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeObject,
					Schema: []configuration.Field{
						{
							Name:        "name",
							Label:       "Alert Type",
							Type:        configuration.FieldTypeSelect,
							Required:    true,
							Description: "Alert rule to configure for this synthetic check",
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Failed Checks", Value: "ProbeFailedExecutionsTooHigh"},
										{Label: "TLS Target Certificate Close To Expiring", Value: "TLSTargetCertificateCloseToExpiring"},
										{Label: "HTTP Request Duration Too High Avg", Value: "HTTPRequestDurationTooHighAvg"},
									},
								},
							},
						},
						{
							Name:        "threshold",
							Label:       "Threshold",
							Type:        configuration.FieldTypeNumber,
							Required:    true,
							Description: "Alert threshold value",
							TypeOptions: &configuration.TypeOptions{
								Number: &configuration.NumberTypeOptions{
									Min: alertThresholdMin,
								},
							},
						},
						{
							Name:        "period",
							Label:       "Period",
							Type:        configuration.FieldTypeSelect,
							Required:    false,
							Description: "Evaluation period for alerts that support a period",
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "name", Values: []string{"ProbeFailedExecutionsTooHigh", "HTTPRequestDurationTooHighAvg"}},
							},
							RequiredConditions: []configuration.RequiredCondition{
								{Field: "name", Values: []string{"ProbeFailedExecutionsTooHigh", "HTTPRequestDurationTooHighAvg"}},
							},
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "5 min", Value: "5m"},
										{Label: "10 min", Value: "10m"},
										{Label: "15 min", Value: "15m"},
										{Label: "20 min", Value: "20m"},
										{Label: "30 min", Value: "30m"},
										{Label: "1 h", Value: "1h"},
									},
								},
							},
						},
						{
							Name:        "runbookUrl",
							Label:       "Runbook URL",
							Type:        configuration.FieldTypeString,
							Required:    false,
							Description: "Optional runbook URL included on the alert",
						},
					},
				},
			},
		},
	}
}
