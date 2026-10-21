package factories

import (
	"github.com/superplanehq/superplane/pkg/triggers/schedule"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const scheduledIntakeTemplateID = "scheduled-intake"
const scheduledIntakeDefaultPrompt = "Inspect the repository and propose one small, concrete improvement. Verify the problem in the code. Explain what to change, why it matters, and how to verify the result. Do not propose work that is already complete."
const scheduledIntakeOutputInstructions = `

Do not edit repository files, commit, push, or open a pull request.
Write your proposal to /tmp/proposal.json as a JSON object with non-empty string fields "title" and "description".
The title is a short imperative sentence. The description explains the change, the evidence, and how to verify it.
If there is no useful proposal, write exactly {} to /tmp/proposal.json.
Your final chat message is not the structured result.`

func requireScheduledIntake(org *models.Organization) error {
	if !org.HasExperimentalFeature(features.FeatureFactoryScheduledIntake) {
		return grpcerrors.FailedPrecondition(nil, "Scheduled intake is not enabled for this organization.")
	}
	return nil
}

func scheduledIntakeTemplate(provider string) factoryAppTemplate {
	return factoryAppTemplate{
		id: scheduledIntakeTemplateID, provider: provider, entrypointNodeID: "schedule",
		canvasFile:  "templates/github/scheduled-intake.canvas.yaml",
		consoleFile: "templates/github/scheduled-intake.console.yaml",
		columnKey:   models.CanvasColumnKeyBacklog,
	}
}

func scheduledIntakeCanvas(db *gorm.DB, factory *models.Factory, name string, params map[string]string) (*yaml.Canvas, error) {
	config := factory.OnboardingConfigValue()
	if strings.TrimSpace(config.AppRepository) == "" {
		return nil, invalidArgument("Connect a repository in workspace setup before you add scheduled intake.")
	}
	provider := config.EffectiveVCSProvider()
	agent := resolveIntakeAgent(db, factory)
	if agent == nil {
		return nil, invalidArgument("Connect an agent or configure SuperPlane-hosted models before you add scheduled intake.")
	}
	// Repository and credentials are workspace settings, not client install parameters.
	input := factoryTemplateInput{
		appName: name,
		installParams: map[string]string{
			"appRepository": config.AppRepository, "defaultBranch": config.DefaultBranch,
			"hour": params["hour"], "minute": params["minute"], "timezone": params["timezone"], "prompt": params["prompt"],
			"type": params["type"], "cronExpression": params["cronExpression"],
		},
		integrations: map[string]factoryTemplateIntegration{provider: {name: resolveVCSInstallationName(db, factory)}},
		agent:        factoryTemplateAgentFromIntake(agent),
	}
	result, err := materializeFactoryTemplate(scheduledIntakeTemplateID, provider, input)
	if err != nil {
		return nil, err
	}
	return yaml.CanvasFromYAML([]byte(result.canvasYAML))
}

func configureScheduledIntake(canvas *yaml.Canvas, provider string, params map[string]string) error {
	config, err := scheduledIntakeSchedule(params)
	if err != nil {
		return err
	}
	repository := strings.TrimSpace(params["appRepository"])
	if repository == "" {
		return invalidArgument("repository is required for scheduled intake")
	}
	host := strings.TrimPrefix(githubcommon.DotCom().WebURL(), "https://")
	if provider == models.ProviderBitbucket {
		host = "bitbucket.org"
	}
	prompt := strings.TrimSpace(params["prompt"])
	if prompt == "" {
		prompt = scheduledIntakeDefaultPrompt
	}
	for i := range canvas.Spec.Nodes {
		node := &canvas.Spec.Nodes[i]
		if node.ID == "schedule" {
			node.Configuration = map[string]any{"type": config.Type, "timezone": *config.Timezone}
			if config.Type == schedule.TypeCron {
				node.Configuration["cronExpression"] = *config.CronExpression
			} else {
				node.Configuration["daysInterval"], node.Configuration["hour"], node.Configuration["minute"] = 1, *config.Hour, *config.Minute
			}
		}
		if node.ID != "scheduled-agent" {
			continue
		}
		node.Configuration["environmentFrom"] = []any{map[string]any{"source": "integration", "integration": map[string]any{"name": provider}}}
		node.Configuration["environment"] = []any{
			map[string]any{"name": "REPO_URL", "value": "https://" + host + "/" + repository + ".git", "valueSource": "literal"},
			map[string]any{"name": "BASE", "value": params["defaultBranch"], "valueSource": "literal"},
		}
		steps := node.Configuration["steps"].([]any)
		// Checkout is inserted by the runner. This step only clears a proposal from an earlier run.
		steps[0].(map[string]any)["command"] = "set -euo pipefail\nrm -f /tmp/proposal.json"
		steps[1].(map[string]any)["prompt"] = prompt + scheduledIntakeOutputInstructions
	}
	return nil
}

func scheduledIntakeNumber(params map[string]string, key string, fallback, maximum int) (int, error) {
	if strings.TrimSpace(params[key]) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(params[key])
	if err != nil || value < 0 || value > maximum {
		return 0, invalidArgument("invalid scheduled intake " + key)
	}
	return value, nil
}

func scheduledIntakeSchedule(params map[string]string) (schedule.Configuration, error) {
	config := schedule.Configuration{Type: params["type"]}
	if config.Type == "" {
		config.Type = schedule.TypeDays
	}
	if config.Type != schedule.TypeDays && config.Type != schedule.TypeCron {
		return config, invalidArgument("Choose a daily schedule or cron expression.")
	}
	offset := strings.TrimSpace(params["timezone"])
	if offset == "" {
		offset = "0"
	}
	value, err := strconv.ParseFloat(offset, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < -12 || value > 14 {
		return config, invalidArgument("invalid schedule timezone")
	}
	config.Timezone = &offset
	if config.Type == schedule.TypeCron {
		expression := strings.TrimSpace(params["cronExpression"])
		config.CronExpression = &expression
	} else {
		hour, err := scheduledIntakeNumber(params, "hour", 9, 23)
		if err != nil {
			return config, err
		}
		minute, err := scheduledIntakeNumber(params, "minute", 0, 59)
		if err != nil {
			return config, err
		}
		interval := 1
		config.DaysInterval, config.Hour, config.Minute = &interval, &hour, &minute
	}
	if _, err := schedule.NextTrigger(config, time.Now()); err != nil {
		return config, invalidArgument(err.Error())
	}
	return config, nil
}

// The proposal instructions are user-owned; the output contract is managed.
func scheduledIntakePromptFromSteps(raw any) string {
	steps, _ := raw.([]any)
	for _, item := range steps {
		step, _ := item.(map[string]any)
		if step["type"] != "prompt" {
			continue
		}
		prompt, _ := step["prompt"].(string)
		return strings.TrimSuffix(strings.TrimRight(prompt, "\r\n"), scheduledIntakeOutputInstructions)
	}
	return ""
}
