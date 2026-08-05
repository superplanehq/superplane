package factory

import (
	"github.com/google/uuid"
)

const (
	// Work order events
	EventTypeOrderOpened           = "order.opened"
	EventTypeOrderClosed           = "order.closed"
	EventTypeOrderAssigneesUpdated = "order.assignees.updated"
	EventTypeOrderStatusUpdated    = "order.status.updated"
	EventTypeOrderCommentAdded     = "order.comment.added"
	EventTypeOrderArtifactAdded    = "order.artifact.added"

	// Factory line events
	EventTypeLineStepExecutionCreated  = "step.execution.created"
	EventTypeLineStepExecutionFinished = "step.execution.finished"
)

// Comment author kinds. `automation` covers any canvas-run comment;
// the specific tool is exposed via the `Automation` payload.
const (
	CommentAuthorKindUser       = "user"
	CommentAuthorKindAutomation = "automation"
	CommentAuthorKindSystem     = "system"
)

// Artifact types
const (
	ArtifactTypePR       = "pr"
	ArtifactTypeMarkdown = "markdown"
)

// Events

type WorkOrderOpened struct {
	Order      *WorkOrderRef  `json:"order,omitempty"`
	User       *UserRef       `json:"user,omitempty"`
	Automation *AutomationRef `json:"automation,omitempty"`
}

type WorkOrderClosed struct {
	Order      *WorkOrderRef  `json:"order,omitempty"`
	User       *UserRef       `json:"user,omitempty"`
	Automation *AutomationRef `json:"automation,omitempty"`
	Result     *string        `json:"result,omitempty"`
}

type WorkOrderAssigneesUpdated struct {
	Order      *WorkOrderRef `json:"order,omitempty"`
	User       *UserRef      `json:"user,omitempty"`
	Assigned   []UserRef     `json:"assigned,omitempty"`
	Unassigned []UserRef     `json:"unassigned,omitempty"`
}

type WorkOrderStatusUpdated struct {
	Order      *WorkOrderRef  `json:"order,omitempty"`
	User       *UserRef       `json:"user,omitempty"`
	Automation *AutomationRef `json:"automation,omitempty"`
	Run        *RunRef        `json:"run,omitempty"`
	FromState  string         `json:"fromState"`
	ToState    string         `json:"toState"`
	FromResult string         `json:"fromResult,omitempty"`
	ToResult   string         `json:"toResult,omitempty"`
}

// AutomationRef snapshots the canvas node + app + factory line/step
// behind an automated event. Captured at write time so later renames
// don't retro-edit history.
type AutomationRef struct {
	NodeID    string    `json:"nodeId,omitempty"`
	NodeName  string    `json:"nodeName,omitempty"`
	AppID     uuid.UUID `json:"appId,omitempty"`
	AppName   string    `json:"appName,omitempty"`
	LineID    uuid.UUID `json:"lineId,omitempty"`
	LineName  string    `json:"lineName,omitempty"`
	StepIndex int       `json:"stepIndex,omitempty"`
	StepName  string    `json:"stepName,omitempty"`
}

type WorkOrderCommentAuthor struct {
	Kind       string         `json:"kind"`
	UserID     *string        `json:"userId,omitempty"`
	Automation *AutomationRef `json:"automation,omitempty"`
}

type WorkOrderCommentAdded struct {
	Order  *WorkOrderRef           `json:"order,omitempty"`
	Body   string                  `json:"body"`
	Author *WorkOrderCommentAuthor `json:"author,omitempty"`
	Run    *RunRef                 `json:"run,omitempty"`
}

type WorkOrderArtifactAdded struct {
	Order      *WorkOrderRef  `json:"order,omitempty"`
	Artifact   *ArtifactRef   `json:"artifact,omitempty"`
	User       *UserRef       `json:"user,omitempty"`
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
	ID uuid.UUID `json:"id"`
}

type RunRef struct {
	ID     uuid.UUID `json:"id"`
	State  string    `json:"state"`
	Result *string   `json:"result,omitempty"`
}

type ArtifactRef struct {
	ID    uuid.UUID      `json:"id"`
	Type  string         `json:"type"`
	URL   string         `json:"url,omitempty"`
	Title string         `json:"title,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
}
