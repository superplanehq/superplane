package runagent

import "time"

const (
	payloadType             = "claude.runAgent.finished"
	defaultChannel          = "default"
	sessionStatusIdle       = "idle"
	sessionStatusTerminated = "terminated"
	initialPoll             = 15 * time.Second
	maxPollInterval         = 5 * time.Minute
	maxPollAttempts         = 200
	maxPollErrors           = 5
	finalMessageReads       = 5
	finalMessageDelay       = time.Second
)

// Spec is the workflow node configuration for claude.runAgent.
type Spec struct {
	// Agent is the managed agent id (use latest if Version is nil, else pin to Version).
	Agent         string   `json:"agent" mapstructure:"agent"`
	Version       *int     `json:"version" mapstructure:"version"`
	EnvironmentID string   `json:"environmentId" mapstructure:"environmentId"`
	Prompt        string   `json:"prompt" mapstructure:"prompt"`
	VaultIDs      []string `json:"vaultIds" mapstructure:"vaultIds"`
}

// ExecutionMetadata is persisted for the run.
type ExecutionMetadata struct {
	Session *SessionMetadata `json:"session,omitempty" mapstructure:"session,omitempty"`
}

// SessionMetadata tracks the Managed Agents session.
type SessionMetadata struct {
	ID     string `json:"id" mapstructure:"id"`
	Status string `json:"status" mapstructure:"status"`
}

// OutputPayload is emitted on the default channel when the run completes.
type OutputPayload struct {
	Status      string `json:"status"`
	SessionID   string `json:"sessionId"`
	LastMessage string `json:"lastMessage"`
}

func isSessionTerminal(status string) bool {
	return status == sessionStatusIdle || status == sessionStatusTerminated
}

func buildOutput(status, sessionID string, lastMessage ...string) OutputPayload {
	out := OutputPayload{
		Status:    status,
		SessionID: sessionID,
	}
	if len(lastMessage) > 0 {
		out.LastMessage = lastMessage[0]
	}
	return out
}

func mergeSessionIntoMetadata(metadata *ExecutionMetadata, s *ManagedSession) {
	if metadata.Session == nil {
		metadata.Session = &SessionMetadata{}
	}
	if s == nil {
		return
	}
	if s.ID != "" {
		metadata.Session.ID = s.ID
	}
	if s.Status != "" {
		metadata.Session.Status = s.Status
	}
}
