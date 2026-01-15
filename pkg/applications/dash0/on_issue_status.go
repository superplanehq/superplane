package dash0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueStatus struct{}

type OnIssueStatusConfiguration struct {
	MinutesInterval *int `json:"minutesInterval"`
}

type OnIssueStatusMetadata struct {
	NextTrigger   *string `json:"nextTrigger"`
	ReferenceTime *string `json:"referenceTime"` // Time when schedule was first set up
}

func (t *OnIssueStatus) Name() string {
	return "dash0.onIssueStatus"
}

func (t *OnIssueStatus) Label() string {
	return "On Issue Status"
}

func (t *OnIssueStatus) Description() string {
	return "Periodically check Dash0 for issues and trigger when issues are detected"
}

func (t *OnIssueStatus) Icon() string {
	return "alert-triangle"
}

func (t *OnIssueStatus) Color() string {
	return "red"
}

func (t *OnIssueStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "minutesInterval",
			Label:       "Check every (minutes)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     intPtr(5),
			Description: "Number of minutes between checks (1-59)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(59),
				},
			},
		},
	}
}

func (t *OnIssueStatus) Setup(ctx core.TriggerContext) error {
	config := OnIssueStatusConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.MinutesInterval == nil {
		return fmt.Errorf("minutesInterval is required")
	}

	if *config.MinutesInterval < 1 || *config.MinutesInterval > 59 {
		return fmt.Errorf("minutesInterval must be between 1 and 59, got: %d", *config.MinutesInterval)
	}

	var metadata OnIssueStatusMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	now := time.Now()

	if metadata.ReferenceTime == nil {
		referenceTime := now.Format(time.RFC3339)
		metadata.ReferenceTime = &referenceTime
	}

	nextTrigger, err := t.nextTrigger(*config.MinutesInterval, now, metadata.ReferenceTime)
	if err != nil {
		return err
	}

	//
	// If the configuration didn't change, don't schedule a new action.
	//
	if metadata.NextTrigger != nil {
		currentTrigger, err := time.Parse(time.RFC3339, *metadata.NextTrigger)
		if err != nil {
			return fmt.Errorf("error parsing next trigger: %v", err)
		}

		if currentTrigger.Sub(*nextTrigger).Abs() < time.Second {
			return nil
		}
	}

	//
	// Always schedule the next and save the next trigger in the metadata.
	//
	err = ctx.Requests.ScheduleActionCall("checkIssues", map[string]any{}, time.Until(*nextTrigger))
	if err != nil {
		return fmt.Errorf("error scheduling action call: %w", err)
	}

	formatted := nextTrigger.Format(time.RFC3339)
	return ctx.Metadata.Set(OnIssueStatusMetadata{
		NextTrigger:   &formatted,
		ReferenceTime: metadata.ReferenceTime,
	})
}

func (t *OnIssueStatus) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "checkIssues",
			UserAccessible: false,
		},
	}
}

func (t *OnIssueStatus) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkIssues":
		return nil, t.checkIssues(ctx)
	}

	return nil, fmt.Errorf("action %s not supported", ctx.Name)
}

func (t *OnIssueStatus) checkIssues(ctx core.TriggerActionContext) error {
	config := OnIssueStatusConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.MinutesInterval == nil {
		return fmt.Errorf("minutesInterval is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating dash0 client: %w", err)
	}

	query := `{otel_metric_name="dash0.issue.status"} >= 1`
	dataset := "default"

	response, err := client.ExecutePrometheusInstantQuery(query, dataset)
	if err != nil {
		ctx.Logger.Warnf("Error executing Prometheus query: %v", err)
		// Continue to reschedule even if query fails
	} else {
		// The response["data"] is a PrometheusResponseData struct, not a map
		// We need to convert it to access the result field
		dataValue := response["data"]
		var dataMap map[string]any

		if dataMapValue, ok := dataValue.(map[string]any); ok {
			dataMap = dataMapValue
		} else {
			// If it's a struct, marshal and unmarshal it to convert to map
			jsonBytes, marshalErr := json.Marshal(dataValue)
			if marshalErr != nil {
				ctx.Logger.Warnf("Failed to marshal response data: %v", marshalErr)
			} else {
				unmarshalErr := json.Unmarshal(jsonBytes, &dataMap)
				if unmarshalErr != nil {
					ctx.Logger.Warnf("Failed to unmarshal response data: %v", unmarshalErr)
				}
			}
		}

		if dataMap != nil {
			result, ok := dataMap["result"].([]interface{})
			if !ok {
				ctx.Logger.Warnf("Unexpected response format: result is not an array, got %T", dataMap["result"])
			} else if len(result) > 0 {
				// Issues detected - emit event
				payload := map[string]any{
					"query":   query,
					"dataset": dataset,
					"results": result,
					"count":   len(result),
				}

				err = ctx.Events.Emit("dash0.issue.detected", payload)
				if err != nil {
					ctx.Logger.Errorf("Error emitting event: %v", err)
					// Continue to reschedule even if emit fails
				} else {
					ctx.Logger.Infof("Issues detected: %d issue(s) found", len(result))
				}
			}
		}
	}

	// Reschedule next check
	var existingMetadata OnIssueStatusMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &existingMetadata)
	if err != nil {
		// Use current time as reference if metadata is invalid
		nowStr := time.Now().Format(time.RFC3339)
		existingMetadata = OnIssueStatusMetadata{
			ReferenceTime: &nowStr,
		}
	}

	nowUTC := time.Now()
	nextTrigger, err := t.nextTrigger(*config.MinutesInterval, nowUTC, existingMetadata.ReferenceTime)
	if err != nil {
		return fmt.Errorf("error calculating next trigger: %w", err)
	}

	err = ctx.Requests.ScheduleActionCall("checkIssues", map[string]any{}, time.Until(*nextTrigger))
	if err != nil {
		return fmt.Errorf("error rescheduling action call: %w", err)
	}

	formatted := nextTrigger.Format(time.RFC3339)
	return ctx.Metadata.Set(OnIssueStatusMetadata{
		NextTrigger:   &formatted,
		ReferenceTime: existingMetadata.ReferenceTime,
	})
}

func (t *OnIssueStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnIssueStatus) nextTrigger(interval int, now time.Time, referenceTime *string) (*time.Time, error) {
	if interval < 1 || interval > 59 {
		return nil, fmt.Errorf("interval must be between 1 and 59 minutes, got: %d", interval)
	}

	nowInTZ := now

	var reference time.Time
	if referenceTime != nil {
		var err error
		reference, err = time.Parse(time.RFC3339, *referenceTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reference time: %w", err)
		}
		reference = reference.In(nowInTZ.Location())
	} else {
		reference = nowInTZ
	}

	minutesElapsed := int(nowInTZ.Sub(reference).Minutes())

	if minutesElapsed < 0 {
		minutesElapsed = 0
	}
	completedIntervals := minutesElapsed / interval

	nextTriggerMinutes := (completedIntervals + 1) * interval
	nextTrigger := reference.Add(time.Duration(nextTriggerMinutes) * time.Minute)

	if nextTrigger.Before(nowInTZ) || nextTrigger.Equal(nowInTZ) {
		nextTrigger = nextTrigger.Add(time.Duration(interval) * time.Minute)
	}

	utcResult := nextTrigger.UTC()
	return &utcResult, nil
}

func intPtr(v int) *int {
	return &v
}
