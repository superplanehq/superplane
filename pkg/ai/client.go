package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const defaultEndpoint = "https://api.openai.com/v1/chat/completions"
const defaultModel = "gpt-4o"

var codeBlockRe = regexp.MustCompile("(?s)```(?:typescript|javascript|ts|js)?\\s*\\n(.*?)```")

const systemPrompt = `You are a SuperPlane plugin code generator. You write TypeScript plugins that use the @superplane/sdk API.

A SuperPlane plugin is a single file that exports an activate() function. The activate function receives a plugin context and uses it to register components and triggers.

Example plugin structure:

const superplane = require("@superplane/sdk");

export function activate(ctx) {
  ctx.components.register("my-component", {
    label: "My Component",
    description: "Does something useful",
    configuration: [
      { name: "param1", label: "Parameter 1", type: "string", required: true }
    ],
    outputChannels: [{ name: "default", label: "Default" }],
    async execute(execCtx) {
      const config = execCtx.configuration;
      // Do work here using execCtx.http.request() for external API calls
      const result = await execCtx.http.request({
        method: "POST",
        url: "https://api.example.com/action",
        headers: { "Authorization": "Bearer " + await execCtx.secrets.getKey("api-key") },
        body: JSON.stringify({ param: config.param1 })
      });
      execCtx.emit("default", { result: JSON.parse(result.body) });
    }
  });
}

export function deactivate() {}

Available context methods:
- ctx.components.register(name, handler) - Register a component
- ctx.triggers.register(name, handler) - Register a trigger
- execCtx.emit(channel, data) - Emit data to an output channel
- execCtx.pass() - Mark execution as passed
- execCtx.fail(reason, message) - Mark execution as failed
- execCtx.configuration - Access component configuration values
- execCtx.secrets.getKey(name) - Get a secret value
- execCtx.http.request({method, url, headers, body}) - Make HTTP requests
- execCtx.metadata.get(key) / execCtx.metadata.set(key, value) - Read/write metadata

Rules:
- Always export activate() and deactivate() functions
- Use require("@superplane/sdk") for the SDK
- Keep the code in a single file
- Use async/await for asynchronous operations
- Include proper error handling
- Return the complete, runnable code
- When writing JSON objects or configuration objects, always format them with proper indentation (one key per line), never as a single long line
- NEVER use external npm packages (e.g. axios, node-fetch, telnet-client). Only built-in Node.js modules and @superplane/sdk are available. For HTTP requests, always use execCtx.http.request()
- In execute(), use only ONE terminal action: emit OR pass OR fail
- NEVER call execCtx.pass() after execCtx.emit(...), because pass can overwrite emitted output

When the user asks you to modify existing code, update the existing code rather than rewriting from scratch.
Respond with a brief explanation followed by the complete code in a typescript code block.`

type Client struct {
	apiKey   string
	endpoint string
	model    string
	client   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: defaultEndpoint,
		model:    defaultModel,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) GenerateScript(ctx context.Context, userMessage string, existingSource string) (string, string, error) {
	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
	}

	if existingSource != "" {
		messages = append(messages, chatMessage{
			Role:    "user",
			Content: fmt.Sprintf("Here is the current script code:\n```typescript\n%s\n```", existingSource),
		})
		messages = append(messages, chatMessage{
			Role:    "assistant",
			Content: "I see the current code. What changes would you like me to make?",
		})
	}

	messages = append(messages, chatMessage{
		Role:    "user",
		Content: userMessage,
	})

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("reading response: %w", err)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", "", fmt.Errorf("parsing response: %w", err)
	}

	if chatResp.Error != nil {
		return "", "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", "", fmt.Errorf("no response from AI")
	}

	fullResponse := chatResp.Choices[0].Message.Content
	source := extractCode(fullResponse)

	return fullResponse, source, nil
}

func extractCode(response string) string {
	matches := codeBlockRe.FindAllStringSubmatch(response, -1)
	if len(matches) == 0 {
		return ""
	}

	// Return the last code block (most likely the complete code)
	lastMatch := matches[len(matches)-1]
	return strings.TrimSpace(lastMatch[1])
}
