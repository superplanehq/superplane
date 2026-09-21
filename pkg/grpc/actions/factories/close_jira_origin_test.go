package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func TestResolveJiraCloseTarget(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}
	db := database.Conn()

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	integrationID := createReadyJiraIntakeIntegration(t, r.Organization.ID, "ENG")

	response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
		FactoryId:         factory.ID.String(),
		Source:            pb.FactoryIntake_SOURCE_JIRA_ISSUES,
		IntegrationId:     integrationID,
		ResourceId:        "ENG",
		SkipInitialImport: true,
	})
	require.NoError(t, err)

	orderFor := func(t *testing.T, originURL string) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factory.CreateWorkOrderWithOrigin(db, "Fix issue", "", nil, nil, nil, models.WorkOrderOrigin{
			URL:   originURL,
			Label: "issue",
		})
		require.NoError(t, err)
		return order
	}

	t.Run("matches the intake project and site", func(t *testing.T) {
		target, err := resolveJiraCloseTarget(db, factory, orderFor(t, "https://acme.atlassian.net/browse/ENG-42"))
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, "ENG-42", target.IssueKey)
		assert.True(t, target.MoveOnComplete)
		assert.Empty(t, target.Column)
		assert.Equal(t, integrationID, target.Integration.ID.String())
	})

	t.Run("ignores a different project on the same site", func(t *testing.T) {
		target, err := resolveJiraCloseTarget(db, factory, orderFor(t, "https://acme.atlassian.net/browse/OPS-42"))
		require.NoError(t, err)
		assert.Nil(t, target)
	})

	t.Run("ignores a different Jira site", func(t *testing.T) {
		target, err := resolveJiraCloseTarget(db, factory, orderFor(t, "https://other.atlassian.net/browse/ENG-42"))
		require.NoError(t, err)
		assert.Nil(t, target)
	})

	t.Run("uses the stored column and honors a disabled move", func(t *testing.T) {
		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  response.GetIntake().GetId(),
			Settings: &pb.FactoryIntake_Settings{
				NewIssues:            proto.Bool(true),
				ReopenedIssues:       proto.Bool(true),
				JiraMoveOnComplete:   proto.Bool(false),
				JiraCompletionColumn: "QA",
			},
		})
		require.NoError(t, err)

		target, err := resolveJiraCloseTarget(db, factory, orderFor(t, "https://acme.atlassian.net/browse/ENG-9"))
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.False(t, target.MoveOnComplete)
		assert.Equal(t, "QA", target.Column)
	})
}

func TestResolveJiraCloseTargetPrefersSourceRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}
	db := database.Conn()

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	integrationID := createReadyJiraIntakeIntegration(t, r.Organization.ID, "ENG")

	first, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
		FactoryId:         factory.ID.String(),
		Source:            pb.FactoryIntake_SOURCE_JIRA_ISSUES,
		IntegrationId:     integrationID,
		ResourceId:        "ENG",
		SkipInitialImport: true,
	})
	require.NoError(t, err)

	second, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
		FactoryId:         factory.ID.String(),
		Source:            pb.FactoryIntake_SOURCE_JIRA_ISSUES,
		IntegrationId:     integrationID,
		ResourceId:        "ENG",
		SkipInitialImport: true,
	})
	require.NoError(t, err)

	_, err = UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
		FactoryId: factory.ID.String(),
		IntakeId:  first.GetIntake().GetId(),
		Settings: &pb.FactoryIntake_Settings{
			NewIssues:            proto.Bool(true),
			ReopenedIssues:       proto.Bool(true),
			JiraMoveOnComplete:   proto.Bool(true),
			JiraCompletionColumn: "To Do",
		},
	})
	require.NoError(t, err)

	_, err = UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
		FactoryId: factory.ID.String(),
		IntakeId:  second.GetIntake().GetId(),
		Settings: &pb.FactoryIntake_Settings{
			NewIssues:            proto.Bool(true),
			ReopenedIssues:       proto.Bool(true),
			JiraMoveOnComplete:   proto.Bool(true),
			JiraCompletionColumn: "QA",
		},
	})
	require.NoError(t, err)

	run, err := models.CreateCanvasRunInTransaction(
		db,
		uuid.MustParse(second.GetIntake().GetCanvasId()),
		intakeTriggerNodeID,
		models.CanvasRunStateFinished,
		"",
	)
	require.NoError(t, err)

	order, err := factory.CreateWorkOrderWithOrigin(db, "Fix ENG-42", "", nil, nil, &run.ID, models.WorkOrderOrigin{
		URL:   "https://acme.atlassian.net/browse/ENG-42",
		Label: "ENG-42",
	})
	require.NoError(t, err)

	target, err := resolveJiraCloseTarget(db, factory, order)
	require.NoError(t, err)
	require.NotNil(t, target)
	assert.Equal(t, "QA", target.Column)
}

func TestJiraCompletionCommentFor(t *testing.T) {
	orgID := uuid.New()
	factory := &models.Factory{Key: "SP"}
	order := &models.FactoryWorkOrder{OrganizationID: orgID, Number: 7}

	comment := jiraCompletionCommentFor("https://app.example.com/", factory, order)
	assert.Equal(t, jiraCompletionComment+" https://app.example.com"+order.URLPath("SP"), comment)
	assert.Equal(t, jiraCompletionComment, jiraCompletionCommentFor("", factory, order))
}
