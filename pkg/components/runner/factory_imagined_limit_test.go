package runner_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestAppendFactoryImaginedLimitPromptSkipsCanvasWithoutFactory(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	prompt := "Implement the task. Finish within 25 minutes."
	steps := []runner.AgentStep{{Name: "Implement", Type: runner.AgentStepPrompt, Prompt: &prompt}}

	got := runner.AppendFactoryImaginedLimitPrompt(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, steps)

	assert.Equal(t, steps, got)
	assert.NotContains(t, *got[0].Prompt, runner.FactoryImaginedLimitPrompt)
	assert.Contains(t, *got[0].Prompt, "Finish within 25 minutes.")
}

func TestAppendFactoryImaginedLimitPromptSkipsInvalidCanvasID(t *testing.T) {
	prompt := "Implement the task."
	steps := []runner.AgentStep{{Name: "Implement", Type: runner.AgentStepPrompt, Prompt: &prompt}}

	got := runner.AppendFactoryImaginedLimitPrompt(core.ExecutionContext{
		OrganizationID: "not-a-uuid",
		WorkflowID:     "also-not",
	}, steps)

	assert.Equal(t, steps, got)
}

func TestAppendFactoryImaginedLimitPromptSkipsPlanningSession(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Planning")
	runID := uuid.New()
	now := time.Now()
	session := &models.FactoryPlanningSession{
		ID:              uuid.New(),
		OrganizationID:  r.Organization.ID,
		FactoryID:       factory.ID,
		CreatedByUserID: &r.User,
		Repository:      "acme/payments",
		Kind:            models.PlanningSessionKindTaskCreation,
		State:           models.PlanningSessionStateRunning,
		CanvasID:        &canvas.ID,
		CanvasRunID:     &runID,
		HeartbeatAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, db.Create(session).Error)
	prompt := "Refine the task."
	steps := []runner.AgentStep{{Name: "Plan", Type: runner.AgentStepPrompt, Prompt: &prompt}}

	got := runner.AppendFactoryImaginedLimitPrompt(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		RunID:          runID,
	}, steps)

	assert.Equal(t, prompt, *got[0].Prompt)
	assert.NotContains(t, *got[0].Prompt, runner.FactoryImaginedLimitPrompt)
}

func TestAppendFactoryImaginedLimitPromptAddsInstructionToEachPromptStep(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	command := "git status"
	first := "Implement the task. Finish within 25 minutes. Token budget is 10000."
	second := "Write the pull request title."
	already := "Review the change.\n\n" + runner.FactoryImaginedLimitPrompt
	steps := []runner.AgentStep{
		{Name: "Status", Type: runner.AgentStepBash, Command: &command},
		{Name: "Implement", Type: runner.AgentStepPrompt, Prompt: &first},
		{Name: "Describe", Type: runner.AgentStepPrompt, Prompt: &second},
		{Name: "Review", Type: runner.AgentStepPrompt, Prompt: &already},
	}

	got := runner.AppendFactoryImaginedLimitPrompt(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		RunID:          uuid.New(),
	}, steps)

	assert.Equal(t, command, *got[0].Command)
	assert.Equal(t, first, *steps[1].Prompt)
	assert.Contains(t, *got[1].Prompt, "Finish within 25 minutes.")
	assert.Contains(t, *got[1].Prompt, "Token budget is 10000.")
	assert.Contains(t, *got[1].Prompt, runner.FactoryImaginedLimitPrompt)
	assert.Contains(t, *got[2].Prompt, "Write the pull request title.")
	assert.Contains(t, *got[2].Prompt, runner.FactoryImaginedLimitPrompt)
	assert.Equal(t, 1, strings.Count(*got[1].Prompt, runner.FactoryImaginedLimitPrompt))
	assert.Equal(t, 1, strings.Count(*got[3].Prompt, runner.FactoryImaginedLimitPrompt))

	again := runner.AppendFactoryImaginedLimitPrompt(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		RunID:          uuid.New(),
	}, got)
	assert.Equal(t, 1, strings.Count(*again[1].Prompt, runner.FactoryImaginedLimitPrompt))
	assert.Equal(t, 1, strings.Count(*again[2].Prompt, runner.FactoryImaginedLimitPrompt))
}
