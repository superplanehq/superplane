package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/services"
)

func Test__workOrderEmailDisplayStatus(t *testing.T) {
	open := &models.FactoryWorkOrder{State: models.FactoryWorkOrderStateOpen}
	active := []models.FactoryWorkOrderExecutionRecord{{
		FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
			Status: models.FactoryWorkOrderExecutionStatusRunning,
		},
	}}

	assert.Equal(t, "draft", workOrderEmailDisplayStatus(&models.FactoryWorkOrder{
		State: models.FactoryWorkOrderStateDraft,
	}, nil))
	assert.Equal(t, "waiting", workOrderEmailDisplayStatus(open, nil))
	assert.Equal(t, "running", workOrderEmailDisplayStatus(open, active))
	assert.Equal(t, "completed", workOrderEmailDisplayStatus(&models.FactoryWorkOrder{
		State:  models.FactoryWorkOrderStateClosed,
		Result: models.FactoryWorkOrderResultCompleted,
	}, nil))
	assert.Equal(t, "failed", workOrderEmailDisplayStatus(&models.FactoryWorkOrder{
		State:  models.FactoryWorkOrderStateClosed,
		Result: models.FactoryWorkOrderResultFailed,
	}, nil))
	assert.Equal(t, "cancelled", workOrderEmailDisplayStatus(&models.FactoryWorkOrder{
		State:  models.FactoryWorkOrderStateClosed,
		Result: models.FactoryWorkOrderResultRejected,
	}, nil))
}

func Test__applyWorkOrderEmailCard(t *testing.T) {
	now := time.Now()
	bugsLine := uuid.New()
	ciLine := uuid.New()
	order := &models.FactoryWorkOrder{
		Title:     "Metrics on list of lines",
		State:     models.FactoryWorkOrderStateOpen,
		UpdatedAt: now.Add(-8 * time.Hour),
		Assignees: []models.FactoryWorkOrderAssignee{
			{User: &models.User{Name: "Ana Souza"}},
			{User: &models.User{Name: "Bo Chen"}},
		},
	}
	executions := []models.FactoryWorkOrderExecutionRecord{
		{
			FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
				LineID:    bugsLine,
				StepName:  "CI Loop",
				Status:    models.FactoryWorkOrderExecutionStatusFinished,
				UpdatedAt: now.Add(-2 * time.Hour),
			},
			LineName: "Bugs",
		},
		{
			FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
				LineID:    ciLine,
				StepName:  "Verify",
				Status:    models.FactoryWorkOrderExecutionStatusFinished,
				UpdatedAt: now.Add(-9 * time.Hour),
			},
			LineName: "Hotfix",
		},
	}

	data := services.WorkOrderNotificationTemplateData{}
	applyWorkOrderEmailCard(&data, order, executions, now)

	assert.Equal(t, "Waiting", data.StatusLabel)
	assert.Equal(t, "#b45309", data.StatusFg)
	assert.Equal(t, "#fffbeb", data.StatusBg)
	assert.Equal(t, "#fde68a", data.StatusBorder)
	assert.Equal(t, "#f59e0b", data.StatusDot)
	assert.Equal(t, "Bugs · CI Loop · +1 more line", data.LineStepLabel)
	assert.Equal(t, "8h ago", data.UpdatedLabel)
	assert.Equal(t, "AS", data.AssigneeInitials)
	assert.Equal(t, "+1", data.AssigneeOverflow)
}

func Test__workOrderEmailInitials(t *testing.T) {
	assert.Equal(t, "AS", workOrderEmailInitials("Ana Souza"))
	assert.Equal(t, "M", workOrderEmailInitials("Maria"))
	assert.Equal(t, "?", workOrderEmailInitials("   "))
	assert.Equal(t, "AB", workOrderEmailInitials("Ada Lovelace Byron"))
}

func Test__formatWorkOrderEmailTimeAgo(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	assert.Equal(t, "12s ago", formatWorkOrderEmailTimeAgo(now.Add(-12*time.Second), now))
	assert.Equal(t, "3m ago", formatWorkOrderEmailTimeAgo(now.Add(-3*time.Minute), now))
	assert.Equal(t, "8h ago", formatWorkOrderEmailTimeAgo(now.Add(-8*time.Hour), now))
	assert.Equal(t, "2d ago", formatWorkOrderEmailTimeAgo(now.Add(-48*time.Hour), now))
	assert.Equal(t, "", formatWorkOrderEmailTimeAgo(time.Time{}, now))
}
