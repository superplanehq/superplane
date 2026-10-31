package runnerapi

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerpayload "github.com/superplanehq/superplane/pkg/runners/api"
	"gorm.io/gorm"
)

// Mint the run token at delivery so queue time does not consume its lifetime.
// Login and catalog tokens keep their deadline because their leases expire in queue.
func (s *Server) refreshCodexSubscriptionTaskToken(ctx context.Context, task *models.RunnerTask, payload []byte) ([]byte, error) {
	db := database.DB(ctx)
	lease, err := models.FindCodexLeaseByTask(db, task.ID)
	if err != nil {
		return nil, err
	}
	if lease != nil && lease.RevokedAt != nil {
		return nil, models.ErrCodexConnectionStale
	}
	if lease == nil {
		return payload, nil
	}
	if err := task.RequireSubscriptionLeaseContinuation(db); err != nil {
		return nil, err
	}
	if lease.Login() || lease.Catalog() {
		return payload, nil
	}
	if lease.Purpose != models.CodexLeasePurposeRun || lease.FactoryID == nil {
		return nil, models.ErrCodexConnectionStale
	}
	connection, err := models.FindFactoryCodexConnectionByID(db, lease.ConnectionID)
	if err != nil {
		return nil, err
	}
	if connection.RevocationPendingAt != nil || connection.Status != models.CodexConnectionStatusReady {
		return nil, models.ErrCodexConnectionNotReady
	}
	var document map[string]json.RawMessage
	var spec runnerpayload.TaskPayload
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(payload, &spec); err != nil {
		return nil, err
	}
	timeout := runnerpayload.DefaultExecutionTimeoutSeconds
	if spec.ExecutionTimeoutSeconds != nil && *spec.ExecutionTimeoutSeconds > 0 {
		timeout = *spec.ExecutionTimeoutSeconds
	}
	token, err := runneraction.MintCodexSubscriptionToken(s.signer, runneraction.CodexSubscriptionScope{
		OrganizationID: connection.OrganizationID,
		FactoryID:      *lease.FactoryID,
		ConnectionID:   connection.ID,
		TaskID:         task.ID,
		Generation:     lease.Generation,
	}, time.Duration(timeout)*time.Second)
	if err != nil {
		return nil, err
	}
	for i := range spec.Environment {
		if spec.Environment[i].Name == runneraction.EnvCodexSubscriptionToken {
			spec.Environment[i].Value = token
		}
	}
	document["environment"], err = json.Marshal(spec.Environment)
	if err != nil {
		return nil, err
	}
	return json.Marshal(document)
}

// Keep only the quota recovery contract. The runner never chooses the owner.
func subscriptionCompletionResult(tx *gorm.DB, task *models.RunnerTask, lease *models.FactoryCodexLease, raw json.RawMessage) (json.RawMessage, error) {
	var document map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &document) != nil || document == nil {
		return raw, nil
	}
	failureJSON, supplied := document["failure"]
	if !supplied {
		return raw, nil
	}
	delete(document, "failure")
	var failure struct {
		Reason   string `json:"reason"`
		LimitID  string `json:"limit_id"`
		ResetsAt string `json:"resets_at"`
	}
	if lease != nil && lease.Purpose == models.CodexLeasePurposeRun && json.Unmarshal(failureJSON, &failure) == nil && failure.Reason == models.SubscriptionFailureChatGPTQuotaExhausted {
		connection, err := models.FindFactoryCodexConnectionByID(tx, lease.ConnectionID)
		if err != nil {
			return nil, err
		}
		if connection.OrganizationID == task.OrganizationID {
			allowed := models.SubscriptionFailure{Reason: failure.Reason, SubscriptionOwnerID: connection.OwnerUserID.String()}
			if len(failure.LimitID) <= 200 {
				allowed.LimitID = strings.TrimSpace(failure.LimitID)
			}
			if reset, err := time.Parse(time.RFC3339, failure.ResetsAt); err == nil && reset.After(time.Now()) {
				allowed.ResetsAt = &reset
			}
			encoded, err := json.Marshal(allowed)
			if err != nil {
				return nil, err
			}
			document["failure"] = encoded
		}
	}
	return json.Marshal(document)
}
