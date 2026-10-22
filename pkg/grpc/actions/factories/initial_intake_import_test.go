package factories

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestInitialWorkspaceIntakeImport(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		initial, complete, skip bool
		want                    int32
	}{
		{name: "first workspace imports at most three", initial: true, want: 3},
		{name: "later workspace also imports at most three", want: 3},
		{name: "new intake in an existing workspace imports at most three", initial: true, complete: true, want: 3},
		{name: "explicit skip does not import", skip: true, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(req.URL.Path, "/issues/") {
					_, _ = w.Write([]byte(sentrySeedIssuePage(1, 5, time.Now())))
					return
				}
				http.NotFound(w, req)
			}))
			defer server.Close()
			factory, err := models.CreateFactory(db, r.Organization.ID, "Workspace", "", "")
			require.NoError(t, err)
			if tc.initial {
				require.NoError(t, factory.SetInitialOnboardingAttempt(db, uuid.New()))
			}
			if tc.complete {
				require.NoError(t, db.Model(factory).Update("onboarding_completed_at", time.Now()).Error)
			}
			integration := models.NewIntegration(uuid.New(), r.Organization.ID, "sentry", "sentry", map[string]any{
				"baseUrl": server.URL, "userToken": base64.StdEncoding.EncodeToString([]byte("test-token")),
			})
			integration.State = models.IntegrationStateReady
			integration.Metadata = datatypes.NewJSONType(map[string]any{"organization": map[string]any{"slug": "example"}})
			require.NoError(t, db.Create(integration).Error)
			ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
			result, err := CreateFactoryIntake(ctx, IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}, r.Organization.ID.String(), &pb.CreateFactoryIntakeRequest{
				FactoryId: factory.ID.String(), Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS, IntegrationId: integration.ID.String(), ResourceId: "payments", SkipInitialImport: tc.skip,
			})
			require.NoError(t, err)
			if tc.skip {
				require.Equal(t, pb.FactoryIntake_INITIAL_IMPORT_STATUS_SKIPPED, result.Intake.GetInitialImportStatus())
			} else {
				require.Equal(t, pb.FactoryIntake_INITIAL_IMPORT_STATUS_COMPLETED, result.Intake.GetInitialImportStatus())
				require.Equal(t, tc.want, result.Intake.GetInitialImportItemCount())
			}
			runs, err := ListFactoryIntakeRuns(ctx, r.Organization.ID.String(), &pb.ListFactoryIntakeRunsRequest{FactoryId: factory.ID.String(), IntakeId: result.Intake.Id})
			require.NoError(t, err)
			require.Len(t, runs.Runs, int(tc.want))
		})
	}
}
