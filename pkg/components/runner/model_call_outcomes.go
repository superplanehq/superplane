package runner

import (
	"encoding/json"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm/modelid"
	"github.com/superplanehq/superplane/pkg/models"
)

type parsedModelCallOutcome struct {
	Model      string `json:"model"`
	Outcome    string `json:"outcome"`
	OccurredAt string `json:"occurredAt"`
}

func RecordRunnerModelCallOutcomes(logger *log.Entry, taskID, finishedEventType string, configuration any, result json.RawMessage, notBefore time.Time) {
	if !IsHostedCredentials(configurationCredentialsSource(configuration)) {
		return
	}
	provider, ok := providerForFinishedEvent(finishedEventType, configuration)
	if !ok || !models.IsHostedModelStatusProvider(provider) {
		return
	}
	inputs := parseRunnerModelCallOutcomes(provider, taskID, result)
	clampModelCallOccurredAt(inputs, notBefore)
	if len(inputs) == 0 {
		return
	}
	if err := models.InsertHostedModelCallOutcomes(database.Conn(), inputs); err != nil && logger != nil {
		logger.WithError(err).Error("failed to record hosted model call outcomes")
	}
}

func clampModelCallOccurredAt(inputs []models.HostedModelCallOutcomeInput, notBefore time.Time) {
	if notBefore.IsZero() {
		return
	}
	for i := range inputs {
		if !inputs[i].OccurredAt.IsZero() && inputs[i].OccurredAt.Before(notBefore) {
			inputs[i].OccurredAt = notBefore
		}
	}
}

func parseRunnerModelCallOutcomes(provider, taskID string, result json.RawMessage) []models.HostedModelCallOutcomeInput {
	if strings.TrimSpace(taskID) == "" || len(result) == 0 {
		return nil
	}
	var payload struct {
		ModelCallOutcomes []parsedModelCallOutcome `json:"modelCallOutcomes"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return nil
	}
	inputs := make([]models.HostedModelCallOutcomeInput, 0, len(payload.ModelCallOutcomes))
	for i, row := range payload.ModelCallOutcomes {
		model := modelid.CatalogModelID(row.Model)
		if model == "" {
			continue
		}
		inputs = append(inputs, models.HostedModelCallOutcomeInput{
			Provider:     provider,
			Model:        model,
			Outcome:      strings.TrimSpace(row.Outcome),
			OccurredAt:   parseOccurredAt(row.OccurredAt),
			RunnerTaskID: taskID,
			Sequence:     i + 1,
		})
	}
	return inputs
}

func parseOccurredAt(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	return time.Time{}
}
