package jsruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// APIHandler provides HTTP handlers for managing JS components and
// generating them via AI.
type APIHandler struct {
	Dir     string
	Runtime *Runtime
}

func NewAPIHandler(dir string, rt *Runtime) *APIHandler {
	return &APIHandler{Dir: dir, Runtime: rt}
}

type jsComponentInfo struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Source string `json:"source"`
}

// ListComponents returns all JS component files with their source and parsed label.
func (h *APIHandler) ListComponents(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(h.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]any{"components": []any{}})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to read directory: %v", err)
		return
	}

	components := []jsComponentInfo{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}

		if !validFilenamePattern.MatchString(entry.Name()) {
			continue
		}

		source, err := os.ReadFile(filepath.Join(h.Dir, entry.Name()))
		if err != nil {
			continue
		}

		label := strings.TrimSuffix(entry.Name(), ".js")
		if def, err := h.Runtime.ParseDefinition(string(source)); err == nil && def.Label != "" {
			label = def.Label
		}

		components = append(components, jsComponentInfo{
			Name:   strings.TrimSuffix(entry.Name(), ".js"),
			Label:  label,
			Source: string(source),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"components": components})
}

type saveComponentRequest struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

// SaveComponent writes a JS component file to disk. The file watcher will
// pick it up and register/reload it in the registry automatically.
func (h *APIHandler) SaveComponent(w http.ResponseWriter, r *http.Request) {
	var req saveComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Source == "" {
		writeError(w, http.StatusBadRequest, "name and source are required")
		return
	}

	filename := req.Name + ".js"
	if !validFilenamePattern.MatchString(filename) {
		writeError(w, http.StatusBadRequest,
			"invalid name: must be lowercase alphanumeric with hyphens (e.g. my-component)")
		return
	}

	if len(req.Source) > maxCodeSize {
		writeError(w, http.StatusBadRequest, "source exceeds maximum size of %d bytes", maxCodeSize)
		return
	}

	if _, err := h.Runtime.ParseDefinition(req.Source); err != nil {
		writeError(w, http.StatusBadRequest, "invalid component: %v", err)
		return
	}

	if err := os.MkdirAll(h.Dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create directory: %v", err)
		return
	}

	path := filepath.Join(h.Dir, filename)
	if err := os.WriteFile(path, []byte(req.Source), 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write file: %v", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"name": req.Name, "filename": filename})
}

// DeleteComponent removes a JS component file from disk.
func (h *APIHandler) DeleteComponent(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name query parameter is required")
		return
	}

	filename := name + ".js"
	if !validFilenamePattern.MatchString(filename) {
		writeError(w, http.StatusBadRequest, "invalid name")
		return
	}

	path := filepath.Join(h.Dir, filename)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "component not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete: %v", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": name})
}

type generateRequest struct {
	Messages []generateMessage `json:"messages"`
	Source   string            `json:"source,omitempty"`
}

type generateMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Generate calls the OpenAI API to create or refine a JS component based
// on a chat conversation. Requires OPENAI_API_KEY env var.
func (h *APIHandler) Generate(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		writeError(w, http.StatusServiceUnavailable,
			"AI generation is not configured. Set the OPENAI_API_KEY environment variable.")
		return
	}

	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages are required")
		return
	}

	systemPrompt := buildSystemPrompt(req.Source)

	openAIMessages := []map[string]string{
		{"role": "system", "content": systemPrompt},
	}
	for _, m := range req.Messages {
		role := m.Role
		if role == "" {
			role = "user"
		}
		openAIMessages = append(openAIMessages, map[string]string{"role": role, "content": m.Content})
	}

	body, _ := json.Marshal(map[string]any{
		"model":      "gpt-4o",
		"max_tokens": 4096,
		"messages":   openAIMessages,
	})

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		"https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to read AI response")
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(respBody))
		writeError(w, http.StatusBadGateway, "AI service returned an error")
		return
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		writeError(w, http.StatusBadGateway, "failed to parse AI response")
		return
	}

	text := ""
	if len(openAIResp.Choices) > 0 {
		text = openAIResp.Choices[0].Message.Content
	}

	writeJSON(w, http.StatusOK, map[string]any{"response": text})
}

// ValidateSource parses JS source and returns any validation errors.
func (h *APIHandler) ValidateSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	def, err := h.Runtime.ParseDefinition(req.Source)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false, "error": err.Error()})
		return
	}

	if def.Label == "" {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false, "error": "component must define a label"})
		return
	}

	if !def.HasExecute {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false, "error": "component must define an execute function"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":       true,
		"label":       def.Label,
		"description": def.Description,
		"icon":        def.Icon,
		"color":       def.Color,
	})
}

func buildSystemPrompt(existingSource string) string {
	prompt := `You are a SuperPlane component builder. You help users create JavaScript components
for the SuperPlane workflow automation platform.

A SuperPlane JS component is a single .js file that calls superplane.component() with a definition object.

IMPORTANT RULES:
- The JavaScript runtime is ECMAScript 5.1 (goja). Do NOT use ES6+ features like const, let,
  arrow functions, template literals, destructuring, spread, or classes.
- Use "var" for all variable declarations.
- Use regular function expressions, not arrow functions.
- Use string concatenation with +, not template literals.

Here is the component structure:

superplane.component({
  label: "My Component",           // Required: display name
  description: "What it does",     // Required: what the component does
  icon: "zap",                     // Optional: Lucide icon name
  color: "blue",                   // Optional: UI color

  configuration: [                 // Optional: configuration fields
    {
      name: "fieldName",
      label: "Field Label",
      type: "string",              // Types: string, expression, secret-key, number, boolean, select, text, url
      required: true,
      description: "Help text",
      placeholder: "example",
    },
  ],

  outputChannels: [                // Optional: defaults to single "default" channel
    { name: "default", label: "Default" },
    { name: "error", label: "Error" },
  ],

  setup: function(ctx) {           // Optional: validates configuration when node is saved
    var config = ctx.configuration;
    if (!config.fieldName) {
      throw new Error("fieldName is required");
    }
  },

  execute: function(ctx) {         // Required: runs when the component executes
    var config = ctx.configuration;
    var input = ctx.input;

    // Make HTTP requests
    var response = ctx.http.request("POST", "https://api.example.com/data", {
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: "value" }),
    });

    // Access secrets
    var token = ctx.secrets.getKey(config.tokenSecret.secret, config.tokenSecret.key);

    // Log messages
    ctx.log.info("Processing...");

    // Emit results to output channel
    ctx.emit("default", "event.type", { result: "data" });

    // Or mark as failed
    // ctx.fail("error", "Something went wrong");

    // Or pass through without emitting
    // ctx.pass();
  },
});

Configuration field types:
- "string": simple text input
- "expression": text that supports {{ expressions }} for dynamic values from previous steps
- "secret-key": references a secret stored in SuperPlane (provides .secret and .key)
- "number": numeric input
- "boolean": toggle/checkbox
- "select": dropdown with options array [{value: "x", label: "X"}]
- "text": multiline text input
- "url": URL input

When the user asks you to create or modify a component:
1. Generate the COMPLETE component code
2. Wrap the code in a ` + "```javascript" + ` code fence
3. Explain what the component does and how to configure it

If the user provides feedback, refine the component based on their input.`

	if existingSource != "" {
		prompt += fmt.Sprintf("\n\nThe user is editing an existing component. Here is the current source code:\n\n```javascript\n%s\n```\n\nModify this component based on their request.", existingSource)
	}

	return prompt
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}
