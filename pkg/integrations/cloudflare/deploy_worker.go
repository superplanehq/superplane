package cloudflare

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	deployWorkerScriptSourceInline = "inline"
	deployWorkerScriptSourceURL    = "url"
	maxWorkerScriptDownloadBytes   = 2 * 1024 * 1024
)

type DeployWorker struct{}

type DeployWorkerSpec struct {
	AccountID          string  `json:"accountId"`
	ScriptName         string  `json:"scriptName"`
	Source             string  `json:"source"`
	InlineCode         *string `json:"inlineCode,omitempty"`
	ScriptURL          *string `json:"scriptUrl,omitempty"`
	CompatibilityDate  string  `json:"compatibilityDate"`
	CompatibilityFlags string  `json:"compatibilityFlags"`
	DeploymentMessage  string  `json:"deploymentMessage"`
}

func (d *DeployWorker) Name() string {
	return "cloudflare.deployWorker"
}

func (d *DeployWorker) Label() string {
	return "Deploy Worker"
}

func (d *DeployWorker) Description() string {
	return "Upload a Worker script and deploy a new version to production traffic"
}

func (d *DeployWorker) Documentation() string {
	return `The Deploy Worker component uploads a single-module Worker (` + "`worker.js`" + `) and creates a deployment so that version serves 100% of traffic.

## Script source

- **Inline**: paste the full contents of ` + "`worker.js`" + ` (ES module exporting ` + "`fetch`" + `).
- **URL**: provide an **https** URL that returns the script body (max 2 MiB).

## Configuration

- **Script name**: Cloudflare Worker script name (used in routes and the dashboard).
- **Compatibility date** (optional): Workers compatibility date (for example ` + "`2024-01-01`" + `). Recommended; if omitted, Cloudflare applies account defaults.
- **Compatibility flags** (optional): comma- or newline-separated flags (for example ` + "`nodejs_compat`" + `).
- **Deployment message** (optional): annotation stored on the deployment.

## Output

Emits the account ID, script name, new version ID, and deployment object.`
}

func (d *DeployWorker) Icon() string {
	return "cloud"
}

func (d *DeployWorker) Color() string {
	return "orange"
}

func (d *DeployWorker) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeployWorker) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "scriptName",
			Label:       "Worker script name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the Worker script in your Cloudflare account",
			Placeholder: "my-worker",
		},
		{
			Name:        "source",
			Label:       "Script source",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Whether the script is pasted inline or downloaded from a URL",
			Default:     deployWorkerScriptSourceInline,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Inline", Value: deployWorkerScriptSourceInline},
						{Label: "URL (https)", Value: deployWorkerScriptSourceURL},
					},
				},
			},
		},
		{
			Name:        "inlineCode",
			Label:       "Worker script (worker.js)",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Full ES module source for worker.js (must export fetch)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "source", Values: []string{deployWorkerScriptSourceInline}},
			},
		},
		{
			Name:        "scriptUrl",
			Label:       "Script URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "HTTPS URL that returns the worker.js source (max 2 MiB)",
			Placeholder: "https://example.com/worker.js",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "source", Values: []string{deployWorkerScriptSourceURL}},
			},
		},
		{
			Name:        "compatibilityDate",
			Label:       "Compatibility date",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Workers compatibility date (YYYY-MM-DD)",
			Placeholder: "2024-01-01",
		},
		{
			Name:        "compatibilityFlags",
			Label:       "Compatibility flags",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional flags, comma- or newline-separated",
			Placeholder: "nodejs_compat",
		},
		{
			Name:        "deploymentMessage",
			Label:       "Deployment message",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional human-readable message stored on the deployment",
		},
	}
}

func (d *DeployWorker) Setup(ctx core.SetupContext) error {
	spec := DeployWorkerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if strings.TrimSpace(spec.ScriptName) == "" {
		return errors.New("scriptName is required")
	}

	switch spec.Source {
	case "", deployWorkerScriptSourceInline:
		if spec.InlineCode == nil || strings.TrimSpace(*spec.InlineCode) == "" {
			return errors.New("inlineCode is required when source is inline")
		}
	case deployWorkerScriptSourceURL:
		if spec.ScriptURL == nil || strings.TrimSpace(*spec.ScriptURL) == "" {
			return errors.New("scriptUrl is required when source is url")
		}
	default:
		return fmt.Errorf("source must be %q or %q", deployWorkerScriptSourceInline, deployWorkerScriptSourceURL)
	}

	return nil
}

func (d *DeployWorker) Execute(ctx core.ExecutionContext) error {
	spec := DeployWorkerSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	source := spec.Source
	if source == "" {
		source = deployWorkerScriptSourceInline
	}

	var module string
	switch source {
	case deployWorkerScriptSourceInline:
		if spec.InlineCode == nil {
			return errors.New("inlineCode is required when source is inline")
		}
		module = strings.TrimSpace(*spec.InlineCode)
		if module == "" {
			return errors.New("inlineCode is required when source is inline")
		}
	case deployWorkerScriptSourceURL:
		if spec.ScriptURL == nil {
			return errors.New("scriptUrl is required when source is url")
		}
		body, err := fetchWorkerScriptFromHTTPSURL(ctx.HTTP, strings.TrimSpace(*spec.ScriptURL))
		if err != nil {
			return err
		}
		module = body
	default:
		return fmt.Errorf("source must be %q or %q", deployWorkerScriptSourceInline, deployWorkerScriptSourceURL)
	}

	metadata := map[string]any{}
	if strings.TrimSpace(spec.CompatibilityDate) != "" {
		metadata["compatibility_date"] = strings.TrimSpace(spec.CompatibilityDate)
	}
	if flags := parseCommaOrNewlineList(spec.CompatibilityFlags); len(flags) > 0 {
		metadata["compatibility_flags"] = flags
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	versionID, err := client.UploadWorkerScriptVersion(accountID, strings.TrimSpace(spec.ScriptName), metadata, module)
	if err != nil {
		return fmt.Errorf("failed to upload worker version: %w", err)
	}

	var annotations map[string]string
	if msg := strings.TrimSpace(spec.DeploymentMessage); msg != "" {
		annotations = map[string]string{"workers/message": msg}
	}

	deployment, err := client.CreateWorkerDeployment(accountID, strings.TrimSpace(spec.ScriptName), versionID, annotations)
	if err != nil {
		return fmt.Errorf("failed to create worker deployment: %w", err)
	}

	result := map[string]any{
		"accountId":  accountID,
		"scriptName": strings.TrimSpace(spec.ScriptName),
		"versionId":  versionID,
		"deployment": deployment,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.worker.deployed",
		[]any{result},
	)
}

func fetchWorkerScriptFromHTTPSURL(httpCtx core.HTTPContext, raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("scriptUrl must be a valid https URL with a host")
	}

	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return "", fmt.Errorf("error building script download request: %w", err)
	}

	res, err := httpCtx.Do(req)
	if err != nil {
		return "", fmt.Errorf("error downloading script: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return "", fmt.Errorf("script URL returned status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	limited := io.LimitReader(res.Body, maxWorkerScriptDownloadBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("error reading script body: %w", err)
	}
	if len(body) > maxWorkerScriptDownloadBytes {
		return "", fmt.Errorf("script from URL exceeds maximum size of %d bytes", maxWorkerScriptDownloadBytes)
	}

	return string(body), nil
}

func parseCommaOrNewlineList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (d *DeployWorker) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeployWorker) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeployWorker) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeployWorker) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeployWorker) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeployWorker) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
