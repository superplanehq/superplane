package public

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/test/support"
	sentrytest "github.com/superplanehq/superplane/test/support/sentry"
	"gorm.io/gorm"
)

func TestCodexCheckpointReportsSafeErrors(t *testing.T) {
	const secret = "checkpoint-credential-must-stay-private"
	for _, test := range []struct {
		name          string
		queryError    error
		encryptErr    error
		commitFailure bool
		body          string
		status        int
		step          string
		cause         string
	}{
		{
			name: "database failure",
			queryError: fmt.Errorf("query contains %s: %w", secret, &pgconn.PgError{
				Code: "40P01", Message: secret, Detail: secret, InternalQuery: secret,
			}),
			status: http.StatusInternalServerError, step: "lock_connection", cause: "SQLSTATE 40P01",
		},
		{
			name: "wrapped timeout", queryError: fmt.Errorf("%s: %w", secret, context.DeadlineExceeded),
			status: http.StatusInternalServerError, step: "lock_connection", cause: context.DeadlineExceeded.Error(),
		},
		{
			name: "encryption failure", encryptErr: errors.New(secret),
			status: http.StatusInternalServerError, step: "encrypt_bundle", cause: "*errors.errorString",
		},
		{
			name: "commit failure", commitFailure: true,
			status: http.StatusInternalServerError, step: "commit_transaction", cause: "SQLSTATE 23503",
		},
		{
			name: "invalid SQLSTATE", queryError: &pgconn.PgError{Code: secret, Message: secret},
			status: http.StatusInternalServerError, step: "lock_connection", cause: "*pgconn.PgError",
		},
		{name: "invalid bundle", body: `{"bundle":{}}`, status: http.StatusBadRequest},
		{name: "missing connection", queryError: models.ErrCodexConnectionNotFound, status: http.StatusNotFound},
		{name: "stale lease", queryError: models.ErrCodexConnectionStale, status: http.StatusConflict},
		{name: "connection not ready", queryError: models.ErrCodexConnectionNotReady, status: http.StatusConflict},
		{name: "ineligible member", queryError: models.ErrCodexConnectionNotEligible, status: http.StatusForbidden},
		{name: "unavailable connection", queryError: models.ErrCodexConnectionUnavailable, status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
			require.NoError(t, err)
			taskID := uuid.New()
			require.NoError(t, connection.BeginLogin(db, taskID, time.Now()))
			signer := jwt.NewSigner("test-checkpoint-diagnostics")
			scope := runneraction.CodexSubscriptionScope{
				OrganizationID: r.Organization.ID, ConnectionID: connection.ID,
				TaskID: taskID, Generation: connection.ReservationGeneration,
			}
			token, err := runneraction.MintCodexSubscriptionToken(signer, scope, time.Hour)
			require.NoError(t, err)
			server := &Server{jwt: signer, encryptor: r.Encryptor}
			if test.encryptErr != nil {
				server.encryptor = checkpointErrorEncryptor{Encryptor: r.Encryptor, err: test.encryptErr}
			}
			const callback = "test:checkpoint_query_failure"
			if test.queryError != nil {
				require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
					_ = tx.AddError(test.queryError)
				}))
				t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })
			}
			if test.commitFailure {
				require.NoError(t, db.Callback().Update().After("gorm:update").Register(callback, func(tx *gorm.DB) {
					if tx.Statement.Table != "factory_codex_connections" {
						return
					}
					// The constraint fails only at COMMIT, after bundle storage succeeds.
					isolated := tx.Session(&gorm.Session{NewDB: true})
					require.NoError(t, isolated.Exec(`CREATE TEMP TABLE codex_checkpoint_commit_failure (
						id INT PRIMARY KEY,
						parent INT REFERENCES codex_checkpoint_commit_failure(id) DEFERRABLE INITIALLY DEFERRED
					) ON COMMIT DROP`).Error)
					require.NoError(t, isolated.Exec("INSERT INTO codex_checkpoint_commit_failure VALUES (1, 2)").Error)
				}))
				t.Cleanup(func() { require.NoError(t, db.Callback().Update().Remove(callback)) })
			}
			transport := sentrytest.BindHub(t)
			logger := log.New()
			var logs bytes.Buffer
			logger.SetOutput(&logs)
			logger.SetFormatter(&log.JSONFormatter{})
			handler := middleware.LoggingMiddleware(logger)(http.HandlerFunc(server.handleRunnerCodexSubscriptionCheckpoint))
			body := `{"bundle":{"access_token":"` + secret + `","refresh_token":"refresh-` + secret + `","id_token":"id-` + secret + `"},"private":"body-must-stay-private"}`
			if test.body != "" {
				body = test.body
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/checkpoint", strings.NewReader(body))
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("X-Request-ID", "checkpoint-request")
			request.Header.Set("X-Cloud-Trace-Context", "efe5990b82441bfd4017a1e10a84b213/1;o=1")
			response := httptest.NewRecorder()
			sentry.CurrentHub().WithScope(func(scope *sentry.Scope) {
				// Exercise body omission even when an outer scope buffers the request.
				scope.SetRequest(request)
				handler.ServeHTTP(response, request)
			})
			require.Equal(t, test.status, response.Code, response.Body.String())

			var accessLog map[string]any
			require.NoError(t, json.Unmarshal(logs.Bytes(), &accessLog))
			if test.status != http.StatusInternalServerError {
				assert.Empty(t, transport.Events())
				assert.NotContains(t, accessLog, "error")
				return
			}
			assert.Equal(t, "ChatGPT subscription request failed\n", response.Body.String())
			assert.Equal(t, "error", accessLog["level"])
			assert.Contains(t, accessLog["error"], test.cause)
			assert.Contains(t, accessLog["error"], test.step)
			assert.Equal(t, "checkpoint-request", accessLog["request_id"])
			assert.Equal(t, "efe5990b82441bfd4017a1e10a84b213", accessLog["trace_id"])
			events := transport.Events()
			require.Len(t, events, 1)
			event := events[0]
			assert.Contains(t, sentrytest.ExceptionText(event), test.cause)
			for key, value := range map[string]string{
				"organization_id": scope.OrganizationID.String(), "task_id": taskID.String(),
				"connection_id": connection.ID.String(), "operation": "checkpoint", "checkpoint_step": test.step,
				"lease_generation": strconv.FormatInt(scope.Generation, 10),
			} {
				assert.Equal(t, value, accessLog[key], key)
				assert.Equal(t, value, event.Tags[key], key)
			}
			require.NotNil(t, event.Request)
			assert.Empty(t, event.Request.Data)
			encoded, err := json.Marshal(event)
			require.NoError(t, err)
			for _, private := range []string{secret, token, "body-must-stay-private"} {
				assert.NotContains(t, string(encoded), private)
				assert.NotContains(t, logs.String(), private)
				assert.NotContains(t, response.Body.String(), private)
			}
			if test.encryptErr != nil {
				server.encryptor = r.Encryptor
				retry := httptest.NewRequest(http.MethodPost, request.URL.Path, strings.NewReader(body))
				retry.Header.Set("Authorization", "Bearer "+token)
				retried := httptest.NewRecorder()
				handler.ServeHTTP(retried, retry)
				require.Equal(t, http.StatusOK, retried.Code, retried.Body.String())
				assert.Len(t, transport.Events(), 1)
			}
		})
	}
}

type checkpointErrorEncryptor struct {
	crypto.Encryptor
	err error
}

func (e checkpointErrorEncryptor) Encrypt(context.Context, []byte, []byte) ([]byte, error) {
	return nil, e.err
}
