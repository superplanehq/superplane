package public

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	maxCodexSubscriptionBodyBytes  = 256*1024 + 4096
	codexRefreshOperationTimeout   = 30 * time.Second
	codexRefreshExchangeTimeout    = 15 * time.Second
	codexRefreshPersistenceReserve = 5 * time.Second
)

var (
	errInvalidCodexBundle            = errors.New("chatgpt subscription credentials are invalid")
	errInvalidCodexCatalogCheckpoint = errors.New("catalog checkpoints can contain only usage metadata")
)

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
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handleRunnerCodexSubscriptionCredentials(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateCodexSubscriptionRunner(w, r)
	if !ok {
		return
	}
	var bundle any
	var version int64
	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
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
		plain, decryptErr := s.encryptor.Decrypt(r.Context(), connection.BundleCiphertext, []byte(connection.ID.String()))
		if decryptErr != nil {
			return decryptErr
		}
		if lease.Purpose == models.CodexLeasePurposeRun || lease.Catalog() {
			secret, secretErr := s.encryptor.Decrypt(r.Context(), lease.SecretCiphertext, []byte(connection.ID.String()))
			if secretErr != nil {
				return secretErr
			}
			plain, decryptErr = runneraction.CodexRunnerAuthBundle(plain, string(secret), time.Now())
			if decryptErr != nil {
				return decryptErr
			}
		}
		if unmarshalErr := json.Unmarshal(plain, &bundle); unmarshalErr != nil || bundle == nil {
			return errors.New("stored chatgpt subscription credentials are invalid")
		}
		version = connection.CredentialVersion
		return nil
	})
	if err != nil {
		writeCodexSubscriptionError(w, err)
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
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
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
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		connection, lockErr := models.LockScopedFactoryCodexConnection(tx, scope.OrganizationID, scope.ConnectionID)
		if lockErr != nil {
			return lockErr
		}
		lease, err := scopedCodexLease(tx, connection, scope)
		if err != nil {
			return err
		}
		if lease.Catalog() {
			for field := range fields {
				if field != "usage" {
					return errInvalidCodexCatalogCheckpoint
				}
			}
		}
		if lease.Purpose == models.CodexLeasePurposeRun || lease.Catalog() {
			if err := connection.StoreUsage(tx, limits, availableModels, time.Now()); err != nil {
				return err
			}
			version = connection.CredentialVersion
			return nil
		}
		prepared, label, prepareErr := runneraction.PrepareCodexAuthBundle(request.Bundle)
		if prepareErr != nil {
			return errInvalidCodexBundle
		}
		next := request.Version
		if next == 0 {
			next = connection.CredentialVersion + 1
		}
		ciphertext, encryptErr := s.encryptor.Encrypt(context.Background(), prepared, []byte(connection.ID.String()))
		if encryptErr != nil {
			return encryptErr
		}
		if storeErr := connection.StoreBundle(tx, scope.TaskID, scope.Generation, next, ciphertext, label, request.Final, time.Now()); storeErr != nil {
			return storeErr
		}
		// Catalog and quota data are independent and optional.
		if err := connection.StoreUsage(tx, limits, availableModels, time.Now()); err != nil {
			return err
		}
		version = connection.CredentialVersion
		return nil
	})
	if errors.Is(err, errInvalidCodexCatalogCheckpoint) {
		http.Error(w, "Catalog checkpoints can contain only usage metadata", http.StatusBadRequest)
		return
	}
	if errors.Is(err, errInvalidCodexBundle) {
		http.Error(w, "ChatGPT subscription credentials are invalid", http.StatusBadRequest)
		return
	}
	if err != nil {
		writeCodexSubscriptionError(w, err)
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
		lease, err := scopedCodexLease(tx, connection, scope)
		if err != nil {
			return err
		}
		if !lease.Login() {
			return models.ErrCodexConnectionStale
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
		lease, err := models.FindCodexRefreshLeaseBySecret(tx, secret)
		if err != nil {
			return err
		}
		connection, err := models.FindFactoryCodexConnectionByID(tx, lease.ConnectionID)
		if err != nil {
			return err
		}
		locked, err := models.LockScopedFactoryCodexConnection(tx, connection.OrganizationID, lease.ConnectionID)
		if err != nil {
			return err
		}
		scope := &runneraction.CodexSubscriptionScope{
			OrganizationID: locked.OrganizationID, ConnectionID: locked.ID,
			TaskID: lease.TaskID, Generation: lease.Generation,
		}
		if lease.FactoryID != nil {
			scope.FactoryID = *lease.FactoryID
		}
		if _, err := scopedCodexLease(tx, locked, scope); err != nil {
			return err
		}
		if locked.Status == models.CodexConnectionStatusReconnectRequired {
			permanent = &runneraction.CodexTokenRefreshError{Permanent: true, Code: "refresh_token_invalidated"}
			return nil
		}
		if len(locked.BundleCiphertext) == 0 {
			return models.ErrCodexConnectionNotReady
		}
		plain, err := s.encryptor.Decrypt(operation, locked.BundleCiphertext, []byte(locked.ID.String()))
		if err != nil {
			return err
		}
		accessToken = runneraction.CodexAuthToken(plain, "access_token")
		idToken = runneraction.CodexAuthToken(plain, "id_token")
		if !runneraction.CodexAccessTokenNeedsRefresh(accessToken, time.Now()) {
			return nil
		}
		refreshed, refreshErr := s.refreshCodexToken(operation, runneraction.CodexAuthToken(plain, "refresh_token"))
		var tokenErr *runneraction.CodexTokenRefreshError
		if errors.As(refreshErr, &tokenErr) && tokenErr.Permanent {
			if err := locked.MarkReconnectRequired(tx, time.Now()); err != nil {
				return err
			}
			permanent = tokenErr
			changedOrganizationID = locked.OrganizationID
			return nil
		}
		if refreshErr != nil {
			return refreshErr
		}
		if refreshed.RefreshToken == "" {
			refreshed.RefreshToken = runneraction.CodexAuthToken(plain, "refresh_token")
		}
		updated, err := runneraction.ReplaceCodexAuthTokens(plain, refreshed.AccessToken, refreshed.RefreshToken, refreshed.IDToken)
		if err != nil {
			return err
		}
		prepared, label, err := runneraction.PrepareCodexAuthBundle(updated)
		if err != nil {
			return err
		}
		ciphertext, err := s.encryptor.Encrypt(operation, prepared, []byte(locked.ID.String()))
		if err != nil {
			return err
		}
		if err := locked.StoreRefreshedBundle(tx, ciphertext, label, time.Now()); err != nil {
			return err
		}
		accessToken = refreshed.AccessToken
		if refreshed.IDToken != "" {
			idToken = refreshed.IDToken
		}
		return nil
	})
	if err == nil && changedOrganizationID != uuid.Nil {
		_ = messages.PublishOrganizationCodexConnectionUpdated(database.DB(operation), changedOrganizationID)
	}
	if err != nil {
		var tokenErr *runneraction.CodexTokenRefreshError
		if errors.As(err, &tokenErr) {
			_ = writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "temporarily_unavailable"})
			return
		}
		if errors.Is(err, models.ErrCodexConnectionStale) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		writeCodexSubscriptionError(w, err)
		return
	}
	if permanent != nil {
		code := permanent.Code
		if code == "" {
			code = "refresh_token_invalidated"
		}
		_ = writeJSON(w, http.StatusUnauthorized, map[string]any{"error": code})
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"id_token":      idToken,
		"refresh_token": secret,
	})
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
	switch lease.Purpose {
	case models.CodexLeasePurposeLogin:
		if scope.FactoryID != uuid.Nil || lease.FactoryID != nil || !time.Now().Before(lease.CreatedAt.Add(models.CodexLoginTimeout)) {
			return nil, models.ErrCodexConnectionStale
		}
	case models.CodexLeasePurposeCatalog:
		if scope.FactoryID != uuid.Nil || lease.FactoryID != nil || !time.Now().Before(lease.CreatedAt.Add(models.CodexCatalogTimeout)) {
			return nil, models.ErrCodexConnectionStale
		}
	case models.CodexLeasePurposeRun:
		if lease.FactoryID == nil || *lease.FactoryID == uuid.Nil || *lease.FactoryID != scope.FactoryID {
			return nil, models.ErrCodexConnectionStale
		}
	default:
		return nil, models.ErrCodexConnectionStale
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
	switch {
	case errors.Is(err, models.ErrCodexConnectionNotFound):
		http.Error(w, "ChatGPT subscription connection not found", http.StatusNotFound)
	case errors.Is(err, models.ErrCodexConnectionStale):
		http.Error(w, "The ChatGPT sign-in update is out of date", http.StatusConflict)
	case errors.Is(err, models.ErrCodexConnectionNotReady):
		http.Error(w, "Connect ChatGPT before this task can run", http.StatusConflict)
	case errors.Is(err, models.ErrCodexConnectionNotEligible), errors.Is(err, models.ErrCodexConnectionUnavailable):
		http.Error(w, "Connect a ChatGPT subscription as an active workspace member", http.StatusForbidden)
	default:
		http.Error(w, "ChatGPT subscription request failed", http.StatusInternalServerError)
	}
}
