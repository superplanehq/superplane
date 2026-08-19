package core

import "errors"

// ErrWorkOrderNotFound is returned by FindWorkOrder when nothing matches
// the given lookup. Components that treat "not found" as benign (e.g. a
// PR merge with no tracked order) check for it with errors.Is.
var ErrWorkOrderNotFound = errors.New("work order not found")

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
	// UpdateWorkOrderArtifact merges Data into an artifact already
	// attached to the work order, resolved by the key it was given at
	// attach time (AddWorkOrderArtifactParams.Key). This is how a
	// PR artifact's `state` stays in sync with GitHub after the initial
	// attach — see the updateWorkOrderArtifact component.
	UpdateWorkOrderArtifact(params UpdateWorkOrderArtifactParams) (*WorkOrderArtifact, error)
	// ReportWorkOrderCheck upserts a scored check on the work order,
	// keyed by CheckKey: the first report creates the check, later
	// reports with the same key update it in place and keep the prior
	// score as PreviousScore. Every report also lands an
	// `order.check.reported` timeline event.
	ReportWorkOrderCheck(params ReportWorkOrderCheckParams) (*WorkOrderCheck, error)
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

// UpdateWorkOrderArtifactParams targets an existing artifact by the key
// it was attached with, rather than by id — the same key
// FindWorkOrder(by: artifactKey) uses, typically a pull request's URL.
// Data is shallow-merged into the artifact's existing data, not
// replaced wholesale, so e.g. sending only `{"state": "merged"}` leaves
// `title`/`number` untouched.
type UpdateWorkOrderArtifactParams struct {
	// OrderID identifies the work order to target; see
	// UpdateWorkOrderStatusParams.OrderID.
	OrderID string
	// Key is required: the artifactKey the artifact was attached with.
	Key  string
	Data map[string]any
}

// ReportWorkOrderCheckParams carries one check report. Format must be
// "fraction" or "percent" (empty defaults to fraction); Level must be
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

type WorkOrder struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Result      string `json:"result,omitempty"`
}

type WorkOrderArtifact struct {
	ID          string         `json:"id"`
	WorkOrderID string         `json:"workOrderId"`
	Type        string         `json:"type"`
	Data        map[string]any `json:"data,omitempty"`
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
}
