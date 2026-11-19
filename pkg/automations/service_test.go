package automations_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

const automationSource = `# Preserve this comment and the final newline.
apiVersion: v2
kind: Automation
name: Observe
triggers:
  - use: manual
steps:
  - name: Print marker
    bash: echo ready
`

func TestServiceKeepsSourceAndBackendRevisionTogether(t *testing.T) {
	ctx := t.Context()
	resources := support.Setup(t)
	orgID := resources.Organization.ID
	db := database.DB(ctx)
	factory, err := models.CreateFactory(db, orgID, "Automation test", "", "AAC")
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureFactoryCustomAutomations))
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureAutomationsAsCode))

	service := automations.Service{Backend: backendWithoutCanvases{}}
	placement := automations.Placement{OrganizationID: orgID, FactoryID: factory.ID, Stage: string(automationspec.Backlog)}
	doc, err := service.Validate(ctx, placement, automationSource)
	require.NoError(t, err)
	require.Equal(t, "echo ready", doc.Steps[0].Bash)
	summaries, err := service.List(ctx, placement)
	require.NoError(t, err)
	require.Empty(t, summaries)

	failedSource := strings.Replace(automationSource, "echo ready", "echo fail", 1)
	_, err = service.Create(ctx, placement, resources.User, failedSource)
	require.ErrorIs(t, err, errCannotPublish)
	summaries, err = service.List(ctx, placement)
	require.NoError(t, err)
	require.Empty(t, summaries)
	currentFactory, err := models.FindFactory(db, orgID, factory.ID)
	require.NoError(t, err)
	require.Empty(t, currentFactory.Description)

	first, err := service.Create(ctx, placement, resources.User, automationSource)
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Revision)
	require.Equal(t, automationSource, first.YAML)
	require.Equal(t, placement.FactoryID, first.FactoryID)
	require.Equal(t, placement.Stage, first.Stage)

	record, err := models.FindAutomationSource(db, orgID, first.ID)
	require.NoError(t, err)
	require.NotEqual(t, first.ID.String(), record.ArtifactID)

	readSource := func() automations.Automation {
		t.Helper()
		read, err := service.Read(ctx, orgID, first.ID)
		require.NoError(t, err)
		return read
	}
	require.Equal(t, first, readSource())
	_, err = service.Update(ctx, orgID, first.ID, resources.User, automationSource+"# A second save\n")
	require.NoError(t, err)
	beforeFailure := readSource()
	require.EqualValues(t, 2, beforeFailure.Revision)

	_, err = service.Update(ctx, orgID, first.ID, resources.User, failedSource)
	require.ErrorIs(t, err, errCannotPublish)
	require.Equal(t, beforeFailure, readSource())
	currentFactory, err = models.FindFactory(db, orgID, factory.ID)
	require.NoError(t, err)
	require.Equal(t, "echo ready", currentFactory.Description)

	type saveResult struct {
		automation automations.Automation
		err        error
	}
	saves := make(chan saveResult, 2)
	for _, command := range []string{"echo alpha", "echo beta"} {
		go func() {
			saved, err := service.Update(ctx, orgID, first.ID, resources.User, strings.Replace(automationSource, "echo ready", command, 1))
			saves <- saveResult{saved, err}
		}()
	}
	for _, saved := range []saveResult{<-saves, <-saves} {
		require.NoError(t, saved.err)
		require.Equal(t, first.ID, saved.automation.ID)
	}
	final := readSource()
	require.EqualValues(t, 4, final.Revision)
	currentFactory, err = models.FindFactory(db, orgID, factory.ID)
	require.NoError(t, err)
	require.Contains(t, final.YAML, currentFactory.Description)
	require.Contains(t, []string{"echo alpha", "echo beta"}, currentFactory.Description)

	summaries, err = service.List(ctx, placement)
	require.NoError(t, err)
	require.Equal(t, []automations.Summary{first.Summary}, summaries)
	require.NoError(t, service.Delete(ctx, orgID, first.ID))
	_, err = service.Read(ctx, orgID, first.ID)
	require.Error(t, err)
	currentFactory, err = models.FindFactory(db, orgID, factory.ID)
	require.NoError(t, err)
	require.Empty(t, currentFactory.Description)
}

var errCannotPublish = errors.New("backend could not publish")

type backendWithoutCanvases struct{}

func (backendWithoutCanvases) Check(context.Context, *gorm.DB, automations.Placement, *automationspec.Document) error {
	return nil
}

func (backendWithoutCanvases) Apply(
	_ context.Context, tx *gorm.DB, placement automations.Placement, _, _ uuid.UUID, previous automations.Artifact, doc *automationspec.Document,
) (automations.Artifact, func(), error) {
	command := doc.Steps[0].Bash
	factory, err := models.FindFactory(tx, placement.OrganizationID, placement.FactoryID)
	if err != nil {
		return automations.Artifact{}, nil, err
	}
	if err := factory.Update(tx, nil, &command, nil); err != nil {
		return automations.Artifact{}, nil, err
	}
	if command == "echo fail" {
		return automations.Artifact{}, nil, errCannotPublish
	}
	if previous.ID == "" {
		previous.ID = "runner://artifact/91"
	}
	previous.Revision = uuid.NewString()
	return previous, nil, nil
}

func (backendWithoutCanvases) Remove(_ context.Context, tx *gorm.DB, placement automations.Placement, _ automations.Artifact) (func(), error) {
	factory, err := models.FindFactory(tx, placement.OrganizationID, placement.FactoryID)
	if err != nil {
		return nil, err
	}
	empty := ""
	return nil, factory.Update(tx, nil, &empty, nil)
}
