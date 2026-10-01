package factories

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/integrations/productive"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__ProductiveTaskItem(t *testing.T) {
	item := productiveTaskItem(productive.Task{
		ID:          "91",
		Number:      "512",
		Title:       "Fix payment retries",
		Description: "Retries fail silently.",
	}, "12345")

	assert.Equal(t, "91", item.ID)
	assert.Equal(t, "#512", item.Key)
	assert.Equal(t, "Fix payment retries", item.Title)
	assert.Equal(t, "Retries fail silently.", item.Body)
	assert.Equal(t, "https://app.productive.io/12345/tasks/91", item.URL)
}

func TestParseGitHubIssueURL(t *testing.T) {
	repository, number, ok := parseGitHubIssueURL("https://github.com/acme/payments/issues/12#issuecomment-1")
	assert.True(t, ok)
	assert.Equal(t, "acme/payments", repository)
	assert.Equal(t, 12, number)

	_, _, ok = parseGitHubIssueURL("https://github.com/acme/payments/pull/12")
	assert.False(t, ok)
	_, _, ok = parseGitHubIssueURL("https://example.com/acme/payments/issues/12")
	assert.False(t, ok)
}

func TestParseProductiveTaskURL(t *testing.T) {
	organizationID, taskID, ok := parseProductiveTaskURL("https://app.productive.io/12345/tasks/91")
	assert.True(t, ok)
	assert.Equal(t, "12345", organizationID)
	assert.Equal(t, "91", taskID)

	_, _, ok = parseProductiveTaskURL("https://app.productive.io/12345/projects/91")
	assert.False(t, ok)
	_, _, ok = parseProductiveTaskURL("https://example.com/12345/tasks/91")
	assert.False(t, ok)
}

type stubIntakeItemSource struct {
	items []IntakeItem
	err   error
}

func (s stubIntakeItemSource) Search(context.Context, string, int) ([]IntakeItem, error) {
	return s.items, s.err
}

type linearFileIntakeSource struct {
	stubIntakeItemSource
	files []linear.IssueFile
	links []linear.IssueLink
}

func (s linearFileIntakeSource) IssueFiles(context.Context, string, string) ([]linear.IssueFile, []linear.IssueLink, error) {
	return s.files, s.links, nil
}

func (s stubIntakeItemSource) Get(_ context.Context, id string) (*IntakeItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	for i := range s.items {
		if s.items[i].ID == id {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, errIntakeItemNotFound
}

func Test__SearchFactoryIntakeItems(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	item := IntakeItem{
		ID:    "12",
		Key:   "#12",
		Title: "Handle duplicate refunds",
		Body:  "Retrying a refund posts twice.",
		URL:   "https://github.com/acme/payments/issues/12",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createIntake := func(t *testing.T, factory *models.Factory) *models.FactoryIntake {
		t.Helper()
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")
		intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		return intake
	}

	deps := func(items []IntakeItem, sourceErr error) IntakeDependencies {
		return IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				if sourceErr != nil {
					return nil, sourceErr
				}
				return stubIntakeItemSource{items: items}, nil
			},
		}
	}

	t.Run("returns items from the intake source", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		response, err := SearchFactoryIntakeItems(ctx, deps([]IntakeItem{item}, nil), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, response.GetItems(), 1)
		assert.Equal(t, item.ID, response.GetItems()[0].GetId())
		assert.Equal(t, item.Key, response.GetItems()[0].GetKey())
		assert.Equal(t, item.Title, response.GetItems()[0].GetTitle())
		assert.Equal(t, item.URL, response.GetItems()[0].GetUrl())
	})

	t.Run("unconnected intake is a failed precondition", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		_, err := SearchFactoryIntakeItems(ctx, deps(nil, errIntakeNotConnected), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "Connect this intake first.")
	})

	t.Run("search still returns items while a Jira intake is paused", func(t *testing.T) {
		factory := newFactory(t)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Jira issues")
		intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, models.FactoryIntakeSourceJiraIssues)
		require.NoError(t, err)
		require.NoError(t, intake.SetPaused(database.DB(t.Context()), true))

		jiraItem := IntakeItem{
			ID:    "10001",
			Key:   "ENG-1",
			Title: "Login fails",
			URL:   "https://example.atlassian.net/browse/ENG-1",
		}
		response, err := SearchFactoryIntakeItems(ctx, deps([]IntakeItem{jiraItem}, nil), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, response.GetItems(), 1)
		assert.Equal(t, jiraItem.ID, response.GetItems()[0].GetId())
		assert.Equal(t, jiraItem.Key, response.GetItems()[0].GetKey())
		assert.Equal(t, jiraItem.Title, response.GetItems()[0].GetTitle())
	})

	t.Run("unsupported intake is a failed precondition", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		_, err := SearchFactoryIntakeItems(ctx, deps(nil, errIntakeSearchUnsupported), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Contains(t, message, "This intake cannot search items yet.")
	})

	t.Run("Datadog 403 from search is a failed precondition the UI can show", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		_, err := SearchFactoryIntakeItems(ctx, deps(nil, datadog.ErrErrorTrackingForbidden), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, datadog.ErrorTrackingForbiddenMessage, message)
	})

	assertSearch := func(t *testing.T, sourceErr error) (codes.Code, string) {
		t.Helper()
		factory := newFactory(t)
		intake := createIntake(t, factory)
		_, err := SearchFactoryIntakeItems(ctx, deps(nil, sourceErr), orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
		})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		return code, message
	}

	for _, source := range liveIntakeClientErrors() {
		t.Run(source.name+" 403 from search is a failed precondition", func(t *testing.T) {
			code, message := assertSearch(t, source.statusError(http.StatusForbidden))
			assert.Equal(t, codes.FailedPrecondition, code)
			assert.Equal(t, intakeConnectFirstMessage, message)
		})
	}

	t.Run("a non-auth 4xx from search is a failed precondition", func(t *testing.T) {
		code, message := assertSearch(t, &jira.APIError{StatusCode: http.StatusUnprocessableEntity})
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, intakeCouldNotLoadItemsMessage, message)
	})

	t.Run("a 5xx from search stays Internal", func(t *testing.T) {
		code, message := assertSearch(t, sentry.StatusError(http.StatusBadGateway))
		assert.Equal(t, codes.Internal, code)
		assert.Equal(t, "failed to search factory intake items", message)
	})

	t.Run("a 429 from search stays Internal", func(t *testing.T) {
		code, message := assertSearch(t, &jira.APIError{StatusCode: http.StatusTooManyRequests})
		assert.Equal(t, codes.Internal, code)
		assert.Equal(t, "failed to search factory intake items", message)
	})

	t.Run("a 408 from search stays Internal", func(t *testing.T) {
		code, message := assertSearch(t, &jira.APIError{StatusCode: http.StatusRequestTimeout})
		assert.Equal(t, codes.Internal, code)
		assert.Equal(t, "failed to search factory intake items", message)
	})

	t.Run("integration error description is returned instead of unsupported search", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")
		created, err := CreateFactoryIntake(ctx, IntakeDependencies{
			Registry:       r.Registry,
			Encryptor:      r.Encryptor,
			AuthService:    r.AuthService,
			WebhookBaseURL: "http://localhost:8000",
		}, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			IntegrationId: integrationID,
			ResourceId:    "payments",
		})
		require.NoError(t, err)

		require.NoError(t, database.DB(t.Context()).Model(&models.Integration{}).
			Where("id = ?", uuid.MustParse(integrationID)).
			Updates(map[string]any{
				"state":             models.IntegrationStateError,
				"state_description": "invalid credentials: Forbidden",
			}).Error)

		_, err = SearchFactoryIntakeItems(ctx, IntakeDependencies{
			Registry:  r.Registry,
			Encryptor: r.Encryptor,
		}, orgID, &pb.SearchFactoryIntakeItemsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  created.GetIntake().GetId(),
		})
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "invalid credentials: Forbidden", message)
		assert.NotContains(t, message, "This intake cannot search items yet.")
		assert.NotContains(t, message, "Connect this intake first.")
	})
}

func Test__ImportFactoryIntakeItem(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	item := IntakeItem{
		ID:    "12",
		Key:   "#12",
		Title: "Handle duplicate refunds",
		Body:  "Retrying a refund posts twice.",
		URL:   "https://github.com/acme/payments/issues/12",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createIntake := func(t *testing.T, factory *models.Factory) *models.FactoryIntake {
		t.Helper()
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "GitHub issues")
		intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		return intake
	}

	deps := IntakeDependencies{
		NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
			return stubIntakeItemSource{items: []IntakeItem{item}}, nil
		},
	}

	t.Run("creates a draft work order with the ticket origin", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)

		response, err := ImportFactoryIntakeItem(ctx, deps, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    item.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response.GetOrder())
		assert.Equal(t, item.Title, response.GetOrder().GetTitle())
		assert.Equal(t, item.Body, response.GetOrder().GetDescription())
		assert.Equal(t, pb.WorkOrder_STATE_DRAFT, response.GetOrder().GetState())
		require.NotNil(t, response.GetOrder().GetOrigin())
		assert.Equal(t, item.URL, response.GetOrder().GetOrigin().GetUrl())
		assert.Equal(t, "acme/payments#12", response.GetOrder().GetOrigin().GetLabel())
		assert.Equal(t, r.User.String(), response.GetOrder().GetCreatedBy().GetUser().GetId())
		require.Len(t, response.GetOrder().GetAssignees(), 1)
		assert.Equal(t, r.User.String(), response.GetOrder().GetAssignees()[0].GetId())
	})

	t.Run("forwards a description that already includes imported comments", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)
		itemWithComments := IntakeItem{
			ID:    "13",
			Key:   "#13",
			Title: "Handle duplicate refunds",
			Body: strings.Join([]string{
				"Retrying a refund posts twice.",
				importedCommentsSeparator,
				importedCommentsHeader,
				"ana 2026-08-01T10:00:00Z",
				"Confirmed on staging.",
			}, "\n"),
			URL: "https://github.com/acme/payments/issues/13",
		}
		depsWithComments := IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return stubIntakeItemSource{items: []IntakeItem{itemWithComments}}, nil
			},
		}

		response, err := ImportFactoryIntakeItem(ctx, depsWithComments, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    itemWithComments.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response.GetOrder())
		assert.Equal(t, itemWithComments.Title, response.GetOrder().GetTitle())
		assert.Equal(t, itemWithComments.Body, response.GetOrder().GetDescription())
		assert.Contains(t, response.GetOrder().GetDescription(), importedCommentsSeparator)
		assert.Contains(t, response.GetOrder().GetDescription(), importedCommentsHeader)
		require.NotNil(t, response.GetOrder().GetOrigin())
		assert.Equal(t, itemWithComments.URL, response.GetOrder().GetOrigin().GetUrl())
	})

	t.Run("importing a chosen item still works while the intake is paused", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)
		require.NoError(t, intake.SetPaused(database.DB(t.Context()), true))

		response, err := ImportFactoryIntakeItem(ctx, deps, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    item.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response.GetOrder())
		assert.Equal(t, item.Title, response.GetOrder().GetTitle())
		require.NotNil(t, response.GetOrder().GetOrigin())
		assert.Equal(t, item.URL, response.GetOrder().GetOrigin().GetUrl())
	})

	t.Run("importing a chosen Jira item still works while the intake is paused", func(t *testing.T) {
		factory := newFactory(t)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Jira issues")
		intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, models.FactoryIntakeSourceJiraIssues)
		require.NoError(t, err)
		require.NoError(t, intake.SetPaused(database.DB(t.Context()), true))

		jiraItem := IntakeItem{
			ID:    "10001",
			Key:   "ENG-1",
			Title: "Login fails",
			Body:  "Users cannot sign in.",
			URL:   "https://example.atlassian.net/browse/ENG-1",
		}
		jiraDeps := IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return stubIntakeItemSource{items: []IntakeItem{jiraItem}}, nil
			},
		}

		response, err := ImportFactoryIntakeItem(ctx, jiraDeps, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    jiraItem.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response.GetOrder())
		assert.Equal(t, jiraItem.Title, response.GetOrder().GetTitle())
		require.NotNil(t, response.GetOrder().GetOrigin())
		assert.Equal(t, jiraItem.URL, response.GetOrder().GetOrigin().GetUrl())
	})

	t.Run("stores the Datadog issue title as the source label", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)
		issueID := "da226b38-baac-11f1-bad1-da7ad0900005"
		datadogItem := IntakeItem{
			ID:    issueID,
			Title: "TimeoutError: checkout timed out",
			Body:  "checkout timed out",
			URL:   "https://app.datadoghq.eu/error-tracking/issue/" + issueID,
		}
		datadogDeps := IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return stubIntakeItemSource{items: []IntakeItem{datadogItem}}, nil
			},
		}

		response, err := ImportFactoryIntakeItem(ctx, datadogDeps, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    datadogItem.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response.GetOrder().GetOrigin())
		assert.Equal(t, datadogItem.URL, response.GetOrder().GetOrigin().GetUrl())
		assert.Equal(t, datadogItem.Title, response.GetOrder().GetOrigin().GetLabel())

		orderID, err := uuid.Parse(response.GetOrder().GetId())
		require.NoError(t, err)
		stored, err := factory.FindWorkOrder(database.DB(t.Context()), orderID)
		require.NoError(t, err)
		require.NotNil(t, stored.OriginLabel)
		assert.Equal(t, datadogItem.Title, *stored.OriginLabel)
	})

	t.Run("importing a Linear issue stores private files and keeps other links", func(t *testing.T) {
		t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
		t.Setenv("BASE_URL", "http://files.test")
		store, err := filesystem.New(t.TempDir())
		require.NoError(t, err)
		blob.SetCurrent(store)
		t.Cleanup(func() { blob.SetCurrent(nil) })

		factory := newFactory(t)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Linear issues")
		intake, err := factory.CreateIntake(database.DB(t.Context()), canvas.ID, models.FactoryIntakeSourceLinearIssues)
		require.NoError(t, err)

		uploadURL := "https://uploads.linear.app/file/abc/shot.png"
		linearItem := IntakeItem{
			ID:    "issue-1",
			Key:   "ENG-1",
			Title: "Deploy pipeline fails",
			Body:  "See " + uploadURL,
			URL:   "https://linear.app/acme/issue/ENG-1",
		}
		linearDeps := IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return linearFileIntakeSource{
					stubIntakeItemSource: stubIntakeItemSource{items: []IntakeItem{linearItem}},
					files: []linear.IssueFile{{
						Name:        "shot.png",
						ContentType: "image/png",
						Body:        []byte("png-bytes"),
						ReplaceURLs: []string{uploadURL},
					}},
					links: []linear.IssueLink{{Title: "Pull request", URL: "https://github.com/acme/repo/pull/1"}},
				}, nil
			},
		}

		response, err := ImportFactoryIntakeItem(ctx, linearDeps, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    linearItem.ID,
		})
		require.NoError(t, err)

		orderID, err := uuid.Parse(response.GetOrder().GetId())
		require.NoError(t, err)
		stored, err := factory.FindWorkOrder(database.DB(t.Context()), orderID)
		require.NoError(t, err)
		assert.NotContains(t, stored.Description, uploadURL)
		assert.Contains(t, stored.Description, blob.FileRefScheme+"://")
		assert.Contains(t, stored.Description, "https://github.com/acme/repo/pull/1")
	})

	t.Run("a second import of the same ticket creates a new work order", func(t *testing.T) {
		factory := newFactory(t)
		intake := createIntake(t, factory)
		req := &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    item.ID,
		}

		first, err := ImportFactoryIntakeItem(ctx, deps, orgID, req)
		require.NoError(t, err)
		second, err := ImportFactoryIntakeItem(ctx, deps, orgID, req)
		require.NoError(t, err)
		assert.NotEqual(t, first.GetOrder().GetId(), second.GetOrder().GetId())
		assert.Equal(t, item.URL, second.GetOrder().GetOrigin().GetUrl())
	})
}
