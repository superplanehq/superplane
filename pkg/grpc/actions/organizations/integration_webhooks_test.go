package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"
)

// A failed review-monitoring hook must be visible even while its integration
// remains ready. Retrying one hook must not reconfigure other subscriptions.
func TestIntegrationWebhookFailureRecovery(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	ctx := context.Background()
	integration := models.NewIntegration(uuid.New(), r.Organization.ID, "github", "GitHub", map[string]any{})
	integration.State = models.IntegrationStateReady
	require.NoError(t, db.Create(integration).Error)
	hook := &models.Webhook{
		AppInstallationID: &integration.ID, State: models.WebhookStateFailed, RetryCount: 3,
		Secret: []byte("must-not-be-returned"),
		Configuration: datatypes.NewJSONType[any](map[string]any{
			"repository": "acme/first", "repositories": []string{"acme/first", "acme/api"},
			"eventTypes": []string{"pull_request", "pull_request_review"},
		}),
		Metadata: datatypes.NewJSONType[any](map[string]any{
			"id":         123,
			"setupError": `error creating webhook: POST https://api.github.com/repos/acme/api/hooks: 422 Validation Failed [{Resource:Hook Field: Code:custom Message:The "pull_request" event cannot have more than 20 hooks}]`,
			"private":    "must-not-be-returned",
		}),
	}
	require.NoError(t, db.Create(hook).Error)

	described, err := DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), integration.ID.String())
	require.NoError(t, err)
	require.Len(t, described.Integration.Status.WebhookFailures, 1)
	failure := described.Integration.Status.WebhookFailures[0]
	require.Equal(t, hook.ID.String(), failure.Id)
	require.Equal(t, []string{"acme/api"}, failure.Repositories)
	require.Equal(t, []string{"pull_request", "pull_request_review"}, failure.Events)
	require.Equal(t, "pull_request", failure.LimitEvent)
	require.Contains(t, failure.Message, "20")
	encoded, err := protojson.Marshal(described)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "must-not-be-returned")

	listed, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listed.Integrations, 1)
	require.Len(t, listed.Integrations[0].Status.WebhookFailures, 1)

	other, err := models.CreateOrganization("other-"+uuid.NewString(), "")
	require.NoError(t, err)
	_, err = RetryIntegrationWebhook(ctx, other.ID.String(), integration.ID.String(), hook.ID.String())
	code, _, _ := grpcerrors.HandlerStatus(err)
	require.Equal(t, codes.NotFound, code)
	_, err = RetryIntegrationWebhook(ctx, r.Organization.ID.String(), integration.ID.String(), uuid.NewString())
	code, _, _ = grpcerrors.HandlerStatus(err)
	require.Equal(t, codes.NotFound, code)

	for range 2 {
		_, err = RetryIntegrationWebhook(ctx, r.Organization.ID.String(), integration.ID.String(), hook.ID.String())
		require.NoError(t, err)
	}
	var saved models.Webhook
	require.NoError(t, db.First(&saved, hook.ID).Error)
	require.Equal(t, models.WebhookStatePending, saved.State)
	require.Zero(t, saved.RetryCount)
	expectedConfig, err := hook.Configuration.MarshalJSON()
	require.NoError(t, err)
	savedConfig, err := saved.Configuration.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, string(expectedConfig), string(savedConfig))
	require.Equal(t, hook.Secret, saved.Secret)
	require.Equal(t, float64(123), saved.Metadata.Data().(map[string]any)["id"])
	described, err = DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), integration.ID.String())
	require.NoError(t, err)
	require.Len(t, described.Integration.Status.WebhookFailures, 1, "keep the repair visible while setup runs")
	require.Equal(t, "pending", described.Integration.Status.WebhookFailures[0].State)
	require.Equal(t, "pull_request", described.Integration.Status.WebhookFailures[0].LimitEvent)

	require.NoError(t, saved.ReadyWithMetadata(db, map[string]any{"id": 123}))
	described, err = DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), integration.ID.String())
	require.NoError(t, err)
	require.Empty(t, described.Integration.Status.WebhookFailures)
	_, err = RetryIntegrationWebhook(ctx, r.Organization.ID.String(), integration.ID.String(), hook.ID.String())
	code, _, _ = grpcerrors.HandlerStatus(err)
	require.Equal(t, codes.FailedPrecondition, code)
}

func TestRetryLegacyWebhookKeepsFailureVisibleUntilReady(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	ctx := context.Background()
	integration := models.NewIntegration(uuid.New(), r.Organization.ID, "github", "GitHub", map[string]any{})
	integration.State = models.IntegrationStateReady
	require.NoError(t, db.Create(integration).Error)

	for _, tc := range []struct {
		name     string
		metadata any
	}{
		{name: "empty metadata", metadata: map[string]any{}},
		{name: "null metadata"},
		{name: "empty error", metadata: map[string]any{"setupError": ""}},
		{name: "registration without error", metadata: map[string]any{"id": 123}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hook := &models.Webhook{
				AppInstallationID: &integration.ID, State: models.WebhookStateFailed,
				Secret:        []byte("test-secret"),
				Configuration: datatypes.NewJSONType[any](map[string]any{"repository": "acme/api", "eventType": "pull_request"}),
				Metadata:      datatypes.NewJSONType(tc.metadata),
			}
			require.NoError(t, db.Create(hook).Error)
			_, err := RetryIntegrationWebhook(ctx, r.Organization.ID.String(), integration.ID.String(), hook.ID.String())
			require.NoError(t, err)

			for _, state := range []string{models.WebhookStatePending, models.WebhookStateProvisioning} {
				if state == models.WebhookStateProvisioning {
					require.NoError(t, hook.MarkProvisioning(db))
				}
				described, err := DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), integration.ID.String())
				require.NoError(t, err)
				require.Len(t, described.Integration.Status.WebhookFailures, 1, "keep the settings warning until setup succeeds")
				require.Equal(t, state, described.Integration.Status.WebhookFailures[0].State)
				listed, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String())
				require.NoError(t, err)
				require.Len(t, listed.Integrations[0].Status.WebhookFailures, 1, "keep the workspace warning until setup succeeds")
			}

			require.NoError(t, hook.ReadyWithMetadata(db, map[string]any{"id": 456}))
			described, err := DescribeIntegration(ctx, r.Registry, r.Organization.ID.String(), integration.ID.String())
			require.NoError(t, err)
			require.Empty(t, described.Integration.Status.WebhookFailures)
		})
	}
}
