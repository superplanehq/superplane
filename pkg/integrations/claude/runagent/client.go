package runagent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	defaultBaseURL             = "https://api.anthropic.com/v1"
	anthropicVersionValue      = "2023-06-01"
	anthropicBetaManagedAgents = "managed-agents-2026-04-01,files-api-2025-04-14"
	sessionEventsPageLimit     = "20"
)

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
	// sawSessionOutputs records whether an events page mentioned the session
	// outputs directory; set while paging in GetSessionMessages.
	sawSessionOutputs bool
}

type claudeErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// CreateManagedSessionRequest is the body for POST /v1/sessions.
type CreateManagedSessionRequest struct {
	// Agent is the agent ID string, or the ID used with AgentVersion for a specific version.
	Agent         string
	AgentVersion  *int
	EnvironmentID string
	VaultIDs      []string
	Resources     []FileResource
}

// FileResource is a file uploaded via the Files API to mount into the session.
type FileResource struct {
	FileID    string
	MountPath string
}

// ManagedSession is a subset of the session resource returned by the API.
type ManagedSession struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ManagedSessionEvent struct {
	Type    string                       `json:"type"`
	Content []ManagedSessionContentBlock `json:"content"`
}

type ManagedSessionContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// sessionResourceBody is a file resource mounted into the session.
type sessionResourceBody struct {
	Type      string `json:"type"`
	FileID    string `json:"file_id"`
	MountPath string `json:"mount_path"`
}

// createManagedSessionBody is the JSON body for session creation.
type createManagedSessionBody struct {
	Agent         any                   `json:"agent"`
	EnvironmentID string                `json:"environment_id"`
	VaultIDs      []string              `json:"vault_ids,omitempty"`
	Resources     []sessionResourceBody `json:"resources,omitempty"`
}

type userMessageTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type userMessageEvent struct {
	Type    string                 `json:"type"`
	Content []userMessageTextBlock `json:"content"`
}

// sendSessionEventsRequest wraps events for POST .../sessions/{id}/events.
type sendSessionEventsRequest struct {
	Events []userMessageEvent `json:"events"`
}

type listSessionEventsResponse struct {
	Data     []ManagedSessionEvent `json:"data"`
	NextPage string                `json:"next_page"`
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	return &Client{
		APIKey:  string(apiKey),
		BaseURL: defaultBaseURL,
		http:    httpClient,
	}, nil
}

// buildCreateSessionBody maps CreateManagedSessionRequest to JSON.
func buildCreateSessionBody(req CreateManagedSessionRequest) (createManagedSessionBody, error) {
	if req.EnvironmentID == "" {
		return createManagedSessionBody{}, fmt.Errorf("environmentId is required")
	}

	agentID := strings.TrimSpace(req.Agent)
	if agentID == "" {
		return createManagedSessionBody{}, fmt.Errorf("agent is required")
	}

	var agent any = agentID
	if req.AgentVersion != nil {
		agent = map[string]any{
			"type":    "agent",
			"id":      agentID,
			"version": *req.AgentVersion,
		}
	}

	body := createManagedSessionBody{
		Agent:         agent,
		EnvironmentID: req.EnvironmentID,
		VaultIDs:      nonEmptyStrings(req.VaultIDs),
	}

	if len(req.Resources) > 0 {
		body.Resources = make([]sessionResourceBody, len(req.Resources))
		for i, r := range req.Resources {
			body.Resources[i] = sessionResourceBody{
				Type:      "file",
				FileID:    r.FileID,
				MountPath: r.MountPath,
			}
		}
	}

	return body, nil
}

func nonEmptyStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// CreateManagedSession creates a Managed Agents session.
func (c *Client) CreateManagedSession(req CreateManagedSessionRequest) (*ManagedSession, error) {
	body, err := buildCreateSessionBody(req)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session request: %w", err)
	}
	responseBody, err := c.execRequestWithBeta(http.MethodPost, c.BaseURL+"/sessions", bytes.NewBuffer(b), anthropicBetaManagedAgents)
	if err != nil {
		return nil, err
	}
	var out ManagedSession
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session response: %w", err)
	}
	return &out, nil
}

// GetManagedSession retrieves a session by ID (GET /v1/sessions/{id}).
func (c *Client) GetManagedSession(sessionID string) (*ManagedSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID)
	responseBody, err := c.execRequestWithBeta(http.MethodGet, URL, nil, anthropicBetaManagedAgents)
	if err != nil {
		return nil, err
	}
	var out ManagedSession
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	return &out, nil
}

func (c *Client) listManagedSessionEventsPage(sessionID, page string) ([]ManagedSessionEvent, string, error) {
	if sessionID == "" {
		return nil, "", fmt.Errorf("session id is required")
	}

	params := url.Values{}
	params.Set("limit", sessionEventsPageLimit)
	params.Set("order", "desc")
	if page != "" {
		params.Set("page", page)
	}

	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID) + "/events?" + params.Encode()
	responseBody, err := c.execRequestWithBeta(http.MethodGet, URL, nil, anthropicBetaManagedAgents)
	if err != nil {
		return nil, "", err
	}

	var out listSessionEventsResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal session events: %w", err)
	}

	// The raw page is scanned for the session outputs directory: tool inputs
	// (and messages) mention it whenever the agent wrote deliverables, and our
	// event structs only decode text blocks.
	if strings.Contains(string(responseBody), sessionOutputsDir) {
		c.sawSessionOutputs = true
	}
	return out.Data, out.NextPage, nil
}

// sessionOutputsDir is where agents must save deliverables for them to be
// captured by the Files API.
const sessionOutputsDir = "/mnt/session/outputs"

// SessionMessages holds all agent messages and completion status from a session's events.
type SessionMessages struct {
	Messages    []string // all agent.message texts in chronological order
	LastMessage string   // the final agent.message text
	Complete    bool     // true if session.status_idle or session.status_terminated is in the events
	// ExpectsArtifacts is true when the session events mention the outputs
	// directory, i.e. the agent (very likely) wrote deliverables.
	ExpectsArtifacts bool
}

func (c *Client) GetSessionMessages(sessionID string) (*SessionMessages, error) {
	result := &SessionMessages{}
	page := ""
	var allEvents []ManagedSessionEvent

	// Fetch all events (desc order from API)
	c.sawSessionOutputs = false
	for {
		events, nextPage, err := c.listManagedSessionEventsPage(sessionID, page)
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, events...)
		if nextPage == "" {
			break
		}
		page = nextPage
	}
	result.ExpectsArtifacts = c.sawSessionOutputs

	// Check for terminal event (events are in desc order, so status_idle is first)
	for _, e := range allEvents {
		if e.Type == "session.status_idle" || e.Type == "session.status_terminated" {
			result.Complete = true
			break
		}
	}

	// Collect agent messages in chronological order (reverse the desc list)
	for i := len(allEvents) - 1; i >= 0; i-- {
		e := allEvents[i]
		if e.Type == "agent.message" || e.Type == "assistant.message" {
			text := lastAgentMessageFromEvents([]ManagedSessionEvent{e})
			if text != "" {
				result.Messages = append(result.Messages, text)
			}
		}
	}

	if len(result.Messages) > 0 {
		result.LastMessage = result.Messages[len(result.Messages)-1]
	}

	return result, nil
}

// GetSessionMessagesWithRetry polls until events are fully written
// (session.status_idle present) or retries are exhausted.
func (c *Client) GetSessionMessagesWithRetry(sessionID string, attempts int, delay time.Duration) (*SessionMessages, error) {
	if attempts < 1 {
		attempts = 1
	}

	var result *SessionMessages
	for i := 0; i < attempts; i++ {
		var err error
		result, err = c.GetSessionMessages(sessionID)
		if err != nil {
			return nil, err
		}

		// If events are fully written, we have everything
		if result.Complete {
			return result, nil
		}

		if i < attempts-1 {
			time.Sleep(delay)
		}
	}

	return result, nil
}

// Deprecated: use GetSessionMessages instead.
func (c *Client) GetLastManagedSessionAgentMessage(sessionID string) (string, []ManagedSessionEvent, error) {
	seen := []ManagedSessionEvent{}
	page := ""
	for {
		events, nextPage, err := c.listManagedSessionEventsPage(sessionID, page)
		if err != nil {
			return "", seen, err
		}
		seen = append(seen, events...)

		message := lastAgentMessageFromEvents(events)
		if message != "" || nextPage == "" {
			return message, seen, nil
		}

		page = nextPage
	}
}

func (c *Client) GetLastManagedSessionAgentMessageWithRetry(sessionID string, attempts int, delay time.Duration) (string, []ManagedSessionEvent, error) {
	if attempts < 1 {
		attempts = 1
	}

	var events []ManagedSessionEvent
	for i := 0; i < attempts; i++ {
		var err error
		var message string
		message, events, err = c.GetLastManagedSessionAgentMessage(sessionID)
		if err != nil {
			return "", events, err
		}

		if message != "" {
			return message, events, nil
		}

		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return "", events, nil
}

func lastAgentMessageFromEvents(events []ManagedSessionEvent) string {
	for _, event := range events {
		if event.Type != "agent.message" && event.Type != "assistant.message" {
			continue
		}

		parts := []string{}
		for _, block := range event.Content {
			if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
				parts = append(parts, block.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func managedSessionEventTypes(events []ManagedSessionEvent) string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.Type)
	}
	return strings.Join(types, ", ")
}

// SendManagedSessionUserMessage appends a user.message event to the session.
// The events endpoint uses ?beta=true per the Managed Agents API.
func (c *Client) SendManagedSessionUserMessage(sessionID, text string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	if text == "" {
		return fmt.Errorf("message text is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID) + "/events?beta=true"
	payload := sendSessionEventsRequest{
		Events: []userMessageEvent{{
			Type: "user.message",
			Content: []userMessageTextBlock{{
				Type: "text",
				Text: text,
			}},
		}},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}
	_, err = c.execRequestWithBeta(http.MethodPost, URL, bytes.NewBuffer(b), anthropicBetaManagedAgents)
	return err
}

// SendManagedSessionInterrupt sends a user.interrupt event (stop agent mid-execution).
func (c *Client) SendManagedSessionInterrupt(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID) + "/events?beta=true"
	payload := map[string]any{
		"events": []map[string]any{
			{"type": "user.interrupt"},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal interrupt: %w", err)
	}
	_, err = c.execRequestWithBeta(http.MethodPost, URL, bytes.NewBuffer(b), anthropicBetaManagedAgents)
	return err
}

// UploadFile uploads a file to the Anthropic Files API and returns its ID.
// The file can then be mounted into a session via CreateManagedSessionRequest.Resources.
// AddSessionResource attaches a file resource to an existing session.
// POST /v1/sessions/{id}/resources
func (c *Client) AddSessionResource(sessionID string, resource FileResource) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID) + "/resources"
	body := map[string]string{
		"type":       "file",
		"file_id":    resource.FileID,
		"mount_path": resource.MountPath,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal resource: %w", err)
	}
	_, err = c.execRequestWithBeta(http.MethodPost, URL, bytes.NewBuffer(b), anthropicBetaManagedAgents)
	return err
}

func (c *Client) UploadFile(content io.Reader, filename string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return "", fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, content); err != nil {
		return "", fmt.Errorf("copy file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/files", &body)
	if err != nil {
		return "", fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", anthropicVersionValue)
	req.Header.Set("anthropic-beta", anthropicBetaManagedAgents)

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read upload response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("upload file failed (%d): %s", res.StatusCode, string(resBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resBody, &result); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("upload returned empty file ID")
	}
	return result.ID, nil
}

// DeleteManagedSession removes a session (DELETE /v1/sessions/{id}).
// The API does not allow deleting a running session without interrupting first.
func (c *Client) DeleteManagedSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID)
	_, err := c.execRequestWithBeta(http.MethodDelete, URL, nil, anthropicBetaManagedAgents)
	return err
}

// SessionFile is a file surfaced by the Files API for a Managed Agents
// session. Files the agent writes to /mnt/session/outputs/ are captured with
// downloadable=true; input files mounted into the session are not downloadable.
type SessionFile struct {
	ID           string `json:"id"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mime_type"`
	SizeBytes    int64  `json:"size_bytes"`
	CreatedAt    string `json:"created_at"`
	Downloadable bool   `json:"downloadable"`
}

type listSessionFilesResponse struct {
	Data    []SessionFile `json:"data"`
	LastID  string        `json:"last_id"`
	HasMore bool          `json:"has_more"`
}

// maxSessionFilePages caps forward pagination when listing session files so a
// runaway has_more loop can never hang an execution.
const maxSessionFilePages = 10

// ListSessionFiles lists the files scoped to a session (GET /v1/files?scope_id=...),
// paginating forward with after_id.
func (c *Client) ListSessionFiles(sessionID string) ([]SessionFile, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}

	var files []SessionFile
	afterID := ""
	for range maxSessionFilePages {
		params := url.Values{}
		params.Set("scope_id", sessionID)
		params.Set("limit", "1000")
		if afterID != "" {
			params.Set("after_id", afterID)
		}

		responseBody, err := c.execRequestWithBeta(http.MethodGet, c.BaseURL+"/files?"+params.Encode(), nil, anthropicBetaManagedAgents)
		if err != nil {
			return nil, err
		}

		var response listSessionFilesResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session files: %w", err)
		}

		files = append(files, response.Data...)
		if !response.HasMore || response.LastID == "" {
			break
		}
		afterID = response.LastID
	}
	return files, nil
}

// ListSessionFilesWithRetry lists session files, retrying while the listing
// has no downloadable entries — the agent's outputs can take a few seconds to
// be indexed after the session goes idle, and mounted input copies (which are
// never downloadable) may appear before them.
func (c *Client) ListSessionFilesWithRetry(sessionID string, attempts int, delay time.Duration) ([]SessionFile, error) {
	if attempts < 1 {
		attempts = 1
	}

	var files []SessionFile
	for i := 0; i < attempts; i++ {
		var err error
		files, err = c.ListSessionFiles(sessionID)
		if err != nil {
			return nil, err
		}
		if hasDownloadableFile(files) {
			return files, nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return files, nil
}

func hasDownloadableFile(files []SessionFile) bool {
	for _, f := range files {
		if f.Downloadable {
			return true
		}
	}
	return false
}

// FileContentURL returns the programmatic download link for a file. Requests
// to it require the API key headers, including the beta header.
func (c *Client) FileContentURL(fileID string) string {
	return c.BaseURL + "/files/" + url.PathEscape(fileID) + "/content"
}

// DownloadFileContent fetches a file's raw content (GET /v1/files/{id}/content).
func (c *Client) DownloadFileContent(fileID string) ([]byte, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file id is required")
	}
	return c.execRequestWithBeta(http.MethodGet, c.FileContentURL(fileID), nil, anthropicBetaManagedAgents)
}

// DeleteFile removes an uploaded file (DELETE /v1/files/{id}).
func (c *Client) DeleteFile(fileID string) error {
	if fileID == "" {
		return nil
	}
	URL := c.BaseURL + "/files/" + url.PathEscape(fileID)
	_, err := c.execRequestWithBeta(http.MethodDelete, URL, nil, anthropicBetaManagedAgents)
	return err
}

// CleanupFiles deletes a list of uploaded files, logging failures.
func (c *Client) CleanupFiles(fileIDs []string, logWarn func(string, ...any)) {
	for _, id := range fileIDs {
		if err := c.DeleteFile(id); err != nil {
			if logWarn != nil {
				logWarn("Failed to delete uploaded file %s: %v", id, err)
			}
		}
	}
}

// CreateVault creates a temporary vault and returns its ID.
func (c *Client) CreateVault(displayName string, metadata map[string]string) (string, error) {
	payload := map[string]any{"display_name": displayName}
	if len(metadata) > 0 {
		payload["metadata"] = metadata
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal vault request: %w", err)
	}
	respBody, err := c.execRequestWithBeta(http.MethodPost, c.BaseURL+"/vaults", bytes.NewBuffer(b), anthropicBetaManagedAgents)
	if err != nil {
		return "", err
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode vault response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("vault creation returned empty ID")
	}
	return result.ID, nil
}

// CreateEnvVarCredential adds an environment_variable credential to a vault.
// The secret is injected at the network egress layer; the agent never sees the raw value.
func (c *Client) CreateEnvVarCredential(vaultID, displayName, envName, secretValue string, allowedHosts []string) error {
	if vaultID == "" || envName == "" || secretValue == "" {
		return fmt.Errorf("vaultID, envName, and secretValue are required")
	}
	networking := map[string]any{"type": "unrestricted"}
	if len(allowedHosts) > 0 {
		networking = map[string]any{
			"type":          "limited",
			"allowed_hosts": allowedHosts,
		}
	}
	payload := map[string]any{
		"display_name": displayName,
		"auth": map[string]any{
			"type":         "environment_variable",
			"secret_name":  envName,
			"secret_value": secretValue,
			"networking":   networking,
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal credential request: %w", err)
	}
	_, err = c.execRequestWithBeta(http.MethodPost, c.BaseURL+"/vaults/"+url.PathEscape(vaultID)+"/credentials", bytes.NewBuffer(b), anthropicBetaManagedAgents)
	return err
}

// DeleteVault removes a vault and its credentials.
func (c *Client) DeleteVault(vaultID string) error {
	if vaultID == "" {
		return nil
	}
	_, err := c.execRequestWithBeta(http.MethodDelete, c.BaseURL+"/vaults/"+url.PathEscape(vaultID), nil, anthropicBetaManagedAgents)
	return err
}

func (c *Client) execRequestWithBeta(method, URL string, body io.Reader, beta string) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", anthropicVersionValue)
	if beta != "" {
		req.Header.Set("anthropic-beta", beta)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		var apiErr claudeErrorResponse
		var errorMessage string
		if err := json.Unmarshal(responseBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			errorMessage = apiErr.Error.Message
		} else {
			errorMessage = string(responseBody)
		}

		if res.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("Claude credentials are invalid or expired: %s", errorMessage)
		}

		return nil, fmt.Errorf("request failed (%d): %s", res.StatusCode, errorMessage)
	}
	return responseBody, nil
}
