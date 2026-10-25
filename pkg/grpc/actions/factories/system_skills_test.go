package factories

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/features"

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
	enableWorkspaceSkills(t, r.Organization.ID)
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

func TestSystemSkillRestoreAndState(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	template, ok := runner.SystemSkillTemplateByID("refine-task")
	require.True(t, ok)
	for _, tc := range []struct {
		name       string
		original   string
		edit       bool
		templateID string
		state      pb.FactoryAgentResource_SystemSkill_State
		restore    bool
	}{
		{"latest", "", false, "refine-task", pb.FactoryAgentResource_SystemSkill_STATE_LATEST, true},
		{"custom", "", true, "refine-task", pb.FactoryAgentResource_SystemSkill_STATE_CUSTOM, true},
		{"older instructions", "# Previously supplied instructions", false, "refine-task", pb.FactoryAgentResource_SystemSkill_STATE_UPDATE_AVAILABLE, true},
		{"unavailable template", "# Unavailable", false, "missing-template", pb.FactoryAgentResource_SystemSkill_STATE_UNAVAILABLE, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			original := tc.original
			version := int32(1)
			if original == "" {
				original = template.Markdown("workspace-refine")
				version = template.Version
			}
			resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "workspace-refine", true, models.FactoryAgentResourceConfig{
				Markdown: original, SystemSkill: &models.FactorySystemSkill{TemplateID: tc.templateID, Version: version},
			})
			require.NoError(t, err)
			if tc.edit {
				edit := "# My custom instructions"
				_, err = UpdateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryAgentResourceRequest{
					FactoryId: factory.ID.String(), ResourceId: resource.ID.String(), Markdown: &edit,
				})
				require.NoError(t, err)
			}
			before, err := factory.FindAgentResource(db, resource.ID)
			require.NoError(t, err)
			listed, err := ListFactoryAgentResources(t.Context(), r.Organization.ID.String(), &pb.ListFactoryAgentResourcesRequest{FactoryId: factory.ID.String(), Kind: pb.FactoryAgentResource_KIND_SKILL})
			require.NoError(t, err)
			require.Len(t, listed.Resources, 1)
			assert.Equal(t, tc.state, listed.Resources[0].GetSystemSkill().GetState())
			assert.Equal(t, version, listed.Resources[0].GetSystemSkill().GetVersion())
			if tc.state == pb.FactoryAgentResource_SystemSkill_STATE_UPDATE_AVAILABLE {
				assert.Greater(t, listed.Resources[0].GetSystemSkill().GetLatestVersion(), version)
			}
			// Merely reading a stale template must not update it in this slice.
			afterRead, err := factory.FindAgentResource(db, resource.ID)
			require.NoError(t, err)
			assert.Equal(t, before.Config, afterRead.Config)
			assert.Equal(t, before.UpdatedAt, afterRead.UpdatedAt)
			response, err := RestoreFactorySystemSkill(t.Context(), r.Organization.ID.String(), &pb.RestoreFactorySystemSkillRequest{FactoryId: factory.ID.String(), ResourceId: resource.ID.String()})
			if !tc.restore {
				require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, pb.FactoryAgentResource_SystemSkill_STATE_LATEST, response.GetResource().GetSystemSkill().GetState())
			assert.Equal(t, template.Version, response.GetResource().GetSystemSkill().GetVersion())
			assert.Contains(t, response.GetResource().GetMarkdown(), "You refine draft tasks")
			restored, err := factory.FindAgentResource(db, resource.ID)
			require.NoError(t, err)
			assert.Equal(t, "workspace-refine", restored.Name)
			assert.True(t, restored.Enabled)
			assert.Equal(t, template.Markdown(restored.Name), restored.Config.Data().Markdown)
			assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256([]byte(restored.Config.Data().Markdown))), restored.Config.Data().SystemSkill.ContentHash)
			// Saving unchanged content keeps Latest; an actual edit becomes Custom.
			for _, edit := range []struct {
				markdown string
				state    pb.FactoryAgentResource_SystemSkill_State
			}{
				{restored.Config.Data().Markdown, pb.FactoryAgentResource_SystemSkill_STATE_LATEST},
				{"# Changed after restore", pb.FactoryAgentResource_SystemSkill_STATE_CUSTOM},
			} {
				saved, err := UpdateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryAgentResourceRequest{FactoryId: factory.ID.String(), ResourceId: resource.ID.String(), Markdown: &edit.markdown})
				require.NoError(t, err)
				assert.Equal(t, edit.state, saved.GetResource().GetSystemSkill().GetState())
			}
		})
	}
}

func TestRestoreSystemSkillRejectsInvalidTargetsAndDisabledFeature(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ordinary, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "ordinary", true, models.FactoryAgentResourceConfig{Markdown: "# My skill"})
	require.NoError(t, err)
	for _, tc := range []struct {
		factoryID, resourceID string
		code                  codes.Code
	}{
		{factory.ID.String(), ordinary.ID.String(), codes.FailedPrecondition},
		{factory.ID.String(), "invalid", codes.InvalidArgument},
		{factory.ID.String(), uuid.NewString(), codes.NotFound},
	} {
		_, err := RestoreFactorySystemSkill(t.Context(), r.Organization.ID.String(), &pb.RestoreFactorySystemSkillRequest{FactoryId: tc.factoryID, ResourceId: tc.resourceID})
		assert.Equal(t, tc.code, grpcerrors.Code(err))
	}
	other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("other"), "", "")
	require.NoError(t, err)
	_, err = RestoreFactorySystemSkill(t.Context(), r.Organization.ID.String(), &pb.RestoreFactorySystemSkillRequest{FactoryId: other.ID.String(), ResourceId: ordinary.ID.String()})
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	system, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "refine-task", true, models.FactoryAgentResourceConfig{
		Markdown: "# Previous instructions", SystemSkill: &models.FactorySystemSkill{TemplateID: "refine-task", Version: 1},
	})
	require.NoError(t, err)
	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureWorkspaceSkills))
	_, err = RestoreFactorySystemSkill(t.Context(), r.Organization.ID.String(), &pb.RestoreFactorySystemSkillRequest{FactoryId: factory.ID.String(), ResourceId: system.ID.String()})
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	found, err := factory.FindAgentResource(db, ordinary.ID)
	require.NoError(t, err)
	assert.Equal(t, "# My skill", found.Config.Data().Markdown)
}
