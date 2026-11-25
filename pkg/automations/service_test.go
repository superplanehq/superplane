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
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureFactoryAsCode))

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

func TestServiceValidatesSecretsInOneOrganizationScopedQuery(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, "Secrets", "", "SEC")
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	for _, name := range []string{"FIRST", "SECOND"} {
		_, err := models.CreateSecret(name, "local", r.User.String(), models.DomainTypeOrganization, r.Organization.ID, []byte("{}"))
		require.NoError(t, err)
	}
	_, err = models.CreateSecret("FOREIGN", "local", r.User.String(), models.DomainTypeOrganization, uuid.New(), []byte("{}"))
	require.NoError(t, err)

	type queryScope struct{}
	ctx := context.WithValue(t.Context(), queryScope{}, true)
	queries := 0
	callback := "test:automation_secret_queries"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(query *gorm.DB) {
		if query.Statement.Context.Value(queryScope{}) == true && query.Statement.Table == "secrets" {
			queries++
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })
	service := automations.Service{Backend: backendWithoutCanvases{}}
	placement := automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: string(automationspec.Backlog)}
	source := automationSource + "    secrets: [FIRST, SECOND]\n  - name: Again\n    bash: echo again\n    secrets: [FIRST]\n"
	_, err = service.Validate(ctx, placement, source)
	require.NoError(t, err)
	require.Equal(t, 1, queries)

	queries = 0
	_, err = service.Validate(ctx, placement, strings.Replace(source, "[FIRST]", "[FOREIGN]", 1))
	require.ErrorContains(t, err, "steps[1].secrets[0]")
	require.ErrorContains(t, err, "FOREIGN")
	require.Equal(t, 1, queries)

	queries = 0
	_, err = service.Validate(ctx, placement, automationSource)
	require.NoError(t, err)
	require.Zero(t, queries)
}

func TestServiceCustomStageRequiresUnifiedFlagAndFactoryAsCode(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, "Custom stage", "", "")
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "Main", nil)
	require.NoError(t, err)
	stageID := uuid.New()
	stage := models.FactoryLineStage{ID: stageID, Key: "custom-" + stageID.String(), Name: "QA", Kind: models.FactoryLineStagePhase}
	stages := line.StagesValue()
	stages = append(stages[:1], append([]models.FactoryLineStage{stage}, stages[1:]...)...)
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	service := automations.Service{Backend: backendWithoutCanvases{}}
	placement := automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: stage.Key}
	source := strings.Replace(automationSource, "use: manual", "use: task_enters_stage", 1)

	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	_, err = service.Validate(t.Context(), placement, source)
	require.ErrorContains(t, err, "Factory as code")
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	_, err = service.Validate(t.Context(), placement, source)
	require.NoError(t, err)
	placement.Stage = "custom-missing"
	_, err = service.Validate(t.Context(), placement, source)
	require.ErrorContains(t, err, "Select a custom stage")
	placement.Stage = stage.Key
	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	_, err = service.Validate(t.Context(), placement, source)
	require.ErrorContains(t, err, "Custom Automations")
}

var errCannotPublish = errors.New("backend could not publish")

type backendWithoutCanvases struct{}

func (backendWithoutCanvases) ArtifactStages(_ *gorm.DB, _ automations.Placement, artifactIDs []string) (map[string]string, error) {
	stages := make(map[string]string, len(artifactIDs))
	for _, id := range artifactIDs {
		stages[id] = strings.TrimPrefix(id, "runner://artifact/")
	}
	return stages, nil
}

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
		previous.ID = "runner://artifact/" + placement.Stage
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
