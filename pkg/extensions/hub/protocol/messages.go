package protocol

import (
	"encoding/json"
)

const (
	QueryToken = "token"
)

const (
	MessageTypeJobAssign   = "job.assign"
	MessageTypeJobComplete = "job.complete"
	MessageTypeJobCancel   = "job.cancel"
	MessageTypePing        = "ping"
	MessageTypePong        = "pong"

	JobTypeInvokeExtension = "invoke-extension"
)

type Envelope struct {
	Type string `json:"type"`
}

type JobAssignMessage struct {
	Type            string           `json:"type"`
	JobID           string           `json:"jobId"`
	JobType         string           `json:"jobType"`
	InvokeExtension *InvokeExtension `json:"invokeExtension"`
}

type InvokeExtension struct {
	OrganizationID string          `json:"organizationId"`
	Extension      *ExtensionRef   `json:"extension"`
	Version        *VersionRef     `json:"version"`
	BundleToken    string          `json:"bundleToken"`
	Invocation     json.RawMessage `json:"invocation,omitempty"`
}

type ExtensionRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type VersionRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

type JobCompleteMessage struct {
	Type            string                 `json:"type"`
	JobID           string                 `json:"jobId"`
	JobType         string                 `json:"jobType"`
	InvokeExtension *InvokeExtensionOutput `json:"invokeExtension,omitempty"`
}

type InvokeExtensionOutput struct {
	Success bool            `json:"success"`
	Error   *JobError       `json:"error,omitempty"`
	Output  json.RawMessage `json:"output,omitempty"`
}

type JobError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewSuccessfulInvokeExtensionOutput(jobID string, jobType string, output json.RawMessage) JobCompleteMessage {
	return JobCompleteMessage{
		Type:    MessageTypeJobComplete,
		JobID:   jobID,
		JobType: jobType,
		InvokeExtension: &InvokeExtensionOutput{
			Success: true,
			Output:  output,
		},
	}
}

func NewFailedInvokeExtensionOutput(jobID string, jobType string, error *JobError) JobCompleteMessage {
	return JobCompleteMessage{
		Type:    MessageTypeJobComplete,
		JobID:   jobID,
		JobType: jobType,
		InvokeExtension: &InvokeExtensionOutput{
			Success: false,
			Error:   error,
		},
	}
}

type JobCancelMessage struct {
	Type  string `json:"type"`
	JobID string `json:"jobId"`
}

type PingMessage struct {
	Type string `json:"type"`
}

func NewPing() PingMessage {
	return PingMessage{Type: MessageTypePing}
}

type PongMessage struct {
	Type string `json:"type"`
}

func NewPong() PongMessage {
	return PongMessage{Type: MessageTypePong}
}
