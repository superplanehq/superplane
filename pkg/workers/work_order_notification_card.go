package workers

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/services"
	"gorm.io/gorm"
)

type workOrderEmailStatusStyle struct {
	Label  string
	Fg     string
	Bg     string
	Border string
	Dot    string
}

// Light-theme colors from web_src/src/App.css so the email card matches
// the Kanban card in a typical inbox (white background).
var workOrderEmailStatusStyles = map[string]workOrderEmailStatusStyle{
	"draft": {
		Label:  "Draft",
		Fg:     "#52525b",
		Bg:     "#f4f4f5",
		Border: "#e4e4e7",
		Dot:    "#a1a1aa",
	},
	"running": {
		Label:  "Running",
		Fg:     "#1d4ed8",
		Bg:     "#eff6ff",
		Border: "#bfdbfe",
		Dot:    "#3b82f6",
	},
	"waiting": {
		Label:  "Waiting",
		Fg:     "#b45309",
		Bg:     "#fffbeb",
		Border: "#fde68a",
		Dot:    "#f59e0b",
	},
	"completed": {
		Label:  "Completed",
		Fg:     "#15803d",
		Bg:     "#ecfdf5",
		Border: "#a7f3d0",
		Dot:    "#16a34a",
	},
	"failed": {
		Label:  "Failed",
		Fg:     "#b91c1c",
		Bg:     "#fef2f2",
		Border: "#fecaca",
		Dot:    "#ef4444",
	},
	"cancelled": {
		Label:  "Cancelled",
		Fg:     "#52525b",
		Bg:     "#f4f4f5",
		Border: "#e4e4e7",
		Dot:    "#a1a1aa",
	},
}

func applyWorkOrderEmailCard(
	data *services.WorkOrderNotificationTemplateData,
	order *models.FactoryWorkOrder,
	executions []models.FactoryWorkOrderExecutionRecord,
	now time.Time,
) {
	style := workOrderEmailStatusStyleFor(workOrderEmailDisplayStatus(order, executions))
	data.StatusLabel = style.Label
	data.StatusFg = style.Fg
	data.StatusBg = style.Bg
	data.StatusBorder = style.Border
	data.StatusDot = style.Dot
	data.LineStepLabel = workOrderEmailLineStepLabel(executions)
	data.UpdatedLabel = formatWorkOrderEmailTimeAgo(order.UpdatedAt, now)
	data.AssigneeInitials, data.AssigneeOverflow = workOrderEmailAssignees(order)
}

func loadWorkOrderExecutionsForEmail(db *gorm.DB, orderID uuid.UUID) []models.FactoryWorkOrderExecutionRecord {
	byOrder, err := models.ListFactoryWorkOrderExecutionsByWorkOrderIDs(db, []uuid.UUID{orderID})
	if err != nil {
		log.Warnf("Failed to load work order executions for notification email: %v", err)
		return nil
	}
	return byOrder[orderID]
}

func workOrderEmailDisplayStatus(
	order *models.FactoryWorkOrder,
	executions []models.FactoryWorkOrderExecutionRecord,
) string {
	if order.State == models.FactoryWorkOrderStateClosed {
		switch order.Result {
		case models.FactoryWorkOrderResultRejected:
			return "cancelled"
		case models.FactoryWorkOrderResultFailed:
			return "failed"
		default:
			return "completed"
		}
	}

	if order.State == models.FactoryWorkOrderStateDraft {
		return "draft"
	}

	if workOrderEmailHasActiveExecution(executions) {
		return "running"
	}

	return "waiting"
}

func workOrderEmailStatusStyleFor(status string) workOrderEmailStatusStyle {
	if style, ok := workOrderEmailStatusStyles[status]; ok {
		return style
	}
	return workOrderEmailStatusStyles["waiting"]
}

func workOrderEmailHasActiveExecution(executions []models.FactoryWorkOrderExecutionRecord) bool {
	for _, execution := range executions {
		if isActiveWorkOrderEmailExecution(execution) {
			return true
		}
	}
	return false
}

func isActiveWorkOrderEmailExecution(execution models.FactoryWorkOrderExecutionRecord) bool {
	return execution.Status == models.FactoryWorkOrderExecutionStatusPending ||
		execution.Status == models.FactoryWorkOrderExecutionStatusRunning
}

func workOrderEmailLineStepLabel(executions []models.FactoryWorkOrderExecutionRecord) string {
	latest := latestWorkOrderEmailExecution(executions)
	if latest == nil {
		return ""
	}

	label := joinNonEmpty(" · ", strings.TrimSpace(latest.LineName), strings.TrimSpace(latest.StepName))
	if label == "" {
		return ""
	}

	otherLines := distinctWorkOrderEmailLineCount(executions) - 1
	if otherLines < 1 {
		return label
	}
	if otherLines == 1 {
		return label + " · +1 more line"
	}
	return fmt.Sprintf("%s · +%d more lines", label, otherLines)
}

func latestWorkOrderEmailExecution(
	executions []models.FactoryWorkOrderExecutionRecord,
) *models.FactoryWorkOrderExecutionRecord {
	if len(executions) == 0 {
		return nil
	}

	winner := executions[0]
	winnerAt := workOrderEmailExecutionTime(winner)
	for _, execution := range executions[1:] {
		at := workOrderEmailExecutionTime(execution)
		if at.After(winnerAt) || (at.Equal(winnerAt) && isActiveWorkOrderEmailExecution(execution)) {
			winner = execution
			winnerAt = at
		}
	}
	return &winner
}

func workOrderEmailExecutionTime(execution models.FactoryWorkOrderExecutionRecord) time.Time {
	if !execution.UpdatedAt.IsZero() {
		return execution.UpdatedAt
	}
	return execution.CreatedAt
}

func distinctWorkOrderEmailLineCount(executions []models.FactoryWorkOrderExecutionRecord) int {
	seen := map[uuid.UUID]struct{}{}
	for _, execution := range executions {
		if execution.LineID == uuid.Nil {
			continue
		}
		seen[execution.LineID] = struct{}{}
	}
	return len(seen)
}

func workOrderEmailAssignees(order *models.FactoryWorkOrder) (initials, overflow string) {
	names := make([]string, 0, len(order.Assignees))
	for _, assignee := range order.Assignees {
		if assignee.User == nil {
			continue
		}
		names = append(names, assignee.User.Name)
	}
	if len(names) == 0 {
		return "", ""
	}

	initials = workOrderEmailInitials(names[0])
	hidden := len(names) - 1
	if hidden < 1 {
		return initials, ""
	}
	return initials, fmt.Sprintf("+%d", hidden)
}

func workOrderEmailInitials(name string) string {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		return firstLetter(parts[0])
	}
	return firstLetter(parts[0]) + firstLetter(parts[len(parts)-1])
}

func firstLetter(value string) string {
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return string(unicode.ToUpper(r))
		}
	}
	r, _ := utf8.DecodeRuneInString(value)
	if r == utf8.RuneError {
		return "?"
	}
	return string(unicode.ToUpper(r))
}

func formatWorkOrderEmailTimeAgo(updatedAt, now time.Time) string {
	if updatedAt.IsZero() {
		return ""
	}

	delta := now.Sub(updatedAt)
	if delta < 0 {
		return "in " + formatWorkOrderEmailDuration(-delta)
	}
	return formatWorkOrderEmailDuration(delta) + " ago"
}

func formatWorkOrderEmailDuration(delta time.Duration) string {
	seconds := int(delta.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}

	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dd", hours/24)
}

func joinNonEmpty(sep string, parts ...string) string {
	filled := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		filled = append(filled, part)
	}
	return strings.Join(filled, sep)
}
