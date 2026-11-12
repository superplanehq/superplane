package runner

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm/modelid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const modelCallOutcomeInsertTimeout = 3 * time.Second

// Tests set these so an insert can run inline, fail, or panic.
var runModelCallOutcomeInsert = func(fn func()) {
	go fn()
}

var insertHostedModelCallOutcomes = func(tx *gorm.DB, inputs []models.HostedModelCallOutcomeInput) error {
	return models.InsertHostedModelCallOutcomes(tx, inputs)
}

// The agent can write the task directory, including the usage sidecar.
// One task cannot add more than this many rows to the shared status page.
const maxModelCallOutcomesPerTask = 500

type parsedModelCallOutcome struct {
	Model      string `json:"model"`
	Outcome    string `json:"outcome"`
	OccurredAt string `json:"occurredAt"`
}

func RecordRunnerModelCallOutcomes(logger *log.Entry, taskID, finishedEventType string, configuration any, result json.RawMessage, notBefore time.Time) {
	// A bad result must not panic the task status path.
	defer func() {
		if recovered := recover(); recovered != nil {
			warnModelCallOutcomePanic(logger, recovered)
		}
	}()
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
	// Detach the insert. A slow or failed sample must not change the task.
	runModelCallOutcomeInsert(func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				warnModelCallOutcomePanic(logger, recovered)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), modelCallOutcomeInsertTimeout)
		defer cancel()
		if err := insertHostedModelCallOutcomes(database.Conn().WithContext(ctx), inputs); err != nil {
			warnModelCallOutcome(logger, err)
		}
	})
}

func warnModelCallOutcome(logger *log.Entry, err error) {
	if logger == nil || err == nil {
		return
	}
	logger.WithError(err).Warn("failed to record hosted model call outcomes")
}

func warnModelCallOutcomePanic(logger *log.Entry, recovered any) {
	if logger == nil {
		return
	}
	logger.WithField("panic", recovered).Warn("panic recording hosted model call outcomes")
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
		if len(inputs) == maxModelCallOutcomesPerTask {
			break
		}
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
