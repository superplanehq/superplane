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
	finalMessageReads       = 15
	finalMessageDelay       = 2 * time.Second
	// Session outputs can take a few seconds to be indexed by the Files API
	// after the session goes idle, so an empty listing is retried once.
	sessionFileListReads = 2
	sessionFileListDelay = 2 * time.Second
)

// Spec is the workflow node configuration for claude.runAgent.
type Spec struct {
	// Agent is the managed agent id (use latest if Version is nil, else pin to Version).
	Agent         string          `json:"agent" mapstructure:"agent"`
	Version       *int            `json:"version" mapstructure:"version"`
	EnvironmentID string          `json:"environmentId" mapstructure:"environmentId"`
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
	Status      string            `json:"status"`
	SessionID   string            `json:"sessionId"`
	LastMessage string            `json:"lastMessage"`
	Messages    []string          `json:"messages"`
	Artifacts   []SessionArtifact `json:"artifacts,omitempty"`
}

// SessionArtifact is a file the agent generated during the session (written to
// /mnt/session/outputs/), stored in the Files API and downloadable with the
// API key.
type SessionArtifact struct {
	FileID      string `json:"fileId"`
	Filename    string `json:"filename"`
	MimeType    string `json:"mimeType"`
	SizeBytes   int64  `json:"sizeBytes"`
	DownloadURL string `json:"downloadUrl"`
}

// CollectSessionArtifacts lists the files the agent generated during the
// session and resolves them into artifacts with download links. Collection is
// best-effort: a listing failure is logged and yields no artifacts rather than
// failing a run whose real output is already available.
func CollectSessionArtifacts(client *Client, sessionID string, logWarn func(string, ...any)) []SessionArtifact {
	files, err := client.ListSessionFilesWithRetry(sessionID, sessionFileListReads, sessionFileListDelay)
	if err != nil {
		if logWarn != nil {
			logWarn("Failed to list session files for %s: %v", sessionID, err)
		}
		return nil
	}

	artifacts := make([]SessionArtifact, 0, len(files))
	for _, f := range files {
		// Input files mounted into the session are listed too, but only
		// agent-generated outputs are downloadable.
		if !f.Downloadable || f.ID == "" {
			continue
		}
		artifacts = append(artifacts, SessionArtifact{
			FileID:      f.ID,
			Filename:    f.Filename,
			MimeType:    f.MimeType,
			SizeBytes:   f.SizeBytes,
			DownloadURL: client.FileContentURL(f.ID),
		})
	}
	if len(artifacts) == 0 {
		return nil
	}
	return artifacts
}

func isSessionTerminal(status string) bool {
	return status == sessionStatusIdle || status == sessionStatusTerminated
}

func buildOutputFromSessionMessages(status, sessionID string, sm *SessionMessages) OutputPayload {
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
