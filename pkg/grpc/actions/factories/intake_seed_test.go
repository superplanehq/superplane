package factories

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__IntakeSeed(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		GitProvider:    r.GitProvider,
		WebhookBaseURL: "http://localhost:8000",
	}

	t.Run("seeded issues analyze at once", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)
		intake := response.GetIntake()

		titles := []string{
			"Handle duplicate refunds on retry",
			"Return 409 when the invoice is already paid",
			"Show a clearer empty state on the billing page",
			"Upgrade the Node 20 base image",
			"Add a flake retry to the checkout e2e suite",
		}

		events, err := gitHubIssueEvents(gitHubIssuePage(titles), "acme/backlog")
		require.NoError(t, err)
		require.NoError(t, emitIntakeEvents(
			database.DB(t.Context()),
			uuid.MustParse(intake.GetCanvasId()),
			intakeGitHubIssuePayloadType,
			events,
		))

		runs, err := ListFactoryIntakeRuns(ctx, orgID, &pb.ListFactoryIntakeRunsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
		})
		require.NoError(t, err)
		require.Len(t, runs.GetRuns(), len(titles))

		// The newest issue leads the list. The work order is not created yet.
		reported := []string{}
		for _, run := range runs.GetRuns() {
			assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_ANALYZING, run.GetPlacement())
			reported = append(reported, run.GetTitle())
		}
		assert.Equal(t, titles, reported)
	})

	t.Run("an intake without a connection starts empty", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		response, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		runs, err := ListFactoryIntakeRuns(ctx, orgID, &pb.ListFactoryIntakeRunsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  response.GetIntake().GetId(),
		})
		require.NoError(t, err)
		assert.Empty(t, runs.GetRuns())
	})
}

func Test__GitHubIssueEvents(t *testing.T) {
	t.Run("the newest issue ends up on top of the intake", func(t *testing.T) {
		events, err := gitHubIssueEvents(gitHubIssuePage([]string{"Newest issue", "Older issue"}), "acme/backlog")
		require.NoError(t, err)
		require.Len(t, events, 2)

		// Events are emitted oldest first, so the newest issue ends up on top
		// of the intake list.
		assert.Equal(t, "Older issue", issueEventTitle(t, events[0]))
		assert.Equal(t, "Newest issue", issueEventTitle(t, events[1]))
	})

	t.Run("an event carries what the graph reads", func(t *testing.T) {
		issue := &github.Issue{
			Number: github.Ptr(42),
			Title:  github.Ptr("Handle duplicate refunds on retry"),
			Body:   github.Ptr("A retried refund charges the customer twice."),
		}

		events, err := gitHubIssueEvents([]*github.Issue{issue}, "acme/backlog")
		require.NoError(t, err)
		require.Len(t, events, 1)

		event := events[0]
		assert.Equal(t, "opened", event["action"])
		assert.Equal(t, map[string]any{"full_name": "acme/backlog"}, event["repository"])

		payload, ok := event["issue"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Handle duplicate refunds on retry", payload["title"])
		assert.Equal(t, "A retried refund charges the customer twice.", payload["body"])

		// The filters read both lists without a guard, and GitHub leaves them
		// out of the response when the issue has none.
		assert.Equal(t, []any{}, payload["labels"])
		assert.Equal(t, []any{}, payload["assignees"])
	})
}

// gitHubIssuePage builds a page as the API returns it, so the titles are given
// newest first.
func gitHubIssuePage(titles []string) []*github.Issue {
	issues := make([]*github.Issue, 0, len(titles))
	for i, title := range titles {
		issues = append(issues, &github.Issue{
			Number: github.Ptr(len(titles) - i),
			Title:  github.Ptr(title),
			Body:   github.Ptr(fmt.Sprintf("Body of %s", title)),
		})
	}

	return issues
}

func issueEventTitle(t *testing.T, event map[string]any) string {
	t.Helper()

	payload, ok := event["issue"].(map[string]any)
	require.True(t, ok)
	title, ok := payload["title"].(string)
	require.True(t, ok)

	return title
}
