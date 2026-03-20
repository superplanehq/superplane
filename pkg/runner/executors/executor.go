package executors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

const denoModuleRunnerSource = `
const [bundleUrl, jobPath] = Deno.args;
const job = JSON.parse(await Deno.readTextFile(jobPath));
const mod = await import(bundleUrl);
if (typeof mod.run !== "function") {
  throw new Error("bundle does not export run");
}
const result = await mod.run(job);
await Deno.stdout.write(new TextEncoder().encode(JSON.stringify(result)));
`

const denoExecuteCodeRunnerSource = `
const [runtimeUrl, moduleUrl, jobPath] = Deno.args;
const runtime = await import(runtimeUrl);
if (typeof runtime.runExecuteCodeModule !== "function") {
  throw new Error("runtime does not export runExecuteCodeModule");
}
const mod = await import(moduleUrl);
const job = JSON.parse(await Deno.readTextFile(jobPath));
const result = await runtime.runExecuteCodeModule(mod, job);
await Deno.stdout.write(new TextEncoder().encode(JSON.stringify(result)));
`

type Config struct {
	HubURL     string
	CacheDir   string
	DenoBinary string
	HTTPClient *http.Client
	Runner     commandRunner
}

func (c *Config) Validate() error {
	if c.HubURL == "" {
		return fmt.Errorf("HUB_URL is required")
	}

	if c.CacheDir == "" {
		return fmt.Errorf("CACHE_DIR is required")
	}

	if c.DenoBinary == "" {
		return fmt.Errorf("DENO_BINARY is required")
	}

	if c.Runner == nil {
		return fmt.Errorf("RUNNER is required")
	}

	return nil
}

func (c *Config) HTTP() *http.Client {
	if c.HTTPClient == nil {
		return &http.Client{Timeout: 30 * time.Second}
	}

	return c.HTTPClient
}

type commandRunner interface {
	Run(ctx context.Context, name string, args []string, stdout io.Writer, stderr io.Writer) error
}

type Executor struct {
	hubURL     string
	cacheDir   string
	denoBinary string
	httpClient *http.Client
	runner     commandRunner
}

func New(config Config) (*Executor, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Executor{
		hubURL:     config.HubURL,
		cacheDir:   config.CacheDir,
		denoBinary: config.DenoBinary,
		httpClient: config.HTTP(),
		runner:     config.Runner,
	}, nil
}

func (e *Executor) HandleJob(ctx context.Context, message protocol.JobAssignMessage) (json.RawMessage, error) {
	switch message.JobType {
	case protocol.JobTypeExecuteCode:
		return e.handleExecuteCodeJob(ctx, message)
	case protocol.JobTypeInvokeExtension:
		return e.handleInvokeExtensionJob(ctx, message)
	default:
		return nil, fmt.Errorf("job type %s is not supported", message.JobType)
	}
}

func (e *Executor) handleExecuteCodeJob(ctx context.Context, message protocol.JobAssignMessage) (json.RawMessage, error) {
	log.Printf("Handling execute code job %s", message.JobID)

	if message.ExecuteCode == nil {
		return nil, fmt.Errorf("missing execute code specification")
	}

	return e.executeCode(ctx, message.ExecuteCode.Code, message.ExecuteCode.Timeout, message.ExecuteCode.Invocation)
}

func (e *Executor) executeCode(ctx context.Context, code string, timeout int, invocation json.RawMessage) (json.RawMessage, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("execute-code job is missing code")
	}

	if timeout <= 0 {
		timeout = 30
	}

	log.Printf("Executing code with timeout %d seconds", timeout)

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	tempDir, err := os.MkdirTemp("", "superplane-execute-code-*")
	if err != nil {
		return nil, fmt.Errorf("create execute-code temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	runnerPath := filepath.Join(tempDir, "runner.js")
	if err := os.WriteFile(runnerPath, []byte(denoExecuteCodeRunnerSource), 0o600); err != nil {
		return nil, fmt.Errorf("write execute-code runner: %w", err)
	}

	modulePath := filepath.Join(tempDir, "module.js")
	if err := os.WriteFile(modulePath, []byte(code), 0o600); err != nil {
		return nil, fmt.Errorf("write execute-code module: %w", err)
	}

	jobPayload, err := buildExecuteCodeJob(invocation)
	if err != nil {
		return nil, err
	}

	jobPath := filepath.Join(tempDir, "job.json")
	if err := os.WriteFile(jobPath, jobPayload, 0o600); err != nil {
		return nil, fmt.Errorf("write execute-code job: %w", err)
	}

	runtimePath, err := findPackageEntryPoint(".", filepath.Join("extensions", "runtime", "ts", "src", "index.ts"))
	if err != nil {
		return nil, err
	}

	args := []string{
		"run",
		"--quiet",
		"--no-prompt",
		"--sloppy-imports",
		fmt.Sprintf("--allow-read=%s,%s,%s", tempDir, runtimePath, filepath.Dir(runtimePath)),
		"--allow-net",
		runnerPath,
		fileURL(runtimePath),
		fileURL(modulePath),
		jobPath,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := e.runner.Run(runCtx, e.denoBinary, args, &stdout, &stderr); err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("execute code timed out after %d seconds", timeout)
		}

		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("run execute-code module: %s", message)
	}

	output := bytes.TrimSpace(stdout.Bytes())
	if !json.Valid(output) {
		return nil, fmt.Errorf("execute-code returned invalid JSON: %s", string(output))
	}

	return json.RawMessage(output), nil
}

func (e *Executor) handleInvokeExtensionJob(ctx context.Context, message protocol.JobAssignMessage) (json.RawMessage, error) {
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

func (e *Executor) ensureBundle(ctx context.Context, message protocol.JobAssignMessage) (string, error) {
	log.Printf("Ensuring bundle for job %s", message.JobID)

	bundlePath := filepath.Join(
		e.cacheDir,
		message.OrganizationID,
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

func (e *Executor) invokeBundle(ctx context.Context, bundlePath string, invocation json.RawMessage) (json.RawMessage, error) {
	//
	// TODO
	// Since we have a temp dir for each execution,
	// we can probably pass a ctx.fs in the RuntimeContext.
	//
	tempDir, err := os.MkdirTemp("", "superplane-extension-exec-*")
	if err != nil {
		return nil, fmt.Errorf("create execution temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	runnerPath := filepath.Join(tempDir, "runner.js")
	if err := os.WriteFile(runnerPath, []byte(denoModuleRunnerSource), 0o600); err != nil {
		return nil, fmt.Errorf("write runner script: %w", err)
	}

	jobPayload, err := buildInvokeExtensionJob(invocation)
	if err != nil {
		return nil, err
	}

	jobPath := filepath.Join(tempDir, "job.json")
	if err := os.WriteFile(jobPath, jobPayload, 0o600); err != nil {
		return nil, fmt.Errorf("write invoke-extension job: %w", err)
	}

	bundleURL := fileURL(bundlePath)
	args := []string{
		"run",
		"--quiet",
		fmt.Sprintf("--allow-read=%s,%s,%s", bundlePath, runnerPath, jobPath),
		runnerPath,
		bundleURL,
		jobPath,
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

func buildInvokeExtensionJob(invocation json.RawMessage) ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":    protocol.JobTypeInvokeExtension,
		"payload": json.RawMessage(invocation),
	})
}

func buildExecuteCodeJob(invocation json.RawMessage) ([]byte, error) {
	job := map[string]any{
		"type": protocol.JobTypeExecuteCode,
	}

	if len(invocation) > 0 {
		job["invocation"] = json.RawMessage(invocation)
	}

	return json.Marshal(job)
}

func (e *Executor) bundleURL(invokeExtension *protocol.InvokeExtension) (string, error) {
	URL, err := url.Parse(e.hubURL)
	if err != nil {
		return "", err
	}

	URL.Path = "/api/v1/extensions/bundle.js"
	query := URL.Query()
	query.Set(protocol.QueryToken, invokeExtension.BundleToken)
	URL.RawQuery = query.Encode()
	return URL.String(), nil
}

func fileURL(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "file://" + path
	}

	return (&url.URL{Scheme: "file", Path: absolutePath}).String()
}

func findPackageEntryPoint(startDir string, relativePath string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve start directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, relativePath)
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find %s", relativePath)
		}

		dir = parent
	}
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args []string, stdout io.Writer, stderr io.Writer) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}
