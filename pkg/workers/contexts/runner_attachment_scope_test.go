package contexts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerAttachmentsUseInvocationScopeInsteadOfSignedURLAppearance(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, "invocation workspace", "", "")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, "other workspace", "", "")
	require.NoError(t, err)
	_, _, run := setupRunnerAppExecution(t, r, factory.ID, "runnerClaudeCode")
	_, _, standalone := setupRunnerAppExecution(t, r, factory.ID, "runnerClaudeCode")
	order, err := factory.CreateWorkOrder(db, "invocation task", "", &r.User, nil, nil)
	require.NoError(t, err)
	otherOrder, err := factory.CreateWorkOrder(db, "other task", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)
	fleet := models.DefaultInstallationRunnerFleets("dev")[0]
	require.NoError(t, fleet.Create(db))
	for _, test := range []struct {
		name                          string
		factoryID, workOrderID, runID uuid.UUID
		allowed                       bool
	}{
		{"workspace attachment", factory.ID, uuid.Nil, run.ID, true},
		{"own task attachment", factory.ID, order.ID, run.ID, true},
		{"other workspace", otherFactory.ID, uuid.Nil, run.ID, false},
		{"other task", factory.ID, otherOrder.ID, run.ID, false},
		{"task without invocation task", factory.ID, order.ID, standalone.ID, false},
		{"missing invocation", factory.ID, uuid.Nil, uuid.Nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			scope := blob.ScopeWorkspace
			if test.workOrderID != uuid.Nil {
				scope = blob.ScopeTask
			}
			file, err := models.CreatePendingFile(db, models.CreateFileParams{Scope: scope, OrganizationID: r.Organization.ID, FactoryID: test.factoryID, WorkOrderID: test.workOrderID, Filename: "notes.txt", ContentType: "text/plain", CreatedByID: r.User})
			require.NoError(t, err)
			require.NoError(t, file.MarkReady(db, 12, "checksum"))
			id := uuid.New()
			// A URL marker and a known file ID do not authorize refresh. The
			// invocation must independently permit its workspace and task.
			url := fmt.Sprintf("https://files.example/api/v1/public/files/%s?sp_file=1&sig=forged&expires=0", file.ID)
			payload, err := json.Marshal(runnerapi.TaskPayload{ID: id.String(), Script: "read " + url})
			require.NoError(t, err)
			err = NewRunnerTaskContext(db, crypto.NewNoOpEncryptor(), r.Organization.ID).WithRunScope(test.runID).Create(id.String(), fleet.Slug, payload)
			if !test.allowed {
				require.Error(t, err)
				_, findErr := models.FindRunnerTask(db, id)
				require.ErrorIs(t, findErr, models.ErrRunnerTaskNotFound)
				return
			}
			require.NoError(t, err)
			job, err := models.FindRunnerTask(db, id)
			require.NoError(t, err)
			var saved runnerapi.TaskPayload
			require.NoError(t, json.Unmarshal(job.PayloadCiphertext, &saved))
			require.Len(t, saved.CredentialRefreshes, 1)
			require.Equal(t, run.ID.String(), saved.CredentialRefreshes[0].Scope["run_id"])
			require.Equal(t, factory.ID.String(), saved.CredentialRefreshes[0].Scope["factory_id"])
		})
	}
}
