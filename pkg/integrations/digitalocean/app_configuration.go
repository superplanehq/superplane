package digitalocean

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const appPollInterval = 15 * time.Second

// appDeploymentMetadata stores the app and deployment IDs for polling.
type appDeploymentMetadata struct {
	AppID        string `json:"appID" mapstructure:"appID"`
	DeploymentID string `json:"deploymentID" mapstructure:"deploymentID"`
}

// pollDeployment polls a deployment's phase and emits, fails, or reschedules accordingly.
func pollDeployment(ctx core.ActionContext, eventType string) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata appDeploymentMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	deployment, err := client.GetDeployment(metadata.AppID, metadata.DeploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %v", err)
	}

	switch deployment.Phase {
	case "ACTIVE":
		app, err := client.GetApp(metadata.AppID)
		if err != nil {
			return fmt.Errorf("failed to get app: %v", err)
		}

		return emitAppOutput(ctx.ExecutionState, eventType, app)

	case "ERROR":
		message := "deployment failed with phase ERROR"
		if deployment.Cause != "" {
			message = fmt.Sprintf("deployment failed: %s", deployment.Cause)
		}

		if errorSteps := collectErrorSteps(deployment.Progress); len(errorSteps) > 0 {
			message = fmt.Sprintf("%s (failed steps: %s)", message, strings.Join(errorSteps, ", "))
		}

		return ctx.ExecutionState.Fail("deployment_failed", message)

	case "SUPERSEDED":
		return ctx.ExecutionState.Fail("deployment_superseded", "deployment was superseded by a newer deployment")

	case "CANCELED":
		return ctx.ExecutionState.Fail("deployment_canceled", "deployment was canceled")

	case "PENDING_BUILD", "PENDING_DEPLOY", "BUILDING", "DEPLOYING":
		// In-progress phases — keep polling
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{
			"appID":        metadata.AppID,
			"deploymentID": metadata.DeploymentID,
		}, appPollInterval)

	default:
		return fmt.Errorf("deployment reached unexpected phase %q", deployment.Phase)
	}
}

// emitAppOutput emits the standard app output payload.
func emitAppOutput(state core.ExecutionStateContext, eventType string, app *App) error {
	deploymentStatus := ""
	if app.ActiveDeployment != nil {
		deploymentStatus = app.ActiveDeployment.Phase
	}

	name := ""
	if app.Spec != nil {
		name = app.Spec.Name
	}

	return state.Emit(
		core.DefaultOutputChannel.Name,
		eventType,
		[]any{map[string]any{
			"id":               app.ID,
			"name":             name,
			"region":           app.Region,
			"liveURL":          app.LiveURL,
			"defaultIngress":   app.DefaultIngress,
			"deploymentStatus": deploymentStatus,
		}},
	)
}

func collectErrorSteps(progress *DeploymentProgress) []string {
	if progress == nil {
		return nil
	}

	var names []string
	for _, step := range progress.Steps {
		names = append(names, collectErrorStepsRecursive(step)...)
	}

	return names
}

func collectErrorStepsRecursive(step *DeploymentProgressStep) []string {
	if step == nil {
		return nil
	}

	var names []string
	if step.Status == "ERROR" {
		names = append(names, step.Name)
	}

	for _, child := range step.Steps {
		names = append(names, collectErrorStepsRecursive(child)...)
	}

	return names
}

type appFieldVisibility struct {
	ServiceWorkerJob      []configuration.VisibilityCondition
	ServiceOnly           []configuration.VisibilityCondition
	StaticSite            []configuration.VisibilityCondition
	ServiceStaticSite     []configuration.VisibilityCondition
	DeployOnPushTogglable bool
}

func appConfigurationFields(v *appFieldVisibility) []configuration.Field {
	if v == nil {
		v = &appFieldVisibility{}
	}

	fields := []configuration.Field{
		// Deploy on push
		{
			Name:        "deployOnPush",
			Label:       "Deploy on Push",
			Type:        configuration.FieldTypeBool,
			Togglable:   v.DeployOnPushTogglable,
			Default:     true,
			Description: "Automatically deploy when code is pushed to the configured branch",
		},

		// Build & runtime
		{
			Name:        "environmentSlug",
			Label:       "Environment / Buildpack",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "The runtime environment or buildpack (e.g., go, node-js, python, html, ruby, php, hugo, gatsby, dotnet)",
			Placeholder: "node-js",
		},
		{
			Name:        "buildCommand",
			Label:       "Build Command",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "Custom build command to run during the build phase",
			Placeholder: "npm install && npm run build",
		},
		{
			Name:                 "runCommand",
			Label:                "Run Command",
			Type:                 configuration.FieldTypeString,
			Togglable:            true,
			Description:          "Custom run command to start the application",
			Placeholder:          "npm start",
			VisibilityConditions: v.ServiceWorkerJob,
		},
		{
			Name:        "sourceDir",
			Label:       "Source Directory",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "Path to the source code within the repository (useful for monorepos)",
			Placeholder: "/",
		},

		// Instance settings
		{
			Name:                 "httpPort",
			Label:                "HTTP Port",
			Type:                 configuration.FieldTypeNumber,
			Togglable:            true,
			Description:          "The internal port on which the service listens for HTTP traffic",
			Placeholder:          "8080",
			VisibilityConditions: v.ServiceOnly,
		},
		{
			Name:                 "instanceSizeSlug",
			Label:                "Instance Size",
			Type:                 configuration.FieldTypeString,
			Togglable:            true,
			Description:          "The instance size slug (e.g., apps-s-1vcpu-0.5gb, apps-s-1vcpu-1gb, apps-d-1vcpu-0.5gb)",
			Placeholder:          "apps-s-1vcpu-1gb",
			VisibilityConditions: v.ServiceWorkerJob,
		},
		{
			Name:                 "instanceCount",
			Label:                "Instance Count",
			Type:                 configuration.FieldTypeNumber,
			Togglable:            true,
			Description:          "The number of instances to run",
			Default:              1,
			VisibilityConditions: v.ServiceWorkerJob,
		},

		// Static site fields
		{
			Name:                 "outputDir",
			Label:                "Output Directory",
			Type:                 configuration.FieldTypeString,
			Togglable:            true,
			Description:          "The directory where the build output is placed (for static sites)",
			Placeholder:          "build",
			VisibilityConditions: v.StaticSite,
		},
		{
			Name:                 "indexDocument",
			Label:                "Index Document",
			Type:                 configuration.FieldTypeString,
			Togglable:            true,
			Description:          "The name of the index document relative to the root of the site (defaults to index.html)",
			Placeholder:          "index.html",
			VisibilityConditions: v.StaticSite,
		},
		{
			Name:                 "errorDocument",
			Label:                "Error Document",
			Type:                 configuration.FieldTypeString,
			Togglable:            true,
			Description:          "The name of the error document relative to the root of the site (e.g., 404.html)",
			Placeholder:          "404.html",
			VisibilityConditions: v.StaticSite,
		},
		{
			Name:                 "catchallDocument",
			Label:                "Catchall Document",
			Type:                 configuration.FieldTypeString,
			Togglable:            true,
			Description:          "The name of the catchall document for single-page applications (e.g., index.html)",
			Placeholder:          "index.html",
			VisibilityConditions: v.StaticSite,
		},

		// Environment variables
		{
			Name:        "envVars",
			Label:       "Environment Variables",
			Type:        configuration.FieldTypeList,
			Togglable:   true,
			Description: "Environment variables to set for the app (format: KEY=VALUE)",
			Placeholder: "DATABASE_URL=postgres://...",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},

		// Ingress
		{
			Name:                 "ingressPath",
			Label:                "Ingress Path",
			Type:                 configuration.FieldTypeString,
			Togglable:            true,
			Description:          "The path prefix for routing traffic to this component (e.g., /api)",
			Placeholder:          "/",
			VisibilityConditions: v.ServiceStaticSite,
		},
		{
			Name:                 "corsAllowOrigins",
			Label:                "CORS Allow Origins",
			Type:                 configuration.FieldTypeList,
			Togglable:            true,
			Description:          "Allowed origins for Cross-Origin Resource Sharing (e.g., https://example.com)",
			Placeholder:          "https://example.com",
			VisibilityConditions: v.ServiceStaticSite,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Origin",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:                 "corsAllowMethods",
			Label:                "CORS Allow Methods",
			Type:                 configuration.FieldTypeMultiSelect,
			Togglable:            true,
			Description:          "HTTP methods allowed for CORS requests",
			VisibilityConditions: v.ServiceStaticSite,
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET", Value: "GET"},
						{Label: "POST", Value: "POST"},
						{Label: "PUT", Value: "PUT"},
						{Label: "PATCH", Value: "PATCH"},
						{Label: "DELETE", Value: "DELETE"},
						{Label: "HEAD", Value: "HEAD"},
						{Label: "OPTIONS", Value: "OPTIONS"},
					},
				},
			},
		},

		// Database
		{
			Name:        "addDatabase",
			Label:       "Add Database",
			Type:        configuration.FieldTypeBool,
			Default:     false,
			Description: "Attach a database to the app. A dev database is free and suitable for development, while a managed database uses an existing DigitalOcean database cluster.",
		},
		{
			Name:        "databaseName",
			Label:       "Database Component Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The name for the database component (used to reference it in environment variables, e.g., ${db.DATABASE_URL})",
			Placeholder: "db",
			Default:     "db",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "addDatabase", Values: []string{"true"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "addDatabase", Values: []string{"true"}},
			},
		},
		{
			Name:        "databaseEngine",
			Label:       "Database Engine",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "PG",
			Description: "The database engine to use",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "addDatabase", Values: []string{"true"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "addDatabase", Values: []string{"true"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "PostgreSQL", Value: "PG"},
						{Label: "MySQL", Value: "MYSQL"},
						{Label: "Redis", Value: "REDIS"},
						{Label: "MongoDB", Value: "MONGODB"},
					},
				},
			},
		},
		{
			Name:        "databaseVersion",
			Label:       "Database Version",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "The version of the database engine (e.g., 16 for PostgreSQL, 8 for MySQL)",
			Placeholder: "16",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "addDatabase", Values: []string{"true"}},
			},
		},
		{
			Name:        "databaseProduction",
			Label:       "Use Managed Database",
			Type:        configuration.FieldTypeBool,
			Default:     false,
			Description: "When enabled, connects to an existing DigitalOcean Managed Database cluster instead of creating a dev database",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "addDatabase", Values: []string{"true"}},
			},
		},
		{
			Name:        "databaseClusterName",
			Label:       "Database Cluster Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The name of the existing DigitalOcean Managed Database cluster to attach",
			Placeholder: "my-db-cluster",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "addDatabase", Values: []string{"true"}},
				{Field: "databaseProduction", Values: []string{"true"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "databaseProduction", Values: []string{"true"}},
			},
		},
		{
			Name:        "databaseDBName",
			Label:       "Database Name",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "The name of the database within the cluster",
			Placeholder: "my_database",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "addDatabase", Values: []string{"true"}},
				{Field: "databaseProduction", Values: []string{"true"}},
			},
		},
		{
			Name:        "databaseDBUser",
			Label:       "Database User",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "The database user to connect as",
			Placeholder: "app_user",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "addDatabase", Values: []string{"true"}},
				{Field: "databaseProduction", Values: []string{"true"}},
			},
		},

		// VPC
		{
			Name:        "vpcID",
			Label:       "VPC",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "The ID of the VPC to deploy the app into. Apps in a VPC can communicate with other resources in the same VPC over the private network.",
			Placeholder: "5218b393-8cef-41a3-a436-72a20de7cba4",
		},
	}

	return fields
}

func parseEnvVars(envVars []string) []*AppEnvVar {
	var result []*AppEnvVar
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			result = append(result, &AppEnvVar{
				Key:   parts[0],
				Value: parts[1],
				Scope: "RUN_AND_BUILD_TIME",
				Type:  "GENERAL",
			})
		}
	}
	return result
}

func validateDatabaseConfig(addDatabase bool, engine, name, clusterName string, production bool) error {
	if !addDatabase {
		return nil
	}

	if name == "" {
		return errors.New("databaseName is required when addDatabase is enabled")
	}

	if engine == "" {
		return errors.New("databaseEngine is required when addDatabase is enabled")
	}

	switch engine {
	case "PG", "MYSQL", "REDIS", "MONGODB":
		// valid
	default:
		return fmt.Errorf("unsupported database engine: %s", engine)
	}

	if production && clusterName == "" {
		return errors.New("databaseClusterName is required when using a managed database")
	}

	return nil
}

func buildDatabaseConfig(addDatabase bool, name, engine, version string, production bool, clusterName, dbName, dbUser string) *AppDatabase {
	if !addDatabase {
		return nil
	}

	db := &AppDatabase{
		Name:       name,
		Engine:     engine,
		Version:    version,
		Production: production,
	}

	if production {
		db.ClusterName = clusterName
		db.DBName = dbName
		db.DBUser = dbUser
	}

	return db
}

func buildIngressConfig(componentName, ingressPath string, corsOrigins, corsMethods []string) *AppIngress {
	if ingressPath == "" && len(corsOrigins) == 0 && len(corsMethods) == 0 {
		return nil
	}

	rule := &AppIngressRule{
		Component: &AppIngressRuleComponent{
			Name: componentName,
		},
	}

	if ingressPath != "" {
		rule.Match = &AppIngressRuleMatch{
			Path: &AppIngressRuleMatchPath{
				Prefix: ingressPath,
			},
		}
	}

	if len(corsOrigins) > 0 || len(corsMethods) > 0 {
		cors := &AppCORS{}

		for _, origin := range corsOrigins {
			cors.AllowOrigins = append(cors.AllowOrigins, &AppCORSAllowOrigin{
				Exact: origin,
			})
		}

		if len(corsMethods) > 0 {
			cors.AllowMethods = corsMethods
		}

		rule.CORS = cors
	}

	return &AppIngress{
		Rules: []*AppIngressRule{rule},
	}
}

// mergeIngressConfig selectively updates an existing ingress configuration.
// Only fields whose corresponding "set" flag is true are modified;
// the rest are preserved from the existing config.
func mergeIngressConfig(
	existing *AppIngress,
	componentName string,
	ingressPath string, setPath bool,
	corsOrigins []string, setOrigins bool,
	corsMethods []string, setMethods bool,
) *AppIngress {
	// Find or create the rule for this component.
	var rule *AppIngressRule
	if existing != nil {
		for _, r := range existing.Rules {
			if r.Component != nil && r.Component.Name == componentName {
				rule = r
				break
			}
		}
	}

	if rule == nil {
		rule = &AppIngressRule{
			Component: &AppIngressRuleComponent{Name: componentName},
		}
	}

	// Update path if toggled on.
	if setPath {
		if ingressPath != "" {
			rule.Match = &AppIngressRuleMatch{
				Path: &AppIngressRuleMatchPath{Prefix: ingressPath},
			}
		} else {
			rule.Match = nil
		}
	}

	// Update CORS if any CORS field is toggled on.
	if setOrigins || setMethods {
		if rule.CORS == nil {
			rule.CORS = &AppCORS{}
		}

		if setOrigins {
			var origins []*AppCORSAllowOrigin
			for _, origin := range corsOrigins {
				origins = append(origins, &AppCORSAllowOrigin{Exact: origin})
			}
			rule.CORS.AllowOrigins = origins
		}

		if setMethods {
			rule.CORS.AllowMethods = corsMethods
		}

		// If both fields are cleared, remove the CORS object.
		if len(rule.CORS.AllowOrigins) == 0 && len(rule.CORS.AllowMethods) == 0 {
			rule.CORS = nil
		}
	}

	// If the existing ingress had other rules, preserve them.
	if existing != nil {
		var otherRules []*AppIngressRule
		for _, r := range existing.Rules {
			if r.Component == nil || r.Component.Name != componentName {
				otherRules = append(otherRules, r)
			}
		}
		return &AppIngress{Rules: append(otherRules, rule)}
	}

	return &AppIngress{Rules: []*AppIngressRule{rule}}
}

func updateSourceConfig(github *GitHubSource, gitlab *GitLabSource, bitbucket *BitbucketSource, branch string, deployOnPush *bool) {
	if branch != "" {
		if github != nil {
			github.Branch = branch
		}
		if gitlab != nil {
			gitlab.Branch = branch
		}
		if bitbucket != nil {
			bitbucket.Branch = branch
		}
	}

	if deployOnPush != nil {
		if github != nil {
			github.DeployOnPush = *deployOnPush
		}
		if gitlab != nil {
			gitlab.DeployOnPush = *deployOnPush
		}
		if bitbucket != nil {
			bitbucket.DeployOnPush = *deployOnPush
		}
	}
}

// hasConfigKey returns true if the given key is present in the raw configuration map.
// This is used to distinguish between a togglable field that is toggled off (key absent)
// and one that is toggled on with an empty/zero value (key present).
func hasConfigKey(configuration any, key string) bool {
	configMap, ok := configuration.(map[string]any)
	if !ok {
		return false
	}

	_, exists := configMap[key]
	return exists
}

// getDeploymentIDForPolling returns the deployment ID to monitor after a create/update call.
// DigitalOcean may omit pending_deployment for no-op updates, so in-progress is a valid fallback.
func getDeploymentIDForPolling(app *App) string {
	if app == nil {
		return ""
	}

	if app.PendingDeployment.ID != "" {
		return app.PendingDeployment.ID
	}

	if app.InProgressDeployment != nil && app.InProgressDeployment.ID != "" {
		return app.InProgressDeployment.ID
	}

	return ""
}
