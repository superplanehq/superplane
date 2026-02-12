package cursor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// --- CONSTANTS ---

const (
	LaunchAgentPayloadType            = "cursor.launchAgent.finished"
	LaunchAgentDefaultChannel         = "default"
	LaunchAgentStatusCreating         = "CREATING"
	LaunchAgentStatusRunning          = "RUNNING"
	LaunchAgentStatusFinished         = "FINISHED"
	LaunchAgentStatusDone             = "done"
	LaunchAgentStatusSucceeded        = "succeeded"
	LaunchAgentStatusFailed           = "failed"
	LaunchAgentStatusError            = "error"
	LaunchAgentDefaultBranch          = "main"
	LaunchAgentBranchPrefix           = "cursor/agent-"
	LaunchAgentSkipReviewerRequest    = false
	LaunchAgentInitialPollInterval    = 30 * time.Second
	LaunchAgentMaxPollInterval        = 10 * time.Minute
	LaunchAgentMaxPollAttempts        = 100
	LaunchAgentMaxPollErrors          = 5
	LaunchAgentWebhookSignatureHeader = "X-Webhook-Signature"
)

// --- CONFIGURATION STRUCTS ---

type LaunchAgentSpec struct {
	Prompt       string `json:"prompt" mapstructure:"prompt"`
	Model        string `json:"model" mapstructure:"model"`
	SourceMode   string `json:"sourceMode" mapstructure:"sourceMode"`
	Repository   string `json:"repository" mapstructure:"repository"`
	Branch       string `json:"branch" mapstructure:"branch"`
	PrURL        string `json:"prUrl" mapstructure:"prUrl"`
	AutoCreatePr bool   `json:"autoCreatePr" mapstructure:"autoCreatePr"`
	UseCursorBot bool   `json:"useCursorBot" mapstructure:"useCursorBot"`
}

// --- STATE STRUCTS (DB PERSISTENCE) ---

type LaunchAgentExecutionMetadata struct {
	Agent  *AgentMetadata  `json:"agent,omitempty" mapstructure:"agent,omitempty"`
	Target *TargetMetadata `json:"target,omitempty" mapstructure:"target,omitempty"`
	Source *SourceMetadata `json:"source,omitempty" mapstructure:"source,omitempty"`
}

type AgentMetadata struct {
	ID      string `json:"id" mapstructure:"id"`
	Name    string `json:"name,omitempty" mapstructure:"name,omitempty"`
	Status  string `json:"status" mapstructure:"status"`
	URL     string `json:"url,omitempty" mapstructure:"url,omitempty"`
	Summary string `json:"summary,omitempty" mapstructure:"summary,omitempty"`
}

type TargetMetadata struct {
	BranchName string `json:"branchName,omitempty" mapstructure:"branchName,omitempty"`
	PrURL      string `json:"prUrl,omitempty" mapstructure:"prUrl,omitempty"`
}

type SourceMetadata struct {
	Repository string `json:"repository,omitempty" mapstructure:"repository,omitempty"`
	Ref        string `json:"ref,omitempty" mapstructure:"ref,omitempty"`
}

// --- API DTOs (EXTERNAL CONTRACT) ---

type launchAgentRequest struct {
	Prompt  launchAgentPrompt  `json:"prompt"`
	Model   string             `json:"model,omitempty"`
	Source  launchAgentSource  `json:"source"`
	Target  launchAgentTarget  `json:"target,omitempty"`
	Webhook launchAgentWebhook `json:"webhook,omitempty"`
}

type launchAgentPrompt struct {
	Text string `json:"text"`
}

type launchAgentSource struct {
	Repository string `json:"repository,omitempty"`
	Ref        string `json:"ref,omitempty"`
	PrURL      string `json:"prUrl,omitempty"`
}

type launchAgentTarget struct {
	AutoCreatePr          *bool  `json:"autoCreatePr,omitempty"`
	OpenAsCursorGithubApp *bool  `json:"openAsCursorGithubApp,omitempty"`
	BranchName            string `json:"branchName,omitempty"`
	AutoBranch            *bool  `json:"autoBranch,omitempty"`
	SkipReviewerRequest   *bool  `json:"skipReviewerRequest,omitempty"`
}

type launchAgentWebhook struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

type LaunchAgentResponse struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name,omitempty"`
	Status    string                     `json:"status"`
	Source    *launchAgentSourceResponse `json:"source,omitempty"`
	Target    *launchAgentTargetResponse `json:"target,omitempty"`
	Summary   string                     `json:"summary,omitempty"`
	CreatedAt string                     `json:"createdAt,omitempty"`
}

type launchAgentSourceResponse struct {
	Repository string `json:"repository,omitempty"`
	Ref        string `json:"ref,omitempty"`
}

type launchAgentTargetResponse struct {
	BranchName            string `json:"branchName,omitempty"`
	URL                   string `json:"url,omitempty"`
	PrURL                 string `json:"prUrl,omitempty"`
	AutoCreatePr          *bool  `json:"autoCreatePr,omitempty"`
	OpenAsCursorGithubApp *bool  `json:"openAsCursorGithubApp,omitempty"`
	SkipReviewerRequest   *bool  `json:"skipReviewerRequest,omitempty"`
}

type launchAgentWebhookPayload struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	PrURL   string `json:"prUrl,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type LaunchAgentOutputPayload struct {
	Status     string `json:"status"`
	AgentID    string `json:"agentId"`
	PrURL      string `json:"prUrl,omitempty"`
	Summary    string `json:"summary,omitempty"`
	BranchName string `json:"branchName,omitempty"`
}

// --- HELPER FUNCTIONS ---

func isSuccessStatus(status string) bool {
	return status == LaunchAgentStatusFinished ||
		status == LaunchAgentStatusDone ||
		status == LaunchAgentStatusSucceeded
}

func isFailureStatus(status string) bool {
	return status == LaunchAgentStatusFailed ||
		status == LaunchAgentStatusError
}

func isTerminalStatus(status string) bool {
	return isSuccessStatus(status) || isFailureStatus(status)
}

func buildOutputPayload(status, agentID, prURL, summary, branchName string) LaunchAgentOutputPayload {
	return LaunchAgentOutputPayload{
		Status:     status,
		AgentID:    agentID,
		PrURL:      prURL,
		Summary:    summary,
		BranchName: branchName,
	}
}

func verifyWebhookSignature(body []byte, signature, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}

	// Cursor sends signature in format "sha256=<hex_digest>"
	// Strip the "sha256=" prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return false
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant-time comparison
	return hmac.Equal([]byte(signature), []byte(expectedSig))
}
