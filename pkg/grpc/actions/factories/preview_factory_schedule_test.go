package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestPreviewFactorySchedule(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("schedule"), "", "")
	require.NoError(t, err)
	req := &pb.PreviewFactoryScheduleRequest{FactoryId: factory.ID.String(), InstallParams: map[string]string{"hour": "11", "minute": "45", "timezone": "5.5"}}
	_, err = PreviewFactorySchedule(t.Context(), r.Organization.ID.String(), req)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.FailedPrecondition, code)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryScheduledIntake))
	response, err := PreviewFactorySchedule(t.Context(), r.Organization.ID.String(), req)
	require.NoError(t, err)
	local := response.NextRun.AsTime().In(time.FixedZone("browser", 19800))
	tomorrow := time.Now().In(local.Location()).AddDate(0, 0, 1)
	require.Equal(t, tomorrow.Day(), local.Day())
	require.Equal(t, 11, local.Hour())
	require.Equal(t, 45, local.Minute())
	for _, expression := range []string{"0 9 * * MON-FRI", "*/10 * * * * *"} {
		req.InstallParams = map[string]string{"type": "cron", "cronExpression": expression, "timezone": "5.5"}
		before := time.Now()
		response, err := PreviewFactorySchedule(t.Context(), r.Organization.ID.String(), req)
		require.NoError(t, err)
		require.True(t, response.NextRun.AsTime().After(before))
		if expression == "*/10 * * * * *" {
			require.Zero(t, response.NextRun.AsTime().Second()%10)
			require.Less(t, response.NextRun.AsTime().Sub(before), 11*time.Second)
		} else {
			local := response.NextRun.AsTime().In(time.FixedZone("browser", 19800))
			require.Equal(t, 9, local.Hour())
			require.NotEqual(t, time.Saturday, local.Weekday())
			require.NotEqual(t, time.Sunday, local.Weekday())
		}
	}
	req.InstallParams["cronExpression"] = "not a cron expression"
	_, err = PreviewFactorySchedule(t.Context(), r.Organization.ID.String(), req)
	code, _, ok = grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, code)
	// Preview must not create a canvas or activate a scheduled job.
	canvases, err := factory.ListCanvases(db)
	require.NoError(t, err)
	require.Empty(t, canvases)
	req.FactoryId = uuid.NewString()
	_, err = PreviewFactorySchedule(t.Context(), r.Organization.ID.String(), req)
	code, _, ok = grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, code)
}

func TestScheduledIntakeCronTemplate(t *testing.T) {
	for _, expression := range []string{"0 9 * * MON-FRI", "*/10 * * * * *"} {
		result, err := materializeFactoryTemplate(scheduledIntakeTemplateID, models.ProviderGitHub, factoryTemplateInput{installParams: map[string]string{"appRepository": "acme/app", "type": "cron", "cronExpression": expression}})
		require.NoError(t, err)
		canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
		require.NoError(t, err)
		trigger := findYAMLNode(t, canvas, "schedule")
		require.Equal(t, "cron", trigger.Configuration["type"])
		require.Equal(t, expression, trigger.Configuration["cronExpression"])
		require.NotContains(t, trigger.Configuration, "daysInterval")
	}
	_, err := materializeFactoryTemplate(scheduledIntakeTemplateID, models.ProviderGitHub, factoryTemplateInput{installParams: map[string]string{"appRepository": "acme/app", "type": "cron", "cronExpression": "nonsense"}})
	require.Error(t, err)
}
