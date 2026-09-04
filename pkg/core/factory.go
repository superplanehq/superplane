package core

import (
	"errors"
	"time"
)

// ErrWorkOrderNotFound is returned by FindWorkOrder when nothing matches
// the given lookup. Components that treat "not found" as benign (e.g. a
// PR merge with no tracked order) check for it with errors.Is.
var ErrWorkOrderNotFound = errors.New("work order not found")

// ErrPullRequestNotFound is returned by FindPullRequest when nothing
// matches. Components that treat "not found" as benign check for it with
// errors.Is.
var ErrPullRequestNotFound = errors.New("pull request not found")

// ErrPullRequestActivityAlreadyActive is returned by AddPullRequestActivity
// when another active activity already owns the same handler and revision.
var ErrPullRequestActivityAlreadyActive = errors.New("pull request activity already active for this handler and revision")

type FactoryContext interface {
	CreateWorkOrder(params WorkOrderParams) (*WorkOrder, error)
	// FindWorkOrder resolves a work order by id or by one of its
	// artifacts' keys, without requiring the current run to be attached
	// to a `factory_work_order_executions` row. Returns ErrWorkOrderNotFound
	// (wrapped) when nothing matches.
	FindWorkOrder(params FindWorkOrderParams) (*WorkOrder, error)
	// UpdateWorkOrderStatus reports whether the row actually transitioned
	// via the second return value; callers must skip downstream emits
	// when `changed` is false so a no-op doesn't leak into the timeline.
	UpdateWorkOrderStatus(params UpdateWorkOrderStatusParams) (order *WorkOrder, changed bool, err error)
	AddWorkOrderComment(params AddWorkOrderCommentParams) error
	AddWorkOrderArtifact(params AddWorkOrderArtifactParams) (*WorkOrderArtifact, error)
	// ReportWorkOrderCheck upserts a scored check on the work order,
	// keyed by CheckKey: the first report creates the check, later
	// reports with the same key update it in place and keep the prior
	// score as PreviousScore. Every report also lands an
	// `order.check.reported` timeline event.
	ReportWorkOrderCheck(params ReportWorkOrderCheckParams) (*WorkOrderCheck, error)
	// SetWorkOrderStatusNote upserts a status note on the work order,
	// keyed by NoteKey: the first set creates the note, later sets with
	// the same key update it in place, and a different key sits beside
	// it. Any lifecycle transition clears the whole set. The order must
	// be open.
	SetWorkOrderStatusNote(params SetWorkOrderStatusNoteParams) (*WorkOrderStatusNote, error)
	AddPullRequest(params AddPullRequestParams) (*PullRequest, error)
	UpdatePullRequest(params UpdatePullRequestParams) (*PullRequest, error)
	FindPullRequest(params FindPullRequestParams) (*PullRequestMatch, error)
	// ListPullRequests returns the factory pull requests for a repository,
	// filtered by state. Used by components that need every matching row,
	// not a single lookup (e.g. rechecking mergeability after a base branch
	// push).
	ListPullRequests(params ListPullRequestsParams) ([]*PullRequest, error)
	AddPullRequestActivity(params AddPullRequestActivityParams) (*PullRequestActivityResult, error)
	UpdatePullRequestActivity(params UpdatePullRequestActivityParams) (*PullRequestActivityResult, error)
}

type WorkOrderParams struct {
	Title       string
	Description string
}

// FindWorkOrderParams configures FactoryContext.FindWorkOrder. By selects
// the lookup strategy: "id" resolves OrderID directly, "artifactKey"
// resolves the work order that owns the artifact tagged with ArtifactKey.
type FindWorkOrderParams struct {
	By          string
	OrderID     string
	ArtifactKey string
}

type UpdateWorkOrderStatusParams struct {
	// OrderID identifies the work order to target. Required; the
	// component field defaults to `{{ order().id }}` (the work order
	// driving the current run) but callers must always resolve and pass
	// an explicit id.
	OrderID string
	State   string
	Result  string
}

type AddWorkOrderCommentParams struct {
	// OrderID identifies the work order to target; see
	// UpdateWorkOrderStatusParams.OrderID.
	OrderID string
	Body    string
}

type AddWorkOrderArtifactParams struct {
	// OrderID identifies the work order to target; see
	// UpdateWorkOrderStatusParams.OrderID.
	OrderID string
	Type    string
	Data    map[string]any
	// Key optionally tags the artifact with a queryable key so a later
	// FindWorkOrder(by: artifactKey) can resolve the work order from it.
	Key string
}

// ReportWorkOrderCheckParams carries one check report. Format must be
// "fraction", "percent", or "boolean" (empty defaults to fraction; a
// boolean check pins Score to 0/1 and MaxScore to 1); Level must be
// "positive", "neutral", "caution", or "critical" (empty defaults to
// neutral) — the reporting component computes it from its thresholds.
type ReportWorkOrderCheckParams struct {
	// OrderID identifies the work order to target; see
	// UpdateWorkOrderStatusParams.OrderID.
	OrderID string
	// CheckKey identifies the check across reports (e.g. "risk-review").
	CheckKey string
	Name     string
	Score    float64
	MaxScore float64
	Format   string
	Level    string
	Summary  string
	Analysis string
}

// SetWorkOrderStatusNoteParams carries one status note. NoteKey identifies
// the note across sets (e.g. "pr-closure"). Kind is "info" (the default
// when empty); Headline is required; CtaLabel and CtaURL must be set
// together and the URL must be absolute http(s).
type SetWorkOrderStatusNoteParams struct {
	// OrderID identifies the work order to target; see
	// UpdateWorkOrderStatusParams.OrderID.
	OrderID string
	// NoteKey identifies the note across sets (e.g. "pr-closure").
	NoteKey             string
	Kind                string
	Headline            string
	Body                string
	CtaLabel            string
	CtaURL              string
	ShowOnlyWhenWaiting bool
}

type WorkOrder struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Result      string `json:"result,omitempty"`
	Number      int64  `json:"number,omitempty"`
	Key         string `json:"key,omitempty"`
}

type PullRequest struct {
	ID          string `json:"id"`
	WorkOrderID string `json:"workOrderId"`
	Provider    string `json:"provider"`
	ExternalID  string `json:"externalId,omitempty"`
	Repository  string `json:"repository"`
	Number      int64  `json:"number"`
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	State       string `json:"state"`
}

type PullRequestMatch struct {
	PullRequest *PullRequest
	WorkOrder   *WorkOrder
}

type AddPullRequestParams struct {
	OrderID    string
	Provider   string
	ExternalID string
	Repository string
	Number     int64
	URL        string
	Title      string
	State      string
	MergedAt   *time.Time
	ClosedAt   *time.Time
}

type UpdatePullRequestParams struct {
	PullRequestID string
	ExternalID    *string
	Repository    *string
	URL           *string
	Title         *string
	State         *string
	MergedAt      *time.Time
	ClosedAt      *time.Time
}

type FindPullRequestParams struct {
	ID         string
	Provider   string
	ExternalID string
	Repository string
	Number     int64
	URL        string
}

// ListPullRequestsParams configures FactoryContext.ListPullRequests. States
// selects which pull request states to include; an empty list defaults to
// open and draft.
type ListPullRequestsParams struct {
	Repository string
	States     []string
}

const (
	PullRequestActivityAccessConcurrent = "concurrent"
	PullRequestActivityAccessExclusive  = "exclusive"

	PullRequestActivityOutcomeReady        = "ready"
	PullRequestActivityOutcomeWaiting      = "waiting"
	PullRequestActivityOutcomeLimitReached = "limitReached"
)

type AddPullRequestActivityParams struct {
	PullRequestID string
	Description   string
	Revision      string
	Access        string
}

type UpdatePullRequestActivityParams struct {
	Description *string
	Access      string
}

type PullRequestRevision struct {
	SHA        string `json:"sha"`
	ObservedAt string `json:"observedAt,omitempty"`
}

type PullRequestActivity struct {
	Description  string               `json:"description,omitempty"`
	Access       string               `json:"access"`
	State        string               `json:"state"`
	Attempt      *int                 `json:"attempt,omitempty"`
	AttemptLimit *int                 `json:"attemptLimit,omitempty"`
	Revision     *PullRequestRevision `json:"revision,omitempty"`
}

type PullRequestActivityResult struct {
	PullRequest     *PullRequest
	WorkOrder       *WorkOrder
	Activity        *PullRequestActivity
	CurrentRevision *PullRequestRevision
	CurrentHeadSHA  string
	Outcome         string
}

type WorkOrderArtifact struct {
	ID          string         `json:"id"`
	WorkOrderID string         `json:"workOrderId"`
	Type        string         `json:"type"`
	Data        map[string]any `json:"data,omitempty"`
}

type WorkOrderStatusNote struct {
	WorkOrderID         string `json:"workOrderId"`
	Key                 string `json:"key"`
	Kind                string `json:"kind"`
	Headline            string `json:"headline"`
	Body                string `json:"body,omitempty"`
	CtaLabel            string `json:"ctaLabel,omitempty"`
	CtaURL              string `json:"ctaUrl,omitempty"`
	ShowOnlyWhenWaiting bool   `json:"showOnlyWhenWaiting,omitempty"`
}

type WorkOrderCheck struct {
	ID            string   `json:"id"`
	WorkOrderID   string   `json:"workOrderId"`
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Score         float64  `json:"score"`
	MaxScore      float64  `json:"maxScore"`
	Format        string   `json:"format"`
	Level         string   `json:"level"`
	PreviousScore *float64 `json:"previousScore,omitempty"`
	// RecentScores lists the latest report scores, oldest first and
	// ending with Score, capped server-side.
	RecentScores []float64 `json:"recentScores,omitempty"`
}
