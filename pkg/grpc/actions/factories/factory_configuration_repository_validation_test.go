package factories

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/proto"
)

func TestFactoryConfigurationDiagnosesRepositoryFieldsBeforeWrites(t *testing.T) {
	f := newConfigurationFixture(t)
	original := f.export(t)
	db := database.DB(t.Context())
	before, err := models.FindFactory(db, f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	for _, test := range []struct{ name, section, field string }{
		{"missing connection", "binding: {}\n  primary: {repository: acme/app, ref: main}", "repositories.binding"},
		{"missing primary", "binding: {integration_name: github}", "repositories.primary"},
		{"missing name", "binding: {integration_name: Missing}\n  primary: {repository: acme/app, ref: main}", "repositories.binding.integration_name"},
		{"unavailable backlog", "binding: {integration_name: github}\n  primary: {repository: acme/app, ref: main}\n  backlog_repository: acme/unavailable", "repositories.backlog_repository"},
		{"both connection references", "binding: {integration_name: github, integration_id: old}\n  primary: {repository: acme/app}", "repositories.binding.integration_name"},
		{"missing access", "binding: {integration_name: github}\n  primary: {repository: acme/app, ref: main}\n  additional: [{repository: acme/docs}]", "repositories.additional[0].access"},
		{"invalid access", "binding: {integration_name: github}\n  primary: {repository: acme/app, ref: main}\n  additional: [{repository: acme/docs, access: admin}]", "repositories.additional[0].access"},
		{"invalid enabled", "binding: {integration_name: github}\n  primary: {repository: acme/app, ref: main}\n  additional: [{repository: acme/docs, access: read, enabled: 'false'}]", "repositories.additional[0].enabled"},
		{"diagnostic preserves order", "binding: {integration_name: github}\n  primary: {repository: acme/app, ref: main}\n  additional: [{repository: z/docs, access: read}, {repository: invalid, access: read}]", "repositories.additional[1]"},
		{"duplicate repository", "binding: {integration_name: github}\n  primary: {repository: acme/app, ref: main}\n  additional: [{repository: acme/docs, access: read}, {repository: ACME/DOCS, access: write}]", "repositories.additional[1].repository"},
	} {
		t.Run(test.name, func(t *testing.T) {
			bundle := proto.Clone(original).(*pb.FactoryConfigurationBundle)
			// Replace the exported section with the exact input shape under validation.
			start := strings.Index(bundle.Files[0].Content, "repositories:")
			end := strings.Index(bundle.Files[0].Content, "stages:")
			require.Greater(t, end, start)
			test.section = strings.ReplaceAll(test.section, "integration_name: github", "integration_name: "+strconv.Quote(integrationName(t, f.r.Organization.ID, f.factory.OnboardingConfigValue().VCSIntegrationID)))
			bundle.Files[0].Content = bundle.Files[0].Content[:start] + "repositories:\n  " + test.section + "\n" + bundle.Files[0].Content[end:]
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			require.Equal(t, test.field, plan.Diagnostics[0].Field)
			require.Empty(t, plan.BaseStateToken)
			_, err := f.apply(bundle, "invalid")
			require.Error(t, err)
			after, err := models.FindFactory(db, f.r.Organization.ID, f.factory.ID)
			require.NoError(t, err)
			require.Equal(t, before.OnboardingConfig, after.OnboardingConfig)
		})
	}
}

func TestFactoryConfigurationUsesProviderDefaultForOmittedPrimaryRef(t *testing.T) {
	f := newConfigurationFixture(t)
	f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{gitHubDefault: func(string) (string, error) { return "trunk", nil }})
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) { manifest.Repositories.Primary.Ref = "" })
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	saved, err := models.FindFactory(database.DB(t.Context()), f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, "trunk", saved.OnboardingConfigValue().DefaultBranch)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}
