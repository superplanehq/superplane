package runnerapi

import "encoding/json"

const (
	messageTypeHello           = "hello"
	messageTypeComplete        = "complete"
	messageTypePong            = "pong"
	messageTypeShutdownRequest = "shutdown_request"
	messageTypeTask            = "task"
	messageTypeCancel          = "cancel"
	messageTypeShutdown        = "shutdown"
	messageTypeReconnect       = "reconnect"
	messageTypePing            = "ping"
	messageTypeAck             = "ack"
	messageTypeError           = "error"
)

const shutdownReasonSignal = "signal"

type messageEnvelope struct {
	Type string `json:"type"`
}

type helloMessage struct {
	Type          string `json:"type"`
	RunnerID      string `json:"runner_id"`
	FleetID       string `json:"fleet_id"`
	Version       string `json:"version"`
	CurrentTaskID string `json:"current_task_id,omitempty"`
}

type completeMessage struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id"`
	TaskID    string          `json:"task_id"`
	ExitCode  int32           `json:"exit_code"`
	Error     string          `json:"error,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Canceled  bool            `json:"canceled,omitempty"`
}

type taskMessage struct {
	Type string          `json:"type"`
	Task json.RawMessage `json:"task"`
}

type taskControlMessage struct {
	Type      string `json:"type"`
	TaskID    string `json:"task_id"`
	RequestID string `json:"request_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type shutdownRequestMessage struct {
	Type          string `json:"type"`
	RequestID     string `json:"request_id"`
	Reason        string `json:"reason"`
	CurrentTaskID string `json:"current_task_id,omitempty"`
}

type typedMessage struct {
	Type string `json:"type"`
}

type ackMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
}

type protocolErrorMessage struct {
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}
