package public

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

const (
	maxCodexSubscriptionBodyBytes  = 256*1024 + 4096
	codexRefreshOperationTimeout   = 30 * time.Second
	codexRefreshExchangeTimeout    = 15 * time.Second
	codexRefreshPersistenceReserve = 5 * time.Second
)

var errInvalidCodexBundle = errors.New("chatgpt subscription credentials are invalid")

type codexCheckpointRequest struct {
	Bundle  json.RawMessage `json:"bundle"`
	Version int64           `json:"version"`
	Final   bool            `json:"final"`
	Usage   json.RawMessage `json:"usage"`
}

type codexProgressRequest struct {
	VerificationURL  string `json:"verificationUrl"`
	VerificationCode string `json:"verificationCode"`
}

type codexRefreshProxyRequest struct {
	RefreshToken        string `json:"refresh_token"`
	RejectedAccessToken string `json:"rejected_access_token"`
}

func (s *Server) handleRunnerCodexSubscriptionCredentials(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateCodexSubscriptionRunner(w, r)
	if !ok {
		return
	}
	operation, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), codexRefreshOperationTimeout)
	defer cancel()
	var bundle any
	var version int64
	var permanent *runneraction.CodexTokenRefreshError
	var changedOrganizationID uuid.UUID
	err := database.DB(operation).Transaction(func(tx *gorm.DB) error {
		connection, lockErr := models.LockScopedFactoryCodexConnection(tx, scope.OrganizationID, scope.ConnectionID)
		if lockErr != nil {
			return lockErr
		}
		lease, err := scopedCodexLease(tx, connection, scope)
		if err != nil {
			return err
		}
		if len(connection.BundleCiphertext) == 0 {
			return models.ErrCodexConnectionNotReady
		}
		plain, failure, changed, refreshErr := s.refreshStoredCodexBundle(operation, tx, connection, "")
		if refreshErr != nil {
			return refreshErr
		}
		if failure != nil {
			permanent = failure
			if changed {
				changedOrganizationID = connection.OrganizationID
			}
			return nil
		}
		if changed {
			changedOrganizationID = connection.OrganizationID
		}
		if lease.Purpose == models.CodexLeasePurposeRun {
			secret, secretErr := s.encryptor.Decrypt(operation, lease.SecretCiphertext, []byte(connection.ID.String()))
			if secretErr != nil {
				return secretErr
			}
			plain, err = runneraction.CodexRunnerAuthBundle(plain, string(secret))
			if err != nil {
				return err
			}
		}
		if unmarshalErr := json.Unmarshal(plain, &bundle); unmarshalErr != nil || bundle == nil {
			return errors.New("stored chatgpt subscription credentials are invalid")
		}
		version = connection.CredentialVersion
		return nil
	})
	if err == nil && changedOrganizationID != uuid.Nil {
		_ = messages.PublishOrganizationCodexConnectionUpdated(database.DB(operation), changedOrganizationID)
	}
	if permanent != nil {
		writeCodexRefreshFailure(w, permanent)
		return
	}
	if err != nil {
		if operation.Err() != nil {
			writeCodexRefreshFailure(w, operation.Err())
		} else {
			writeCodexRefreshFailure(w, err)
		}
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"bundle": bundle, "version": version})
}

func (s *Server) handleRunnerCodexSubscriptionCheckpoint(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateCodexSubscriptionRunner(w, r)
	if !ok {
		return
	}
	body, err := readCodexSubscriptionBody(w, r)
	if err != nil {
		return
	}
	var request codexCheckpointRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	var usage struct {
		Limits json.RawMessage `json:"limits"`
		Models json.RawMessage `json:"models"`
	}
	_ = json.Unmarshal(request.Usage, &usage)
	var limits []models.CodexUsageLimit
	if json.Unmarshal(usage.Limits, &limits) != nil || models.ValidateCodexUsageLimits(limits) != nil {
		limits = nil
	}
	var availableModels []models.CodexModel
	if json.Unmarshal(usage.Models, &availableModels) != nil || models.ValidateCodexModels(availableModels) != nil {
		availableModels = nil
	}
	var version int64
	checkpointStep := "begin_transaction"
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		checkpointStep = "lock_connection"
		connection, lockErr := models.LockScopedFactoryCodexConnection(tx, scope.OrganizationID, scope.ConnectionID)
		if lockErr != nil {
			return lockErr
		}
		checkpointStep = "validate_lease"
		lease, err := scopedCodexLease(tx, connection, scope)
		if err != nil {
			return err
		}
		if lease.Purpose == models.CodexLeasePurposeRun {
			checkpointStep = "store_usage"
			if err := connection.StoreUsage(tx, limits, availableModels, time.Now()); err != nil {
				return err
			}
			version = connection.CredentialVersion
			checkpointStep = "commit_transaction"
			return nil
		}
		checkpointStep = "prepare_bundle"
		prepared, label, prepareErr := runneraction.PrepareCodexAuthBundle(request.Bundle)
		if prepareErr != nil {
			return errInvalidCodexBundle
		}
		next := request.Version
		if next == 0 {
			next = connection.CredentialVersion + 1
		}
		checkpointStep = "encrypt_bundle"
		ciphertext, encryptErr := s.encryptor.Encrypt(context.Background(), prepared, []byte(connection.ID.String()))
		if encryptErr != nil {
			return encryptErr
		}
		checkpointStep = "store_bundle"
		if storeErr := connection.StoreBundle(tx, scope.TaskID, scope.Generation, next, ciphertext, label, request.Final, time.Now()); storeErr != nil {
			return storeErr
		}
		// Catalog and quota data are independent and optional.
		checkpointStep = "store_usage"
		if err := connection.StoreUsage(tx, limits, availableModels, time.Now()); err != nil {
			return err
		}
		version = connection.CredentialVersion
		checkpointStep = "commit_transaction"
		return nil
	})
	if errors.Is(err, errInvalidCodexBundle) {
		http.Error(w, "ChatGPT subscription credentials are invalid", http.StatusBadRequest)
		return
	}
	if err != nil {
		message, status := codexSubscriptionErrorResponse(err)
		if status == http.StatusInternalServerError {
			middleware.SetServerError(r.Context(), fmt.Errorf("codex checkpoint %s: %s", checkpointStep, safeCodexCheckpointCause(err)), map[string]string{
				"operation":        "checkpoint",
				"checkpoint_step":  checkpointStep,
				"organization_id":  scope.OrganizationID.String(),
				"connection_id":    scope.ConnectionID.String(),
				"task_id":          scope.TaskID.String(),
				"lease_generation": fmt.Sprint(scope.Generation),
			})
		}
		http.Error(w, message, status)
		return
	}
	_ = messages.PublishOrganizationCodexConnectionUpdated(database.DB(r.Context()), scope.OrganizationID)
	_ = writeJSON(w, http.StatusOK, map[string]any{"accepted": true, "version": version})
}

func (s *Server) handleRunnerCodexSubscriptionProgress(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateCodexSubscriptionRunner(w, r)
	if !ok {
		return
	}
	body, err := readCodexSubscriptionBody(w, r)
	if err != nil {
		return
	}
	var request codexProgressRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	verificationURL := strings.TrimSpace(request.VerificationURL)
	verificationCode := strings.TrimSpace(request.VerificationCode)
	if !strings.HasPrefix(verificationURL, "https://") || len(verificationURL) > 2048 || verificationCode == "" || len(verificationCode) > 64 {
		http.Error(w, "Invalid device sign-in details", http.StatusBadRequest)
		return
	}
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		connection, lockErr := models.LockScopedFactoryCodexConnection(tx, scope.OrganizationID, scope.ConnectionID)
		if lockErr != nil {
			return lockErr
		}
		if _, err := scopedCodexLease(tx, connection, scope); err != nil {
			return err
		}
		return connection.RecordDeviceCode(tx, scope.TaskID, scope.Generation, verificationURL, verificationCode, time.Now())
	})
	if err != nil {
		writeCodexSubscriptionError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"accepted": true})
}

func (s *Server) handleRunnerCodexSubscriptionRefresh(w http.ResponseWriter, r *http.Request) {
	body, err := readCodexSubscriptionBody(w, r)
	if err != nil {
		return
	}
	var request codexRefreshProxyRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	secret := strings.TrimSpace(request.RefreshToken)
	if secret == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// The provider can rotate its refresh token before the client disconnects.
	// Finish this bounded operation so a disconnect cannot discard the new token.
	operation, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), codexRefreshOperationTimeout)
	defer cancel()
	var accessToken string
	var idToken string
	var permanent *runneraction.CodexTokenRefreshError
	var changedOrganizationID uuid.UUID
	err = database.DB(operation).Transaction(func(tx *gorm.DB) error {
		lease, err := models.FindCodexRunLeaseBySecret(tx, secret)
		if err != nil {
			return err
		}
		connection, err := models.FindFactoryCodexConnectionByID(tx, lease.ConnectionID)
		if err != nil {
			return err
		}
		if lease.FactoryID == nil {
			return models.ErrCodexConnectionStale
		}
		locked, err := models.LockScopedFactoryCodexConnection(tx, connection.OrganizationID, lease.ConnectionID)
		if err != nil {
			return err
		}
		if _, err := scopedCodexLease(tx, locked, &runneraction.CodexSubscriptionScope{OrganizationID: locked.OrganizationID, FactoryID: *lease.FactoryID, ConnectionID: locked.ID, TaskID: lease.TaskID, Generation: lease.Generation}); err != nil {
			return err
		}
		if len(locked.BundleCiphertext) == 0 {
			return models.ErrCodexConnectionNotReady
		}
		plain, failure, changed, err := s.refreshStoredCodexBundle(operation, tx, locked, request.RejectedAccessToken)
		if err != nil {
			return err
		}
		if failure != nil {
			permanent = failure
			if changed {
				changedOrganizationID = locked.OrganizationID
			}
			return nil
		}
		if changed {
			changedOrganizationID = locked.OrganizationID
		}
		accessToken = runneraction.CodexAuthToken(plain, "access_token")
		idToken = runneraction.CodexAuthToken(plain, "id_token")
		return nil
	})
	if err == nil && changedOrganizationID != uuid.Nil {
		_ = messages.PublishOrganizationCodexConnectionUpdated(database.DB(operation), changedOrganizationID)
	}
	if err != nil {
		if errors.Is(err, models.ErrCodexConnectionStale) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if operation.Err() != nil {
			writeCodexRefreshFailure(w, operation.Err())
		} else {
			writeCodexRefreshFailure(w, err)
		}
		return
	}
	if permanent != nil {
		writeCodexRefreshFailure(w, permanent)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"id_token":      idToken,
		"refresh_token": secret,
	})
}

// The connection row is locked by the caller. Keep the provider exchange and
// encrypted persistence in the same transaction so parallel runs share a token.
func (s *Server) refreshStoredCodexBundle(ctx context.Context, tx *gorm.DB, connection *models.FactoryCodexConnection, rejectedAccessToken string) ([]byte, *runneraction.CodexTokenRefreshError, bool, error) {
	if connection.Status == models.CodexConnectionStatusReconnectRequired {
		return nil, &runneraction.CodexTokenRefreshError{Permanent: true, Code: "refresh_token_invalidated"}, false, nil
	}
	plain, err := s.encryptor.Decrypt(ctx, connection.BundleCiphertext, []byte(connection.ID.String()))
	if err != nil {
		return nil, nil, false, err
	}
	accessToken := runneraction.CodexAuthToken(plain, "access_token")
	if rejectedAccessToken != "" && rejectedAccessToken != accessToken {
		return plain, nil, false, nil
	}
	if rejectedAccessToken == "" && !runneraction.CodexAccessTokenNeedsRefresh(accessToken, time.Now()) {
		return plain, nil, false, nil
	}
	refreshed, err := s.refreshCodexToken(ctx, runneraction.CodexAuthToken(plain, "refresh_token"))
	var tokenErr *runneraction.CodexTokenRefreshError
	if errors.As(err, &tokenErr) && tokenErr.Permanent {
		if markErr := connection.MarkReconnectRequired(tx, time.Now()); markErr != nil {
			return nil, nil, false, markErr
		}
		return nil, tokenErr, true, nil
	}
	if err != nil {
		return nil, nil, false, err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = runneraction.CodexAuthToken(plain, "refresh_token")
	}
	updated, err := runneraction.ReplaceCodexAuthTokens(plain, refreshed.AccessToken, refreshed.RefreshToken, refreshed.IDToken)
	if err != nil {
		return nil, nil, false, err
	}
	prepared, label, err := runneraction.PrepareCodexAuthBundle(updated)
	if err != nil {
		return nil, nil, false, err
	}
	ciphertext, err := s.encryptor.Encrypt(ctx, prepared, []byte(connection.ID.String()))
	if err != nil {
		return nil, nil, false, err
	}
	if err := connection.StoreRefreshedBundle(tx, ciphertext, label, time.Now()); err != nil {
		return nil, nil, false, err
	}
	return prepared, nil, true, nil
}

func writeCodexRefreshFailure(w http.ResponseWriter, err error) {
	var tokenErr *runneraction.CodexTokenRefreshError
	if errors.As(err, &tokenErr) {
		if tokenErr.Permanent {
			code := tokenErr.Code
			if code == "" {
				code = "refresh_token_invalidated"
			}
			_ = writeJSON(w, http.StatusUnauthorized, map[string]any{"error": code})
			return
		}
		_ = writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "temporarily_unavailable"})
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		_ = writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "connection_busy"})
		return
	}
	writeCodexSubscriptionError(w, err)
}

func (s *Server) refreshCodexToken(ctx context.Context, refreshToken string) (runneraction.CodexTokenSet, error) {
	// Leave a persistence reserve even if row-lock acquisition used part of the
	// operation deadline. A depleted reserve prevents a new exchange from starting.
	deadline := time.Now().Add(codexRefreshExchangeTimeout)
	if operationDeadline, ok := ctx.Deadline(); ok && operationDeadline.Add(-codexRefreshPersistenceReserve).Before(deadline) {
		deadline = operationDeadline.Add(-codexRefreshPersistenceReserve)
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return runneraction.CodexTokenSet{}, &runneraction.CodexTokenRefreshError{}
	}
	if s.codexTokenRefresh != nil {
		return s.codexTokenRefresh(ctx, refreshToken)
	}
	return runneraction.RefreshCodexTokens(ctx, nil, refreshToken)
}

func (s *Server) authenticateCodexSubscriptionRunner(w http.ResponseWriter, r *http.Request) (*runneraction.CodexSubscriptionScope, bool) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	scope, err := runneraction.ParseCodexSubscriptionToken(s.jwt, token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return scope, true
}

func readCodexSubscriptionBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCodexSubscriptionBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return nil, err
	}
	return body, nil
}

func scopedCodexLease(tx *gorm.DB, connection *models.FactoryCodexConnection, scope *runneraction.CodexSubscriptionScope) (*models.FactoryCodexLease, error) {
	if connection.ID != scope.ConnectionID || connection.OrganizationID != scope.OrganizationID || connection.RevocationPendingAt != nil {
		return nil, models.ErrCodexConnectionStale
	}
	lease, err := models.FindCodexLeaseForTask(tx, connection.ID, scope.TaskID, scope.Generation)
	if err != nil {
		return nil, err
	}
	if lease.Login() {
		if scope.FactoryID != uuid.Nil || lease.FactoryID != nil {
			return nil, models.ErrCodexConnectionStale
		}
	} else if lease.FactoryID == nil || *lease.FactoryID != scope.FactoryID {
		return nil, models.ErrCodexConnectionStale
	}
	if lease.Login() {
		return lease, nil
	}
	task, err := models.FindRunnerTask(tx, lease.TaskID)
	if err != nil {
		return nil, err
	}
	if err := task.RequireSubscriptionLeaseContinuation(tx); err != nil {
		return nil, err
	}
	return lease, nil
}

func writeCodexSubscriptionError(w http.ResponseWriter, err error) {
	message, status := codexSubscriptionErrorResponse(err)
	http.Error(w, message, status)
}

func codexSubscriptionErrorResponse(err error) (string, int) {
	switch {
	case errors.Is(err, models.ErrCodexConnectionNotFound):
		return "ChatGPT subscription connection not found", http.StatusNotFound
	case errors.Is(err, models.ErrCodexConnectionStale):
		return "The ChatGPT sign-in update is out of date", http.StatusConflict
	case errors.Is(err, models.ErrCodexConnectionNotReady):
		return "Connect ChatGPT before this task can run", http.StatusConflict
	case errors.Is(err, models.ErrCodexConnectionNotEligible):
		return "Connect a ChatGPT subscription as an active workspace member", http.StatusForbidden
	case errors.Is(err, models.ErrCodexConnectionUnavailable):
		return err.Error(), http.StatusForbidden
	default:
		return "ChatGPT subscription request failed", http.StatusInternalServerError
	}
}

func safeCodexCheckpointCause(err error) string {
	// Error messages and PostgreSQL details can contain credentials or SQL values.
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && len(postgresError.Code) == 5 && strings.Trim(postgresError.Code, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ") == "" {
		return "PostgreSQL error (SQLSTATE " + postgresError.Code + ")"
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, sql.ErrConnDone, sql.ErrTxDone, driver.ErrBadConn, gorm.ErrRecordNotFound} {
		if errors.Is(err, cause) {
			return cause.Error()
		}
	}
	for errors.Unwrap(err) != nil {
		err = errors.Unwrap(err)
	}
	return fmt.Sprintf("unexpected error (%T)", err)
}
