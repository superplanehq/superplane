package runcloudagent

import (
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
)

const (
	payloadType             = "claude.runCloudAgent"
	defaultChannel          = "default"
	sessionStatusIdle       = "idle"
	sessionStatusTerminated = "terminated"
	initialPoll             = 15 * time.Second
	maxPollInterval         = 5 * time.Minute
	maxPollAttempts         = 200
	maxPollErrors           = 5
	finalMessageReads       = 15
	finalMessageDelay       = 2 * time.Second
)

// Spec is the workflow node configuration for claude.runCloudAgent.
type Spec struct {
	// Agent is the managed agent id (use latest if Version is nil, else pin to Version).
	Agent         string          `json:"agent" mapstructure:"agent"`
	Version       *int            `json:"version" mapstructure:"version"`
	EnvironmentID string          `json:"environmentId" mapstructure:"environmentId"`
	Repository    string          `json:"repository" mapstructure:"repository"`
	Branch        string          `json:"branch" mapstructure:"branch"`
	Prompt        string          `json:"prompt" mapstructure:"prompt"`
	VaultIDs      []string        `json:"vaultIds" mapstructure:"vaultIds"`
	Files         []string        `json:"files" mapstructure:"files"`
	Secrets       []SecretBinding `json:"secrets" mapstructure:"secrets"`
}

// SecretBinding maps a SuperPlane secret to an environment variable in the agent session.
type SecretBinding struct {
	EnvName      string    `json:"envName" mapstructure:"envName"`
	Value        SecretRef `json:"value" mapstructure:"value"`
	AllowedHosts []string  `json:"allowedHosts" mapstructure:"allowedHosts"`
}

// SecretRef references a SuperPlane secret by name and key.
type SecretRef struct {
	Secret string `json:"secret" mapstructure:"secret"`
	Key    string `json:"key" mapstructure:"key"`
}

// NodeMetadata stores the resolved agent and environment names for display on
// the component card. It is resolved at Setup time (not per execution) so the
// card shows names as soon as the node is configured.
type NodeMetadata struct {
	AgentID         string `json:"agentId" mapstructure:"agentId"`
	AgentName       string `json:"agentName" mapstructure:"agentName"`
	EnvironmentID   string `json:"environmentId" mapstructure:"environmentId"`
	EnvironmentName string `json:"environmentName" mapstructure:"environmentName"`
}

// ExecutionMetadata is persisted for the run.
type ExecutionMetadata struct {
	Session    *SessionMetadata `json:"session,omitempty" mapstructure:"session,omitempty"`
	Repository string           `json:"repository,omitempty" mapstructure:"repository,omitempty"`
	Branch     string           `json:"branch,omitempty" mapstructure:"branch,omitempty"`
}

// SessionMetadata tracks the Managed Agents session.
type SessionMetadata struct {
	ID     string `json:"id" mapstructure:"id"`
	Status string `json:"status" mapstructure:"status"`
}

// OutputPayload is emitted on the default channel when the run completes.
type OutputPayload struct {
	Status      string   `json:"status"`
	SessionID   string   `json:"sessionId"`
	LastMessage string   `json:"lastMessage"`
	Messages    []string `json:"messages"`
}

func isSessionTerminal(status string) bool {
	return status == sessionStatusIdle || status == sessionStatusTerminated
}

func buildOutputFromSessionMessages(status, sessionID string, sm *runagent.SessionMessages) OutputPayload {
	out := OutputPayload{
		Status:    status,
		SessionID: sessionID,
	}
	if sm != nil {
		out.LastMessage = sm.LastMessage
		out.Messages = sm.Messages
	}
	return out
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

func mergeSessionIntoMetadata(metadata *ExecutionMetadata, s *runagent.ManagedSession) {
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

// buildRepositoryPrompt prepends a clone instruction to the task so the managed
// agent checks out the configured repository into its workspace before working.
// The Managed Agents API has no first-class repository field; the agent clones
// the repository itself using its built-in tools.
func buildRepositoryPrompt(repository, branch, task string) string {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return task
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Clone the git repository %s into your working directory", repository))
	if strings.TrimSpace(branch) != "" {
		b.WriteString(fmt.Sprintf(" and check out the %q branch", strings.TrimSpace(branch)))
	}
	b.WriteString(". Then complete the following task:\n\n")
	b.WriteString(task)
	return b.String()
}
