package factories

import (
	"bytes"
	"context"
	"io"
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
	"github.com/superplanehq/superplane/pkg/integrations/productive"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
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

type stubIntakeItemSource struct {
	items []IntakeItem
	err   error
}

func (s stubIntakeItemSource) Search(context.Context, string, int) ([]IntakeItem, error) {
	return s.items, s.err
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

// stubRemoteFetcher stands in for a GitHub client's HTTPDo: it lets a test
// intake source satisfy the optional `RemoteFetch` interface that
// ImportFactoryIntakeItem uses to copy remote image bytes into workspace
// storage.
type stubRemoteFetcher struct {
	body        []byte
	contentType string
	calls       int
}

func (f *stubRemoteFetcher) RemoteFetch(_ context.Context, _ *http.Request) (*http.Response, error) {
	f.calls++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(f.body)),
		Header:     http.Header{"Content-Type": []string{f.contentType}},
	}, nil
}

type stubIntakeItemSourceWithRemoteFetch struct {
	stubIntakeItemSource
	*stubRemoteFetcher
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

	t.Run("copies GitHub-hosted images into workspace storage", func(t *testing.T) {
		t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
		t.Setenv("BASE_URL", "http://files.test")
		store, err := filesystem.New(t.TempDir())
		require.NoError(t, err)
		blob.SetCurrent(store)
		t.Cleanup(func() { blob.SetCurrent(nil) })

		factory := newFactory(t)
		intake := createIntake(t, factory)
		imageURL := "https://user-images.githubusercontent.com/1/2.png"
		itemWithImage := IntakeItem{
			ID:    "42",
			Key:   "#42",
			Title: "Broken screenshot on a private repo",
			Body:  "See ![screenshot](" + imageURL + ") for details.",
			URL:   "https://github.com/acme/payments/issues/42",
		}
		fetcher := &stubRemoteFetcher{body: []byte("png-bytes"), contentType: "image/png"}
		depsWithFetch := IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return stubIntakeItemSourceWithRemoteFetch{
					stubIntakeItemSource: stubIntakeItemSource{items: []IntakeItem{itemWithImage}},
					stubRemoteFetcher:    fetcher,
				}, nil
			},
		}

		response, err := ImportFactoryIntakeItem(ctx, depsWithFetch, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    itemWithImage.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, response.GetOrder())
		assert.Equal(t, 1, fetcher.calls)
		assert.NotContains(t, response.GetOrder().GetDescription(), imageURL)

		require.Len(t, response.GetOrder().GetFiles(), 1)
		file := response.GetOrder().GetFiles()[0]
		assert.Equal(t, "image/png", file.GetContentType())
		assert.NotEmpty(t, file.GetDownloadUrl())
		assert.Contains(t, response.GetOrder().GetDescription(), blob.FileRefScheme+"://"+file.GetId())

		// Prove the image bytes were actually persisted to storage, not just
		// that a download URL was minted from the file ID and signing key.
		fileID, err := uuid.Parse(file.GetId())
		require.NoError(t, err)
		record, err := models.FindFile(database.DB(t.Context()), fileID)
		require.NoError(t, err)
		reader, err := store.Get(ctx, record.StorageKey)
		require.NoError(t, err)
		defer reader.Close()
		stored, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, fetcher.body, stored)
	})
}
