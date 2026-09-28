package factories

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func Test__CreateWorkOrder__AssignsTheCreator(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	other := support.CreateUser(t, r, r.Organization.ID)

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := CreateWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Ship the refunds line",
		AssigneeIds: []string{other.ID.String()},
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
	assert.Equal(t, r.UserModel.Name, resp.Order.Assignees[0].Name)
	assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
}

func Test__CreateWorkOrder__ReparentsWorkspaceFiles(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(ctx, database.Conn(), store, file, bytes.NewReader([]byte("png-bytes"))))

	resp, err := CreateWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Login screenshot",
		Description: "See ![bug](" + blob.FileRef(file.ID) + ")",
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Files, 1)
	assert.Equal(t, file.ID.String(), resp.Order.Files[0].Id)
	assert.NotEmpty(t, resp.Order.Files[0].DownloadUrl)
	assert.Contains(t, resp.Order.Description, blob.FileRef(file.ID))
	assert.NotContains(t, resp.Order.Description, "http://")

	reparented, err := models.FindFile(database.Conn(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, blob.ScopeTask, reparented.Scope)
}

func Test__CreateWorkOrder__MirrorsManualTasksAsGitHubIssues(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	issueURL := "https://github.com/acme/payments/issues/42"

	t.Run("creates a GitHub issue and saves it as the origin", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42",
			"title": "Ship the refunds line"
		}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Ship the refunds line",
			Description: "Stop double charges.",
		})
		require.NoError(t, err)
		assert.Equal(t, issueURL, resp.Order.GetOrigin().GetUrl())
		assert.Equal(t, "acme/payments#42", resp.Order.GetOrigin().GetLabel())

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/acme/payments/issues", httpCtx.Requests[0].URL.Path)
		body, err := io.ReadAll(httpCtx.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Ship the refunds line")
		assert.Contains(t, string(body), "Stop double charges.")
	})

	t.Run("does not call GitHub when the setting is off", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{}`)
		factoryModel := githubIssueFactory(t, r, false, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Nil(t, resp.Order.GetOrigin())
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("does not call GitHub when the intake is paused", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{}`)
		factoryModel := githubIssueFactory(t, r, true, true, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Nil(t, resp.Order.GetOrigin())
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("creates the task without an origin when GitHub fails", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusForbidden, `{"message":"Resource not accessible by integration"}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Equal(t, "Ship the refunds line", resp.Order.GetTitle())
		assert.Nil(t, resp.Order.GetOrigin())
		require.Len(t, httpCtx.Requests, 1)

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		require.Len(t, orders, 1)
		assert.Nil(t, orders[0].OriginURL)
	})

	t.Run("uses the oldest enabled intake", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 7,
			"html_url": "https://github.com/acme/older/issues/7"
		}`)
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		older := time.Now().Add(-time.Hour)
		seedGitHubIssueIntake(t, r, factoryModel, true, false, "acme/older", &older)
		seedGitHubIssueIntake(t, r, factoryModel, true, false, "acme/newer", nil)

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Use the first intake",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://github.com/acme/older/issues/7", resp.Order.GetOrigin().GetUrl())
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "/repos/acme/older/issues", httpCtx.Requests[0].URL.Path)
	})
}

func githubIssueDeps(r *support.ResourceRegistry, httpCtx *contexts.HTTPContext) IntakeDependencies {
	return IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
		HTTP:           httpCtx,
	}
}

func githubIssueHTTP(status int, body string) *contexts.HTTPContext {
	return &contexts.HTTPContext{Responses: []*http.Response{{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}}}
}

func githubIssueFactory(
	t *testing.T,
	r *support.ResourceRegistry,
	createIssue bool,
	paused bool,
	repository string,
) *models.Factory {
	t.Helper()

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedGitHubIssueIntake(t, r, factoryModel, createIssue, paused, repository, nil)
	return factoryModel
}

func seedGitHubIssueIntake(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryModel *models.Factory,
	createIssue bool,
	paused bool,
	repository string,
	createdAt *time.Time,
) {
	t.Helper()

	db := database.Conn()
	integration := createReadyGitHubIssueIntegration(t, r)
	const triggerNodeID = "trigger"
	integrationID := integration.ID.String()
	metadata := map[string]any{intakeMetadataGitHubCreateIssueForManualTasks: createIssue}

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: triggerNodeID,
			Type:   models.NodeTypeTrigger,
			Name:   "On Issue",
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "github.onIssue"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"repository": repository,
				"actions":    []any{"opened"},
			}),
			Metadata: datatypes.NewJSONType(metadata),
		}},
		nil,
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	nodes := append([]models.Node(nil), liveVersion.Nodes...)
	for i := range nodes {
		if nodes[i].ID != triggerNodeID {
			continue
		}
		nodes[i].IntegrationID = &integrationID
		nodes[i].Ref = models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}
		nodes[i].Configuration = map[string]any{"repository": repository, "actions": []any{"opened"}}
		nodes[i].Metadata = metadata
	}
	require.NoError(t, db.Model(liveVersion).Update("nodes", datatypes.NewJSONSlice(nodes)).Error)

	intake, err := factoryModel.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
	require.NoError(t, err)
	if paused {
		require.NoError(t, intake.SetPaused(db, true))
	}
	if createdAt != nil {
		require.NoError(t, db.Model(intake).Update("created_at", *createdAt).Error)
	}
}

func createReadyGitHubIssueIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("github"),
		map[string]any{},
	)
	require.NoError(t, err)

	setup := datatypes.NewJSONType(models.SetupState{})
	integration.State = models.IntegrationStateReady
	integration.SetupState = &setup
	integration.Properties = datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
		{Name: common.PropertyAuthMethod, Value: common.AuthMethodPAT},
		{Name: common.PropertyOwner, Value: "acme"},
		{Name: common.PropertyOwnerType, Value: common.OwnerTypeOrganization},
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	encrypted, err := r.Encryptor.Encrypt(t.Context(), []byte("ghp_test"), []byte(integration.ID.String()))
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, database.Conn().Create(&models.IntegrationSecret{
		OrganizationID: integration.OrganizationID,
		InstallationID: integration.ID,
		Name:           common.SecretPAT,
		Value:          encrypted,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}).Error)

	return integration
}
