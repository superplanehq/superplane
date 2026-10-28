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

func RecordRunnerModelCallOutcomes(logger *log.Entry, taskID, finishedEventType string, configuration any, result json.RawMessage) {
	if !IsHostedCredentials(configurationCredentialsSource(configuration)) {
		return
	}
	provider, ok := providerForFinishedEvent(finishedEventType, configuration)
	if !ok || !isHostedModelStatusProvider(provider) {
		return
	}
	inputs := parseRunnerModelCallOutcomes(provider, taskID, result)
	if len(inputs) == 0 {
		return
	}
	if err := models.InsertHostedModelCallOutcomes(database.Conn(), inputs); err != nil && logger != nil {
		logger.WithError(err).Error("failed to record hosted model call outcomes")
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

func isHostedModelStatusProvider(provider string) bool {
	switch provider {
	case models.UsageProviderAnthropic, models.UsageProviderOpenAI, models.UsageProviderOpenRouter:
		return true
	default:
		return false
	}
}
