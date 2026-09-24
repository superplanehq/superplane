package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

// FactoryVelocityPullRequest is one factory pull request that merged or closed
// inside a velocity window, joined to the work order that produced it. The
// attribution columns let the velocity report group merges by intake source and
// by the member assigned to the task.
type FactoryVelocityPullRequest struct {
	WorkOrderID uuid.UUID
	Repository  string
	URL         string
	Number      int64
	MergedAt    *time.Time
	ClosedAt    *time.Time
	// Member who opened the work order. Nil when an automation opened it.
	CreatedByID *uuid.UUID
	// Members assigned to the work order, earliest assignment first. Tied
	// assignments sort by user ID so the credited member stays stable.
	AssigneeIDs []uuid.UUID
	// Source of the intake that opened the work order (for example
	// `github-issues`). Empty when the order did not come from an intake.
	IntakeSource string
	// External ticket the work order was imported from. Imported orders carry
	// no intake run, so this is the only signal of where they came from.
	OriginURL string
	// Hours from the first execution start to the close event. Nil when the
	// order never ran or is still open.
	CycleHours *float64
}

// Merged reports whether the pull request landed. A pull request that carries
// no merge stamp and a close stamp is factory waste.
func (p *FactoryVelocityPullRequest) Merged() bool {
	return p.MergedAt != nil
}

// FactoryVelocityMember is an organization member with the identity fields the
// velocity People table renders. GitHubLogin ties the member to the pull
// requests they authored in the connected repository.
type FactoryVelocityMember struct {
	UserID    uuid.UUID
	Name      string
	Email     string
	AvatarURL string
	// The column tag is required: the default naming strategy maps this field to
	// "git_hub_login", so the query alias would not bind and every member would
	// look like they have no GitHub identity.
	GitHubLogin string `gorm:"column:github_login"`
}

// ListFactoryVelocityPullRequests returns every factory pull request that
// merged or closed without merging in [from, to), with the work order
// attribution the velocity report groups by.
//
// Close time comes from the latest `order.status.updated` event that moved the
// order to closed, matching how line metrics measure a closed order.
func ListFactoryVelocityPullRequests(
	tx *gorm.DB,
	factoryID uuid.UUID,
	from, to time.Time,
) ([]FactoryVelocityPullRequest, error) {
	var scanned []factoryVelocityPullRequestScan
	err := tx.Raw(listFactoryVelocityPullRequestsSQL,
		factoryID,
		factoryID,
		factoryID,
		factory.EventTypeOrderStatusUpdated,
		FactoryWorkOrderStateClosed,
		factoryID,
		from, to,
		from, to,
	).Scan(&scanned).Error
	if err != nil {
		return nil, err
	}

	rows := make([]FactoryVelocityPullRequest, len(scanned))
	for i := range scanned {
		row, err := scanned[i].asPullRequest()
		if err != nil {
			return nil, err
		}
		rows[i] = row
	}
	return rows, nil
}

const listFactoryVelocityPullRequestsSQL = `
WITH intake_source AS (
	SELECT wo.id AS work_order_id, fi.source
	FROM factory_work_orders wo
	INNER JOIN workflow_runs r ON r.id = wo.source_run_id
	INNER JOIN factory_intakes fi ON fi.canvas_id = r.workflow_id
	WHERE wo.factory_id = ?
),
first_execution AS (
	SELECT x.work_order_id, MIN(x.created_at) AS started_at
	FROM factory_work_order_executions x
	WHERE x.factory_id = ?
	GROUP BY x.work_order_id
),
latest_close AS (
	SELECT DISTINCT ON (e.work_order_id)
		e.work_order_id,
		e.created_at AS closed_at
	FROM factory_work_order_events e
	INNER JOIN factory_work_orders wo ON wo.id = e.work_order_id
	WHERE wo.factory_id = ?
		AND e.type = ?
		AND e.data->>'toState' = ?
	ORDER BY e.work_order_id, e.created_at DESC
)
SELECT
	p.work_order_id,
	p.repository,
	p.url,
	p.number,
	p.merged_at,
	p.closed_at,
	wo.created_by_id,
	COALESCE(asn.ids, '{}'::uuid[])::text AS assignee_ids,
	COALESCE(ins.source, '') AS intake_source,
	COALESCE(wo.origin_url, '') AS origin_url,
	CASE
		WHEN fe.started_at IS NOT NULL
			AND lc.closed_at IS NOT NULL
			AND lc.closed_at > fe.started_at
		THEN EXTRACT(EPOCH FROM (lc.closed_at - fe.started_at)) / 3600.0
	END AS cycle_hours
FROM factory_pull_requests p
INNER JOIN factory_work_orders wo ON wo.id = p.work_order_id
LEFT JOIN intake_source ins ON ins.work_order_id = wo.id
LEFT JOIN first_execution fe ON fe.work_order_id = wo.id
LEFT JOIN latest_close lc ON lc.work_order_id = wo.id
LEFT JOIN LATERAL (
	SELECT array_agg(a.user_id ORDER BY a.created_at ASC, a.user_id ASC) AS ids
	FROM factory_work_order_assignees a
	WHERE a.work_order_id = p.work_order_id
) asn ON true
WHERE p.factory_id = ?
	AND (
		(p.merged_at IS NOT NULL AND p.merged_at >= ? AND p.merged_at < ?)
		OR (
			p.merged_at IS NULL
			AND p.closed_at IS NOT NULL
			AND p.closed_at >= ? AND p.closed_at < ?
		)
	)
`

// ListFactoryVelocityMembers returns the human members of an organization with
// their GitHub identity, when they linked one.
//
// A member can arrive at a GitHub login two ways: they linked a GitHub account
// on purpose, or they sign in with GitHub. The link wins, because a member who
// links an account states which author they are. Signing in with GitHub stays a
// fallback so members who joined that way keep their attribution.
func ListFactoryVelocityMembers(tx *gorm.DB, orgID uuid.UUID) ([]FactoryVelocityMember, error) {
	var members []FactoryVelocityMember
	err := tx.Raw(listFactoryVelocityMembersSQL, ProviderGitHub, ProviderGitHub, orgID, UserTypeHuman).Scan(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

const listFactoryVelocityMembersSQL = `
SELECT DISTINCT ON (u.id)
	u.id AS user_id,
	u.name,
	COALESCE(NULLIF(l.avatar_url, ''), p.avatar_url, '') AS avatar_url,
	COALESCE(u.email, '') AS email,
	COALESCE(NULLIF(l.username, ''), p.username, '') AS github_login
FROM users u
LEFT JOIN accounts a ON a.id = u.account_id
LEFT JOIN account_linked_accounts l ON l.account_id = a.id AND l.provider = ?
LEFT JOIN account_providers p ON p.account_id = a.id AND p.provider = ?
WHERE u.organization_id = ?
	AND u.type = ?
	AND u.deleted_at IS NULL
ORDER BY u.id, l.linked_at DESC NULLS LAST, p.updated_at DESC NULLS LAST
`

type factoryVelocityPullRequestScan struct {
	WorkOrderID  uuid.UUID
	Repository   string
	URL          string
	Number       int64
	MergedAt     *time.Time
	ClosedAt     *time.Time
	CreatedByID  *uuid.UUID
	AssigneeIDs  string
	IntakeSource string
	OriginURL    string
	CycleHours   *float64
}

func (s factoryVelocityPullRequestScan) asPullRequest() (FactoryVelocityPullRequest, error) {
	assigneeIDs, err := parseUUIDArray(s.AssigneeIDs)
	if err != nil {
		return FactoryVelocityPullRequest{}, err
	}
	return FactoryVelocityPullRequest{
		WorkOrderID:  s.WorkOrderID,
		Repository:   s.Repository,
		URL:          s.URL,
		Number:       s.Number,
		MergedAt:     s.MergedAt,
		ClosedAt:     s.ClosedAt,
		CreatedByID:  s.CreatedByID,
		AssigneeIDs:  assigneeIDs,
		IntakeSource: s.IntakeSource,
		OriginURL:    s.OriginURL,
		CycleHours:   s.CycleHours,
	}, nil
}

func parseUUIDArray(raw string) ([]uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil, nil
	}
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part == "" || strings.EqualFold(part, "null") {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			return nil, fmt.Errorf("parse assignee id %q: %w", part, err)
		}
		out = append(out, id)
	}
	return out, nil
}
