package protocol

import (
	"encoding/json"
	"fmt"
)

const (
	QueryWorkerID = "workerId"
	QueryToken    = "token"
)

const (
	MessageTypeJobAssign   = "job.assign"
	MessageTypeJobComplete = "job.complete"
	MessageTypeJobCancel   = "job.cancel"
	MessageTypePing        = "ping"
	MessageTypePong        = "pong"
)

type Registration struct {
	WorkerID       string `json:"workerId"`
	OrganizationID string `json:"organizationId"`
	WorkerPoolID   string `json:"workerPoolId"`
}

func (r Registration) Validate() error {
	if r.WorkerID == "" {
		return fmt.Errorf("worker ID is required")
	}
	if r.OrganizationID == "" {
		return fmt.Errorf("organization ID is required")
	}
	if r.WorkerPoolID == "" {
		return fmt.Errorf("worker pool ID is required")
	}

	return nil
}

type Envelope struct {
	Type string `json:"type"`
}

type JobAssignMessage struct {
	Type        string          `json:"type"`
	JobID       string          `json:"jobId"`
	ExtensionID string          `json:"extensionId"`
	VersionID   string          `json:"versionId"`
	Digest      string          `json:"digest,omitempty"`
	BundleToken string          `json:"bundleToken"`
	Invocation  json.RawMessage `json:"invocation,omitempty"`
}

type JobCompleteMessage struct {
	Type    string          `json:"type"`
	JobID   string          `json:"jobId"`
	Success bool            `json:"success"`
	Output  json.RawMessage `json:"output,omitempty"`
	Error   *JobError       `json:"error,omitempty"`
}

type JobError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewJobSuccess(jobID string, output json.RawMessage) JobCompleteMessage {
	return JobCompleteMessage{
		Type:    MessageTypeJobComplete,
		JobID:   jobID,
		Success: true,
		Output:  output,
	}
}

func NewJobFailure(jobID string, code string, message string) JobCompleteMessage {
	return JobCompleteMessage{
		Type:    MessageTypeJobComplete,
		JobID:   jobID,
		Success: false,
		Error: &JobError{
			Code:    code,
			Message: message,
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
