package factories

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

func TestFactoryConfigurationExportBatchesMetadataLiveVersionsAndSources(t *testing.T) {
	f := newConfigurationLifecycleFixture(t)
	db := database.DB(t.Context())
	type queryCountKey struct{}
	const callback = "test:configuration_version_queries"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if count, ok := tx.Statement.Context.Value(queryCountKey{}).(*atomic.Int64); ok && slices.Contains([]string{"workflow_versions", "factory_intakes", "factory_pr_feedback_handlers", "automations_as_code"}, tx.Statement.Table) {
			count.Add(1)
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })

	export := func() int64 {
		t.Helper()
		var count atomic.Int64
		ctx := context.WithValue(f.ctx, queryCountKey{}, &count)
		response, err := ExportFactoryConfiguration(ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
			FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(),
		})
		require.NoError(t, err)
		require.NotEmpty(t, response.Bundle.Files)
		return count.Load()
	}
	bundle := f.export(t)
	addConfigurationLifecycleAutomation(t, bundle, "audit", "Audit")
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	singleQueries := export()
	for range 4 {
		resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
	}
	response, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
		FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(),
	})
	require.NoError(t, err)
	bundle = response.Bundle
	for i := range 4 {
		addConfigurationLifecycleAutomation(t, bundle, fmt.Sprintf("audit-%d", i), fmt.Sprintf("Audit %d", i))
	}
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.Positive(t, singleQueries)
	require.Equal(t, singleQueries, export(), "live graph, metadata, and source reads must not grow with automation count")
}
