package runagent

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/configuration/structuredoutput"
)

const (
	payloadType    = "claude.runAgent.finished"
	defaultChannel = "default"
	// latestVersionValue is the sentinel the Version resource field uses for the
	// explicit "Latest" option; it is treated the same as an unset version.
	latestVersionValue      = "latest"
	sessionStatusIdle       = "idle"
	sessionStatusTerminated = "terminated"
	initialPoll             = 15 * time.Second
	maxPollInterval         = 5 * time.Minute
	maxPollAttempts         = 200
	maxPollErrors           = 5
	finalMessageReads       = 15
	// Session outputs can take a few seconds to be indexed by the Files API
	// after the session goes idle (~1-3s documented), so when the session is
	// expected to have written outputs, a listing without them is retried
	// with a budget beyond that lag.
	sessionFileListReads = 3
	sessionFileListDelay = 2 * time.Second
	// maxInlineArtifactSizeBytes caps how large a generated file can be for
	// its content to be embedded in the output payload. Larger files keep
	// their metadata and download link only.
	maxInlineArtifactSizeBytes = 10 * 1024 * 1024
)

// finalMessageDelay is the pause between event-stream reads while waiting for
// the terminal event to be written. A var so tests can shrink it.
var finalMessageDelay = 2 * time.Second

// Spec is the workflow node configuration for claude.runAgent.
type Spec struct {
	// Agent is the managed agent id.
	Agent string `json:"agent" mapstructure:"agent"`
	// Version pins the agent version. It holds the raw resource value (a version
	// number as a string); empty runs the agent's latest version.
	Version string `json:"version" mapstructure:"version"`
	// Environment is stored under the legacy "environmentId" key so existing
	// nodes keep working; it is presented as the Environment resource field.
	Environment string          `json:"environmentId" mapstructure:"environmentId"`
	Prompt      string          `json:"prompt" mapstructure:"prompt"`
	VaultIDs    []string        `json:"vaultIds" mapstructure:"vaultIds"`
	Files       []string        `json:"files" mapstructure:"files"`
	Secrets     []SecretBinding `json:"secrets" mapstructure:"secrets"`
	// PersistSession keeps the Managed Agents session after the run finishes so
	// its transcript stays readable in the Anthropic Console.
	PersistSession bool `json:"persistSession" mapstructure:"persistSession"`
	// OutputSchema is a JSON Schema the agent is asked (via a prompt suffix, not
	// a server-enforced constraint) to match in its final message.
	OutputSchema string `json:"outputSchema" mapstructure:"outputSchema"`
}

// RunAgentNodeMetadata is node-level metadata surfaced in the UI so configured
// options are visible on the node without opening it.
type RunAgentNodeMetadata struct {
	StructuredOutput bool `json:"structuredOutput" mapstructure:"structuredOutput"`
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
	// Parsed is the JSON object extracted from LastMessage when Structured
	// Output is configured and the session completed normally.
	Parsed any `json:"parsed,omitempty"`
}

// SessionArtifact is a file the agent generated during the session (written to
// /mnt/session/outputs/). Its content is embedded in the payload — text files
// as a plain string, everything else base64-encoded — so downstream steps can
// consume it directly. Files over the inline size cap carry metadata and the
// download link only.
type SessionArtifact struct {
	FileID      string `json:"fileId"`
	Filename    string `json:"filename"`
	MimeType    string `json:"mimeType"`
	SizeBytes   int64  `json:"sizeBytes"`
	Encoding    string `json:"encoding,omitempty"`
	Content     string `json:"content,omitempty"`
	DownloadURL string `json:"downloadUrl"`
}

// CollectSessionArtifacts lists the files the agent generated during the
// session and resolves them into artifacts carrying the file content.
// Collection is best-effort: a listing or download failure is logged and
// degrades the artifact (or drops the list) rather than failing a run whose
// real output is already available. The listing is only retried (to cover
// the Files API indexing lag) when the session events indicate the agent
// wrote outputs, so artifact-less runs finish without extra delay.
func CollectSessionArtifacts(client *Client, sessionID string, expectsArtifacts bool, logWarn func(string, ...any)) []SessionArtifact {
	attempts := 1
	if expectsArtifacts {
		attempts = sessionFileListReads
	}
	files, err := client.ListSessionFilesWithRetry(sessionID, attempts, sessionFileListDelay)
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

		artifact := SessionArtifact{
			FileID:      f.ID,
			Filename:    f.Filename,
			MimeType:    f.MimeType,
			SizeBytes:   f.SizeBytes,
			DownloadURL: client.FileContentURL(f.ID),
		}
		if f.SizeBytes <= maxInlineArtifactSizeBytes {
			if content, err := client.DownloadFileContent(f.ID); err == nil {
				artifact.Encoding, artifact.Content = encodeArtifactContent(f.MimeType, content)
			} else if logWarn != nil {
				logWarn("Failed to download session file %s: %v", f.ID, err)
			}
		}
		artifacts = append(artifacts, artifact)
	}
	if len(artifacts) == 0 {
		return nil
	}
	return artifacts
}

// encodeArtifactContent returns the payload encoding and content for a
// downloaded file: text-like content passes through as a plain string,
// everything else is base64-encoded.
func encodeArtifactContent(mimeType string, content []byte) (string, string) {
	if isTextMIME(mimeType) {
		return "text", string(content)
	}
	return "base64", base64.StdEncoding.EncodeToString(content)
}

// isTextMIME reports whether content of the given MIME type is safe to emit as
// plain text; anything else is base64-encoded.
func isTextMIME(mimeType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if i := strings.Index(mimeType, ";"); i >= 0 {
		mimeType = strings.TrimSpace(mimeType[:i])
	}
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	if strings.HasSuffix(mimeType, "+json") || strings.HasSuffix(mimeType, "+xml") {
		return true
	}
	switch mimeType {
	case "application/json",
		"application/xml",
		"application/x-yaml",
		"application/yaml",
		"application/javascript",
		"application/x-sh",
		"application/sql",
		"application/csv":
		return true
	}
	return false
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

// applyStructuredOutput sets out.Parsed by best-effort extracting JSON from
// the agent's final message, when a schema is configured and the session
// completed normally ("idle"). There is no server-side schema enforcement for
// Managed Agents sessions (unlike output_config.format on the Messages API),
// so this is prompt-guided and best-effort — a "terminated" session may have
// errored or been interrupted mid-task, so its message is not trusted either.
func applyStructuredOutput(out *OutputPayload, status string, schema map[string]any) {
	if schema == nil || status != sessionStatusIdle || out.LastMessage == "" {
		return
	}
	if parsed, ok := structuredoutput.ExtractJSON(out.LastMessage); ok {
		out.Parsed = parsed
	}
}

// schemaFromConfiguration re-derives the parsed output schema from the node's
// raw configuration for the async poll path, where the schema was already
// validated at Setup/Execute — a decode or parse failure here is tolerated as
// "no schema" rather than failing an otherwise-complete run.
func schemaFromConfiguration(config any) map[string]any {
	spec, err := decodeSpec(config)
	if err != nil {
		return nil
	}
	schema, _ := structuredoutput.Parse(spec.OutputSchema)
	return schema
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
