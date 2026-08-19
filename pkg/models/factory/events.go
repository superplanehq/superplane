package factory

import (
	"github.com/google/uuid"
)

const (
	// Work order events. `order.status.updated` is the sole authoritative
	// lifecycle event: every FSM transition emits one, enriched with the
	// actor / automation / originating run / app when applicable.
	EventTypeOrderAssigneesUpdated = "order.assignees.updated"
	EventTypeOrderStatusUpdated    = "order.status.updated"
	EventTypeOrderCommentAdded     = "order.comment.added"
	EventTypeOrderArtifactAdded    = "order.artifact.added"
	// EventTypeOrderArtifactUpdated is a websocket-only notification
	// reason (see FactoryContext.UpdateWorkOrderArtifact) — it does not
	// back a timeline event/struct. Flipping a PR artifact's state
	// shouldn't spam the timeline with one entry per open/draft/closed/
	// merged transition; the row is updated in place and this reason
	// just tells the frontend which query to invalidate.
	EventTypeOrderArtifactUpdated = "order.artifact.updated"
	// EventTypeOrderCheckReported records every check report, including
	// re-reports of the same check key. The check row itself is
	// latest-only state (one row per key, updated in place); the events
	// keep the score history on the timeline.
	EventTypeOrderCheckReported = "order.check.reported"

	// Factory line events
	EventTypeLineStepExecutionCreated  = "step.execution.created"
	EventTypeLineStepExecutionFinished = "step.execution.finished"
)

// Comment author kinds. `automation` covers any canvas-run comment;
// the specific tool is exposed via the `Automation` payload.
const (
	CommentAuthorKindUser       = "user"
	CommentAuthorKindAutomation = "automation"
)

// Artifact types
const (
	ArtifactTypePR       = "pr"
	ArtifactTypeMarkdown = "markdown"
	ArtifactTypeBranch   = "branch"
	ArtifactTypeLink     = "link"
)

// Check levels. The reporting component computes the level from its
// declarative thresholds (direction + cautionAt/criticalAt); the model
// only validates and stores it.
const (
	CheckLevelPositive = "positive"
	CheckLevelNeutral  = "neutral"
	CheckLevelCaution  = "caution"
	CheckLevelCritical = "critical"
)

// Check score formats: render as `score/maxScore` or as a percentage.
const (
	CheckFormatFraction = "fraction"
	CheckFormatPercent  = "percent"
	// CheckFormatBoolean is a pass/fail verdict: score 1 (pass) or 0
	// (fail) on a max score of 1.
	CheckFormatBoolean = "boolean"
)

// Events

type WorkOrderAssigneesUpdated struct {
	Order      *WorkOrderRef `json:"order,omitempty"`
	User       *UserRef      `json:"user,omitempty"`
	Assigned   []UserRef     `json:"assigned,omitempty"`
	Unassigned []UserRef     `json:"unassigned,omitempty"`
}

// WorkOrderStatusUpdated is the authoritative lifecycle event: emitted for
// every FSM transition, including the initial `"" → draft` on creation and
// the `→ closed` termination. `User`, `Automation`, and `Run` + `App` are
// attribution channels — populate whichever caused the transition.
type WorkOrderStatusUpdated struct {
	Order      *WorkOrderRef  `json:"order,omitempty"`
	User       *UserRef       `json:"user,omitempty"`
	Automation *AutomationRef `json:"automation,omitempty"`
	App        *AppRef        `json:"app,omitempty"`
	Run        *RunRef        `json:"run,omitempty"`
	FromState  string         `json:"fromState"`
	ToState    string         `json:"toState"`
	FromResult string         `json:"fromResult,omitempty"`
	ToResult   string         `json:"toResult,omitempty"`
}

// AutomationRef snapshots the canvas node + app + factory line/step
// behind an automated event. Captured at write time so later renames
// don't retro-edit history. `StepIndex` is a pointer so consumers can
// distinguish "no line context" from a legitimate index of 0.
type AutomationRef struct {
	NodeID    string    `json:"nodeId,omitempty"`
	NodeName  string    `json:"nodeName,omitempty"`
	AppID     uuid.UUID `json:"appId,omitempty"`
	AppName   string    `json:"appName,omitempty"`
	LineID    uuid.UUID `json:"lineId,omitempty"`
	LineName  string    `json:"lineName,omitempty"`
	StepIndex *int      `json:"stepIndex,omitempty"`
	StepName  string    `json:"stepName,omitempty"`
}

type WorkOrderCommentAuthor struct {
	Kind       string         `json:"kind"`
	UserID     *string        `json:"userId,omitempty"`
	Automation *AutomationRef `json:"automation,omitempty"`
}

type WorkOrderCommentAdded struct {
	Order          *WorkOrderRef           `json:"order,omitempty"`
	CommentID      uuid.UUID               `json:"commentId,omitempty"`
	Body           string                  `json:"body"`
	Author         *WorkOrderCommentAuthor `json:"author,omitempty"`
	Run            *RunRef                 `json:"run,omitempty"`
	MentionedUsers []UserRef               `json:"mentionedUsers,omitempty"`
}

type WorkOrderArtifactAdded struct {
	Order      *WorkOrderRef  `json:"order,omitempty"`
	Artifact   *ArtifactRef   `json:"artifact,omitempty"`
	User       *UserRef       `json:"user,omitempty"`
	Automation *AutomationRef `json:"automation,omitempty"`
	Run        *RunRef        `json:"run,omitempty"`
}

type WorkOrderCheckReported struct {
	Order      *WorkOrderRef  `json:"order,omitempty"`
	Check      *CheckRef      `json:"check,omitempty"`
	Automation *AutomationRef `json:"automation,omitempty"`
	Run        *RunRef        `json:"run,omitempty"`
}

type LineStepExecutionCreated struct {
	StepName string        `json:"stepName"`
	Order    *WorkOrderRef `json:"order,omitempty"`
	Line     *LineRef      `json:"line,omitempty"`
	App      *AppRef       `json:"app,omitempty"`
	Run      *RunRef       `json:"run,omitempty"`
}

type LineStepExecutionFinished struct {
	StepName string        `json:"stepName"`
	Order    *WorkOrderRef `json:"order,omitempty"`
	Line     *LineRef      `json:"line,omitempty"`
	App      *AppRef       `json:"app,omitempty"`
	Run      *RunRef       `json:"run,omitempty"`
}

// Refs

type WorkOrderRef struct {
	ID     uuid.UUID `json:"id"`
	Title  string    `json:"title"`
	Result *string   `json:"result,omitempty"`
}

type UserRef struct {
	ID uuid.UUID `json:"id"`
}

type LineRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type AppRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name,omitempty"`
}

type RunRef struct {
	ID     uuid.UUID `json:"id"`
	State  string    `json:"state"`
	Result *string   `json:"result,omitempty"`
}

type ArtifactRef struct {
	ID   uuid.UUID      `json:"id"`
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

// CheckRef snapshots a check report for the timeline. PreviousScore is
// the score the same check key held before this report, if any, so the
// timeline can show the movement without replaying older events.
type CheckRef struct {
	ID            uuid.UUID `json:"id"`
	Key           string    `json:"key"`
	Name          string    `json:"name"`
	Score         float64   `json:"score"`
	MaxScore      float64   `json:"maxScore"`
	Format        string    `json:"format"`
	Level         string    `json:"level"`
	PreviousScore *float64  `json:"previousScore,omitempty"`
}
