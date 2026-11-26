package factories

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestFactoryConfigurationSanitizesInspectionDatabaseErrors(t *testing.T) {
	f := newConfigurationLifecycleFixture(t)
	bundle := f.export(t)
	addConfigurationLifecycleAutomation(t, bundle, "audit", "Audit")
	bundle.Files[len(bundle.Files)-1].Content = strings.Replace(bundle.Files[len(bundle.Files)-1].Content,
		"triggers:", "agent: {model: sonnet}\ntriggers:", 1)
	planned := f.plan(t, bundle)
	require.Empty(t, planned.Diagnostics)
	_, err := f.apply(bundle, planned.BaseStateToken)
	require.NoError(t, err)
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String()})
	require.NoError(t, err)
	bundle = exported.Bundle
	planned = f.plan(t, bundle)
	require.Empty(t, planned.Diagnostics)
	before := configurationResourceCounts(t)

	type databaseFaultKey struct{}
	const privateDetail = "private-database-host: relation organization_hosted_model_allowlists unavailable (SQLSTATE 57P01)"
	const callback = "test:configuration_inspection_database_failure"
	var faults atomic.Int64
	db := database.DB(t.Context())
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Context.Value(databaseFaultKey{}) == true && tx.Statement.Table == "organization_hosted_model_allowlists" {
			faults.Add(1)
			tx.AddError(errors.New(privateDetail))
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })
	ctx := context.WithValue(f.ctx, databaseFaultKey{}, true)
	for _, operation := range []string{"export", "plan", "apply"} {
		t.Run(operation, func(t *testing.T) {
			faults.Store(0)
			switch operation {
			case "export":
				_, err = ExportFactoryConfiguration(ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String()})
			case "plan":
				_, err = PlanFactoryConfiguration(ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.PlanFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle})
			case "apply":
				_, err = ApplyFactoryConfiguration(ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.ApplyFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle, BaseStateToken: planned.BaseStateToken})
			}
			require.Positive(t, faults.Load(), "the request must reach the failing policy read during custom export")
			require.Error(t, err)
			assert.Equal(t, codes.Internal, grpcerrors.Code(err))
			assert.NotContains(t, grpcerrors.StatusMessage(err), privateDetail)
			assert.Equal(t, before, configurationResourceCounts(t))
		})
	}
	t.Run("unsupported canvas keeps its domain diagnostic", func(t *testing.T) {
		canvas, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Audit")
		require.NoError(t, err)
		live := resetTestLiveVersion(t, canvas)
		agent := selectCustomAutomationAgent(live.Nodes)
		require.NotNil(t, agent)
		unsupported := *agent
		unsupported.ID = "unsupported-agent"
		live.Nodes = append(live.Nodes, unsupported)
		require.NoError(t, db.Model(live).Update("nodes", datatypes.NewJSONSlice(live.Nodes)).Error)
		_, err = ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.ID.String(), LineId: f.line.ID.String()})
		require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), canvas.Name)
	})

}
