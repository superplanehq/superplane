package contexts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"gorm.io/gorm"
)

func IntegrationCredentialIdentity(integration *models.Integration) string {
	identity := map[string]any{"configuration": integration.Configuration.Data(), "properties": integration.Properties}
	for key, value := range integration.Metadata.Data() {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "account") || strings.Contains(lower, "user") || strings.Contains(lower, "installation") || strings.Contains(lower, "workspace") || strings.Contains(lower, "organization") || strings.Contains(lower, "scope") || strings.Contains(lower, "owner") || strings.Contains(lower, "githubapp") || strings.Contains(lower, "repositor") {
			identity[key] = value
		}
	}
	encoded, _ := json.Marshal(identity)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func integrationCredentialIdentity(integration *models.Integration) string {
	return IntegrationCredentialIdentity(integration)
}

// RunnerIntegrationScope binds provider-side grants as well as the integration
// record. A queued job must not gain new repository access while it waits.
func RunnerIntegrationScope(tx *gorm.DB, integration *models.Integration) (string, error) {
	if integration.AppName != "github" && integration.AppName != "bitbucket" {
		return "", nil
	}
	identity := map[string]any{}
	binding, err := models.FindVCSProviderIntegrationBinding(tx, integration.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	if err == nil {
		identity["binding"] = map[string]any{"provider": binding.Provider, "organization_id": binding.OrganizationID, "installation_id": binding.InstallationID}
	}
	repositories, err := models.ListVCSProviderBindingRepositories(tx, integration.ID)
	if err != nil {
		return "", err
	}
	var ids []int64
	for _, repository := range repositories {
		ids = append(ids, repository.RepositoryID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	identity["repositories"] = ids
	if integration.AppName == "github" {
		names, err := models.ListEnabledGitHubContextRepositoryNames(tx, integration.ID)
		if err != nil {
			return "", err
		}
		sort.Strings(names)
		identity["context_repositories"] = names
	}
	encoded, _ := json.Marshal(identity)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func IntegrationCredentialKeyIdentity(tx *gorm.DB, encryptor crypto.Encryptor, integration *models.Integration) (string, error) {
	name := ""
	if integration.AppName == "github" {
		name = "pat"
	}
	if integration.AppName == "openrouter" {
		name = "apiKey"
	}
	if name == "" {
		return "", nil
	}
	secrets, err := models.ListIntegrationSecretsInTransaction(tx, integration.ID)
	if err != nil {
		return "", err
	}
	for _, secret := range secrets {
		if secret.Name != name {
			continue
		}
		plaintext, err := encryption.ForDB(tx, encryptor).Decrypt(context.Background(), secret.Value, []byte(integration.ID.String()))
		if err != nil {
			return "", err
		}
		digest := sha256.Sum256(plaintext)
		return hex.EncodeToString(digest[:]), nil
	}
	return "", nil
}

func (c *RunnerTaskContext) withCredentialRefreshes(payload []byte) ([]byte, error) {
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, err
	}
	descriptors := append([]runnerapi.CredentialRefresh(nil), c.refreshes...)
	if c.secrets != nil {
		descriptors = append(descriptors, c.secrets.refreshes...)
	}
	var task runnerapi.TaskPayload
	if err := json.Unmarshal(payload, &task); err != nil {
		return nil, err
	}
	for _, variable := range task.Environment {
		if variable.Name != "SUPERPLANE_RUN_TOKEN" && variable.Name != "SUPERPLANE_ARTIFACT_TOKEN" && variable.Name != "SUPERPLANE_MERGE_CONFIDENCE_TOKEN" {
			continue
		}
		claims, err := jwt.NewSigner(os.Getenv("JWT_SECRET")).ValidateAndGetClaims(variable.Value)
		if err != nil {
			return nil, fmt.Errorf("validate runner scoped credential: %w", err)
		}
		scope := map[string]string{}
		for key, value := range claims {
			if key == "iat" || key == "nbf" || key == "exp" {
				continue
			}
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid scoped credential claim")
			}
			scope[key] = text
		}
		descriptors = append(descriptors, runnerapi.CredentialRefresh{Version: 1, Kind: "scoped_token", OrganizationID: c.organizationID.String(), ResourceID: scope["canvas_run_id"], Recipient: scope["purpose"], Scope: scope, Current: variable.Value})
	}
	// File URLs can occur inside saved prompts, manifests, and command text.
	var stringsInPayload []string
	walkCredentialStrings(document, nil, func(path []string, value string) {
		if len(path) >= 3 && path[0] == "files" && path[2] == "content" {
			var embedded map[string]any
			if json.Unmarshal([]byte(value), &embedded) == nil {
				walkCredentialStrings(embedded, nil, func(_ []string, text string) { stringsInPayload = append(stringsInPayload, text) })
				return
			}
		}
		stringsInPayload = append(stringsInPayload, value)
	})
	seenURLs := map[string]bool{}
	var attachmentFactoryID, attachmentWorkOrderID uuid.UUID
	attachmentScopeResolved := false
	for _, value := range stringsInPayload {
		for _, raw := range blob.SignedFileURLs(value) {
			if seenURLs[raw] {
				continue
			}
			seenURLs[raw] = true
			id, ok := blob.FileIDFromSignedURL(raw)
			if !ok {
				continue
			}
			if !attachmentScopeResolved {
				var err error
				attachmentFactoryID, attachmentWorkOrderID, err = RunnerAttachmentScope(c.tx, c.organizationID, c.runID)
				if err != nil {
					return nil, err
				}
				attachmentScopeResolved = true
			}
			files, err := models.ListFilesByIDs(c.tx, []uuid.UUID{id})
			if err != nil {
				return nil, err
			}
			if len(files) != 1 || !files[0].IsDispatchable(c.organizationID, attachmentFactoryID, attachmentWorkOrderID) || files[0].Scope == blob.ScopeTask && attachmentWorkOrderID == uuid.Nil {
				return nil, fmt.Errorf("runner attachment is no longer available")
			}
			file := files[0]
			scope := map[string]string{"run_id": c.runID.String(), "factory_id": attachmentFactoryID.String(), "storage_key": file.StorageKey}
			if attachmentWorkOrderID != uuid.Nil {
				scope["work_order_id"] = attachmentWorkOrderID.String()
			}
			descriptors = append(descriptors, runnerapi.CredentialRefresh{Version: 1, Kind: "attachment", OrganizationID: c.organizationID.String(), ResourceID: id.String(), Recipient: file.Scope, Scope: scope, Current: raw})
		}
	}
	var bound []runnerapi.CredentialRefresh
	seenDescriptors := map[string]bool{}
	for _, descriptor := range descriptors {
		if descriptor.Current == "" {
			continue
		}
		identity, _ := json.Marshal(struct {
			Kind, ID, Current string
			Scope             map[string]string
		}{descriptor.Kind, descriptor.ResourceID, descriptor.Current, descriptor.Scope})
		if seenDescriptors[string(identity)] {
			continue
		}
		seenDescriptors[string(identity)] = true
		if descriptor.Kind == "mcp_header" || descriptor.Kind == "mcp_oauth" {
			descriptor.Fields = bindMCPCredentialFields(task, descriptor)
			if len(descriptor.Fields) > 0 {
				bound = append(bound, descriptor)
			}
			continue
		}
		walkCredentialStrings(document, nil, func(path []string, value string) {
			if descriptor.Kind == "attachment" && len(path) >= 3 && path[0] == "files" && path[2] == "content" {
				var embedded map[string]any
				if json.Unmarshal([]byte(value), &embedded) == nil {
					walkCredentialStrings(embedded, nil, func(nested []string, text string) {
						if strings.Contains(text, descriptor.Current) {
							descriptor.Fields = append(descriptor.Fields, runnerapi.CredentialField{Path: append([]string(nil), path...), DocumentPath: append([]string(nil), nested...), Original: descriptor.Current})
						}
					})
					return
				}
			}
			original := descriptor.Current
			encoding := ""
			if len(path) >= 3 && path[0] == "files" && path[2] == "content" && (descriptor.Kind == "mcp_header" || descriptor.Kind == "mcp_oauth") {
				encoded, _ := json.Marshal(original)
				original = string(encoded[1 : len(encoded)-1])
				encoding = "json_string"
			}
			if !strings.Contains(value, original) {
				return
			}
			if descriptor.Kind != "attachment" {
				if len(path) < 3 || path[0] != "environment" || path[2] != "value" || value != descriptor.Current {
					return
				}
			}
			descriptor.Fields = append(descriptor.Fields, runnerapi.CredentialField{Path: append([]string(nil), path...), Original: original, Encoding: encoding})
		})
		if len(descriptor.Fields) > 0 {
			bound = append(bound, descriptor)
		}
	}
	if len(bound) == 0 {
		return payload, nil
	}
	document["credential_refreshes"] = bound
	return json.Marshal(document)
}

// RunnerAttachmentScope derives authorization from the producing run, including
// indexed task links and ancestors for nested Run App invocations.
// File rows never grant their own workspace or task scope.
func RunnerAttachmentScope(tx *gorm.DB, organizationID, runID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	if runID == uuid.Nil {
		return uuid.Nil, uuid.Nil, models.ErrFileForeignReference
	}
	run, err := models.FindUnscopedCanvasRun(tx, runID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	canvas, err := models.FindCanvasInTransaction(tx, organizationID, run.WorkflowID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if canvas.FactoryID == nil {
		return uuid.Nil, uuid.Nil, models.ErrFileForeignReference
	}
	if _, err := models.FindFactory(tx, organizationID, *canvas.FactoryID); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	execution, err := models.FindWorkOrderExecutionForRun(tx, runID)
	if errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
		tasks, err := models.ResolveCanvasRunTasks(tx, canvas, []models.CanvasRun{*run})
		if err != nil {
			return uuid.Nil, uuid.Nil, err
		}
		task, ok := tasks[runID]
		if !ok {
			return *canvas.FactoryID, uuid.Nil, nil
		}
		return runnerAttachmentScopeForOrder(tx, organizationID, *canvas.FactoryID, task.ID)
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if execution.OrganizationID != organizationID {
		return uuid.Nil, uuid.Nil, models.ErrFileForeignReference
	}
	if execution.FactoryID != *canvas.FactoryID {
		return *canvas.FactoryID, uuid.Nil, nil
	}
	return runnerAttachmentScopeForOrder(tx, organizationID, *canvas.FactoryID, execution.WorkOrderID)
}

func runnerAttachmentScopeForOrder(tx *gorm.DB, organizationID, factoryID, orderID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	order, err := models.FindUnscopedWorkOrder(tx, orderID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if order.OrganizationID != organizationID || order.FactoryID != factoryID {
		return uuid.Nil, uuid.Nil, models.ErrFileForeignReference
	}
	return factoryID, order.ID, nil
}

func walkCredentialStrings(value any, path []string, visit func([]string, string)) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key != "credential_refreshes" {
				walkCredentialStrings(child, append(path, key), visit)
			}
		}
	case []any:
		for i, child := range typed {
			walkCredentialStrings(child, append(path, strconv.Itoa(i)), visit)
		}
	case string:
		visit(path, typed)
	}
}

func bindMCPCredentialFields(task runnerapi.TaskPayload, descriptor runnerapi.CredentialRefresh) []runnerapi.CredentialField {
	var fields []runnerapi.CredentialField
	for i, file := range task.Files {
		if file.Path != "workspace_mcp.json" {
			continue
		}
		var document struct {
			Servers []struct {
				Name    string            `json:"name"`
				URL     string            `json:"url"`
				Headers map[string]string `json:"headers"`
			} `json:"servers"`
		}
		if json.Unmarshal([]byte(file.Content), &document) != nil {
			continue
		}
		for j, server := range document.Servers {
			if server.Name != descriptor.Scope["name"] || server.URL != descriptor.Recipient {
				continue
			}
			header := descriptor.Scope["header"]
			if descriptor.Kind == "mcp_oauth" {
				header = "Authorization"
			}
			if !strings.Contains(server.Headers[header], descriptor.Current) {
				continue
			}
			fields = append(fields, runnerapi.CredentialField{Path: []string{"files", strconv.Itoa(i), "content"}, DocumentPath: []string{"servers", strconv.Itoa(j), "headers", header}, Original: descriptor.Current})
		}
	}
	return fields
}
