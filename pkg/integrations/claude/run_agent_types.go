package claude

import "time"

const (
	runAgentPayloadType       = "claude.runAgent.finished"
	runAgentDefaultChannel    = "default"
	sessionStatusIdle         = "idle"
	sessionStatusRunning      = "running"
	sessionStatusRescheduling = "rescheduling"
	sessionStatusTerminated   = "terminated"
	runAgentInitialPoll       = 15 * time.Second
	runAgentMaxPollInterval   = 5 * time.Minute
	runAgentMaxPollAttempts   = 200
	runAgentMaxPollErrors     = 5
)

// RunAgentSpec is the workflow node configuration for claude.runAgent.
type RunAgentSpec struct {
	// Agent is the managed agent id (use latest if Version is nil, else pin to Version).
	Agent         string   `json:"agent" mapstructure:"agent"`
	Version       *int     `json:"version" mapstructure:"version"`
	EnvironmentID string   `json:"environmentId" mapstructure:"environmentId"`
	Prompt        string   `json:"prompt" mapstructure:"prompt"`
	VaultIDs      []string `json:"vaultIds" mapstructure:"vaultIds"`
}

// RunAgentExecutionMetadata is persisted for the run.
type RunAgentExecutionMetadata struct {
	Session *RunAgentSessionMetadata `json:"session,omitempty" mapstructure:"session,omitempty"`
}

// RunAgentSessionMetadata tracks the Managed Agents session.
type RunAgentSessionMetadata struct {
	ID     string `json:"id" mapstructure:"id"`
	Status string `json:"status" mapstructure:"status"`
}

// RunAgentOutputPayload is emitted on the default channel when the run completes.
type RunAgentOutputPayload struct {
	Status    string `json:"status"`
	SessionID string `json:"sessionId"`
}

func isSessionTerminal(status string) bool {
	s := status
	return s == sessionStatusIdle || s == sessionStatusTerminated
}

func buildRunAgentOutput(status, sessionID string) RunAgentOutputPayload {
	return RunAgentOutputPayload{
		Status:    status,
		SessionID: sessionID,
	}
}

func mergeSessionIntoMetadata(metadata *RunAgentExecutionMetadata, s *ManagedSession) {
	if metadata.Session == nil {
		metadata.Session = &RunAgentSessionMetadata{}
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
