package digitalocean

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateApp struct{}

type UpdateAppSpec struct {
	App              string   `json:"app" mapstructure:"app"`
	Name             string   `json:"name" mapstructure:"name"`
	Region           string   `json:"region" mapstructure:"region"`
	Branch           string   `json:"branch" mapstructure:"branch"`
	DeployOnPush     *bool    `json:"deployOnPush" mapstructure:"deployOnPush"`
	EnvironmentSlug  string   `json:"environmentSlug" mapstructure:"environmentSlug"`
	BuildCommand     string   `json:"buildCommand" mapstructure:"buildCommand"`
	RunCommand       string   `json:"runCommand" mapstructure:"runCommand"`
	SourceDir        string   `json:"sourceDir" mapstructure:"sourceDir"`
	HTTPPort         int64    `json:"httpPort" mapstructure:"httpPort"`
	InstanceSizeSlug string   `json:"instanceSizeSlug" mapstructure:"instanceSizeSlug"`
	InstanceCount    int64    `json:"instanceCount" mapstructure:"instanceCount"`
	OutputDir        string   `json:"outputDir" mapstructure:"outputDir"`
	IndexDocument    string   `json:"indexDocument" mapstructure:"indexDocument"`
	ErrorDocument    string   `json:"errorDocument" mapstructure:"errorDocument"`
	CatchallDocument string   `json:"catchallDocument" mapstructure:"catchallDocument"`
	EnvVars          []string `json:"envVars" mapstructure:"envVars"`

	// Ingress configuration
	IngressPath      string   `json:"ingressPath" mapstructure:"ingressPath"`
	CORSAllowOrigins []string `json:"corsAllowOrigins" mapstructure:"corsAllowOrigins"`
	CORSAllowMethods []string `json:"corsAllowMethods" mapstructure:"corsAllowMethods"`

	// Database configuration
	AddDatabase         bool   `json:"addDatabase" mapstructure:"addDatabase"`
	DatabaseName        string `json:"databaseName" mapstructure:"databaseName"`
	DatabaseEngine      string `json:"databaseEngine" mapstructure:"databaseEngine"`
	DatabaseVersion     string `json:"databaseVersion" mapstructure:"databaseVersion"`
	DatabaseProduction  bool   `json:"databaseProduction" mapstructure:"databaseProduction"`
	DatabaseClusterName string `json:"databaseClusterName" mapstructure:"databaseClusterName"`
	DatabaseDBName      string `json:"databaseDBName" mapstructure:"databaseDBName"`
	DatabaseDBUser      string `json:"databaseDBUser" mapstructure:"databaseDBUser"`

	// VPC configuration
	VPCID string `json:"vpcID" mapstructure:"vpcID"`
}

func (u *UpdateApp) Name() string {
	return "digitalocean.updateApp"
}

func (u *UpdateApp) Label() string {
	return "Update App"
}

func (u *UpdateApp) Description() string {
	return "Update a DigitalOcean App Platform application's configuration, environment, and deployment settings"
}

func (u *UpdateApp) Documentation() string {
	return `The Update App component modifies an existing DigitalOcean App Platform application.

## Use Cases

- **Update configuration**: Change app settings like environment variables, branch, build commands, and more
- **Rename apps**: Update the app name
- **Migrate regions**: Move the app to a different region
- **Inject secrets**: Add or update environment variables such as database connection strings
- **Switch branches**: Change the deployed branch without recreating the app
- **Scale resources**: Adjust instance size and count for services, workers, and jobs
- **Configure networking**: Update ingress paths, CORS settings, and VPC connections
- **Manage databases**: Add or update database attachments (dev or managed)

## Configuration

- **App**: The app to update (required)
- **Name**: Update the app name (optional)
- **Region**: Update the region the app is deployed in (optional)
- **Branch**: The branch to deploy from (optional, applies to all components' source providers)
- **Deploy on Push**: Toggle automatic deployment when code is pushed to the branch
- **Environment Slug**: Update the runtime environment/buildpack
- **Build Command**: Update the build command
- **Run Command**: Update the run command (services, workers, jobs)
- **Source Directory**: Update the source directory path
- **HTTP Port**: Update the service listening port
- **Instance Size**: Update the instance size slug
- **Instance Count**: Update the number of instances
- **Output Directory**: Update the static site output directory
- **Index/Error/Catchall Document**: Update static site document settings
- **Environment Variables**: Key-value pairs to add or update (merges with existing)

### Ingress Configuration

- **Ingress Path**: Update the path prefix for routing traffic
- **CORS Allow Origins**: Update allowed origins for Cross-Origin Resource Sharing
- **CORS Allow Methods**: Update HTTP methods allowed for CORS requests

### Database Configuration

- **Add Database**: Attach a new database to the app
- **Database Component Name, Engine, Version**: Configure the database
- **Use Managed Database**: Connect to an existing managed database cluster

### VPC Configuration

- **VPC**: Update the VPC ID for the app

## Output

Returns the updated app including:
- **id**: The unique app ID
- **name**: The app name
- **region**: The region where the app is deployed
- **live_url**: The live URL for the app
- **default_ingress**: The default ingress URL
- **active_deployment**: Information about the updated deployment

## Notes

- Environment variables are merged with existing ones (not replaced)
- Build/runtime settings are applied to all matching components
- Updating an app triggers a new deployment
- The component emits an output once the deployment reaches ACTIVE status
- If the deployment fails, the component will report the failure
- Dev databases are free and suitable for development; use managed databases for production`
}

func (u *UpdateApp) Icon() string {
	return "refresh-cw"
}

func (u *UpdateApp) Color() string {
	return "blue"
}

func (u *UpdateApp) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateApp) Configuration() []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "app",
			Label:       "App",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select an app",
			Description: "The app to update",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "app",
				},
			},
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "Update the app name",
			Placeholder: "my-app",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Togglable:   true,
			Placeholder: "Select a region",
			Description: "Update the region the app is deployed in",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "region",
				},
			},
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Togglable:   true,
			Description: "The branch to deploy from (applies to all components' source providers)",
			Placeholder: "main",
		},
	}

	fields = append(fields, appConfigurationFields(&appFieldVisibility{
		DeployOnPushTogglable: true,
	})...)

	return fields
}

func (u *UpdateApp) Setup(ctx core.SetupContext) error {
	spec := UpdateAppSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.App == "" {
		return errors.New("app is required")
	}

	// Validate database configuration
	if err := validateDatabaseConfig(spec.AddDatabase, spec.DatabaseEngine, spec.DatabaseName, spec.DatabaseClusterName, spec.DatabaseProduction); err != nil {
		return err
	}

	// Resolve metadata for UI display
	err = resolveAppMetadata(ctx, spec.App)
	if err != nil {
		return fmt.Errorf("error resolving app metadata: %v", err)
	}

	return nil
}

func (u *UpdateApp) Execute(ctx core.ExecutionContext) error {
	spec := UpdateAppSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	hasKey := func(key string) bool {
		return hasConfigKey(ctx.Configuration, key)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	app, err := client.GetApp(spec.App)
	if err != nil {
		return fmt.Errorf("failed to get app: %v", err)
	}

	if app.Spec == nil {
		return fmt.Errorf("app %s has no spec", spec.App)
	}

	updatedSpec := app.Spec

	if hasKey("name") {
		updatedSpec.Name = spec.Name
	}
	if hasKey("region") {
		updatedSpec.Region = spec.Region
	}

	newEnvVars := make(map[string]string)
	for _, envVar := range spec.EnvVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			newEnvVars[parts[0]] = parts[1]
		}
	}

	// Helper function to merge environment variables
	mergeEnvVars := func(existing []*AppEnvVar) []*AppEnvVar {
		envMap := make(map[string]*AppEnvVar)
		for _, env := range existing {
			envMap[env.Key] = env
		}

		for key, value := range newEnvVars {
			envMap[key] = &AppEnvVar{
				Key:   key,
				Value: value,
				Scope: "RUN_AND_BUILD_TIME",
				Type:  "GENERAL",
			}
		}

		result := make([]*AppEnvVar, 0, len(envMap))
		for _, env := range envMap {
			result = append(result, env)
		}
		return result
	}

	// Update services
	for i, service := range updatedSpec.Services {
		updateSourceConfig(service.GitHub, service.GitLab, service.Bitbucket, spec.Branch, spec.DeployOnPush)

		if hasKey("environmentSlug") {
			updatedSpec.Services[i].EnvironmentSlug = spec.EnvironmentSlug
		}
		if hasKey("buildCommand") {
			updatedSpec.Services[i].BuildCommand = spec.BuildCommand
		}
		if hasKey("runCommand") {
			updatedSpec.Services[i].RunCommand = spec.RunCommand
		}
		if hasKey("sourceDir") {
			updatedSpec.Services[i].SourceDir = spec.SourceDir
		}
		if hasKey("httpPort") {
			updatedSpec.Services[i].HTTPPort = spec.HTTPPort
		}
		if hasKey("instanceSizeSlug") {
			updatedSpec.Services[i].InstanceSizeSlug = spec.InstanceSizeSlug
		}
		if hasKey("instanceCount") {
			updatedSpec.Services[i].InstanceCount = spec.InstanceCount
		}
		if len(newEnvVars) > 0 {
			updatedSpec.Services[i].EnvVariables = mergeEnvVars(service.EnvVariables)
		}
	}

	// Update workers
	for i, worker := range updatedSpec.Workers {
		updateSourceConfig(worker.GitHub, worker.GitLab, worker.Bitbucket, spec.Branch, spec.DeployOnPush)

		if hasKey("environmentSlug") {
			updatedSpec.Workers[i].EnvironmentSlug = spec.EnvironmentSlug
		}
		if hasKey("buildCommand") {
			updatedSpec.Workers[i].BuildCommand = spec.BuildCommand
		}
		if hasKey("runCommand") {
			updatedSpec.Workers[i].RunCommand = spec.RunCommand
		}
		if hasKey("sourceDir") {
			updatedSpec.Workers[i].SourceDir = spec.SourceDir
		}
		if hasKey("instanceSizeSlug") {
			updatedSpec.Workers[i].InstanceSizeSlug = spec.InstanceSizeSlug
		}
		if hasKey("instanceCount") {
			updatedSpec.Workers[i].InstanceCount = spec.InstanceCount
		}
		if len(newEnvVars) > 0 {
			updatedSpec.Workers[i].EnvVariables = mergeEnvVars(worker.EnvVariables)
		}
	}

	// Update jobs
	for i, job := range updatedSpec.Jobs {
		updateSourceConfig(job.GitHub, job.GitLab, job.Bitbucket, spec.Branch, spec.DeployOnPush)

		if hasKey("environmentSlug") {
			updatedSpec.Jobs[i].EnvironmentSlug = spec.EnvironmentSlug
		}
		if hasKey("buildCommand") {
			updatedSpec.Jobs[i].BuildCommand = spec.BuildCommand
		}
		if hasKey("runCommand") {
			updatedSpec.Jobs[i].RunCommand = spec.RunCommand
		}
		if hasKey("sourceDir") {
			updatedSpec.Jobs[i].SourceDir = spec.SourceDir
		}
		if hasKey("instanceSizeSlug") {
			updatedSpec.Jobs[i].InstanceSizeSlug = spec.InstanceSizeSlug
		}
		if hasKey("instanceCount") {
			updatedSpec.Jobs[i].InstanceCount = spec.InstanceCount
		}
		if len(newEnvVars) > 0 {
			updatedSpec.Jobs[i].EnvVariables = mergeEnvVars(job.EnvVariables)
		}
	}

	// Update static sites
	for i, site := range updatedSpec.StaticSites {
		updateSourceConfig(site.GitHub, site.GitLab, site.Bitbucket, spec.Branch, spec.DeployOnPush)

		if hasKey("environmentSlug") {
			updatedSpec.StaticSites[i].EnvironmentSlug = spec.EnvironmentSlug
		}
		if hasKey("buildCommand") {
			updatedSpec.StaticSites[i].BuildCommand = spec.BuildCommand
		}
		if hasKey("sourceDir") {
			updatedSpec.StaticSites[i].SourceDir = spec.SourceDir
		}
		if hasKey("outputDir") {
			updatedSpec.StaticSites[i].OutputDir = spec.OutputDir
		}
		if hasKey("indexDocument") {
			updatedSpec.StaticSites[i].IndexDocument = spec.IndexDocument
		}
		if hasKey("errorDocument") {
			updatedSpec.StaticSites[i].ErrorDocument = spec.ErrorDocument
		}
		if hasKey("catchallDocument") {
			updatedSpec.StaticSites[i].CatchallDocument = spec.CatchallDocument
		}
		if len(newEnvVars) > 0 {
			updatedSpec.StaticSites[i].EnvVariables = mergeEnvVars(site.EnvVariables)
		}
	}

	// Update ingress if any ingress-related fields are toggled on.
	// Merge with existing ingress to avoid losing fields that weren't toggled.
	if hasKey("ingressPath") || hasKey("corsAllowOrigins") || hasKey("corsAllowMethods") {
		componentName := ""
		if len(updatedSpec.Services) > 0 {
			componentName = updatedSpec.Services[0].Name
		} else if len(updatedSpec.StaticSites) > 0 {
			componentName = updatedSpec.StaticSites[0].Name
		}

		if componentName != "" {
			updatedSpec.Ingress = mergeIngressConfig(
				updatedSpec.Ingress,
				componentName,
				spec.IngressPath, hasKey("ingressPath"),
				spec.CORSAllowOrigins, hasKey("corsAllowOrigins"),
				spec.CORSAllowMethods, hasKey("corsAllowMethods"),
			)
		}
	}

	// Update database if configured
	if db := buildDatabaseConfig(spec.AddDatabase, spec.DatabaseName, spec.DatabaseEngine, spec.DatabaseVersion, spec.DatabaseProduction, spec.DatabaseClusterName, spec.DatabaseDBName, spec.DatabaseDBUser); db != nil {
		updatedSpec.Databases = []*AppDatabase{db}
	}

	// Update VPC if configured
	if hasKey("vpcID") {
		updatedSpec.VPC = &AppVPC{ID: spec.VPCID}
	}

	// Update the app
	updatedApp, err := client.UpdateApp(spec.App, UpdateAppRequest{
		Spec: updatedSpec,
	})
	if err != nil {
		return fmt.Errorf("failed to update app: %v", err)
	}

	deploymentID := getDeploymentIDForPolling(updatedApp)
	if deploymentID == "" {
		// No new deployment was created (for example, a no-op update), so emit the current app state immediately.
		return emitAppOutput(ctx.ExecutionState, "digitalocean.app.updated", updatedApp)
	}

	// Store the app and deployment IDs for polling
	err = ctx.Metadata.Set(appDeploymentMetadata{
		AppID:        updatedApp.ID,
		DeploymentID: deploymentID,
	})
	if err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	// Schedule the first poll
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, appPollInterval)
}

func (u *UpdateApp) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateApp) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateApp) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (u *UpdateApp) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return pollDeployment(ctx, "digitalocean.app.updated")
}

func (u *UpdateApp) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (u *UpdateApp) Cleanup(ctx core.SetupContext) error {
	return nil
}
