package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
)

const denoBootstrapSource = `
const [bundleUrl, payloadPath] = Deno.args;
const payload = JSON.parse(await Deno.readTextFile(payloadPath));
const mod = await import(bundleUrl);
if (typeof mod.invoke !== "function") {
  throw new Error("bundle does not export invoke");
}
const result = await mod.invoke(payload);
await Deno.stdout.write(new TextEncoder().encode(JSON.stringify(result)));
`

type RuntimeExecutorConfig struct {
	HubURL     string
	CacheDir   string
	DenoBinary string
	HTTPClient *http.Client
	Runner     commandRunner
}

type commandRunner interface {
	Run(ctx context.Context, name string, args []string, stdout io.Writer, stderr io.Writer) error
}

type RuntimeExecutor struct {
	hubURL     string
	cacheDir   string
	denoBinary string
	httpClient *http.Client
	runner     commandRunner
}

func NewRuntimeExecutor(config RuntimeExecutorConfig) *RuntimeExecutor {
	cacheDir := config.CacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "superplane-extension-worker-cache")
	}

	denoBinary := config.DenoBinary
	if denoBinary == "" {
		denoBinary = "deno"
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	runner := config.Runner
	if runner == nil {
		runner = execRunner{}
	}

	return &RuntimeExecutor{
		hubURL:     config.HubURL,
		cacheDir:   cacheDir,
		denoBinary: denoBinary,
		httpClient: httpClient,
		runner:     runner,
	}
}

func (e *RuntimeExecutor) HandleJob(ctx context.Context, message protocol.JobAssignMessage) (json.RawMessage, error) {
	switch message.JobType {
	case protocol.JobTypeInvokeExtension:
		return e.handleInvokeExtensionJob(ctx, message)
	default:
		return nil, fmt.Errorf("job type %s is not supported", message.JobType)
	}
}

func (e *RuntimeExecutor) handleInvokeExtensionJob(ctx context.Context, message protocol.JobAssignMessage) (json.RawMessage, error) {
	log.Printf("Handling invoke extension job %s", message.JobID)

	if message.InvokeExtension == nil {
		return nil, fmt.Errorf("missing invoke extension specification")
	}

	bundlePath, err := e.ensureBundle(ctx, message)
	if err != nil {
		return nil, err
	}

	return e.invokeBundle(ctx, bundlePath, message.InvokeExtension.Invocation)
}

func (e *RuntimeExecutor) ensureBundle(ctx context.Context, message protocol.JobAssignMessage) (string, error) {
	log.Printf("Ensuring bundle for job %s", message.JobID)

	bundlePath := filepath.Join(
		e.cacheDir,
		message.InvokeExtension.OrganizationID,
		message.InvokeExtension.Extension.Name,
		message.InvokeExtension.Version.Name,
		message.InvokeExtension.Version.Digest,
		"bundle.js",
	)

	if _, err := os.Stat(bundlePath); err == nil {
		log.Printf("Bundle for job %s already exists at %s", message.JobID, bundlePath)
		return bundlePath, nil
	}

	log.Printf("Creating bundle cache directory for job %s", message.JobID)
	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o755); err != nil {
		return "", fmt.Errorf("create bundle cache directory: %w", err)
	}

	log.Printf("Downloading bundle for job %s", message.JobID)
	bundleURL, err := e.bundleURL(message.InvokeExtension)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, bundleURL, nil)
	if err != nil {
		return "", fmt.Errorf("create bundle request: %w", err)
	}

	response, err := e.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("download bundle: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf("download bundle: unexpected status %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read bundle response: %w", err)
	}

	tempPath := bundlePath + ".tmp"
	if err := os.WriteFile(tempPath, content, 0o600); err != nil {
		return "", fmt.Errorf("write cached bundle: %w", err)
	}
	if err := os.Rename(tempPath, bundlePath); err != nil {
		return "", fmt.Errorf("commit cached bundle: %w", err)
	}

	return bundlePath, nil
}

func (e *RuntimeExecutor) invokeBundle(ctx context.Context, bundlePath string, invocation json.RawMessage) (json.RawMessage, error) {
	tempDir, err := os.MkdirTemp("", "superplane-extension-exec-*")
	if err != nil {
		return nil, fmt.Errorf("create execution temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	bootstrapPath := filepath.Join(tempDir, "bootstrap.js")
	if err := os.WriteFile(bootstrapPath, []byte(denoBootstrapSource), 0o600); err != nil {
		return nil, fmt.Errorf("write bootstrap script: %w", err)
	}

	payloadPath := filepath.Join(tempDir, "payload.json")
	if err := os.WriteFile(payloadPath, invocation, 0o600); err != nil {
		return nil, fmt.Errorf("write invocation payload: %w", err)
	}

	bundleURL := fileURL(bundlePath)
	args := []string{
		"run",
		"--quiet",
		fmt.Sprintf("--allow-read=%s,%s,%s", bundlePath, bootstrapPath, payloadPath),
		bootstrapPath,
		bundleURL,
		payloadPath,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := e.runner.Run(ctx, e.denoBinary, args, &stdout, &stderr); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("run deno bundle: %s", message)
	}

	output := bytes.TrimSpace(stdout.Bytes())
	if !json.Valid(output) {
		return nil, fmt.Errorf("bundle returned invalid JSON: %s", string(output))
	}

	return json.RawMessage(output), nil
}

func (e *RuntimeExecutor) bundleURL(invokeExtension *protocol.InvokeExtension) (string, error) {
	bundleURL, err := joinHTTPURL(e.hubURL, "/api/v1/extensions/bundle.js")
	if err != nil {
		return "", err
	}

	return addQuery(bundleURL, map[string]string{
		protocol.QueryToken: invokeExtension.BundleToken,
	})
}

func fileURL(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "file://" + path
	}

	return (&url.URL{Scheme: "file", Path: absolutePath}).String()
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args []string, stdout io.Writer, stderr io.Writer) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

func joinHTTPURL(base string, path string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(path, "/") {
		parsed.Path = path
	} else {
		parsed.Path = "/" + path
	}

	return parsed.String(), nil
}
