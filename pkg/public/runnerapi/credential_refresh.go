package runnerapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	componentrunner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/openrouter"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	runnerpayload "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

var errInvalidCredentialDescriptor = errors.New("runner credential is no longer authorized")

type preparationHTTP struct {
	ctx        context.Context
	http       core.HTTPContext
	retryAfter time.Duration
	retryable  bool
}

func (h *preparationHTTP) Do(request *http.Request) (*http.Response, error) {
	response, err := h.http.Do(request.Clone(h.ctx))
	h.retryable = err != nil
	if response != nil {
		h.retryable = response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
		value := response.Header.Get("Retry-After")
		if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
			h.retryAfter = max(h.retryAfter, time.Duration(seconds)*time.Second)
		}
		if at, err := http.ParseTime(value); err == nil {
			h.retryAfter = max(h.retryAfter, time.Until(at))
		}
	}
	return response, err
}

func (s *Server) refreshTaskPayload(ctx context.Context, task *models.RunnerTask, preparation *models.RunnerTaskPreparation, httpClient *preparationHTTP) ([]byte, error) {
	payload, err := s.encryptor.Decrypt(ctx, task.PayloadCiphertext, []byte(task.ID.String()))
	if err != nil {
		return nil, errInvalidCredentialDescriptor
	}
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil || document == nil {
		return nil, errInvalidCredentialDescriptor
	}
	if err := s.validateRefreshResources(database.DB(ctx), task, payload); err != nil {
		return nil, err
	}
	var saved runnerpayload.TaskPayload
	if err := json.Unmarshal(payload, &saved); err != nil {
		return nil, errInvalidCredentialDescriptor
	}
	timeout := time.Duration(runnerpayload.DefaultExecutionTimeoutSeconds) * time.Second
	if saved.ExecutionTimeoutSeconds != nil && *saved.ExecutionTimeoutSeconds > 0 {
		timeout = time.Duration(*saved.ExecutionTimeoutSeconds) * time.Second
	}
	for i, descriptor := range saved.CredentialRefreshes {
		if descriptor.Version != 1 || descriptor.OrganizationID != task.OrganizationID.String() || len(descriptor.Fields) == 0 {
			return nil, errInvalidCredentialDescriptor
		}
		value, err := s.refreshCredential(ctx, task, preparation, descriptor, timeout, httpClient)
		if err != nil {
			return nil, err
		}
		if _, ok := descriptor.Scope["prepared_valid_until"]; !ok {
			descriptor.Scope["prepared_valid_until"] = time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339Nano)
		}
		for j, field := range descriptor.Fields {
			replacement := value
			if field.Encoding == "json_string" {
				encoded, _ := json.Marshal(value)
				replacement = string(encoded[1 : len(encoded)-1])
			}
			if field.Encoding != "" && field.Encoding != "json_string" {
				return nil, errInvalidCredentialDescriptor
			}
			if err := replaceCredentialField(document, field, replacement); err != nil {
				return nil, err
			}
			saved.CredentialRefreshes[i].Fields[j].Original = replacement
		}
	}
	if len(saved.CredentialRefreshes) > 0 {
		document["credential_refreshes"] = saved.CredentialRefreshes
	}
	return json.Marshal(document)
}

func (s *Server) refreshCredential(ctx context.Context, task *models.RunnerTask, preparation *models.RunnerTaskPreparation, descriptor runnerpayload.CredentialRefresh, timeout time.Duration, httpClient *preparationHTTP) (string, error) {
	id, err := uuid.Parse(descriptor.ResourceID)
	if err != nil {
		return "", errInvalidCredentialDescriptor
	}
	db := database.DB(ctx)
	switch descriptor.Kind {
	case "secret":
		if descriptor.Recipient != "organization" {
			return "", errInvalidCredentialDescriptor
		}
		return s.refreshOrganizationSecret(ctx, task.OrganizationID, id, descriptor.Scope["key"])
	case "integration":
		integration, err := models.FindIntegrationInTransaction(db, task.OrganizationID, id)
		if err != nil || integration.State != models.IntegrationStateReady || integration.AppName != descriptor.Recipient || contexts.IntegrationCredentialIdentity(integration) != descriptor.Scope["identity"] {
			return "", errInvalidCredentialDescriptor
		}
		implementation, err := s.registry.GetIntegration(integration.AppName)
		if err != nil {
			return "", errInvalidCredentialDescriptor
		}
		provider, ok := registry.UnwrapIntegration(implementation).(core.IntegrationSecretProvider)
		if !ok {
			return "", errInvalidCredentialDescriptor
		}
		resolved, err := provider.ResolveSecrets(core.IntegrationSecretContext{Logger: logging.ForIntegration(*integration), HTTP: httpClient, Integration: contexts.NewIntegrationContext(db, nil, integration, s.encryptor, s.registry, nil)})
		if err != nil {
			return "", err
		}
		value, ok := resolved.Values[descriptor.Scope["key"]]
		if !ok || len(value) == 0 {
			return "", errInvalidCredentialDescriptor
		}
		return string(value), nil
	case "scoped_token":
		descriptor.Scope["prepared_valid_until"] = time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339Nano)
		return s.refreshScopedToken(ctx, task, descriptor, timeout)
	case "attachment":
		file, err := models.FindFile(db, id)
		factoryID, parseErr := uuid.Parse(descriptor.Scope["factory_id"])
		workOrderID, _ := uuid.Parse(descriptor.Scope["work_order_id"])
		if err != nil || parseErr != nil || !file.IsDispatchable(task.OrganizationID, factoryID, workOrderID) || file.Scope == blob.ScopeTask && workOrderID == uuid.Nil || file.Scope != descriptor.Recipient || file.StorageKey != descriptor.Scope["storage_key"] {
			return "", errInvalidCredentialDescriptor
		}
		descriptor.Scope["prepared_valid_until"] = time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339Nano)
		return storedfiles.DownloadURL(ctx, blob.Current(), file, timeout+30*time.Minute)
	case "mcp_header", "mcp_oauth":
		factoryID, err := uuid.Parse(descriptor.Scope["factory_id"])
		if err != nil {
			return "", errInvalidCredentialDescriptor
		}
		factory, err := models.FindFactory(db, task.OrganizationID, factoryID)
		if err != nil {
			return "", errInvalidCredentialDescriptor
		}
		resource, err := factory.FindAgentResource(db, id)
		if err != nil || !resource.Enabled || resource.Kind != models.FactoryAgentResourceKindMCPServer || resource.Config.Data().URL != descriptor.Recipient || resource.Config.Data().MCPAuth() != descriptor.Scope["auth"] {
			return "", errInvalidCredentialDescriptor
		}
		if descriptor.Kind == "mcp_header" {
			secretID, err := uuid.Parse(descriptor.Scope["secret_id"])
			if err != nil {
				return "", errInvalidCredentialDescriptor
			}
			secret, err := models.FindSecretByIDInTransaction(db, models.DomainTypeOrganization, task.OrganizationID, secretID.String())
			if err != nil {
				return "", errInvalidCredentialDescriptor
			}
			allowed := false
			for _, header := range resource.Config.Data().Headers {
				if header.Name == descriptor.Scope["header"] && header.SecretName == secret.Name && header.SecretKey == descriptor.Scope["key"] {
					allowed = true
				}
			}
			if !allowed {
				return "", errInvalidCredentialDescriptor
			}
			return s.refreshOrganizationSecret(ctx, task.OrganizationID, secretID, descriptor.Scope["key"])
		}
		metadata := resource.OAuthMetadata.Data()
		connectedAt := ""
		if resource.OAuthConnectedAt != nil {
			connectedAt = resource.OAuthConnectedAt.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
		}
		if resource.OAuthState() != models.FactoryAgentResourceOAuthConnected || metadata.ClientID != descriptor.Scope["client_id"] || metadata.TokenEndpoint != descriptor.Scope["token_endpoint"] || metadata.AuthorizationEndpoint != descriptor.Scope["authorization_endpoint"] || metadata.Resource != descriptor.Scope["resource"] || connectedAt != descriptor.Scope["connected_at"] {
			return "", errInvalidCredentialDescriptor
		}
		token, err := mcp.MintFactoryAgentResourceAccessTokenForRunner(ctx, s.encryptor, httpClient, db, resource, timeout)
		if expiry, parseErr := time.Parse(time.RFC3339, resource.OAuthMetadata.Data().AccessTokenExpiresAt); parseErr == nil {
			descriptor.Scope["prepared_valid_until"] = expiry.Add(-mcp.RunnerAccessTokenMinimumLifetime(timeout)).UTC().Format(time.RFC3339Nano)
		}
		return token, err
	case "hosted_provider":
		provider, err := models.RequireEnabledHostedLLMProvider(db, descriptor.Recipient)
		if err != nil || provider.ID != id || provider.BaseURL != descriptor.Scope["base_url"] {
			return "", errInvalidCredentialDescriptor
		}
		runID, err := uuid.Parse(descriptor.Scope["run_id"])
		if err != nil {
			return "", errInvalidCredentialDescriptor
		}
		run, err := models.FindUnscopedCanvasRun(db, runID)
		if err != nil || run.State == models.CanvasRunStateFinished || run.State == models.CanvasRunStateCancelling {
			return "", errInvalidCredentialDescriptor
		}
		canvas, err := models.FindCanvasInTransaction(db, task.OrganizationID, run.WorkflowID)
		if err != nil {
			return "", errInvalidCredentialDescriptor
		}
		hosted := contexts.NewHostedLLMContext(db, s.encryptor, task.OrganizationID, canvas.FactoryID)
		if err := hosted.AssertCreditAvailable(); err != nil {
			return "", hostedCredentialError(err)
		}
		if err := hosted.AssertModelSelectable(descriptor.Recipient, "hosted", descriptor.Scope["model"]); err != nil {
			return "", errInvalidCredentialDescriptor
		}
		if descriptor.Recipient == models.UsageProviderOpenRouter {
			descriptor.Scope["prepared_valid_until"] = time.Now().Add(openrouter.ChildKeyTTLSlack - time.Minute).UTC().Format(time.RFC3339Nano)
			return s.mintRunnerProviderKey(ctx, task, preparation, provider, timeout, httpClient)
		}
		return llm.DecryptAPIKey(ctx, s.encryptor, provider.Provider, provider.APIKey)
	default:
		return "", errInvalidCredentialDescriptor
	}
}

func preparedPayloadValidUntil(payload []byte) time.Time {
	var saved runnerpayload.TaskPayload
	var validUntil time.Time
	if json.Unmarshal(payload, &saved) != nil {
		return time.Now()
	}
	for _, descriptor := range saved.CredentialRefreshes {
		expiry, err := time.Parse(time.RFC3339Nano, descriptor.Scope["prepared_valid_until"])
		if err != nil {
			expiry = time.Now().Add(15 * time.Minute)
		}
		if validUntil.IsZero() || expiry.Before(validUntil) {
			validUntil = expiry
		}
	}
	if validUntil.IsZero() {
		return time.Now().Add(15 * time.Minute)
	}
	return validUntil
}

func (s *Server) refreshOrganizationSecret(ctx context.Context, organizationID, id uuid.UUID, key string) (string, error) {
	secret, err := models.FindSecretByIDInTransaction(database.DB(ctx), models.DomainTypeOrganization, organizationID, id.String())
	if err != nil || key == "" {
		return "", errInvalidCredentialDescriptor
	}
	plaintext, err := s.encryptor.Decrypt(ctx, secret.Data, []byte(secret.Name))
	if err != nil {
		return "", errInvalidCredentialDescriptor
	}
	var values map[string]string
	if err := json.Unmarshal(plaintext, &values); err != nil || values[key] == "" {
		return "", errInvalidCredentialDescriptor
	}
	return values[key], nil
}

func (s *Server) refreshScopedToken(ctx context.Context, task *models.RunnerTask, descriptor runnerpayload.CredentialRefresh, timeout time.Duration) (string, error) {
	scope := descriptor.Scope
	if scope["org_id"] != task.OrganizationID.String() || scope["purpose"] != descriptor.Recipient || scope["canvas_run_id"] != descriptor.ResourceID {
		return "", errInvalidCredentialDescriptor
	}
	runID, err := uuid.Parse(scope["canvas_run_id"])
	if err != nil {
		return "", errInvalidCredentialDescriptor
	}
	run, err := models.FindUnscopedCanvasRun(database.DB(ctx), runID)
	if err != nil || run.State == models.CanvasRunStateFinished || run.State == models.CanvasRunStateCancelling {
		return "", errInvalidCredentialDescriptor
	}
	canvas, err := models.FindCanvasInTransaction(database.DB(ctx), task.OrganizationID, run.WorkflowID)
	if err != nil || canvas.FactoryID == nil || canvas.FactoryID.String() != scope["factory_id"] {
		return "", errInvalidCredentialDescriptor
	}
	if _, err := models.FindFactory(database.DB(ctx), task.OrganizationID, *canvas.FactoryID); err != nil {
		return "", errInvalidCredentialDescriptor
	}
	switch descriptor.Recipient {
	case componentrunner.PlanningSessionTokenPurpose:
		session, err := models.FindPlanningSessionByRun(database.DB(ctx), runID)
		if err != nil || session.ID.String() != scope["session_id"] || session.OrganizationID != task.OrganizationID || session.FactoryID != *canvas.FactoryID {
			return "", errInvalidCredentialDescriptor
		}
	case componentrunner.ArtifactUploadTokenPurpose, componentrunner.MergeConfidenceTokenPurpose:
		orderID, err := uuid.Parse(scope["work_order_id"])
		if err != nil {
			return "", errInvalidCredentialDescriptor
		}
		order, err := models.FindUnscopedWorkOrder(database.DB(ctx), orderID)
		if err != nil || order.OrganizationID != task.OrganizationID || order.FactoryID != *canvas.FactoryID {
			return "", errInvalidCredentialDescriptor
		}
		executionID, err := uuid.Parse(scope["node_execution_id"])
		if err != nil {
			return "", errInvalidCredentialDescriptor
		}
		execution, err := models.FindNodeExecutionInTransaction(database.DB(ctx), run.WorkflowID, executionID)
		if err != nil || execution.RunID != runID || execution.State == models.CanvasNodeExecutionStateFinished {
			return "", errInvalidCredentialDescriptor
		}
	default:
		return "", errInvalidCredentialDescriptor
	}
	return s.signer.GenerateWithClaims(timeout+30*time.Minute, scope)
}

func replaceCredentialField(document map[string]any, field runnerpayload.CredentialField, replacement string) error {
	if len(field.Path) < 1 || field.Original == "" || replacement == "" {
		return errInvalidCredentialDescriptor
	}
	var current any = document
	for _, part := range field.Path[:len(field.Path)-1] {
		switch value := current.(type) {
		case map[string]any:
			current = value[part]
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(value) {
				return errInvalidCredentialDescriptor
			}
			current = value[i]
		default:
			return errInvalidCredentialDescriptor
		}
	}
	key := field.Path[len(field.Path)-1]
	var value string
	var setValue func(string)
	switch parent := current.(type) {
	case map[string]any:
		var ok bool
		value, ok = parent[key].(string)
		if !ok {
			return errInvalidCredentialDescriptor
		}
		setValue = func(next string) { parent[key] = next }
	case []any:
		i, err := strconv.Atoi(key)
		if err != nil || i < 0 || i >= len(parent) {
			return errInvalidCredentialDescriptor
		}
		var ok bool
		value, ok = parent[i].(string)
		if !ok {
			return errInvalidCredentialDescriptor
		}
		setValue = func(next string) { parent[i] = next }
	default:
		return errInvalidCredentialDescriptor
	}
	if len(field.DocumentPath) > 0 {
		var embedded map[string]any
		if err := json.Unmarshal([]byte(value), &embedded); err != nil {
			return errInvalidCredentialDescriptor
		}
		if err := replaceCredentialField(embedded, runnerpayload.CredentialField{Path: field.DocumentPath, Original: field.Original}, replacement); err != nil {
			return err
		}
		encoded, err := json.Marshal(embedded)
		if err != nil {
			return err
		}
		setValue(string(encoded))
		return nil
	}
	if !strings.Contains(value, field.Original) {
		return errInvalidCredentialDescriptor
	}
	setValue(strings.ReplaceAll(value, field.Original, replacement))
	return nil
}

func (s *Server) mintRunnerProviderKey(ctx context.Context, task *models.RunnerTask, preparation *models.RunnerTaskPreparation, provider *models.HostedLLMProvider, timeout time.Duration, httpClient core.HTTPContext) (string, error) {
	db := database.DB(ctx)
	previous, err := models.ListUnacceptedRunnerCredentials(db, task.ID)
	if err != nil {
		return "", err
	}
	for i := range previous {
		if err := s.cleanupRunnerCredential(ctx, &previous[i], httpClient); err != nil {
			return "", err
		}
	}
	managementKey, err := llm.DecryptManagementKey(ctx, s.encryptor, provider.Provider, provider.ManagementKey)
	if err != nil {
		return "", errInvalidCredentialDescriptor
	}
	apiKey, err := llm.DecryptAPIKey(ctx, s.encryptor, provider.Provider, provider.APIKey)
	if err != nil {
		return "", &credentialStorageError{cause: err}
	}
	now := time.Now()
	cleanup := &models.RunnerCredentialCleanup{ID: uuid.New(), TaskID: task.ID, AttemptID: *preparation.AttemptID, Provider: provider.Provider, ProviderResourceID: provider.ID, IssuerCredentialIdentity: llm.CredentialIdentity(apiKey, managementKey, provider.BaseURL), Name: "superplane-runner-" + task.ID.String() + "-" + preparation.AttemptID.String(), ExpiresAt: now.Add(timeout + openrouter.ChildKeyTTLSlack), NextRetryAt: now, CreatedAt: now, UpdatedAt: now}
	if err := cleanup.Create(db); err != nil {
		return "", err
	}
	client := openrouter.NewManagementClient(httpClient, managementKey)
	created, err := client.CreateKeyWithContext(ctx, openrouter.CreateKeyRequest{Name: cleanup.Name, ExpiresAt: openrouter.FormatKeyExpiresAt(cleanup.ExpiresAt)})
	if err != nil {
		return "", err
	}
	if err := cleanup.RecordHash(db, created.Hash); err != nil {
		return "", err
	}
	return created.Key, nil
}

func (s *Server) cleanupRunnerCredential(ctx context.Context, record *models.RunnerCredentialCleanup, httpClient core.HTTPContext) error {
	db := database.DB(ctx)
	if !record.ExpiresAt.After(time.Now()) {
		return record.Complete(db, time.Now())
	}
	provider, err := models.FindHostedLLMProvider(db, record.Provider)
	if err != nil || provider.ID != record.ProviderResourceID {
		return errInvalidCredentialDescriptor
	}
	managementKey, err := llm.DecryptManagementKey(ctx, s.encryptor, provider.Provider, provider.ManagementKey)
	if err != nil {
		return err
	}
	apiKey, err := llm.DecryptAPIKey(ctx, s.encryptor, provider.Provider, provider.APIKey)
	if err != nil {
		return err
	}
	if record.IssuerCredentialIdentity == "" || record.IssuerCredentialIdentity != llm.CredentialIdentity(apiKey, managementKey, provider.BaseURL) {
		return errInvalidCredentialDescriptor
	}
	client := openrouter.NewManagementClient(httpClient, managementKey)
	if record.KeyHash != "" {
		if err := client.DeleteKeyWithContext(ctx, record.KeyHash); err != nil {
			return err
		}
		return record.Complete(db, time.Now())
	}
	var hashes []string
	for offset := 0; ; {
		page, err := client.ListKeys(ctx, offset)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			break
		}
		for _, key := range page {
			if key.Name == record.Name {
				hashes = append(hashes, key.Hash)
			}
		}
		offset += len(page)
	}
	if len(hashes) == 0 {
		return record.Retry(db, time.Now())
	}
	for _, hash := range hashes {
		if err := client.DeleteKeyWithContext(ctx, hash); err != nil {
			return err
		}
	}
	return record.Complete(db, time.Now())
}
