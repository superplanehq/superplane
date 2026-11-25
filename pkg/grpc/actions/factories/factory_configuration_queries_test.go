package factories

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

func TestFactoryConfigurationExportBatchesLiveVersions(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	type queryCountKey struct{}
	const callback = "test:configuration_version_queries"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if count, ok := tx.Statement.Context.Value(queryCountKey{}).(*atomic.Int64); ok && tx.Statement.Table == "workflow_versions" {
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
	singleQueries := export()
	for range 4 {
		resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
	}
	require.Positive(t, singleQueries)
	require.Equal(t, singleQueries, export(), "live graph reads must not grow with automation count")
}
