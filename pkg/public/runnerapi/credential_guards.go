package runnerapi

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	runnerpayload "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type credentialStorageError struct{ cause error }

func (e *credentialStorageError) Error() string {
	return "runner credential storage is temporarily unavailable"
}
func (e *credentialStorageError) Unwrap() error { return e.cause }

func credentialLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, models.ErrFactoryNotFound) || errors.Is(err, models.ErrFileNotFound) || errors.Is(err, models.ErrFileForeignReference) || errors.Is(err, models.ErrHostedLLMProviderNotFound) || errors.Is(err, models.ErrFactoryAgentResourceNotFound) || errors.Is(err, models.ErrFactoryPlanningSessionNotFound) || errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
		return errInvalidCredentialDescriptor
	}
	return &credentialStorageError{cause: err}
}

// validateRefreshResources performs no provider calls. When called by Accept
// or BeginDispatch, SHARE locks keep resource identity stable through commit.
func (s *Server) validateRefreshResources(tx *gorm.DB, task *models.RunnerTask, payload []byte) error {
	var saved runnerpayload.TaskPayload
	if json.Unmarshal(payload, &saved) != nil {
		return errInvalidCredentialDescriptor
	}
	lockedDB := func() *gorm.DB {
		return tx.Session(&gorm.Session{NewDB: true}).Clauses(clause.Locking{Strength: "SHARE", Options: "NOWAIT"}).Session(&gorm.Session{})
	}
	for _, descriptor := range saved.CredentialRefreshes {
		id, err := uuid.Parse(descriptor.ResourceID)
		if err != nil || descriptor.Version != 1 || descriptor.OrganizationID != task.OrganizationID.String() || len(descriptor.Fields) == 0 || descriptor.Scope == nil {
			return errInvalidCredentialDescriptor
		}
		switch descriptor.Kind {
		case "secret":
			if descriptor.Recipient != "organization" || descriptor.Scope["key"] == "" {
				return errInvalidCredentialDescriptor
			}
			if err := s.validateSecretCredential(lockedDB(), task.OrganizationID, id.String(), descriptor); err != nil {
				return err
			}
		case "integration":
			integration, err := models.FindIntegrationInTransaction(lockedDB(), task.OrganizationID, id)
			if err != nil {
				return credentialLookupError(err)
			}
			if integration.State != models.IntegrationStateReady || integration.AppName != descriptor.Recipient || contexts.IntegrationCredentialIdentity(integration) != descriptor.Scope["identity"] {
				return errInvalidCredentialDescriptor
			}
			permittedScope, err := contexts.RunnerIntegrationScope(tx, integration)
			if err != nil {
				return &credentialStorageError{cause: err}
			}
			if permittedScope != descriptor.Scope["permitted_scope"] {
				return errInvalidCredentialDescriptor
			}
			keyIdentity, err := contexts.IntegrationCredentialKeyIdentity(lockedDB(), s.encryptor, integration)
			if err != nil {
				return &credentialStorageError{cause: err}
			}
			if keyIdentity != descriptor.Scope["key_identity"] {
				return errInvalidCredentialDescriptor
			}
		case "attachment":
			file, err := models.FindFile(lockedDB(), id)
			if err != nil {
				return credentialLookupError(err)
			}
			factoryID, err := uuid.Parse(descriptor.Scope["factory_id"])
			workOrderID, _ := uuid.Parse(descriptor.Scope["work_order_id"])
			if err != nil || !file.IsDispatchable(task.OrganizationID, factoryID, workOrderID) || file.Scope == blob.ScopeTask && workOrderID == uuid.Nil || file.Scope != descriptor.Recipient || file.StorageKey != descriptor.Scope["storage_key"] {
				return errInvalidCredentialDescriptor
			}
			runID, err := uuid.Parse(descriptor.Scope["run_id"])
			if err != nil {
				return errInvalidCredentialDescriptor
			}
			if err := validateCredentialRun(tx, task, runID, factoryID.String()); err != nil {
				return err
			}
			currentFactory, currentOrder, err := contexts.RunnerAttachmentScope(tx, task.OrganizationID, runID)
			if err != nil {
				return credentialLookupError(err)
			}
			if currentFactory != factoryID || currentOrder != workOrderID {
				return errInvalidCredentialDescriptor
			}
		case "mcp_header", "mcp_oauth":
			factoryID, err := uuid.Parse(descriptor.Scope["factory_id"])
			if err != nil {
				return errInvalidCredentialDescriptor
			}
			factory, err := models.FindFactory(tx, task.OrganizationID, factoryID)
			if err != nil {
				return credentialLookupError(err)
			}
			resource, err := factory.FindAgentResource(lockedDB(), id)
			if err != nil {
				return credentialLookupError(err)
			}
			if !resource.Enabled || resource.Kind != models.FactoryAgentResourceKindMCPServer || resource.Config.Data().URL != descriptor.Recipient || resource.Config.Data().MCPAuth() != descriptor.Scope["auth"] {
				return errInvalidCredentialDescriptor
			}
			if descriptor.Kind == "mcp_header" {
				secret, err := models.FindSecretByIDInTransaction(lockedDB(), models.DomainTypeOrganization, task.OrganizationID, descriptor.Scope["secret_id"])
				if err != nil {
					return credentialLookupError(err)
				}
				allowed := false
				for _, header := range resource.Config.Data().Headers {
					if header.Name == descriptor.Scope["header"] && header.SecretName == secret.Name && header.SecretKey == descriptor.Scope["key"] {
						allowed = true
					}
				}
				if !allowed {
					return errInvalidCredentialDescriptor
				}
				if err := s.validateSecretCredential(lockedDB(), task.OrganizationID, secret.ID.String(), descriptor); err != nil {
					return err
				}
				continue
			}
			metadata := resource.OAuthMetadata.Data()
			connectedAt := ""
			if resource.OAuthConnectedAt != nil {
				connectedAt = resource.OAuthConnectedAt.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
			}
			if resource.OAuthState() != models.FactoryAgentResourceOAuthConnected || metadata.ClientID != descriptor.Scope["client_id"] || metadata.TokenEndpoint != descriptor.Scope["token_endpoint"] || metadata.AuthorizationEndpoint != descriptor.Scope["authorization_endpoint"] || metadata.Resource != descriptor.Scope["resource"] || connectedAt != descriptor.Scope["connected_at"] {
				return errInvalidCredentialDescriptor
			}
		case "hosted_provider":
			provider, err := models.FindHostedLLMProvider(lockedDB(), descriptor.Recipient)
			if err != nil {
				return credentialLookupError(err)
			}
			if provider.ID != id || !provider.CanRunHosted() || provider.BaseURL != descriptor.Scope["base_url"] {
				return errInvalidCredentialDescriptor
			}
			apiKey, err := llm.DecryptAPIKey(tx.Statement.Context, s.encryptor, provider.Provider, provider.APIKey)
			if err != nil {
				return &credentialStorageError{cause: err}
			}
			managementKey := ""
			if len(provider.ManagementKey) > 0 {
				managementKey, err = llm.DecryptManagementKey(tx.Statement.Context, s.encryptor, provider.Provider, provider.ManagementKey)
				if err != nil {
					return &credentialStorageError{cause: err}
				}
			}
			if llm.CredentialIdentity(apiKey, managementKey, provider.BaseURL) != descriptor.Scope["identity"] {
				return errInvalidCredentialDescriptor
			}
			runID, err := uuid.Parse(descriptor.Scope["run_id"])
			if err != nil {
				return errInvalidCredentialDescriptor
			}
			if err := validateCredentialRun(tx, task, runID, ""); err != nil {
				return err
			}
			run, err := models.FindUnscopedCanvasRun(tx, runID)
			if err != nil {
				return credentialLookupError(err)
			}
			canvas, err := models.FindCanvasInTransaction(tx, task.OrganizationID, run.WorkflowID)
			if err != nil {
				return credentialLookupError(err)
			}
			allowed, err := models.ModelIsSelectable(tx, task.OrganizationID, canvas.FactoryID, provider.Provider, models.UsageFundingSourceHosted, descriptor.Scope["model"])
			if err != nil {
				return &credentialStorageError{cause: err}
			}
			if !allowed {
				return errInvalidCredentialDescriptor
			}
			if err := models.AssertHostedRunAllowed(tx, task.OrganizationID, canvas.FactoryID); err != nil {
				return hostedCredentialError(err)
			}
		case "scoped_token":
			if descriptor.Scope["purpose"] != descriptor.Recipient || descriptor.Scope["org_id"] != task.OrganizationID.String() {
				return errInvalidCredentialDescriptor
			}
			if err := validateCredentialRun(tx, task, id, descriptor.Scope["factory_id"]); err != nil {
				return err
			}
			run, err := models.FindUnscopedCanvasRun(tx, id)
			if err != nil {
				return credentialLookupError(err)
			}
			if descriptor.Recipient == "planning_session" {
				session, err := models.FindPlanningSessionByRun(tx, id)
				if err != nil {
					return credentialLookupError(err)
				}
				if session.ID.String() != descriptor.Scope["session_id"] || session.OrganizationID != task.OrganizationID {
					return errInvalidCredentialDescriptor
				}
				continue
			}
			if descriptor.Recipient != "runner_artifact_upload" && descriptor.Recipient != "merge_confidence" {
				return errInvalidCredentialDescriptor
			}
			orderID, err := uuid.Parse(descriptor.Scope["work_order_id"])
			if err != nil {
				return errInvalidCredentialDescriptor
			}
			order, err := models.FindUnscopedWorkOrder(tx, orderID)
			if err != nil {
				return credentialLookupError(err)
			}
			if order.OrganizationID != task.OrganizationID || order.FactoryID.String() != descriptor.Scope["factory_id"] {
				return errInvalidCredentialDescriptor
			}
			executionID, err := uuid.Parse(descriptor.Scope["node_execution_id"])
			if err != nil {
				return errInvalidCredentialDescriptor
			}
			execution, err := models.FindNodeExecutionInTransaction(tx, run.WorkflowID, executionID)
			if err != nil {
				return credentialLookupError(err)
			}
			if execution.RunID != id || execution.State == models.CanvasNodeExecutionStateFinished {
				return errInvalidCredentialDescriptor
			}
		default:
			return errInvalidCredentialDescriptor
		}
	}
	return nil
}

func hostedCredentialError(err error) error {
	if errors.Is(err, models.ErrHostedCreditEmpty) || errors.Is(err, models.ErrFactoryHostedBudgetEmpty) || errors.Is(err, models.ErrHostedSubscriptionRequired) {
		return errInvalidCredentialDescriptor
	}
	return credentialLookupError(err)
}

func (s *Server) validateSecretCredential(tx *gorm.DB, organizationID uuid.UUID, id string, descriptor runnerpayload.CredentialRefresh) error {
	secret, err := models.FindSecretByIDInTransaction(tx, models.DomainTypeOrganization, organizationID, id)
	if err != nil {
		return credentialLookupError(err)
	}
	plaintext, err := s.encryptor.Decrypt(tx.Statement.Context, secret.Data, []byte(secret.Name))
	if err != nil {
		return &credentialStorageError{cause: err}
	}
	var keys map[string]string
	if json.Unmarshal(plaintext, &keys) != nil || keys[descriptor.Scope["key"]] == "" {
		return errInvalidCredentialDescriptor
	}
	if descriptor.Scope["prepared_valid_until"] != "" && len(descriptor.Fields) > 0 && keys[descriptor.Scope["key"]] != descriptor.Fields[0].Original {
		return errInvalidCredentialDescriptor
	}
	return nil
}

func validateCredentialRun(tx *gorm.DB, task *models.RunnerTask, runID uuid.UUID, factoryID string) error {
	run, err := models.FindUnscopedCanvasRun(tx, runID)
	if err != nil {
		return credentialLookupError(err)
	}
	if run.State == models.CanvasRunStateFinished || run.State == models.CanvasRunStateCancelling {
		return errInvalidCredentialDescriptor
	}
	canvas, err := models.FindCanvasInTransaction(tx, task.OrganizationID, run.WorkflowID)
	if err != nil {
		return credentialLookupError(err)
	}
	if factoryID != "" && (canvas.FactoryID == nil || canvas.FactoryID.String() != factoryID) {
		return errInvalidCredentialDescriptor
	}
	return nil
}
