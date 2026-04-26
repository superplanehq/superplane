package templates

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/encoding/protojson"

	// Register the same actions, triggers, and integrations that production uses, so
	// template YAMLs validate like seeding does.
	_ "github.com/superplanehq/superplane/pkg/components/addmemory"
	_ "github.com/superplanehq/superplane/pkg/components/approval"
	_ "github.com/superplanehq/superplane/pkg/components/deletememory"
	_ "github.com/superplanehq/superplane/pkg/components/filter"
	_ "github.com/superplanehq/superplane/pkg/components/graphql"
	_ "github.com/superplanehq/superplane/pkg/components/http"
	_ "github.com/superplanehq/superplane/pkg/components/if"
	_ "github.com/superplanehq/superplane/pkg/components/merge"
	_ "github.com/superplanehq/superplane/pkg/components/noop"
	_ "github.com/superplanehq/superplane/pkg/components/readmemory"
	_ "github.com/superplanehq/superplane/pkg/components/send_email"
	_ "github.com/superplanehq/superplane/pkg/components/ssh"
	_ "github.com/superplanehq/superplane/pkg/components/timegate"
	_ "github.com/superplanehq/superplane/pkg/components/updatememory"
	_ "github.com/superplanehq/superplane/pkg/components/upsertmemory"
	_ "github.com/superplanehq/superplane/pkg/components/wait"
	_ "github.com/superplanehq/superplane/pkg/integrations/aws"
	_ "github.com/superplanehq/superplane/pkg/integrations/azure"
	_ "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	_ "github.com/superplanehq/superplane/pkg/integrations/circleci"
	_ "github.com/superplanehq/superplane/pkg/integrations/claude"
	_ "github.com/superplanehq/superplane/pkg/integrations/cloudflare"
	_ "github.com/superplanehq/superplane/pkg/integrations/cursor"
	_ "github.com/superplanehq/superplane/pkg/integrations/dash0"
	_ "github.com/superplanehq/superplane/pkg/integrations/datadog"
	_ "github.com/superplanehq/superplane/pkg/integrations/daytona"
	_ "github.com/superplanehq/superplane/pkg/integrations/digitalocean"
	_ "github.com/superplanehq/superplane/pkg/integrations/discord"
	_ "github.com/superplanehq/superplane/pkg/integrations/dockerhub"
	_ "github.com/superplanehq/superplane/pkg/integrations/elastic"
	_ "github.com/superplanehq/superplane/pkg/integrations/firehydrant"
	_ "github.com/superplanehq/superplane/pkg/integrations/gcp"
	_ "github.com/superplanehq/superplane/pkg/integrations/github"
	_ "github.com/superplanehq/superplane/pkg/integrations/gitlab"
	_ "github.com/superplanehq/superplane/pkg/integrations/grafana"
	_ "github.com/superplanehq/superplane/pkg/integrations/harness"
	_ "github.com/superplanehq/superplane/pkg/integrations/hetzner"
	_ "github.com/superplanehq/superplane/pkg/integrations/honeycomb"
	_ "github.com/superplanehq/superplane/pkg/integrations/incident"
	_ "github.com/superplanehq/superplane/pkg/integrations/jfrog_artifactory"
	_ "github.com/superplanehq/superplane/pkg/integrations/jira"
	_ "github.com/superplanehq/superplane/pkg/integrations/launchdarkly"
	_ "github.com/superplanehq/superplane/pkg/integrations/logfire"
	_ "github.com/superplanehq/superplane/pkg/integrations/newrelic"
	_ "github.com/superplanehq/superplane/pkg/integrations/oci"
	_ "github.com/superplanehq/superplane/pkg/integrations/octopus"
	_ "github.com/superplanehq/superplane/pkg/integrations/openai"
	_ "github.com/superplanehq/superplane/pkg/integrations/pagerduty"
	_ "github.com/superplanehq/superplane/pkg/integrations/perplexity"
	_ "github.com/superplanehq/superplane/pkg/integrations/prometheus"
	_ "github.com/superplanehq/superplane/pkg/integrations/render"
	_ "github.com/superplanehq/superplane/pkg/integrations/rootly"
	_ "github.com/superplanehq/superplane/pkg/integrations/semaphore"
	_ "github.com/superplanehq/superplane/pkg/integrations/sendgrid"
	_ "github.com/superplanehq/superplane/pkg/integrations/sentry"
	_ "github.com/superplanehq/superplane/pkg/integrations/servicenow"
	_ "github.com/superplanehq/superplane/pkg/integrations/slack"
	_ "github.com/superplanehq/superplane/pkg/integrations/smtp"
	_ "github.com/superplanehq/superplane/pkg/integrations/statuspage"
	_ "github.com/superplanehq/superplane/pkg/integrations/teams"
	_ "github.com/superplanehq/superplane/pkg/integrations/telegram"
	_ "github.com/superplanehq/superplane/pkg/triggers/schedule"
	_ "github.com/superplanehq/superplane/pkg/triggers/start"
	_ "github.com/superplanehq/superplane/pkg/triggers/webhook"
	_ "github.com/superplanehq/superplane/pkg/widgets/annotation"
)

// templateDirForTest returns the repo's templates/canvases directory, independent of
// TEMPLATE_DIR, so CI and local go test can validate the same files as the server
// (which sets TEMPLATE_DIR at deploy time).
func templateDirForTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	return filepath.Join(repoRoot, "templates", "canvases")
}

// TestCanvasesTemplatesParse like SeedTemplates' canvas parsing, ensure every
// built-in template YAML is valid and passes edge and component validation.
func TestCanvasesTemplatesParse(t *testing.T) {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	orgID := models.TemplateOrganizationID.String()

	dir := templateDirForTest(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		jsonData, err := yaml.YAMLToJSON(data)
		if err != nil {
			t.Fatalf("%s: yaml to json: %v", e.Name(), err)
		}
		var canvas pb.Canvas
		if err := protojson.Unmarshal(jsonData, &canvas); err != nil {
			t.Fatalf("%s: protojson: %v", e.Name(), err)
		}
		if canvas.Metadata == nil {
			t.Fatalf("%s: missing metadata", e.Name())
		}
		if canvas.Metadata.Name == "" {
			t.Fatalf("%s: missing name", e.Name())
		}
		canvas.Metadata.IsTemplate = true
		_, _, err = canvases.ParseCanvas(reg, orgID, &canvas)
		if err != nil {
			t.Fatalf("%s: ParseCanvas: %v", e.Name(), err)
		}
	}
}
