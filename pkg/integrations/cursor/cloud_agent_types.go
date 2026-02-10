package cursor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"
)

const (
	CloudAgentPayloadType            = "cursor.cloudAgent.finished"
	CloudAgentPassedChannel          = "passed"
	CloudAgentFailedChannel          = "failed"
	CloudAgentStatusCreating         = "CREATING"
	CloudAgentStatusRunning          = "RUNNING"
	CloudAgentStatusFinished         = "FINISHED"
	CloudAgentStatusDone             = "done"
	CloudAgentStatusSucceeded        = "succeeded"
	CloudAgentStatusFailed           = "failed"
	CloudAgentStatusError            = "error"
	CloudAgentDefaultBranch          = "main"
	CloudAgentBranchPrefix           = "cursor/agent-"
	CloudAgentSkipReviewerRequest    = false
	CloudAgentInitialPollInterval    = 30 * time.Second
	CloudAgentMaxPollInterval        = 10 * time.Minute
	CloudAgentMaxPollAttempts        = 100
	CloudAgentMaxPollErrors          = 5
	CloudAgentWebhookSignatureHeader = "X-Cursor-Signature"
)

type cloudAgentRequest struct {
	Prompt  cloudAgentPrompt  `json:"prompt"`
	Model   string            `json:"model,omitempty"`
	Source  cloudAgentSource  `json:"source"`
	Target  cloudAgentTarget  `json:"target,omitempty"`
	Webhook cloudAgentWebhook `json:"webhook,omitempty"`
}

type cloudAgentPrompt struct {
	Text string `json:"text"`
}

type cloudAgentSource struct {
	Repository string `json:"repository,omitempty"`
	Ref        string `json:"ref,omitempty"`
	PrURL      string `json:"prUrl,omitempty"`
}

type cloudAgentTarget struct {
	AutoCreatePr          *bool  `json:"autoCreatePr,omitempty"`
	OpenAsCursorGithubApp *bool  `json:"openAsCursorGithubApp,omitempty"`
	BranchName            string `json:"branchName,omitempty"`
	AutoBranch            *bool  `json:"autoBranch,omitempty"`
	SkipReviewerRequest   *bool  `json:"skipReviewerRequest,omitempty"`
}

type cloudAgentWebhook struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

// CloudAgentResponse is the Cursor API response for launching/getting an agent.
type CloudAgentResponse struct {
	ID        string                    `json:"id"`
	Name      string                    `json:"name,omitempty"`
	Status    string                    `json:"status"`
	Source    *cloudAgentSourceResponse `json:"source,omitempty"`
	Target    *cloudAgentTargetResponse `json:"target,omitempty"`
	Summary   string                    `json:"summary,omitempty"`
	CreatedAt string                    `json:"createdAt,omitempty"`
}

type cloudAgentSourceResponse struct {
	Repository string `json:"repository,omitempty"`
	Ref        string `json:"ref,omitempty"`
}

type cloudAgentTargetResponse struct {
	BranchName            string `json:"branchName,omitempty"`
	URL                   string `json:"url,omitempty"`
	PrURL                 string `json:"prUrl,omitempty"`
	AutoCreatePr          bool   `json:"autoCreatePr,omitempty"`
	OpenAsCursorGithubApp bool   `json:"openAsCursorGithubApp,omitempty"`
	SkipReviewerRequest   bool   `json:"skipReviewerRequest,omitempty"`
}

type cloudAgentWebhookPayload struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	PrURL   string `json:"prUrl,omitempty"`
	Summary string `json:"summary,omitempty"`
}

func isSuccessStatus(status string) bool {
	return status == CloudAgentStatusFinished ||
		status == CloudAgentStatusDone ||
		status == CloudAgentStatusSucceeded
}

func isFailureStatus(status string) bool {
	return status == CloudAgentStatusFailed ||
		status == CloudAgentStatusError
}

func isTerminalStatus(status string) bool {
	return isSuccessStatus(status) || isFailureStatus(status)
}

func validateURL(urlStr string, fieldName string) error {
	if urlStr == "" {
		return nil
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("%s is not a valid URL: %w", fieldName, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must be an absolute http or https URL", fieldName)
	}
	if u.Host == "" {
		return fmt.Errorf("%s must include a host", fieldName)
	}
	return nil
}

func buildOutputPayload(status, agentID, prURL, summary, branchName string) CloudAgentOutputPayload {
	return CloudAgentOutputPayload{
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

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

type CloudAgentSpec struct {
	Prompt       string `json:"prompt" mapstructure:"prompt"`
	Model        string `json:"model" mapstructure:"model"`
	SourceMode   string `json:"sourceMode" mapstructure:"sourceMode"`
	Repository   string `json:"repository" mapstructure:"repository"`
	Branch       string `json:"branch" mapstructure:"branch"`
	PrURL        string `json:"prUrl" mapstructure:"prUrl"`
	AutoCreatePr bool   `json:"autoCreatePr" mapstructure:"autoCreatePr"`
	UseCursorBot bool   `json:"useCursorBot" mapstructure:"useCursorBot"`
}

type CloudAgentExecutionMetadata struct {
	Agent         *AgentMetadata  `json:"agent,omitempty" mapstructure:"agent,omitempty"`
	Target        *TargetMetadata `json:"target,omitempty" mapstructure:"target,omitempty"`
	Source        *SourceMetadata `json:"source,omitempty" mapstructure:"source,omitempty"`
	WebhookSecret string `json:"webhookSecret,omitempty" mapstructure:"webhookSecret,omitempty"`
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

type CloudAgentOutputPayload struct {
	Status     string `json:"status"`
	AgentID    string `json:"agentId"`
	PrURL      string `json:"prUrl,omitempty"`
	Summary    string `json:"summary,omitempty"`
	BranchName string `json:"branchName,omitempty"`
}
