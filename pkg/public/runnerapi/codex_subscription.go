package runnerapi

import (
	"context"
	"encoding/json"
	"time"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerpayload "github.com/superplanehq/superplane/pkg/runners/api"
)

// Mint the run token at delivery so queue time does not consume its lifetime.
// Login tokens keep their original deadline because login leases expire in queue.
func (s *Server) refreshCodexSubscriptionTaskToken(ctx context.Context, task *models.RunnerTask, payload []byte) ([]byte, error) {
	db := database.DB(ctx)
	lease, err := models.FindCodexLeaseByTask(db, task.ID)
	if err != nil {
		return nil, err
	}
	if lease != nil && lease.RevokedAt != nil {
		return nil, models.ErrCodexConnectionStale
	}
	if lease == nil || lease.Login() {
		return payload, nil
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
		FactoryID:      connection.FactoryID,
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
