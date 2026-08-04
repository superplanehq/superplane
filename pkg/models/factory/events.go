package factory

import (
	"github.com/google/uuid"
)

const (
	//
	// Work order events
	//
	EventTypeOrderOpened           = "order.opened"
	EventTypeOrderClosed           = "order.closed"
	EventTypeOrderAssigneesUpdated = "order.assignees.updated"

	//
	// Factory line events
	//
	EventTypeLineStepExecutionCreated  = "step.execution.created"
	EventTypeLineStepExecutionFinished = "step.execution.finished"
)

//
// Events
//

type WorkOrderOpened struct {
	Order *WorkOrderRef `json:"order,omitempty"`
	User  *UserRef      `json:"user,omitempty"`
}

type WorkOrderClosed struct {
	Order  *WorkOrderRef `json:"order,omitempty"`
	User   *UserRef      `json:"user,omitempty"`
	Result *string       `json:"result,omitempty"`
}

type WorkOrderAssigneesUpdated struct {
	Order      *WorkOrderRef `json:"order,omitempty"`
	User       *UserRef      `json:"user,omitempty"`
	Assigned   []UserRef     `json:"assigned,omitempty"`
	Unassigned []UserRef     `json:"unassigned,omitempty"`
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

//
// Refs
//

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
