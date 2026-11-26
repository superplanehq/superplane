package factories

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	canvasyaml "github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func newConfigurationLifecycleFixture(t *testing.T) configurationFixture {
	t.Helper()
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	vcsID := createReadyOnboardingIntegration(t, f.r.Organization.ID, models.ProviderGitHub)
	repository, branch := "acme/app", "main"
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &vcsID, AppRepository: &repository, DefaultBranch: &branch,
	}))
	support.AssignOpenSystemTrial(t, f.r.Organization.ID)
	upsertHostedOnboardingProvider(t, db)
	f.r.Registry.Integrations[models.ProviderGitHub] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Triggers: []core.Trigger{impl.NewDummyIntegrationTrigger(impl.DummyIntegrationTriggerOptions{Name: prFeedbackTriggerComponent(models.ProviderGitHub)})},
	})
	return f
}

func addConfigurationLifecycleAutomation(t *testing.T, bundle *pb.FactoryConfigurationBundle, id, name string) {
	t.Helper()
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	reference := "automations/" + id + ".yaml"
	manifest.Stages[2].Automations = append(manifest.Stages[2].Automations, factoryConfigurationAutomation{ID: id, Name: name, Uses: reference})
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	bundle.Files = append(bundle.Files, &pb.FactoryConfigurationFile{
		Path:    ".superplane/" + reference,
		Content: "apiVersion: v2\nkind: Automation\nname: " + name + "\nruns_on_task: true\ntriggers:\n  - use: manual\nsteps:\n  - name: Inspect\n    prompt: Inspect the task.\n",
	})
}

func TestFactoryConfigurationCreatesAndRemovesAutomations(t *testing.T) {
	f := newConfigurationLifecycleFixture(t)
	db := database.DB(t.Context())
	optional := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	bundle := f.export(t)
	addConfigurationLifecycleAutomation(t, bundle, "alpha-audit", "Alpha audit")
	addConfigurationLifecycleAutomation(t, bundle, "beta-audit", "Beta audit")
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 2)
	require.Equal(t, before, configurationResourceCounts(t), "planning must not create resources")
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	alpha, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Alpha audit")
	require.NoError(t, err)
	beta, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Beta audit")
	require.NoError(t, err)
	require.Equal(t, "verify", *alpha.ColumnKey)
	alphaSource, err := models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, alpha.ID.String())
	require.NoError(t, err)
	require.Equal(t, resetTestLiveVersion(t, alpha).ID.String(), alphaSource.ArtifactRevision)
	after := configurationResourceCounts(t)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, after, configurationResourceCounts(t), "friendly IDs must remain idempotent")
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
		FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(),
	})
	require.NoError(t, err)
	plan = f.plan(t, exported.Bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)

	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	manifest.Stages[2].Automations = slices.DeleteFunc(manifest.Stages[2].Automations, func(automation factoryConfigurationAutomation) bool {
		return automation.Name != "Beta audit"
	})
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	bundle.Files = slices.DeleteFunc(bundle.Files, func(file *pb.FactoryConfigurationFile) bool {
		return file.Path == ".superplane/automations/alpha-audit.yaml"
	})
	bundle.Files[1].Content = strings.Replace(bundle.Files[1].Content, "Inspect the task.", "Review the task.", 1)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 3)
	result, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	for _, removed := range []*models.Canvas{alpha, optional} {
		_, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, removed.ID)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	}
	_, err = models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, alpha.ID.String())
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	betaSource, err := models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, beta.ID.String())
	require.NoError(t, err)
	require.Contains(t, betaSource.Source, "Review the task.")
	steps, err := customAutomationProtoSteps(selectCustomAutomationAgent(resetTestLiveVersion(t, beta).Nodes).Configuration["steps"])
	require.NoError(t, err)
	require.Equal(t, "Review the task.", steps[0].Prompt)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationRejectsRemovingPhaseRunner(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	manifest.Stages[1].Automations = []factoryConfigurationAutomation{}
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.NotEmpty(t, plan.Diagnostics)
	require.Empty(t, plan.BaseStateToken)
	_, err = f.apply(bundle, "invalid")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationRecreatesDeletedAutomation(t *testing.T) {
	for _, deletion := range []string{"UI deletion", "stale source", "source still in use"} {
		t.Run(deletion, func(t *testing.T) {
			f := newConfigurationLifecycleFixture(t)
			db := database.DB(t.Context())
			bundle := f.export(t)
			addConfigurationLifecycleAutomation(t, bundle, "audit", "Audit")
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			_, err := f.apply(bundle, plan.BaseStateToken)
			require.NoError(t, err)
			original, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Audit")
			require.NoError(t, err)
			source, err := models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, original.ID.String())
			require.NoError(t, err)
			if deletion == "UI deletion" {
				_, err = DeleteFactoryAutomation(f.ctx, f.r.Organization.ID.String(), &pb.DeleteFactoryAutomationRequest{
					FactoryId: f.factory.ID.String(), AutomationId: original.ID.String(),
				})
				require.NoError(t, err)
			} else {
				// Older deletion paths left the source behind after deleting its canvas.
				require.NoError(t, original.SoftDeleteInTransaction(db))
				if deletion == "source still in use" {
					source.ArtifactID = f.canvas.ID.String()
					require.NoError(t, source.Save(db))
				}
			}
			before := configurationResourceCounts(t)
			plan = f.plan(t, bundle)
			require.Equal(t, before, configurationResourceCounts(t), "planning must not repair or create resources")
			if deletion == "source still in use" {
				require.NotEmpty(t, plan.Diagnostics)
				require.Equal(t, "stages[2].automations[0].id", plan.Diagnostics[0].Field)
				_, err = f.apply(bundle, plan.BaseStateToken)
				require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
				require.Equal(t, before, configurationResourceCounts(t))
				return
			}
			require.Empty(t, plan.Diagnostics)
			require.Len(t, plan.Changes, 1)
			result, err := f.apply(bundle, plan.BaseStateToken)
			require.NoError(t, err)
			require.True(t, result.Changed)
			recreated, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Audit")
			require.NoError(t, err)
			require.NotEqual(t, original.ID, recreated.ID)
			stored, err := models.FindAutomationSource(db, f.r.Organization.ID, source.ID)
			require.NoError(t, err)
			require.Equal(t, recreated.ID.String(), stored.ArtifactID)
			plan = f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			require.Empty(t, plan.Changes)
		})
	}
}

func TestFactoryConfigurationRetainsDuplicateBuiltInIdentity(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	first := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	second := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	firstName, secondName := "First confidence", "Second confidence"
	require.NoError(t, canvases.UpdateCanvasInTransaction(db, first, &firstName, nil))
	require.NoError(t, canvases.UpdateCanvasInTransaction(db, second, &secondName, nil))
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	require.Len(t, manifest.Stages[2].Automations, 2)
	manifest.Stages[2].Automations = manifest.Stages[2].Automations[1:]
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	_, err = models.FindCanvasInTransaction(db, f.r.Organization.ID, first.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = models.FindCanvasInTransaction(db, f.r.Organization.ID, second.ID)
	require.NoError(t, err)
	before := configurationResourceCounts(t)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes, "removing the first alias must retain the second automation's identity")
	result, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationRemovesIntakeAndFeedbackAttachments(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	integrationID := createReadyOnboardingIntegration(t, f.r.Organization.ID, models.ProviderGitHub)
	intakeDocument, err := buildIntakeCanvas(intakeCanvasRequest{
		OrganizationID: f.r.Organization.ID, FactoryID: f.factory.ID, Source: models.FactoryIntakeSourceGitHubIssues,
		Settings: intakeSettingsOrDefault(models.FactoryIntakeSourceGitHubIssues, intakeSettings{}),
		Binding:  &intakeBinding{Configuration: map[string]any{"repository": "acme/app"}, Integration: &canvasyaml.IntegrationRef{ID: integrationID}},
	})
	require.NoError(t, err)
	intakeCanvas := support.CreateFactoryCanvas(t, f.r, f.factory.ID, "GitHub intake")
	resetTestPublishCanvas(t, f.r, intakeCanvas, intakeDocument)
	_, err = f.factory.CreateIntake(db, intakeCanvas.ID, models.FactoryIntakeSourceGitHubIssues)
	require.NoError(t, err)
	feedbackCanvas := support.CreateFactoryCanvas(t, f.r, f.factory.ID, "PR feedback")
	resetTestPublishCanvas(t, f.r, feedbackCanvas, buildDiscussionPRFeedbackCanvas(prFeedbackBuildRequest{
		Repository: "acme/app", VCSProvider: models.ProviderGitHub, Agent: &intakeAgent{Component: models.SuperPlaneRunnerComponent},
	}))
	_, err = f.factory.CreatePRFeedbackHandler(db, feedbackCanvas.ID,
		models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion)
	require.NoError(t, err)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	require.Len(t, manifest.Stages[0].Automations, 1)
	require.Len(t, manifest.Stages[2].Automations, 1)
	manifest.Stages[0].Automations = []factoryConfigurationAutomation{}
	manifest.Stages[2].Automations = []factoryConfigurationAutomation{}
	content, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 2)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	intakes, err := f.factory.ListIntakes(db)
	require.NoError(t, err)
	require.Empty(t, intakes)
	handlers, err := f.factory.ListPRFeedbackHandlers(db)
	require.NoError(t, err)
	require.Empty(t, handlers)
	for _, canvas := range []*models.Canvas{intakeCanvas, feedbackCanvas} {
		_, err = models.FindCanvasInTransaction(db, f.r.Organization.ID, canvas.ID)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	}
}

func TestFactoryConfigurationRejectsRemovingForeignAttachments(t *testing.T) {
	for _, attachment := range []string{"intake", "feedback handler"} {
		t.Run(attachment, func(t *testing.T) {
			f := newConfigurationLifecycleFixture(t)
			db := database.DB(t.Context())
			omitted := f.export(t)
			bundle := proto.Clone(omitted).(*pb.FactoryConfigurationBundle)
			addConfigurationLifecycleAutomation(t, bundle, "audit", "Audit")
			planned := f.plan(t, bundle)
			require.Empty(t, planned.Diagnostics)
			_, err := f.apply(bundle, planned.BaseStateToken)
			require.NoError(t, err)
			canvas, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Audit")
			require.NoError(t, err)
			source, err := models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, canvas.ID.String())
			require.NoError(t, err)
			planned = f.plan(t, omitted)
			require.Empty(t, planned.Diagnostics, "the optional automation must be removable before an inconsistent attachment exists")
			require.Len(t, planned.Changes, 1)

			otherOrganization := support.CreateOrganization(t, f.r, f.r.User)
			otherFactory, err := models.CreateFactory(db, otherOrganization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			var savedAttachment any
			if attachment == "intake" {
				_, err = otherFactory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
				require.NoError(t, err)
				savedAttachment, err = models.FindFactoryIntakeByCanvasID(db, canvas.ID)
			} else {
				_, err = otherFactory.CreatePRFeedbackHandler(db, canvas.ID,
					models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion)
				require.NoError(t, err)
				savedAttachment, err = models.FindPRFeedbackHandlerByCanvasID(db, canvas.ID)
			}
			require.NoError(t, err)
			before := configurationResourceCounts(t)
			blocked := f.plan(t, omitted)
			require.Len(t, blocked.Diagnostics, 1)
			require.Contains(t, blocked.Diagnostics[0].Message, "Cannot remove automation")
			require.Empty(t, blocked.BaseStateToken)
			_, err = f.apply(omitted, planned.BaseStateToken)
			require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			require.Equal(t, before, configurationResourceCounts(t))
			retained, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, canvas.ID)
			require.NoError(t, err)
			require.Equal(t, canvas, retained)
			retainedSource, err := models.FindAutomationSourceByArtifact(db, f.r.Organization.ID, canvas.ID.String())
			require.NoError(t, err)
			require.Equal(t, source, retainedSource)
			if attachment == "intake" {
				retained, err := models.FindFactoryIntakeByCanvasID(db, canvas.ID)
				require.NoError(t, err)
				require.Equal(t, savedAttachment, retained)
			} else {
				retained, err := models.FindPRFeedbackHandlerByCanvasID(db, canvas.ID)
				require.NoError(t, err)
				require.Equal(t, savedAttachment, retained)
			}
		})
	}
}

func TestFactoryConfigurationRollsBackAutomationMembershipChanges(t *testing.T) {
	for _, failure := range []string{"stale membership", "publication failure"} {
		t.Run(failure, func(t *testing.T) {
			f := newConfigurationLifecycleFixture(t)
			db := database.DB(t.Context())
			optional := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
			bundle := f.export(t)
			var manifest factoryConfigurationManifest
			require.NoError(t, yaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
			manifest.Stages[2].Automations = []factoryConfigurationAutomation{}
			content, err := yaml.Marshal(manifest)
			require.NoError(t, err)
			bundle.Files[0].Content = string(content)
			addConfigurationLifecycleAutomation(t, bundle, "alpha-audit", "Alpha audit")
			addConfigurationLifecycleAutomation(t, bundle, "beta-audit", "Beta audit")
			plan := f.plan(t, bundle)
			require.Empty(t, plan.Diagnostics)
			if failure == "stale membership" {
				resetTestTemplate(t, f.r, f.factory, "pr-closure", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
			} else {
				require.NoError(t, db.Callback().Create().Before("gorm:create").Register("factory_configuration_membership_failure", func(tx *gorm.DB) {
					canvas, ok := tx.Statement.Dest.(*models.Canvas)
					if ok && canvas.Name == "Beta audit" {
						tx.AddError(errors.New("publication failed"))
					}
				}))
				t.Cleanup(func() { _ = db.Callback().Create().Remove("factory_configuration_membership_failure") })
			}
			before := configurationResourceCounts(t)
			_, err = f.apply(bundle, plan.BaseStateToken)
			require.Error(t, err)
			if failure == "stale membership" {
				require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			}
			require.Equal(t, before, configurationResourceCounts(t))
			_, err = models.FindCanvasInTransaction(db, f.r.Organization.ID, optional.ID)
			require.NoError(t, err, "a failed apply must retain the omitted automation")
		})
	}
}

func TestFactoryConfigurationRestoresDeletedAutomationWithReusedFilename(t *testing.T) {
	f := newConfigurationLifecycleFixture(t)
	db := database.DB(t.Context())
	bundle := f.export(t)
	addConfigurationLifecycleAutomation(t, bundle, "inspect-first", "Inspect Task")
	addConfigurationLifecycleAutomation(t, bundle, "inspect-second", "Inspect Task!")
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	first, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Inspect Task")
	require.NoError(t, err)
	survivor, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Inspect Task!")
	require.NoError(t, err)
	survivorVersion := survivor.LiveVersionID
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.ID.String()})
	require.NoError(t, err)
	saved := exported.Bundle
	_, err = DeleteFactoryAutomation(f.ctx, f.r.Organization.ID.String(), &pb.DeleteFactoryAutomationRequest{
		FactoryId: f.factory.ID.String(), AutomationId: first.ID.String(),
	})
	require.NoError(t, err)
	plan = f.plan(t, saved)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	result, err := f.apply(saved, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, result.Changed)
	restored, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Inspect Task")
	require.NoError(t, err)
	require.NotEqual(t, first.ID, restored.ID)
	current, err := models.FindCanvasInTransaction(db, f.r.Organization.ID, survivor.ID)
	require.NoError(t, err)
	require.Equal(t, "Inspect Task!", current.Name)
	require.Equal(t, survivorVersion, current.LiveVersionID)
	plan = f.plan(t, saved)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}
