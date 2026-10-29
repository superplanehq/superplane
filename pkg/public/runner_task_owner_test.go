package public

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/taskowner"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
)

// This boundary catches capability attachment accidentally gated on visual
// evidence, and proves that the delivered credential reaches both owner APIs.
func TestTaskLinkedRunOwnsTaskWithoutVisualEvidence(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	db := database.DB(t.Context())
	ctx, factory, order := mustTaskLinkedAgentRun(t, r)
	env := runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{{Name: runner.EnvSuperplaneRunnerToken, Value: "untrusted"}}, 60)
	token := runnerTokenFrom(env)
	require.NotEmpty(t, token)
	require.NotEqual(t, "untrusted", token)
	assert.False(t, runner.HasArtifactUploadToken(env))
	assert.False(t, runner.HasMergeConfidenceToken(env))
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	t.Run("code history follows the saved task repository", func(t *testing.T) {
		integration := models.NewIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), nil)
		integration.State = models.IntegrationStateReady
		setupState := datatypes.NewJSONType(models.SetupState{})
		integration.SetupState = &setupState
		integration.Properties = datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
			{Name: common.PropertyAuthMethod, Value: common.AuthMethodPAT},
			{Name: common.PropertyOwner, Value: "acme"},
			{Name: common.PropertyOwnerType, Value: common.OwnerTypeOrganization},
		})
		require.NoError(t, db.Create(integration).Error)
		secrets := contexts.NewIntegrationSecretStorage(db, r.Encryptor, integration)
		require.NoError(t, secrets.Create(core.IntegrationSecretDefinition{Name: common.SecretPAT, Value: "test-token"}))
		currentRepository := "acme/current"
		integrationID := integration.ID.String()
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			AppRepository: &currentRepository, VCSIntegrationID: &integrationID,
		}))
		originalRepository, emptyRepository, blankRepository := "acme/original", "", "  "
		for _, test := range []struct {
			name               string
			savedRepository    *string
			expectedRepository string
		}{
			{"saved repository", &originalRepository, "acme/original"},
			{"no saved repository", nil, "acme/current"},
			{"empty saved repository", &emptyRepository, "acme/current"},
			{"blank saved repository", &blankRepository, "acme/current"},
		} {
			t.Run(test.name, func(t *testing.T) {
				require.NoError(t, db.Model(order).Update("repository", test.savedRepository).Error)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/runner/task/suggest-owners", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				rec := httptest.NewRecorder()
				target, ok := server.runnerOwnerTarget(runner.ScopeOwnerRead)(rec, req)
				require.True(t, ok, rec.Body.String())
				history, err := server.taskOwnerHistory(t.Context(), target.factory, target.repository)
				require.NoError(t, err)
				require.IsType(t, taskowner.GitHubHistory{}, history)
				assert.Equal(t, test.expectedRepository, history.(taskowner.GitHubHistory).Repository)
			})
		}
		empty := ""
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			AppRepository: &empty, VCSIntegrationID: &empty,
		}))
		require.NoError(t, db.Model(order).Update("repository", "").Error)
	})
	owner := mustTaskOwnerMember(t, r, "implementation-dev")
	rec := suggestRunnerOwners(t, server, token, `{"paths":["./pkg/","../secret"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{"pkg"}, decodeSuggestOwners(t, rec).Paths)
	channel, queueName := taskOwnerUpdateQueue(t)
	rec = assignRunnerOwner(t, server, token, `{"user":"implementation-dev","reason":"Changed pkg/runner. https://example.com"}`)
	assertAssignedOwner(t, rec, owner.ID)
	delivery, published, err := channel.Get(queueName, true)
	require.NoError(t, err)
	require.True(t, published)
	var update pb.FactoryWorkOrderUpdatedMessage
	require.NoError(t, proto.Unmarshal(delivery.Body, &update))
	assert.Equal(t, order.ID.String(), update.OrderId)
	assert.Equal(t, factoryevents.EventTypeOrderAssigneesUpdated, update.Reason)
	persisted, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{owner.ID}, persisted.AssigneeIDs())
	source, reason := latestOwnerEvent(t, persisted)
	assert.Equal(t, models.TaskOwnerSourceAgent, source)
	assert.Equal(t, "Changed pkg/runner.", reason)
	rec = suggestRunnerOwners(t, server, token, `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, owner.ID, decodeSuggestOwners(t, rec).CurrentOwners[0].UserID)

	signer := jwt.NewSigner("test")
	scope, err := runner.ParseRunnerToken(signer, token)
	require.NoError(t, err)
	for _, field := range []string{"organization", "factory", "task", "canvas", "run", "execution", "node"} {
		t.Run("mismatched "+field, func(t *testing.T) {
			wrong := *scope
			switch field {
			case "organization":
				wrong.OrganizationID = uuid.New()
			case "factory":
				wrong.FactoryID = uuid.New()
			case "task":
				wrong.WorkOrderID = uuid.New()
			case "canvas":
				wrong.CanvasID = uuid.New()
			case "run":
				wrong.CanvasRunID = uuid.New()
			case "execution":
				wrong.NodeExecutionID = uuid.New()
			case "node":
				wrong.NodeID = "other"
			}
			wrongToken, err := runner.MintRunnerToken(signer, wrong, time.Minute)
			require.NoError(t, err)
			rec := assignRunnerOwner(t, server, wrongToken, `{"user":"implementation-dev","reason":"Owns pkg."}`)
			require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		})
	}
	eventsBefore, err := persisted.ListEvents(db, 0, nil)
	require.NoError(t, err)
	for _, invalid := range []struct {
		name, token, body string
		status            int
	}{
		{"invalid member", token, `{"user":"` + uuid.NewString() + `","reason":"Owns pkg."}`, http.StatusBadRequest},
		{"invalid reason", token, `{"user":"implementation-dev","reason":"https://example.com"}`, http.StatusBadRequest},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			rec := assignRunnerOwner(t, server, invalid.token, invalid.body)
			require.Equal(t, invalid.status, rec.Code, rec.Body.String())
		})
	}
	for _, claims := range []map[string]string{{"purpose": runner.ArtifactUploadTokenPurpose}, {"purpose": runner.RunnerTokenPurpose}} {
		invalidToken, err := signer.GenerateWithClaims(-time.Second, claims)
		if claims["purpose"] == runner.ArtifactUploadTokenPurpose {
			invalidToken, err = signer.GenerateWithClaims(time.Minute, claims)
		}
		require.NoError(t, err)
		rec := assignRunnerOwner(t, server, invalidToken, `{"user":"implementation-dev","reason":"Owns pkg."}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}
	for _, route := range []string{"/api/v1/runner/tasks/owner", "/api/v1/runner/planning-sessions/owner", "/api/v1/runner/artifacts/links", "/api/v1/runner/artifacts/markdown", "/api/v1/runner/merge-confidence/checks"} {
		rec := postPlanningRunner(t, server, token, route, `{}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
	eventsAfter, err := persisted.ListEvents(db, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, eventsBefore, eventsAfter)
	for _, result := range []string{models.FactoryWorkOrderResultManualRejected, models.FactoryWorkOrderResultArchived} {
		require.NoError(t, db.Model(persisted).Updates(map[string]any{"state": models.FactoryWorkOrderStateClosed, "result": result}).Error)
		eventsBefore, err := persisted.ListEvents(db, 0, nil)
		require.NoError(t, err)
		for _, rec := range []*httptest.ResponseRecorder{
			assignRunnerOwner(t, server, token, `{"user":"implementation-dev","reason":"Owns pkg."}`),
			suggestRunnerOwners(t, server, token, `{}`),
		} {
			require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		}
		assert.False(t, runner.HasRunnerToken(runner.AttachRunnerEnv(ctx, nil, 60)))
		eventsAfter, err := persisted.ListEvents(db, 0, nil)
		require.NoError(t, err)
		assert.Equal(t, eventsBefore, eventsAfter)
		unchanged, err := factory.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{owner.ID}, unchanged.AssigneeIDs())
	}
	ctx.RunID = uuid.New()
	assert.False(t, runner.HasRunnerToken(runner.AttachRunnerEnv(ctx, nil, 60)))
}

func assertRunnerOwnerAccess(t *testing.T, server *Server, ctx core.ExecutionContext, env []runner.TaskEnvironmentVariable, orderID uuid.UUID) string {
	t.Helper()
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, env, 60))
	require.NotEmpty(t, token)
	scope, err := runner.ParseRunnerToken(jwt.NewSigner("test"), token)
	require.NoError(t, err)
	assert.Equal(t, orderID, scope.WorkOrderID)
	rec := suggestRunnerOwners(t, server, token, `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotEmpty(t, decodeSuggestOwners(t, rec).Members)
	return token
}

// mustTaskLinkedAgentRun creates a factory line run of one agent node on an
// open task.
func mustTaskLinkedAgentRun(t *testing.T, r *support.ResourceRegistry) (core.ExecutionContext, *models.Factory, *models.FactoryWorkOrder) {
	t.Helper()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Implement", "", nil, nil, nil)
	require.NoError(t, err)
	line, err := factory.CreateLine(db, support.RandomName("line"), nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "agent", Type: models.NodeTypeComponent}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "agent", "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
	require.NoError(t, err)
	execution := createExecutionForCanvasRun(t, run, event.ID, "agent")
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID,
		WorkOrderID: order.ID, LineID: line.ID, LineDispatchID: dispatch.ID,
		RunID: &run.ID, Status: models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error)
	ctx := core.ExecutionContext{ID: execution.ID, RunID: run.ID, WorkflowID: canvas.ID.String(), OrganizationID: r.Organization.ID.String(), NodeID: "agent", BaseURL: "http://app.test"}
	return ctx, factory, order
}

func runnerTokenFrom(env []runner.TaskEnvironmentVariable) string {
	for _, variable := range env {
		if variable.Name == runner.EnvSuperplaneRunnerToken {
			return variable.Value
		}
	}
	return ""
}

func suggestRunnerOwners(t *testing.T, server *Server, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	return runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/suggest-owners", body)
}

func assignRunnerOwner(t *testing.T, server *Server, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	return runnerRequest(t, server, http.MethodPut, token, "/api/v1/runner/task/owner", body)
}

func runnerRequest(t *testing.T, server *Server, method, token, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}
