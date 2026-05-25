package guardrails

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	anthropicAPIBase    = "https://api.anthropic.com/v1"
	anthropicAPIVersion = "2023-06-01"
	// Haiku is fast and cheap — ideal for high-volume classification.
	defaultClassifierModel = "claude-haiku-4-5-20251001"

	classifierSystemPrompt = `You are a security classifier for AI prompt content.
You will receive a list of rule-engine findings from a prompt scan and must assess
whether each finding is a true positive or a false positive, then output an overall
risk score.

Always respond with ONLY valid JSON matching this schema:
{
  "risk_score": <integer 0-100>,
  "confirmed_findings": [<rule_id strings for confirmed true positives>],
  "analysis": "<one sentence summary>"
}`
)

// AnthropicClassifierConfig configures the Anthropic-backed classifier.
type AnthropicClassifierConfig struct {
	APIKey    string
	Model     string
	BaseURL   string
	TimeoutMs int
}

// AnthropicClassifier calls Claude to validate and score rule-engine findings.
type AnthropicClassifier struct {
	cfg        AnthropicClassifierConfig
	httpClient *http.Client
}

// NewAnthropicClassifier returns a classifier backed by the Anthropic Messages API.
func NewAnthropicClassifier(cfg AnthropicClassifierConfig) (*AnthropicClassifier, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("anthropic classifier: APIKey is required")
	}
	if cfg.Model == "" {
		cfg.Model = defaultClassifierModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = anthropicAPIBase
	}
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 25 * time.Second
	}
	return &AnthropicClassifier{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

func (c *AnthropicClassifier) Model() string { return c.cfg.Model }

func (c *AnthropicClassifier) Classify(ctx context.Context, req ClassificationRequest) (*ClassificationResult, error) {
	if len(req.Findings) == 0 {
		return nil, nil
	}

	userMessage := buildClassifierUserMessage(req)

	body := map[string]any{
		"model":      c.cfg.Model,
		"max_tokens": 512,
		"system":     classifierSystemPrompt,
		"messages": []map[string]any{
			{"role": "user", "content": userMessage},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("classifier: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(c.cfg.BaseURL, "/")+"/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("classifier: build request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)
	httpReq.Header.Set("content-type", "application/json")

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	latencyMs := int(time.Since(start).Milliseconds())
	if err != nil {
		return nil, fmt.Errorf("classifier: http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("classifier: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("classifier: API error %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	text, inputTokens, outputTokens, err := extractTextFromResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("classifier: parse response: %w", err)
	}

	riskScore, confirmedIDs := parseClassifierJSON(text, req.Findings)

	confirmedFindings := filterConfirmedFindings(req.Findings, confirmedIDs)

	return &ClassificationResult{
		Model:       c.cfg.Model,
		RiskScore:   riskScore,
		Findings:    confirmedFindings,
		RawResponse: text,
		TokenCount:  inputTokens + outputTokens,
		LatencyMs:   latencyMs,
	}, nil
}

// buildClassifierUserMessage produces a concise prompt from the scan findings.
func buildClassifierUserMessage(req ClassificationRequest) string {
	var sb strings.Builder
	sb.WriteString("Scan findings to classify:\n\n")
	for i, f := range req.Findings {
		sb.WriteString(fmt.Sprintf("%d. rule_id=%q category=%q severity=%q evidence=%q confidence=%.2f\n",
			i+1, f.RuleID, f.Category, f.Severity, f.Evidence, f.Confidence))
	}
	sb.WriteString(fmt.Sprintf("\nContent hash: %s", req.ContentHash))
	return sb.String()
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type classifierJSON struct {
	RiskScore         int      `json:"risk_score"`
	ConfirmedFindings []string `json:"confirmed_findings"`
	Analysis          string   `json:"analysis"`
}

func extractTextFromResponse(raw []byte) (text string, inputTokens, outputTokens int, err error) {
	var resp anthropicResponse
	if err = json.Unmarshal(raw, &resp); err != nil {
		return "", 0, 0, err
	}
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}
	return text, resp.Usage.InputTokens, resp.Usage.OutputTokens, nil
}

func parseClassifierJSON(text string, findings []Finding) (riskScore int, confirmedIDs map[string]bool) {
	// Attempt to extract JSON from the response text.
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	confirmedIDs = make(map[string]bool)

	if start == -1 || end == -1 || end <= start {
		// Fall back: treat all original findings as confirmed.
		for _, f := range findings {
			confirmedIDs[f.RuleID] = true
		}
		return Score(findings), confirmedIDs
	}

	var parsed classifierJSON
	if err := json.Unmarshal([]byte(text[start:end+1]), &parsed); err != nil {
		for _, f := range findings {
			confirmedIDs[f.RuleID] = true
		}
		return Score(findings), confirmedIDs
	}

	if parsed.RiskScore < 0 {
		parsed.RiskScore = 0
	}
	if parsed.RiskScore > 100 {
		parsed.RiskScore = 100
	}

	for _, id := range parsed.ConfirmedFindings {
		confirmedIDs[id] = true
	}
	return parsed.RiskScore, confirmedIDs
}

func filterConfirmedFindings(findings []Finding, confirmedIDs map[string]bool) []Finding {
	if len(confirmedIDs) == 0 {
		return findings
	}
	var out []Finding
	for _, f := range findings {
		if confirmedIDs[f.RuleID] {
			out = append(out, f)
		}
	}
	return out
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
