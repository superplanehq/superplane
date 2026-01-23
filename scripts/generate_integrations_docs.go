package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	pw "github.com/playwright-community/playwright-go"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/workflows"
	spjwt "github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/server"
	"google.golang.org/protobuf/encoding/protojson"
	yamlv3 "gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	// Register integrations used by doc templates.
	_ "github.com/superplanehq/superplane/pkg/components/http"
	_ "github.com/superplanehq/superplane/pkg/triggers/webhook"
)

const docsRoot = "docs/integrations"
const coreIndexPath = "docs/integrations/Core/index.yaml"
const coreOutputPath = "docs/integrations/Core.mdx"

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

type docEntry struct {
	Title           string           `yaml:"title"`
	Subtitle        string           `yaml:"subtitle"`
	Description     string           `yaml:"description"`
	ExampleOutput   string           `yaml:"exampleOutput"`
	ExampleUseCases []exampleUseCase `yaml:"exampleUseCases"`
}

type docIndex struct {
	Title      string     `yaml:"title"`
	Overview   string     `yaml:"overview"`
	Components []docEntry `yaml:"components"`
	Triggers   []docEntry `yaml:"triggers"`
}

type exampleUseCase struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Canvas      string `yaml:"canvas"`
}

func main() {
	if err := os.MkdirAll(docsRoot, 0o755); err != nil {
		exitWithError(err)
	}

	if err := writeCoreDocsFromIndex(coreIndexPath, coreOutputPath); err != nil {
		exitWithError(err)
	}
}

func writeCoreDocsFromIndex(indexPath string, outputPath string) error {
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	var index docIndex
	if err := yamlv3.Unmarshal(indexData, &index); err != nil {
		return err
	}

	var buf bytes.Buffer
	writeCoreFrontMatter(&buf, strings.TrimSpace(index.Title))
	writeImports(&buf, len(index.Components) > 0 || len(index.Triggers) > 0, hasExampleUseCases(index))
	writeOverviewSection(&buf, index.Overview)
	writeCardGridTriggers(&buf, index.Triggers)
	writeCardGridComponents(&buf, index.Components)
	if err := writeTriggerSection(&buf, index.Triggers); err != nil {
		return err
	}
	if err := writeComponentSection(&buf, index.Components); err != nil {
		return err
	}

	return writeFile(outputPath, buf.Bytes())
}

func writeCoreFrontMatter(buf *bytes.Buffer, title string) {
	if title == "" {
		title = "Core"
	}
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("title: \"%s\"\n", escapeQuotes(title)))
	buf.WriteString("sidebar:\n")
	buf.WriteString(fmt.Sprintf("  label: \"%s\"\n", escapeQuotes(title)))
	buf.WriteString(fmt.Sprintf("type: \"%s\"\n", escapeQuotes("core")))
	buf.WriteString("---\n\n")
}

func writeComponentSection(buf *bytes.Buffer, components []docEntry) error {
	if len(components) == 0 {
		return nil
	}

	for _, component := range components {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(component.Title)))
		buf.WriteString(fmt.Sprintf("## %s\n\n", component.Title))
		writeParagraph(buf, component.Description)
		if err := writeExampleUseCasesSection(buf, component.ExampleUseCases); err != nil {
			return err
		}
		if err := writeExampleSection("Example Output", component.ExampleOutput, buf); err != nil {
			return err
		}
	}

	return nil
}

func writeTriggerSection(buf *bytes.Buffer, triggers []docEntry) error {
	if len(triggers) == 0 {
		return nil
	}

	for _, trigger := range triggers {
		buf.WriteString(fmt.Sprintf("<a id=\"%s\"></a>\n\n", slugify(trigger.Title)))
		buf.WriteString(fmt.Sprintf("## %s\n\n", trigger.Title))
		writeParagraph(buf, trigger.Description)
		if err := writeExampleUseCasesSection(buf, trigger.ExampleUseCases); err != nil {
			return err
		}
		if err := writeExampleSection("Example Data", trigger.ExampleOutput, buf); err != nil {
			return err
		}
	}

	return nil
}

func writeCardGridComponents(buf *bytes.Buffer, components []docEntry) {
	if len(components) == 0 {
		return
	}

	buf.WriteString("## Components\n\n")
	buf.WriteString("<CardGrid>\n")
	for _, component := range components {
		description := strings.TrimSpace(component.Subtitle)
		if description == "" {
			description = strings.TrimSpace(component.Description)
		}
		buf.WriteString(fmt.Sprintf("  <LinkCard title=\"%s\" href=\"#%s\" description=\"%s\" />\n",
			escapeQuotes(component.Title),
			slugify(component.Title),
			escapeQuotes(description),
		))
	}
	buf.WriteString("</CardGrid>\n\n")
}

func writeCardGridTriggers(buf *bytes.Buffer, triggers []docEntry) {
	if len(triggers) == 0 {
		return
	}

	buf.WriteString("## Triggers\n\n")
	buf.WriteString("<CardGrid>\n")
	for _, trigger := range triggers {
		description := strings.TrimSpace(trigger.Subtitle)
		if description == "" {
			description = strings.TrimSpace(trigger.Description)
		}
		buf.WriteString(fmt.Sprintf("  <LinkCard title=\"%s\" href=\"#%s\" description=\"%s\" />\n",
			escapeQuotes(trigger.Title),
			slugify(trigger.Title),
			escapeQuotes(description),
		))
	}
	buf.WriteString("</CardGrid>\n\n")
}

func writeParagraph(buf *bytes.Buffer, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	buf.WriteString(trimmed)
	buf.WriteString("\n\n")
}

func writeOverviewSection(buf *bytes.Buffer, description string) {
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return
	}
	buf.WriteString(trimmed)
	buf.WriteString("\n\n")
}

func writeExampleSection(title string, examplePath string, buf *bytes.Buffer) error {
	trimmed := strings.TrimSpace(examplePath)
	if trimmed == "" {
		return nil
	}

	raw, err := os.ReadFile(trimmed)
	if err != nil {
		return err
	}
	trimmedRaw := strings.TrimSpace(string(raw))
	if trimmedRaw == "" {
		return nil
	}
	buf.WriteString(fmt.Sprintf("### %s\n\n", title))
	buf.WriteString("```json\n")
	buf.WriteString(trimmedRaw)
	buf.WriteString("\n```\n\n")
	return nil
}

func writeExampleUseCasesSection(buf *bytes.Buffer, useCases []exampleUseCase) error {
	if len(useCases) == 0 {
		return nil
	}

	buf.WriteString("### Use Cases\n\n")
	for _, useCase := range useCases {
		title := strings.TrimSpace(useCase.Title)
		if title == "" {
			title = "Use Case"
		}
		buf.WriteString(fmt.Sprintf("#### %s\n\n", title))
		writeParagraph(buf, useCase.Description)

		canvasPath := strings.TrimSpace(useCase.Canvas)
		if canvasPath == "" {
			continue
		}

		screenshotPath, err := generateCanvasScreenshot(canvasPath)
		if err != nil {
			return err
		}

		relative := filepath.ToSlash(strings.TrimPrefix(screenshotPath, "docs/integrations/"))
		if !strings.HasPrefix(relative, ".") {
			relative = "./" + relative
		}

		buf.WriteString("<Tabs>\n")
		buf.WriteString("  <TabItem label=\"UI\">\n\n")
		buf.WriteString(fmt.Sprintf("![%s](%s)\n\n", escapeQuotes(title), relative))
		buf.WriteString("  </TabItem>\n")
		buf.WriteString("  <TabItem label=\"YAML\">\n\n")
		buf.WriteString("```yaml\n")
		if err := writeFileContent(buf, canvasPath); err != nil {
			return err
		}
		buf.WriteString("\n```\n\n")
		buf.WriteString("  </TabItem>\n")
		buf.WriteString("</Tabs>\n\n")
	}

	return nil
}

func writeFileContent(buf *bytes.Buffer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	buf.Write(bytes.TrimRight(data, "\n"))
	return nil
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func slugify(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	snake := strings.ReplaceAll(trimmed, "_", "-")
	withDashes := camelBoundary.ReplaceAllString(snake, "$1-$2")
	withDashes = strings.ReplaceAll(withDashes, " ", "-")
	withDashes = strings.ReplaceAll(withDashes, ".", "-")
	return strings.ToLower(withDashes)
}

func writeImports(buf *bytes.Buffer, includeCards bool, includeTabs bool) {
	if includeCards && includeTabs {
		buf.WriteString("import { CardGrid, LinkCard, Tabs, TabItem } from \"@astrojs/starlight/components\";\n\n")
		return
	}
	if includeCards {
		buf.WriteString("import { CardGrid, LinkCard } from \"@astrojs/starlight/components\";\n\n")
		return
	}
	if includeTabs {
		buf.WriteString("import { Tabs, TabItem } from \"@astrojs/starlight/components\";\n\n")
	}
}

func hasExampleUseCases(index docIndex) bool {
	for _, trigger := range index.Triggers {
		if len(trigger.ExampleUseCases) > 0 {
			return true
		}
	}
	for _, component := range index.Components {
		if len(component.ExampleUseCases) > 0 {
			return true
		}
	}
	return false
}

type screenshotEnv struct {
	runner  *pw.Playwright
	browser pw.Browser
	context pw.BrowserContext
	page    pw.Page
	viteCmd *exec.Cmd
	baseURL string
}

func generateCanvasScreenshot(templatePath string) (string, error) {
	workflow, err := readTemplateWorkflow(templatePath)
	if err != nil {
		return "", err
	}
	waitText := firstNodeName(workflow)

	env := &screenshotEnv{}
	if err := env.start(); err != nil {
		return "", err
	}
	defer env.shutdown()

	orgID, userID, accountID, err := seedUserAndOrganization()
	if err != nil {
		return "", err
	}

	if err := ensureWorkflowNameAvailable(orgID, workflow); err != nil {
		return "", err
	}

	reg := registry.NewRegistry(crypto.NewNoOpEncryptor())
	ctx := authentication.SetUserIdInMetadata(context.Background(), userID.String())
	resp, err := workflows.CreateWorkflow(ctx, reg, orgID.String(), workflow)
	if err != nil {
		return "", err
	}

	workflowID, err := uuid.Parse(resp.Workflow.Metadata.Id)
	if err != nil {
		return "", err
	}

	if err := seedWebhookMetadata(workflowID); err != nil {
		return "", err
	}

	if err := env.addAuthCookie(accountID); err != nil {
		return "", err
	}

	if err := env.openWorkflow(orgID, workflowID, waitText); err != nil {
		return "", err
	}
	if err := env.zoomCanvas(2); err != nil {
		return "", err
	}

	screenshotPath := strings.TrimSuffix(templatePath, filepath.Ext(templatePath)) + ".png"
	if err := os.MkdirAll(filepath.Dir(screenshotPath), 0o755); err != nil {
		return "", err
	}

	if err := env.captureCanvasScreenshot(screenshotPath); err != nil {
		return "", err
	}

	return screenshotPath, nil
}

func (e *screenshotEnv) start() error {
	setScreenshotEnv()

	if err := e.startVite(); err != nil {
		return err
	}
	if err := e.startAppServer(); err != nil {
		return err
	}
	if err := e.startPlaywright(); err != nil {
		return err
	}
	return e.launchBrowser()
}

func (e *screenshotEnv) shutdown() {
	if e.page != nil {
		_ = e.page.Close()
	}
	if e.context != nil {
		_ = e.context.Close()
	}
	if e.browser != nil {
		_ = e.browser.Close()
	}
	if e.runner != nil {
		_ = e.runner.Stop()
	}
	if e.viteCmd != nil && e.viteCmd.Process != nil {
		_ = e.viteCmd.Process.Kill()
	}
}

func (e *screenshotEnv) startPlaywright() error {
	runner, err := pw.Run()
	if err != nil {
		return err
	}
	e.runner = runner
	return nil
}

func (e *screenshotEnv) launchBrowser() error {
	browser, err := e.runner.Chromium.Launch()
	if err != nil {
		return err
	}
	context, err := browser.NewContext(pw.BrowserNewContextOptions{
		Viewport: &pw.Size{
			Width:  1920,
			Height: 1080,
		},
	})
	if err != nil {
		return err
	}
	page, err := context.NewPage()
	if err != nil {
		return err
	}
	e.browser = browser
	e.context = context
	e.page = page
	e.baseURL = os.Getenv("BASE_URL")
	return nil
}

func (e *screenshotEnv) addAuthCookie(accountID uuid.UUID) error {
	secret := os.Getenv("JWT_SECRET")
	signer := spjwt.NewSigner(secret)
	token, err := signer.Generate(accountID.String(), 24*time.Hour)
	if err != nil {
		return err
	}

	return e.context.AddCookies([]pw.OptionalCookie{{
		Name:     "account_token",
		Value:    token,
		URL:      pw.String(e.baseURL + "/"),
		HttpOnly: pw.Bool(true),
	}})
}

func (e *screenshotEnv) startAppServer() error {
	baseURL := os.Getenv("BASE_URL")
	aliveURL := baseURL + "/api/v1/canvases/is-alive"
	if isServerAlive(aliveURL) {
		return nil
	}

	go server.Start()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if isServerAlive(aliveURL) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("server did not become ready at %s", aliveURL)
}

func (e *screenshotEnv) startVite() error {
	if isViteAlive("http://127.0.0.1:5173/") {
		return nil
	}

	cmd := exec.Command("npm", "run", "dev", "--", "--host", "127.0.0.1", "--port", "5173")
	cmd.Dir = "web_src"
	cmd.Env = append(os.Environ(), "BROWSER=none", "API_PORT=8001")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return err
	}
	e.viteCmd = cmd

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if isViteAlive("http://127.0.0.1:5173/") {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("vite failed to start")
}

func (e *screenshotEnv) openWorkflow(orgID uuid.UUID, workflowID uuid.UUID, waitText string) error {
	if _, err := e.page.Goto(
		fmt.Sprintf("%s/%s/workflows/%s", e.baseURL, orgID.String(), workflowID.String()),
		pw.PageGotoOptions{WaitUntil: pw.WaitUntilStateDomcontentloaded},
	); err != nil {
		return err
	}

	trimmed := strings.TrimSpace(waitText)
	if trimmed != "" {
		selector := nodeHeaderSelector(trimmed)
		if err := e.page.Locator(selector).WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(10000)}); err != nil {
			return err
		}
	}
	time.Sleep(800 * time.Millisecond)
	return nil
}

func (e *screenshotEnv) zoomCanvas(steps int) error {
	if steps <= 0 {
		return nil
	}
	if err := e.page.Click(".react-flow"); err != nil {
		return err
	}
	for i := 0; i < steps; i++ {
		if err := e.page.Keyboard().Press("Control+="); err != nil {
			return err
		}
		time.Sleep(350 * time.Millisecond)
	}
	return nil
}

func (e *screenshotEnv) captureCanvasScreenshot(path string) error {
	canvas := e.page.Locator(".react-flow")
	if err := canvas.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(10000)}); err != nil {
		return err
	}
	if err := e.page.Locator(".react-flow__node").First().WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(10000)}); err != nil {
		return err
	}

	clip, err := e.page.Evaluate(`() => {
		const nodes = Array.from(document.querySelectorAll('.react-flow__node'));
		if (nodes.length === 0) {
			return null;
		}
		let minX = Infinity;
		let minY = Infinity;
		let maxX = -Infinity;
		let maxY = -Infinity;
		for (const node of nodes) {
			const rect = node.getBoundingClientRect();
			minX = Math.min(minX, rect.left);
			minY = Math.min(minY, rect.top);
			maxX = Math.max(maxX, rect.right);
			maxY = Math.max(maxY, rect.bottom);
		}
		const padding = 80;
		minX = Math.max(0, minX - padding);
		minY = Math.max(0, minY - padding);
		maxX = Math.min(window.innerWidth, maxX + padding);
		maxY = Math.min(window.innerHeight, maxY + padding);
		return { x: minX, y: minY, width: maxX - minX, height: maxY - minY };
	}`)
	if err != nil {
		return err
	}
	if clip == nil {
		return fmt.Errorf("no canvas nodes found for screenshot crop")
	}

	clipMap, ok := clip.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected clip data")
	}

	x, ok := numberFromMap(clipMap, "x")
	if !ok {
		return fmt.Errorf("invalid clip x")
	}
	y, ok := numberFromMap(clipMap, "y")
	if !ok {
		return fmt.Errorf("invalid clip y")
	}
	width, ok := numberFromMap(clipMap, "width")
	if !ok {
		return fmt.Errorf("invalid clip width")
	}
	height, ok := numberFromMap(clipMap, "height")
	if !ok {
		return fmt.Errorf("invalid clip height")
	}

	if _, err := e.page.Screenshot(pw.PageScreenshotOptions{
		Path: pw.String(path),
		Type: pw.ScreenshotTypePng,
		Clip: &pw.Rect{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		},
	}); err != nil {
		return err
	}

	return nil
}

func numberFromMap(values map[string]any, key string) (float64, bool) {
	raw, ok := values[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch value := raw.(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint32:
		return float64(value), true
	case uint64:
		return float64(value), true
	default:
		return 0, false
	}
}

func readTemplateWorkflow(templatePath string) (*pb.Workflow, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}

	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, err
	}

	var workflow pb.Workflow
	if err := protojson.Unmarshal(jsonData, &workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

func firstNodeName(workflow *pb.Workflow) string {
	if workflow == nil || workflow.Spec == nil {
		return ""
	}
	if len(workflow.Spec.Nodes) == 0 {
		return ""
	}
	return workflow.Spec.Nodes[0].Name
}

func nodeHeaderSelector(nodeName string) string {
	safe := slugify(nodeName)
	return fmt.Sprintf("[data-testid=\"node-%s-header\"]", safe)
}

func seedUserAndOrganization() (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	email := "docs@screenshot.superplane.local"
	name := "Docs Screenshot"
	account, err := models.FindAccountByEmail(email)
	if err != nil {
		account, err = models.CreateAccount(name, email)
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
	}

	orgName := "docs-screenshot-org"
	organization, err := models.FindOrganizationByName(orgName)
	if err != nil {
		organization, err = models.CreateOrganization(orgName, "")
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
	}

	user, err := models.FindMaybeDeletedUserByEmail(organization.ID.String(), email)
	if err != nil {
		user, err = models.CreateUser(organization.ID, account.ID, email, name)
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
	} else if user.DeletedAt.Valid {
		if err := user.Restore(); err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
	}

	if svc, err := authorization.NewAuthService(); err == nil {
		tx := database.Conn().Begin()
		if err := svc.SetupOrganization(tx, organization.ID.String(), user.ID.String()); err != nil {
			tx.Rollback()
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
		if err := tx.Commit().Error; err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, err
		}
	}

	return organization.ID, user.ID, account.ID, nil
}

func ensureWorkflowNameAvailable(orgID uuid.UUID, workflow *pb.Workflow) error {
	if workflow == nil || workflow.Metadata == nil {
		return nil
	}
	name := strings.TrimSpace(workflow.Metadata.Name)
	if name == "" {
		return nil
	}

	existing, err := models.FindWorkflowByName(name, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	return existing.SoftDelete()
}

func seedWebhookMetadata(workflowID uuid.UUID) error {
	webhookMetadata := map[string]any{
		"url":            "https://hooks.superplane.example/webhook",
		"authentication": "none",
	}

	return database.Conn().
		Model(&models.WorkflowNode{}).
		Where("workflow_id = ? AND node_id = ?", workflowID, "webhook-trigger").
		Update("metadata", datatypes.NewJSONType(webhookMetadata)).Error
}

func setScreenshotEnv() {
	os.Setenv("DB_NAME", "superplane_test")
	os.Setenv("START_PUBLIC_API", "yes")
	os.Setenv("START_INTERNAL_API", "yes")
	os.Setenv("INTERNAL_API_PORT", "50052")
	os.Setenv("PUBLIC_API_BASE_PATH", "/api/v1")
	os.Setenv("START_WEB_SERVER", "yes")
	os.Setenv("WEB_BASE_PATH", "")
	os.Setenv("START_GRPC_GATEWAY", "yes")
	os.Setenv("GRPC_SERVER_ADDR", "127.0.0.1:50052")
	os.Setenv("START_EVENT_DISTRIBUTER", "yes")
	os.Setenv("START_CONSUMERS", "yes")
	os.Setenv("START_WORKFLOW_EVENT_ROUTER", "yes")
	os.Setenv("START_WORKFLOW_NODE_EXECUTOR", "yes")
	os.Setenv("START_BLUEPRINT_NODE_EXECUTOR", "yes")
	os.Setenv("START_WORKFLOW_NODE_QUEUE_WORKER", "yes")
	os.Setenv("START_NODE_REQUEST_WORKER", "yes")
	os.Setenv("START_WEBHOOK_PROVISIONER", "yes")
	os.Setenv("START_WEBHOOK_CLEANUP_WORKER", "yes")
	os.Setenv("NO_ENCRYPTION", "yes")
	os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	os.Setenv("JWT_SECRET", "docs-jwt-secret")
	os.Setenv("OIDC_KEYS_PATH", "test/fixtures/oidc-keys")
	os.Setenv("PUBLIC_API_PORT", "8001")
	os.Setenv("BASE_URL", "http://127.0.0.1:8001")
	os.Setenv("WEBHOOKS_BASE_URL", "https://superplane.sxmoon.com")
	os.Setenv("APP_ENV", "development")
	os.Setenv("OWNER_SETUP_ENABLED", "yes")
	os.Setenv("ENABLE_PASSWORD_LOGIN", "yes")
}

func isServerAlive(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return true
}

func isViteAlive(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

func escapeQuotes(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
