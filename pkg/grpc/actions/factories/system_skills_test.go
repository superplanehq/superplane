package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestSystemSkillResourceProtection(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "refine-task", true, models.FactoryAgentResourceConfig{
		Markdown:    "# Refine task",
		SystemSkill: &models.FactorySystemSkill{TemplateID: "refine-task", Version: 1},
	})
	require.NoError(t, err)
	orgID := r.Organization.ID.String()
	name, enabled := "renamed", false
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"rename", func() error {
			_, err := UpdateFactoryAgentResource(t.Context(), IntakeDependencies{}, orgID, &pb.UpdateFactoryAgentResourceRequest{FactoryId: factory.ID.String(), ResourceId: resource.ID.String(), Name: &name})
			return err
		}},
		{"disable", func() error {
			_, err := UpdateFactoryAgentResource(t.Context(), IntakeDependencies{}, orgID, &pb.UpdateFactoryAgentResourceRequest{FactoryId: factory.ID.String(), ResourceId: resource.ID.String(), Enabled: &enabled})
			return err
		}},
		{"delete", func() error {
			_, err := DeleteFactoryAgentResource(t.Context(), IntakeDependencies{}, orgID, &pb.DeleteFactoryAgentResourceRequest{FactoryId: factory.ID.String(), ResourceId: resource.ID.String()})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			require.Error(t, err)
			assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			found, err := factory.FindAgentResource(db, resource.ID)
			require.NoError(t, err)
			assert.Equal(t, "refine-task", found.Name)
			assert.True(t, found.Enabled)
		})
	}
	// An editor saves the unchanged name together with edited content.
	sameName, markdown, stillEnabled := "refine-task", "# Customized refinement", true
	updated, err := UpdateFactoryAgentResource(t.Context(), IntakeDependencies{}, orgID, &pb.UpdateFactoryAgentResourceRequest{
		FactoryId: factory.ID.String(), ResourceId: resource.ID.String(), Name: &sameName, Markdown: &markdown, Enabled: &stillEnabled,
	})
	require.NoError(t, err)
	assert.Equal(t, markdown, updated.GetResource().GetMarkdown())
	assert.Equal(t, "refine-task", updated.GetResource().GetSystemSkill().GetTemplateId())
	assert.EqualValues(t, 1, updated.GetResource().GetSystemSkill().GetVersion())

	// The list response must expose the metadata that locks UI controls.
	listed, err := ListFactoryAgentResources(t.Context(), orgID, &pb.ListFactoryAgentResourcesRequest{FactoryId: factory.ID.String(), Kind: pb.FactoryAgentResource_KIND_SKILL})
	require.NoError(t, err)
	require.Len(t, listed.Resources, 1)
	assert.NotNil(t, listed.Resources[0].SystemSkill)
}
