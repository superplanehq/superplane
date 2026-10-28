package models

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	TaskOwnerSourceAgent = "agent"

	taskOwnerReasonMaxRunes = 280
)

// TaskOwnerCandidate is a member who can own a task: an active human member
// of the organization. GitHubLogins and GitHubIDs hold every GitHub identity
// in Associated accounts, so commits from an older identity still map to the
// member. They are empty when the member has no GitHub account. GitHubLogin
// is the active one. GitHubIDs are the numeric GitHub user IDs, which do not
// change when a login is renamed.
type TaskOwnerCandidate struct {
	UserID       uuid.UUID
	Name         string
	Email        string
	IsOwner      bool
	Roles        []string
	GitHubLogin  string
	GitHubLogins []string
	GitHubIDs    []string
	OpenTasks    int
}

// TaskOwnerAssignment is the owner choice of an automated assignment.
type TaskOwnerAssignment struct {
	UserID uuid.UUID
	Source string
	Reason string
}

type taskOwnerCandidateScan struct {
	UserID    uuid.UUID `gorm:"column:user_id"`
	Name      string    `gorm:"column:name"`
	Email     string    `gorm:"column:email"`
	IsOwner   bool      `gorm:"column:is_owner"`
	Login     string    `gorm:"column:login"`
	GitHubID  string    `gorm:"column:github_id"`
	Active    bool      `gorm:"column:active"`
	OpenTasks int       `gorm:"column:open_tasks"`
}

// HasGitHubLogin reports whether login is one of the member's linked GitHub
// logins. GitHub logins are case-insensitive.
func (c *TaskOwnerCandidate) HasGitHubLogin(login string) bool {
	login = normalizeTaskOwnerLogin(login)
	if login == "" {
		return false
	}
	return slices.Contains(c.GitHubLogins, login)
}

// HasGitHubID reports whether id is the numeric user ID of one of the
// member's associated GitHub accounts.
func (c *TaskOwnerCandidate) HasGitHubID(id string) bool {
	id = strings.TrimSpace(id)
	return id != "" && slices.Contains(c.GitHubIDs, id)
}

// FindTaskOwnerCandidate resolves a member by user ID or GitHub login. A
// leading "@" on the login is ignored.
func FindTaskOwnerCandidate(candidates []TaskOwnerCandidate, query string) (TaskOwnerCandidate, bool) {
	query = strings.TrimSpace(query)
	if id, err := uuid.Parse(query); err == nil {
		for _, candidate := range candidates {
			if candidate.UserID == id {
				return candidate, true
			}
		}
		return TaskOwnerCandidate{}, false
	}

	for _, candidate := range candidates {
		if candidate.HasGitHubLogin(query) {
			return candidate, true
		}
	}
	return TaskOwnerCandidate{}, false
}

// ListTaskOwnerCandidates returns the members who can own a task in the
// factory, with the number of open tasks each one owns there. A draft or open
// task counts as open. Roles are not loaded: they live in the authorization
// service.
func ListTaskOwnerCandidates(tx *gorm.DB, orgID, factoryID uuid.UUID) ([]TaskOwnerCandidate, error) {
	var rows []taskOwnerCandidateScan
	err := tx.Raw(
		listTaskOwnerCandidatesSQL,
		factoryID,
		[]string{FactoryWorkOrderStateDraft, FactoryWorkOrderStateOpen},
		ProviderGitHub,
		orgID,
		UserTypeHuman,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	candidates := []TaskOwnerCandidate{}
	indexByUser := map[uuid.UUID]int{}
	for _, row := range rows {
		index, seen := indexByUser[row.UserID]
		if !seen {
			candidates = append(candidates, TaskOwnerCandidate{
				UserID:    row.UserID,
				Name:      row.Name,
				Email:     row.Email,
				IsOwner:   row.IsOwner,
				OpenTasks: row.OpenTasks,
			})
			index = len(candidates) - 1
			indexByUser[row.UserID] = index
		}
		login := normalizeTaskOwnerLogin(row.Login)
		if login == "" {
			continue
		}
		candidate := &candidates[index]
		if !slices.Contains(candidate.GitHubLogins, login) {
			candidate.GitHubLogins = append(candidate.GitHubLogins, login)
		}
		if id := strings.TrimSpace(row.GitHubID); id != "" && !slices.Contains(candidate.GitHubIDs, id) {
			candidate.GitHubIDs = append(candidate.GitHubIDs, id)
		}
		if row.Active || candidate.GitHubLogin == "" {
			candidate.GitHubLogin = strings.TrimSpace(row.Login)
		}
	}
	return candidates, nil
}

const listTaskOwnerCandidatesSQL = `
WITH open_tasks AS (
	SELECT a.user_id, COUNT(*) AS open_tasks
	FROM factory_work_order_assignees a
	INNER JOIN factory_work_orders wo ON wo.id = a.work_order_id
	WHERE wo.factory_id = ?
		AND wo.state IN ?
	GROUP BY a.user_id
)
SELECT
	u.id AS user_id,
	u.name,
	COALESCE(u.email, '') AS email,
	u.is_owner,
	COALESCE(l.username, '') AS login,
	COALESCE(l.provider_id, '') AS github_id,
	COALESCE(l.active, false) AS active,
	COALESCE(ot.open_tasks, 0) AS open_tasks
FROM users u
LEFT JOIN account_linked_accounts l ON l.account_id = u.account_id AND l.provider = ?
LEFT JOIN open_tasks ot ON ot.user_id = u.id
WHERE u.organization_id = ?
	AND u.type = ?
	AND u.deleted_at IS NULL
ORDER BY u.id, l.active DESC, l.linked_at DESC
`

// AssignOwnerIfUnowned makes the person the only owner when the task has no
// owner. It reports false and changes nothing when the task already has an
// owner, so an automated choice never replaces a choice a person made.
func (o *FactoryWorkOrder) AssignOwnerIfUnowned(tx *gorm.DB, assignment TaskOwnerAssignment) (bool, error) {
	assigned := false
	err := tx.Transaction(func(tx *gorm.DB) error {
		if err := lockFactoryWorkOrder(tx, o.ID); err != nil {
			return err
		}

		var owners int64
		if err := tx.Model(&FactoryWorkOrderAssignee{}).Where("work_order_id = ?", o.ID).Count(&owners).Error; err != nil {
			return err
		}
		if owners > 0 {
			return nil
		}

		if err := o.ReplaceAssignees(tx, []uuid.UUID{assignment.UserID}); err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(o).Omit(clause.Associations).Update("updated_at", now).Error; err != nil {
			return err
		}
		o.UpdatedAt = now
		o.Assignees = []FactoryWorkOrderAssignee{{WorkOrderID: o.ID, UserID: assignment.UserID}}

		assigned = true
		return o.recordEvent(tx, factory.EventTypeOrderAssigneesUpdated, factory.WorkOrderAssigneesUpdated{
			Order:    o.Ref(),
			Assigned: []factory.UserRef{{ID: assignment.UserID}},
			Source:   assignment.Source,
			Reason:   NormalizeTaskOwnerReason(assignment.Reason),
		})
	})
	return assigned, err
}

func normalizeTaskOwnerLogin(login string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(login), "@"))
}

var taskOwnerReasonLink = regexp.MustCompile(`(?i)\b(?:[a-z][a-z0-9+.-]*://|www\.)\S*`)

// NormalizeTaskOwnerReason makes the agent's reason safe to show in emails
// and the timeline. The agent reads untrusted intake text, so the reason
// keeps one line of plain text without links.
func NormalizeTaskOwnerReason(reason string) string {
	reason = taskOwnerReasonLink.ReplaceAllString(reason, "")
	reason = strings.Join(strings.Fields(reason), " ")
	runes := []rune(reason)
	if len(runes) <= taskOwnerReasonMaxRunes {
		return reason
	}
	return strings.TrimSpace(string(runes[:taskOwnerReasonMaxRunes]))
}
