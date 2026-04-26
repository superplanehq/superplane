package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/templates"

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

func main() {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		exitWithError(err)
	}

	dir := filepath.Join("templates", "canvases")
	if err := templates.ValidateCanvasTemplates(reg, dir); err != nil {
		exitWithError(err)
	}
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
