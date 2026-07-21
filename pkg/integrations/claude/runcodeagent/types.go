package runcodeagent

import (
	"regexp"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/configuration/structuredoutput"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
)

const (
	payloadType    = "claude.runCodeAgent"
	defaultChannel = "default"

	sourceModeRepository = "repository"
	sourceModePR         = "pr"

	sessionStatusIdle       = "idle"
	sessionStatusTerminated = "terminated"

	initialPoll     = 15 * time.Second
	maxPollInterval = 3 * time.Minute
	// maxPollAttempts caps a run at roughly 4 hours of polling (exponential
	// backoff up to maxPollInterval) before it is declared a timeout.
	maxPollAttempts   = 80
	maxPollErrors     = 5
	finalMessageReads = 15

	// Where attached files are mounted inside the sandbox.
	attachmentsMountDir = "/workspace/attachments"

	networkingUnrestricted = "unrestricted"
	networkingLimited      = "limited"
)

// finalMessageDelay is the pause between event-stream reads while waiting for
// the terminal event to be written. A var so tests can shrink it.
var finalMessageDelay = 2 * time.Second

// defaultGitHubHosts are always allowed when networking is "limited" so the
// agent can clone, push, and open pull requests.
var defaultGitHubHosts = []string{
	"github.com",
	"api.github.com",
	"codeload.github.com",
	"*.githubusercontent.com",
	"objects.githubusercontent.com",
}

// Spec is the workflow node configuration for claude.runCodeAgent.
type Spec struct {
	SourceMode   string    `json:"sourceMode" mapstructure:"sourceMode"`
	Repository   string    `json:"repository" mapstructure:"repository"`
	BaseBranch   string    `json:"baseBranch" mapstructure:"baseBranch"`
	BranchName   string    `json:"branchName" mapstructure:"branchName"`
	AutoCreatePr *bool     `json:"autoCreatePr" mapstructure:"autoCreatePr"`
	PrURL        string    `json:"prUrl" mapstructure:"prUrl"`
	Task         string    `json:"task" mapstructure:"task"`
	GithubToken  SecretRef `json:"githubToken" mapstructure:"githubToken"`
	ActAsBot     *bool     `json:"actAsBot" mapstructure:"actAsBot"`
	Model        string    `json:"model" mapstructure:"model"`
	Networking   string    `json:"networking" mapstructure:"networking"`
	AllowedHosts []string  `json:"allowedHosts" mapstructure:"allowedHosts"`
	Files        []string  `json:"files" mapstructure:"files"`
	// PersistSession keeps the Managed Agents session (and the environment it
	// runs in) after the run finishes so the transcript stays readable in the
	// Anthropic Console.
	PersistSession bool `json:"persistSession" mapstructure:"persistSession"`
	// OutputSchema is a JSON Schema the agent is asked (via a prompt suffix, not
	// a server-enforced constraint) to match in its final message.
	OutputSchema string `json:"outputSchema" mapstructure:"outputSchema"`
}

// SecretRef references a SuperPlane secret by name and key.
type SecretRef struct {
	Secret string `json:"secret" mapstructure:"secret"`
	Key    string `json:"key" mapstructure:"key"`
}

func (r SecretRef) isSet() bool {
	return strings.TrimSpace(r.Secret) != "" && strings.TrimSpace(r.Key) != ""
}

// NodeMetadata is resolved at Setup for display on the component card.
type NodeMetadata struct {
	Repository       string `json:"repository,omitempty" mapstructure:"repository,omitempty"`
	BaseBranch       string `json:"baseBranch,omitempty" mapstructure:"baseBranch,omitempty"`
	PrURL            string `json:"prUrl,omitempty" mapstructure:"prUrl,omitempty"`
	Model            string `json:"model,omitempty" mapstructure:"model,omitempty"`
	SourceMode       string `json:"sourceMode,omitempty" mapstructure:"sourceMode,omitempty"`
	StructuredOutput bool   `json:"structuredOutput" mapstructure:"structuredOutput"`
}

// ExecutionMetadata tracks every resource provisioned for a run so it can be
// displayed and, crucially, always cleaned up.
type ExecutionMetadata struct {
	Session       *SessionMetadata `json:"session,omitempty" mapstructure:"session,omitempty"`
	AgentID       string           `json:"agentId,omitempty" mapstructure:"agentId,omitempty"`
	EnvironmentID string           `json:"environmentId,omitempty" mapstructure:"environmentId,omitempty"`
	VaultID       string           `json:"vaultId,omitempty" mapstructure:"vaultId,omitempty"`
	FileIDs       []string         `json:"fileIds,omitempty" mapstructure:"fileIds,omitempty"`
	Repository    string           `json:"repository,omitempty" mapstructure:"repository,omitempty"`
	Branch        string           `json:"branch,omitempty" mapstructure:"branch,omitempty"`
	PrURL         string           `json:"prUrl,omitempty" mapstructure:"prUrl,omitempty"`
}

// SessionMetadata tracks the Managed Agents session.
type SessionMetadata struct {
	ID     string `json:"id" mapstructure:"id"`
	Status string `json:"status" mapstructure:"status"`
}

// OutputPayload is emitted on the default channel when the run completes.
type OutputPayload struct {
	Status      string                     `json:"status"`
	SessionID   string                     `json:"sessionId"`
	PrURL       string                     `json:"prUrl"`
	Branch      string                     `json:"branch"`
	LastMessage string                     `json:"lastMessage"`
	Artifacts   []runagent.SessionArtifact `json:"artifacts,omitempty"`
	// Parsed is the JSON object extracted from LastMessage when Structured
	// Output is configured and the session completed normally.
	Parsed any `json:"parsed,omitempty"`
}

func isSessionTerminal(status string) bool {
	return status == sessionStatusIdle || status == sessionStatusTerminated
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

// prURLPattern extracts the pull request URL the agent reports on its final line.
var prURLPattern = regexp.MustCompile(`PR_URL=(\S+)`)

// extractPRURL scans the agent's messages for the PR_URL=<url> marker.
func extractPRURL(messages []string, lastMessage string) string {
	candidates := append([]string{}, messages...)
	if lastMessage != "" {
		candidates = append(candidates, lastMessage)
	}
	// Scan newest-first so the most recent PR URL wins.
	for i := len(candidates) - 1; i >= 0; i-- {
		if m := prURLPattern.FindStringSubmatch(candidates[i]); m != nil {
			url := strings.TrimSpace(m[1])
			if url != "" && url != "NO_PR" {
				return url
			}
		}
	}
	return ""
}

func buildOutput(status, sessionID, branch string, sm *runagent.SessionMessages, fallbackPrURL string) OutputPayload {
	out := OutputPayload{Status: status, SessionID: sessionID, Branch: branch, PrURL: fallbackPrURL}
	if sm != nil {
		out.LastMessage = sm.LastMessage
		if pr := extractPRURL(sm.Messages, sm.LastMessage); pr != "" {
			out.PrURL = pr
		}
	}
	return out
}

// applyStructuredOutput sets out.Parsed by best-effort extracting JSON from
// the agent's final message.
func applyStructuredOutput(out *OutputPayload, status string, schema map[string]any) {
	if schema == nil || status != sessionStatusIdle || out.LastMessage == "" {
		return
	}
	if parsed, ok := structuredoutput.ExtractJSON(out.LastMessage); ok {
		out.Parsed = parsed
	}
}

// schemaFromConfiguration re-derives the parsed output schema from the node's
// raw configuration for the async poll path.
func schemaFromConfiguration(config any) map[string]any {
	spec, err := decodeSpec(config)
	if err != nil {
		return nil
	}
	schema, _ := structuredoutput.Parse(spec.OutputSchema)
	return schema
}
