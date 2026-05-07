package contents

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/pathfilter"
)

type OnPush struct{}

type OnPushConfiguration struct {
	Repository string                    `json:"repository" mapstructure:"repository"`
	Refs       []configuration.Predicate `json:"refs" mapstructure:"refs"`
	Paths      []string                  `json:"paths" mapstructure:"paths"`
}

func (p *OnPush) Name() string {
	return "github.onPush"
}

func (p *OnPush) Label() string {
	return "On Push"
}

func (p *OnPush) Description() string {
	return "Listen to GitHub push events"
}

func (p *OnPush) Documentation() string {
	return `The On Push trigger starts a workflow execution when code is pushed to a GitHub repository.

## Use Cases

- **CI/CD automation**: Trigger builds and deployments on code pushes
- **Code quality checks**: Run linting and tests on every push
- **Notification workflows**: Send notifications when code is pushed
- **Documentation updates**: Automatically update documentation on push

## Configuration

- **Repository**: Select the GitHub repository to monitor
- **Refs**: Configure which branches/tags to monitor (e.g., ` + "`refs/heads/main`" + `, ` + "`refs/tags/*`" + `)
- **Paths** *(optional)*: Glob patterns (GitHub Actions–style) for added, modified, and removed files. Use ` + "`!`" + ` prefix to exclude (for example ` + "`billing/**`" + ` with ` + "`!billing/**/*.md`" + `). Patterns starting only with ` + "`!`" + ` assume an include of ` + "`**`" + `. Lists stored as legacy ref-style predicates still honor ` + "`equals`" + ` values as globs; ` + "`matches`" + ` must be replaced with glob patterns. If the webhook has no per-commit file lists (empty ` + "`commits`" + ` or missing ` + "`added`" + `/` + "`modified`" + `/` + "`removed`" + ` arrays), a configured path filter cannot match and the trigger will not fire. Leave empty to fire on all pushes.

## Event Data

Each push event includes:
- **repository**: Repository information
- **ref**: The branch or tag that was pushed to
- **commits**: Array of commit information (each with ` + "`added`" + `, ` + "`modified`" + `, ` + "`removed`" + ` file arrays)
- **pusher**: Information about who pushed
- **before/after**: Commit SHAs before and after the push

## Webhook Setup

This trigger automatically sets up a GitHub webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (p *OnPush) Icon() string {
	return "github"
}

func (p *OnPush) Color() string {
	return "gray"
}

func (p *OnPush) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "refs",
			Label:    "Refs",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: true,
			Default: []map[string]any{
				{
					"type":  configuration.PredicateTypeEquals,
					"value": "refs/heads/main",
				},
			},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:        "paths",
			Label:       "Paths",
			Description: "Optional. GitHub Actions–style globs for changed files. Prefix with ! to exclude. Exclude-only lists assume ** (all paths). Leave empty to fire on all pushes.",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Pattern",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (p *OnPush) Setup(ctx core.TriggerContext) error {
	err := common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	config, err := decodeOnPushConfigurationForStruct(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		EventType:  "push",
		Repository: config.Repository,
	})
}

func (p *OnPush) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *OnPush) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPush) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	ctx = common.WithWebhookLogger(ctx, p.Name())
	ctx.Logger.Infof("Received GitHub webhook")

	config, err := decodeOnPushConfigurationForStruct(ctx.Configuration)
	if err != nil {
		ctx.Logger.Errorf("Failed to decode configuration: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		ctx.Logger.Errorf("Missing X-GitHub-Event header")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-GitHub-Event header")
	}

	if eventType != "push" {
		ctx.Logger.Infof("Ignoring event - event type %q is not a push event", eventType)
		return http.StatusOK, nil, nil
	}

	code, err := common.VerifySignature(ctx)
	if err != nil {
		ctx.Logger.Errorf("Failed to verify signature: %v", err)
		return code, nil, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		ctx.Logger.Errorf("Failed to parse request body: %v", err)
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	//
	// If the event is a push event for branch deletion, ignore it.
	//
	if isBranchDeletionEvent(data) {
		ctx.Logger.Info("Ignoring event - branch deletion")
		return http.StatusOK, nil, nil
	}

	ref, ok := data["ref"]
	if !ok {
		ctx.Logger.Errorf("Missing ref")
		return http.StatusBadRequest, nil, fmt.Errorf("missing ref")
	}

	r, ok := ref.(string)
	if !ok {
		ctx.Logger.Errorf("Invalid ref")
		return http.StatusBadRequest, nil, fmt.Errorf("invalid ref")
	}

	if !configuration.MatchesAnyPredicate(config.Refs, r) {
		ctx.Logger.Infof("Ignoring event - ref %q did not match configured filters", r)
		return http.StatusOK, nil, nil
	}

	pathPatterns := onPushPathsFromConfiguration(ctx.Configuration, config.Paths, ctx.Logger)
	pathPatterns = pathfilter.TrimNonEmptyStrings(pathPatterns)
	if len(pathPatterns) > 0 {
		changedFiles := extractChangedFiles(data)
		if !pathfilter.EvaluatePushPathGlobFilter(
			pathPatterns,
			changedFiles,
			func(pat string) {
				ctx.Logger.Warnf("Invalid path glob syntax (skipping pattern) %q", pat)
			},
			func(pat string, err error) {
				ctx.Logger.Warnf("Path glob match error for pattern %q: %v", pat, err)
			},
			func(reason string) {
				ctx.Logger.Warnf("github.onPush paths: %s", reason)
			},
		) {
			if len(changedFiles) == 0 {
				ctx.Logger.Infof("Ignoring event - path filter active but no changed files found in payload")
			} else {
				ctx.Logger.Infof("Ignoring event - no changed file matched path globs (%d patterns, %d file(s))",
					len(pathPatterns), len(changedFiles))
			}
			return http.StatusOK, nil, nil
		}
	}

	err = ctx.Events.Emit("github.push", data)

	if err != nil {
		ctx.Logger.Errorf("Failed to emit event: %v", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func isBranchDeletionEvent(data map[string]any) bool {
	v, ok := data["deleted"]
	if !ok {
		return false
	}

	deleted, ok := v.(bool)
	if !ok {
		return false
	}

	return deleted
}

func (p *OnPush) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// extractChangedFiles collects all added, modified, and removed file paths
// from every commit in the push payload.
func extractChangedFiles(data map[string]any) []string {
	commitsRaw, ok := data["commits"]
	if !ok {
		return nil
	}

	commits, ok := commitsRaw.([]any)
	if !ok {
		return nil
	}

	var files []string
	for _, c := range commits {
		commit, ok := c.(map[string]any)
		if !ok {
			continue
		}

		for _, key := range []string{"added", "modified", "removed"} {
			listRaw, ok := commit[key]
			if !ok {
				continue
			}

			list, ok := listRaw.([]any)
			if !ok {
				continue
			}

			for _, f := range list {
				if path, ok := f.(string); ok {
					files = append(files, path)
				}
			}
		}
	}

	return files
}
