package github

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnPackagePublished struct{}

type OnPackagePublishedConfiguration struct {
	Repository   string                    `json:"repository"`
	PackageNames []configuration.Predicate `json:"packageNames"`
	PackageTypes []string                  `json:"packageTypes"`
}

func (p *OnPackagePublished) Name() string {
	return "github.onPackagePublished"
}

func (p *OnPackagePublished) Label() string {
	return "On Package Published"
}

func (p *OnPackagePublished) Description() string {
	return "Listen to GitHub package published events"
}

func (p *OnPackagePublished) Icon() string {
	return "github"
}

func (p *OnPackagePublished) Color() string {
	return "gray"
}

func (p *OnPackagePublished) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "packageNames",
			Label:    "Package Names",
			Type:     configuration.FieldTypeAnyPredicateList,
			Required: true,
			Default: []map[string]any{
				{
					"type":  configuration.PredicateTypeMatches,
					"value": ".*",
				},
			},
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
		{
			Name:     "packageTypes",
			Label:    "Package Types",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "NPM", Value: "npm"},
						{Label: "Docker", Value: "docker"},
						{Label: "Container", Value: "container"},
						{Label: "Maven", Value: "maven"},
						{Label: "RubyGems", Value: "rubygems"},
						{Label: "NuGet", Value: "nuget"},
					},
				},
			},
		},
	}
}

func (p *OnPackagePublished) Setup(ctx core.TriggerContext) error {
	err := ensureRepoInMetadata(
		ctx.MetadataContext,
		ctx.AppInstallationContext,
		ctx.Configuration,
	)

	if err != nil {
		return err
	}

	var config OnPackagePublishedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Register webhook at organization level (empty repository = org-level)
	// We still validate the repository in HandleWebhook for filtering
	return ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		EventType:  "package",
		Repository: "",
	})
}

func (p *OnPackagePublished) Actions() []core.Action {
	return []core.Action{}
}

func (p *OnPackagePublished) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPackagePublished) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	log.Printf("[OnPackagePublished] Received webhook - Event Type: %s", ctx.Headers.Get("X-GitHub-Event"))
	log.Printf("[OnPackagePublished] Body: %s", string(ctx.Body))

	config := OnPackagePublishedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	code, err := verifySignature(ctx, "package")
	if err != nil {
		log.Printf("[OnPackagePublished] Signature verification failed: %v", err)
		return code, err
	}

	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	log.Printf("[OnPackagePublished] Parsed data - action: %v, package: %v", data["action"], data["package"])

	//
	// Filter by repository (since webhook is org-level, we get events from all repos)
	//
	repository, ok := data["repository"]
	if ok {
		if repoMap, ok := repository.(map[string]any); ok {
			if repoName, ok := repoMap["name"].(string); ok {
				if repoName != config.Repository {
					log.Printf("[OnPackagePublished] Repository '%s' does not match configured repository '%s'", repoName, config.Repository)
					return http.StatusOK, nil
				}
				log.Printf("[OnPackagePublished] Repository '%s' matches configured repository", repoName)
			}
		}
	}

	//
	// Check action - only process "published" events
	//
	action, ok := data["action"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing action")
	}

	actionStr, ok := action.(string)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid action")
	}

	if actionStr != "published" {
		log.Printf("[OnPackagePublished] Ignoring action: %s (only 'published' is processed)", actionStr)
		return http.StatusOK, nil
	}

	log.Printf("[OnPackagePublished] Processing published action")

	//
	// Extract package information from package object
	//
	packageObj, ok := data["package"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing package")
	}

	packageData, ok := packageObj.(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid package")
	}

	packageName, ok := packageData["name"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing package name")
	}

	packageNameStr, ok := packageName.(string)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid package name")
	}

	//
	// Filter by package name predicates
	//
	if !configuration.MatchesAnyPredicate(config.PackageNames, packageNameStr) {
		log.Printf("[OnPackagePublished] Package name '%s' does not match predicates", packageNameStr)
		return http.StatusOK, nil
	}

	log.Printf("[OnPackagePublished] Package name '%s' matches predicates", packageNameStr)

	//
	// Filter by package type if specified
	//
	if len(config.PackageTypes) > 0 {
		packageType, ok := packageData["package_type"]
		if !ok {
			return http.StatusBadRequest, fmt.Errorf("missing package_type")
		}

		packageTypeStr, ok := packageType.(string)
		if !ok {
			return http.StatusBadRequest, fmt.Errorf("invalid package_type")
		}

		if !slices.Contains(config.PackageTypes, packageTypeStr) {
			log.Printf("[OnPackagePublished] Package type '%s' not in allowed types: %v", packageTypeStr, config.PackageTypes)
			return http.StatusOK, nil
		}

		log.Printf("[OnPackagePublished] Package type '%s' matches allowed types", packageTypeStr)
	}

	log.Printf("[OnPackagePublished] Emitting github.packagePublished event")
	err = ctx.EventContext.Emit("github.packagePublished", data)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
