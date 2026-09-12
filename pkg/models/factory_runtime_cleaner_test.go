package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__FactoryRuntimeCleaner__KeepsWorkspaceAndWipesRuntime(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	completedAt := time.Now().Add(-time.Hour)
	require.NoError(t, db.Model(factoryModel).Update("onboarding_completed_at", completedAt).Error)
	budget := int64(2500)
	require.NoError(t, factoryModel.UpdateHostedSpendBudget(db, &budget))

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("github"),
		map[string]any{},
	)
	require.NoError(t, err)

	line, err := factoryModel.CreateLine(db, "implement", nil)
	require.NoError(t, err)

	intakeCanvas := support.CreateFactoryCanvas(t, r, factoryModel.ID, "GitHub issues")
	intake, err := factoryModel.CreateIntake(db, intakeCanvas.ID, models.FactoryIntakeSourceGitHubIssues)
	require.NoError(t, err)

	handlerCanvas := support.CreateFactoryCanvas(t, r, factoryModel.ID, "PR Closure")
	handler, err := factoryModel.CreatePRFeedbackHandler(
		db,
		handlerCanvas.ID,
		models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
		models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion,
	)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)

	order, err := factoryModel.CreateWorkOrder(db, "Order", "", &r.User, []uuid.UUID{r.User}, nil)
	require.NoError(t, err)
	require.Greater(t, factoryModel.NextWorkOrderNumber, int64(1))

	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, nil)
	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "step",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusFinished,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, db.Create(&execution).Error)

	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/9",
	})
	require.NoError(t, err)
	require.NoError(t, pullRequest.LinkRun(db, run.ID, "Address review"))

	_, err = order.ReportCheck(db, models.FactoryWorkOrderCheckParams{
		Key:      "risk-review",
		Name:     "Risk review",
		Score:    42,
		MaxScore: 100,
	})
	require.NoError(t, err)

	userID := r.User.String()
	_, err = order.RecordCommentAdded(db, models.FactoryWorkOrderCommentParams{
		Body: "Ship after the check passes.",
		Author: factory.WorkOrderCommentAuthor{
			Kind:   factory.CommentAuthorKindUser,
			UserID: &userID,
		},
	})
	require.NoError(t, err)

	_, err = order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeLink,
		Data: map[string]any{
			"url":    "https://github.com/acme/app/pull/9",
			"title":  "Draft PR",
			"number": "9",
		},
		CreatedBy: &r.User,
	})
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		Filename:       "shot.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, file.MarkReady(db, 12, "abc"))

	require.NoError(t, models.AddCanvasMemory(canvas.ID, "notes", map[string]any{"k": "v"}))

	session := &models.AgentSession{
		ID:                uuid.New(),
		OrganizationID:    r.Organization.ID,
		UserID:            r.User,
		CanvasID:          canvas.ID,
		Provider:          "anthropic",
		ProviderSessionID: "sesn_reset",
		Status:            models.AgentSessionStatusIdle,
	}
	require.NoError(t, models.CreateAgentSessionInTransaction(db, session))

	planningCanvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	planningSession, err := factoryModel.StartPlanningSession(db, models.StartPlanningSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/app",
		CanvasID:        planningCanvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)
	require.NoError(t, planningSession.SendUserMessage(db, "Plan the empty state."))

	merge := models.NewFactoryVelocityRepositoryMerge(
		r.Organization.ID,
		factoryModel.ID,
		"acme/app",
		12,
		models.FactoryVelocityMergeSourcePeople,
		time.Now(),
	)
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(
		db,
		factoryModel.ID,
		time.Now().Add(-24*time.Hour),
		time.Now().Add(time.Hour),
		[]models.FactoryVelocityRepositoryMerge{merge},
	))
	_, err = models.ClaimFactoryVelocitySync(db, factoryModel.ID, time.Now().Add(time.Hour))
	require.NoError(t, err)

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}))

	_, err = models.AddAdminLLMCreditGrant(db, r.Organization.ID, 1_000_000, "local top-up", &r.Account.ID)
	require.NoError(t, err)
	expireWelcomeGrant(t, db, r.Organization.ID)
	require.NoError(t, models.SetOrganizationPolarCustomerID(db, r.Organization.ID, "cus_local"))
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":                      models.BillingPlanBusiness,
			"plan_source":               models.BillingPlanSourcePolar,
			"polar_subscription_id":     "sub_local",
			"polar_subscription_status": models.PolarSubscriptionStatusActive,
		}).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := models.NewFactoryRuntimeCleaner(tx, factoryModel).Run(); err != nil {
			return err
		}
		return models.ResetOrganizationBillingTrial(tx, r.Organization.ID)
	}))

	reloaded, err := models.FindFactory(db, r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	assert.Equal(t, factoryModel.Name, reloaded.Name)
	assert.Equal(t, int64(1), reloaded.NextWorkOrderNumber)
	require.NotNil(t, reloaded.HostedSpendBudgetCents)
	assert.Equal(t, budget, *reloaded.HostedSpendBudgetCents)
	require.NotNil(t, reloaded.OnboardingCompletedAt)
	assert.WithinDuration(t, completedAt.UTC(), reloaded.OnboardingCompletedAt.UTC(), time.Second)

	_, err = models.FindIntegrationInTransaction(db, r.Organization.ID, integration.ID)
	require.NoError(t, err)

	foundLine, err := factoryModel.FindLine(db, line.ID)
	require.NoError(t, err)
	assert.Equal(t, line.Name, foundLine.Name)

	foundIntake, err := factoryModel.FindIntake(db, intake.ID)
	require.NoError(t, err)
	assert.Equal(t, intake.CanvasID, foundIntake.CanvasID)

	foundHandler, err := factoryModel.FindPRFeedbackHandler(db, handler.ID)
	require.NoError(t, err)
	assert.Equal(t, handler.CanvasID, foundHandler.CanvasID)

	_, err = models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	_, err = models.FindCanvasInTransaction(db, r.Organization.ID, intakeCanvas.ID)
	require.NoError(t, err)

	assertCount(t, db, &models.FactoryWorkOrder{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.FactoryWorkOrderExecution{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.FactoryPullRequest{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.FactoryWorkOrderComment{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.FactoryWorkOrderArtifact{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.File{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.CanvasRun{}, "workflow_id = ?", canvas.ID, 0)
	assertCount(t, db, &models.AgentSession{}, "canvas_id = ?", canvas.ID, 0)
	assertCount(t, db, &models.CanvasMemory{}, "canvas_id = ?", canvas.ID, 0)
	assertCount(t, db, &models.FactoryPlanningSession{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.FactoryVelocityRepositoryMerge{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.FactoryVelocitySync{}, "factory_id = ?", factoryModel.ID, 0)
	assertCount(t, db, &models.WorkspaceUsageEvent{}, "organization_id = ?", r.Organization.ID, 0)

	_, err = models.FindFile(db, file.ID)
	assert.ErrorIs(t, err, models.ErrFileNotFound)

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceSystem, plan.PlanSource)
	assert.Nil(t, plan.PolarSubscriptionID)
	require.NotNil(t, plan.TrialEndsAt)
	assert.WithinDuration(t, time.Now().Add(models.DefaultWelcomeGrantTTL), *plan.TrialEndsAt, 5*time.Second)

	settings, err := models.FindOrganizationLLMSettings(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, settings)
	assert.Nil(t, settings.PolarCustomerID)

	var extraGrants int64
	require.NoError(t, db.Model(&models.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind <> ?", r.Organization.ID, models.LLMCreditGrantKindWelcome).
		Count(&extraGrants).Error)
	assert.Equal(t, int64(0), extraGrants)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.RemainingMicros)
	require.NotNil(t, summary.WelcomeCreditExpiresAt)
	assert.WithinDuration(t, time.Now().Add(models.DefaultWelcomeGrantTTL), *summary.WelcomeCreditExpiresAt, 5*time.Second)
}

func assertCount(t *testing.T, db *gorm.DB, model any, query string, arg any, want int64) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(model).Where(query, arg).Count(&count).Error)
	assert.Equal(t, want, count)
}
