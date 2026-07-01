package cloudflare

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

// DeployWorkerProvision holds optional Create Worker metadata when provision runs.
type DeployWorkerProvision struct {
	Tags                          string `json:"tags"`
	Logpush                       *bool  `json:"logpush"`
	ObservabilityEnabled          *bool  `json:"observabilityEnabled"`
	ObservabilityHeadSamplingRate string `json:"observabilityHeadSamplingRate"`
	SubdomainEnabled              *bool  `json:"subdomainEnabled"`
	SubdomainPreviewsEnabled      *bool  `json:"subdomainPreviewsEnabled"`
	TailConsumers                 string `json:"tailConsumers"`
}

type DeployWorkerSpec struct {
	AccountID          string                 `json:"accountId"`
	ScriptName         string                 `json:"scriptName"`
	ProvisionIfMissing *bool                  `json:"provisionIfMissing"`
	Provision          *DeployWorkerProvision `json:"provision"`
	Source             string                 `json:"source"`
	InlineCode         *string                `json:"inlineCode,omitempty"`
	ScriptURL          *string                `json:"scriptUrl,omitempty"`
	CompatibilityDate  string                 `json:"compatibilityDate"`
	CompatibilityFlags string                 `json:"compatibilityFlags"`
	DeploymentMessage  string                 `json:"deploymentMessage"`
}

func deployProvisionArgs(spec DeployWorkerSpec) (tags string, logpush *bool, observabilityEnabled *bool, observabilityHeadSamplingRate string, subdomainEnabled *bool, subdomainPreviewsEnabled *bool, tailConsumers string) {
	p := spec.Provision
	if p == nil {
		return "", nil, nil, "", nil, nil, ""
	}
	return p.Tags, p.Logpush, p.ObservabilityEnabled, p.ObservabilityHeadSamplingRate, p.SubdomainEnabled, p.SubdomainPreviewsEnabled, p.TailConsumers
}

func (d *DeployWorker) Name() string {
	return "cloudflare.deployWorker"
}

func (d *DeployWorker) Label() string {
	return "Deploy Worker"
}

func (d *DeployWorker) Description() string {
	return "Provision a Worker if needed, upload a script version, and deploy it to traffic"
}

func (d *DeployWorker) Documentation() string {
	return `The Deploy Worker component uploads a single-module Worker (` + "`worker.js`" + `) to a **Worker script name** in your account. When **Provision Worker if missing** is enabled (default), it first calls Cloudflare's Create Worker API so a **new** name works without a separate step. After upload, SuperPlane always creates a **deployment** so the new version serves 100% of traffic (` + "`cloudflare.worker.deployed`" + `).

## Script source

- **Inline**: paste the full contents of ` + "`worker.js`" + ` (ES module exporting ` + "`fetch`" + `).
- **URL**: provide an **https** URL that returns the script body (max 2 MiB).

## Configuration

- **Worker script name**: Name of the Worker in your account (same name used in routes and the dashboard). Used for both provisioning and upload.
- **Provision Worker if missing** (default on): Calls ` + "`POST .../workers/workers`" + ` with the optional metadata fields in **Provision settings** when that section is visible. If the Worker already exists, the error is ignored and upload continues.
- **Provision settings** (optional): Tags, Logpush, Observability, Subdomain, Tail consumers — sent only on the provision step when it runs; omit toggles to leave those keys out of the API request.
- **Compatibility date** (optional): Passed to the **script upload** metadata. If omitted, SuperPlane sends a recent default so Cloudflare treats the upload as a modern module Worker.
- **Compatibility flags** (optional): comma- or newline-separated flags for the upload metadata.
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
	provisionSchema := []configuration.Field{
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional tags for the Worker resource, comma- or newline-separated",
			Placeholder: "team:platform, env:production",
		},
		{
			Name:        "logpush",
			Label:       "Logpush enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Whether Logpush is enabled (only sent when this toggle is set)",
		},
		{
			Name:        "observabilityEnabled",
			Label:       "Observability enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "When set, sends an observability object with this enabled flag",
		},
		{
			Name:        "observabilityHeadSamplingRate",
			Label:       "Observability sampling rate",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional head sampling rate from 0 to 1 (sent inside observability when observability is included)",
			Placeholder: "1",
		},
		{
			Name:        "subdomainEnabled",
			Label:       "workers.dev subdomain enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Subdomain enabled flag for the provision request",
		},
		{
			Name:        "subdomainPreviewsEnabled",
			Label:       "Preview URLs enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Whether preview URLs are enabled for the Worker",
		},
		{
			Name:        "tailConsumers",
			Label:       "Tail consumer Worker names",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Comma- or newline-separated Worker names that consume logs from this Worker",
			Placeholder: "my-log-consumer",
		},
	}

	return []configuration.Field{
		{
			Name:        "scriptName",
			Label:       "Worker script name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the Worker script in your Cloudflare account (new or existing)",
			Placeholder: "my-worker",
		},
		{
			Name:        "provisionIfMissing",
			Label:       "Provision Worker if missing",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Call Cloudflare Create Worker before upload so a new script name works (ignored if Worker already exists)",
		},
		{
			Name:        "provision",
			Label:       "Provision settings",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional metadata for Create Worker (only used when provision is enabled)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "provisionIfMissing", Values: []string{"true"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: provisionSchema,
				},
			},
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
			Description: "Workers compatibility date for the uploaded script (YYYY-MM-DD)",
			Placeholder: "2024-01-01",
		},
		{
			Name:        "compatibilityFlags",
			Label:       "Compatibility flags",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional flags for the upload, comma- or newline-separated",
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

func (d *DeployWorker) validateObservabilitySamplingRate(rate string) error {
	if s := strings.TrimSpace(rate); s != "" {
		rateVal, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("observabilityHeadSamplingRate must be a number between 0 and 1: %w", err)
		}
		if rateVal < 0 || rateVal > 1 {
			return errors.New("observabilityHeadSamplingRate must be between 0 and 1")
		}
	}
	return nil
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

	if spec.Provision != nil {
		if err := d.validateObservabilitySamplingRate(spec.Provision.ObservabilityHeadSamplingRate); err != nil {
			return err
		}
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
		if strings.TrimSpace(body) == "" {
			return errors.New("script downloaded from scriptUrl is empty")
		}
		module = body
	default:
		return fmt.Errorf("source must be %q or %q", deployWorkerScriptSourceInline, deployWorkerScriptSourceURL)
	}

	uploadMeta := map[string]any{}
	if strings.TrimSpace(spec.CompatibilityDate) != "" {
		uploadMeta["compatibility_date"] = strings.TrimSpace(spec.CompatibilityDate)
	}
	if flags := parseCommaOrNewlineList(spec.CompatibilityFlags); len(flags) > 0 {
		uploadMeta["compatibility_flags"] = flags
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	scriptName := strings.TrimSpace(spec.ScriptName)
	provision := spec.ProvisionIfMissing == nil || *spec.ProvisionIfMissing
	if provision {
		tags, logpush, obsEn, obsRate, subEn, subPrev, tails := deployProvisionArgs(spec)
		provBody, err := buildWorkerProvisionRequestBody(
			scriptName,
			tags,
			logpush,
			obsEn,
			obsRate,
			subEn,
			subPrev,
			tails,
		)
		if err != nil {
			return err
		}
		if err := ensureWorkerProvisioned(client, accountID, provBody); err != nil {
			return fmt.Errorf("failed to provision worker: %w", err)
		}
	}

	versionID, err := client.UploadWorkerScriptVersion(accountID, scriptName, uploadMeta, module)
	if err != nil {
		return fmt.Errorf("failed to upload worker version: %w", err)
	}

	var annotations map[string]string
	if msg := strings.TrimSpace(spec.DeploymentMessage); msg != "" {
		annotations = map[string]string{"workers/message": msg}
	}

	deployment, err := client.CreateWorkerDeployment(accountID, scriptName, versionID, annotations)
	if err != nil {
		return fmt.Errorf("failed to create worker deployment: %w", err)
	}

	result := map[string]any{
		"accountId":  accountID,
		"scriptName": scriptName,
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
