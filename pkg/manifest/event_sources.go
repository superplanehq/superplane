package manifest

// GetManualEventSourceManifest returns the manifest for manual event sources
func GetManualEventSourceManifest() *TypeManifest {
	return &TypeManifest{
		Type:        "manual",
		DisplayName: "Manual",
		Description: "Manually trigger executions via the UI or API",
		Category:    "event_source",
		Icon:        "manual",
		Fields:      []FieldManifest{},
	}
}

// GetScheduledEventSourceManifest returns the manifest for scheduled event sources
func GetScheduledEventSourceManifest() *TypeManifest {
	return &TypeManifest{
		Type:        "scheduled",
		DisplayName: "Scheduled",
		Description: "Trigger executions on a recurring schedule",
		Category:    "event_source",
		Icon:        "scheduled",
		Fields: []FieldManifest{
			{
				Name:        "schedule",
				DisplayName: "Schedule",
				Type:        FieldTypeObject,
				Required:    true,
				Description: "Define when the event source should trigger",
				Fields: []FieldManifest{
					{
						Name:        "type",
						DisplayName: "Schedule Type",
						Type:        FieldTypeSelect,
						Required:    true,
						Description: "How often to trigger",
						Options: []Option{
							{Value: "hourly", Label: "Hourly"},
							{Value: "daily", Label: "Daily"},
							{Value: "weekly", Label: "Weekly"},
						},
					},
					{
						Name:        "hourly",
						DisplayName: "Hourly Configuration",
						Type:        FieldTypeObject,
						Required:    false,
						Description: "Configuration for hourly schedule",
						DependsOn:   "type",
						Fields: []FieldManifest{
							{
								Name:        "minute",
								DisplayName: "Minute",
								Type:        FieldTypeNumber,
								Required:    true,
								Description: "Minute of the hour to trigger (0-59)",
								Placeholder: "0",
								Validation: &Validation{
									Min: intPtr(0),
									Max: intPtr(59),
								},
							},
						},
					},
					{
						Name:        "daily",
						DisplayName: "Daily Configuration",
						Type:        FieldTypeObject,
						Required:    false,
						Description: "Configuration for daily schedule",
						DependsOn:   "type",
						Fields: []FieldManifest{
							{
								Name:        "time",
								DisplayName: "Time (UTC)",
								Type:        FieldTypeString,
								Required:    true,
								Description: "Time of day to trigger in HH:MM format (24-hour UTC)",
								Placeholder: "14:30",
								Validation: &Validation{
									Pattern: "^([01]?[0-9]|2[0-3]):[0-5][0-9]$",
								},
							},
						},
					},
					{
						Name:        "weekly",
						DisplayName: "Weekly Configuration",
						Type:        FieldTypeObject,
						Required:    false,
						Description: "Configuration for weekly schedule",
						DependsOn:   "type",
						Fields: []FieldManifest{
							{
								Name:        "week_day",
								DisplayName: "Day of Week",
								Type:        FieldTypeSelect,
								Required:    true,
								Description: "Day of the week to trigger",
								Options: []Option{
									{Value: "monday", Label: "Monday"},
									{Value: "tuesday", Label: "Tuesday"},
									{Value: "wednesday", Label: "Wednesday"},
									{Value: "thursday", Label: "Thursday"},
									{Value: "friday", Label: "Friday"},
									{Value: "saturday", Label: "Saturday"},
									{Value: "sunday", Label: "Sunday"},
								},
							},
							{
								Name:        "time",
								DisplayName: "Time (UTC)",
								Type:        FieldTypeString,
								Required:    true,
								Description: "Time of day to trigger in HH:MM format (24-hour UTC)",
								Placeholder: "14:30",
								Validation: &Validation{
									Pattern: "^([01]?[0-9]|2[0-3]):[0-5][0-9]$",
								},
							},
						},
					},
				},
			},
		},
	}
}

// GetWebhookEventSourceManifest returns the manifest for webhook event sources
func GetWebhookEventSourceManifest() *TypeManifest {
	return &TypeManifest{
		Type:        "webhook",
		DisplayName: "Webhook",
		Description: "Receive events via HTTP webhook endpoint",
		Category:    "event_source",
		Icon:        "webhook",
		Fields: []FieldManifest{
			{
				Name:        "eventTypes",
				DisplayName: "Event Type Filters",
				Type:        FieldTypeArray,
				ItemType:    FieldTypeObject,
				Required:    false,
				Description: "Filter which events should trigger executions",
				Fields: []FieldManifest{
					{
						Name:        "type",
						DisplayName: "Event Type",
						Type:        FieldTypeString,
						Required:    true,
						Description: "The event type name",
						Placeholder: "push",
					},
					{
						Name:        "filter_operator",
						DisplayName: "Filter Operator",
						Type:        FieldTypeSelect,
						Required:    false,
						Description: "How to combine multiple filters",
						Options: []Option{
							{Value: "and", Label: "AND"},
							{Value: "or", Label: "OR"},
						},
						Default: "and",
					},
					{
						Name:        "filters",
						DisplayName: "Filters",
						Type:        FieldTypeArray,
						ItemType:    FieldTypeObject,
						Required:    false,
						Description: "Conditions to match on event data",
						Fields: []FieldManifest{
							{
								Name:        "type",
								DisplayName: "Filter Type",
								Type:        FieldTypeSelect,
								Required:    true,
								Description: "What to filter on",
								Options: []Option{
									{Value: "data", Label: "Event Data"},
									{Value: "header", Label: "HTTP Header"},
								},
							},
							{
								Name:        "data",
								DisplayName: "Data Filter",
								Type:        FieldTypeObject,
								Required:    false,
								Description: "Filter on event payload data",
								DependsOn:   "type",
								Fields: []FieldManifest{
									{
										Name:        "path",
										DisplayName: "JSON Path",
										Type:        FieldTypeString,
										Required:    true,
										Description: "JSON path to the field (e.g., $.ref)",
										Placeholder: "$.ref",
									},
									{
										Name:        "value",
										DisplayName: "Value",
										Type:        FieldTypeString,
										Required:    true,
										Description: "Value to match",
										Placeholder: "refs/heads/main",
									},
								},
							},
							{
								Name:        "header",
								DisplayName: "Header Filter",
								Type:        FieldTypeObject,
								Required:    false,
								Description: "Filter on HTTP headers",
								DependsOn:   "type",
								Fields: []FieldManifest{
									{
										Name:        "name",
										DisplayName: "Header Name",
										Type:        FieldTypeString,
										Required:    true,
										Description: "HTTP header name",
										Placeholder: "X-GitHub-Event",
									},
									{
										Name:        "value",
										DisplayName: "Value",
										Type:        FieldTypeString,
										Required:    true,
										Description: "Value to match",
										Placeholder: "push",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func intPtr(i int32) *int32 {
	return &i
}
