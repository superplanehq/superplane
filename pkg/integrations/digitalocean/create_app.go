package digitalocean

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateApp struct{}

type CreateAppSpec struct {
	Name             string   `json:"name" mapstructure:"name"`
	Region           string   `json:"region" mapstructure:"region"`
	ComponentType    string   `json:"componentType" mapstructure:"componentType"`
	SourceProvider   string   `json:"sourceProvider" mapstructure:"sourceProvider"`
	GitHubRepo       string   `json:"gitHubRepo" mapstructure:"gitHubRepo"`
	GitHubBranch     string   `json:"gitHubBranch" mapstructure:"gitHubBranch"`
	GitLabRepo       string   `json:"gitLabRepo" mapstructure:"gitLabRepo"`
	GitLabBranch     string   `json:"gitLabBranch" mapstructure:"gitLabBranch"`
	BitbucketRepo    string   `json:"bitbucketRepo" mapstructure:"bitbucketRepo"`
	BitbucketBranch  string   `json:"bitbucketBranch" mapstructure:"bitbucketBranch"`
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

func (c *CreateApp) Name() string {
	return "digitalocean.createApp"
}

func (c *CreateApp) Label() string {
	return "Create App"
}

func (c *CreateApp) Description() string {
	return "Create a new DigitalOcean App Platform application with services, static sites, workers, or jobs"
}

func (c *CreateApp) Documentation() string {
	return `The Create App component provisions a new application on DigitalOcean's App Platform from a GitHub, GitLab, or Bitbucket repository.
The component requires that you have connected your Git provider in your DigitalOcean account and granted access to the repository you want to deploy.
You can do so by creating a sample app in the DigitalOcean control panel as illustrated here: https://docs.digitalocean.com/products/app-platform/getting-started/deploy-sample-apps/

## Use Cases

- **Deploy web services**: Provision web services and APIs with configurable instance sizes and HTTP ports
- **Deploy static sites**: Host static websites and single-page applications with custom build and output directories
- **Deploy workers**: Run background workers for processing tasks
- **Deploy jobs**: Run one-off or scheduled jobs (pre-deploy, post-deploy, or failed-deploy)
- **Automated provisioning**: Create app instances as part of infrastructure automation workflows
- **Multi-environment setup**: Deploy separate app instances for dev, staging, and production

## Configuration

- **Name**: The name for the app (required)
- **Region**: The region to deploy the app in (required)
- **Component Type**: The type of component - Service, Static Site, Worker, or Job (required, defaults to Service)
- **Source Provider**: The source code provider - GitHub, GitLab, or Bitbucket (required)
- **Repository**: The repository in owner/repo format (required, shown based on selected provider)
- **Branch**: The branch to deploy from (defaults to "main", shown based on selected provider)
- **Deploy on Push**: Automatically deploy when code is pushed to the branch (default: true)
- **Environment Slug**: The runtime environment/buildpack (e.g., go, node-js, python, html)
- **Build Command**: Custom build command (e.g., npm install && npm run build)
- **Run Command**: Custom run command for services, workers, and jobs (e.g., npm start)
- **Source Directory**: Path to the source code within the repository (defaults to /)
- **HTTP Port**: The port the service listens on (services only)
- **Instance Size**: The instance size slug (e.g., apps-s-1vcpu-1gb) for services, workers, and jobs
- **Instance Count**: Number of instances to run (services, workers, and jobs)
- **Output Directory**: Build output directory for static sites (e.g., build, dist, public)
- **Index Document**: Index document for static sites (defaults to index.html)
- **Error Document**: Custom error document for static sites (e.g., 404.html)
- **Catchall Document**: Catchall document for single-page applications (e.g., index.html)
- **Environment Variables**: Key-value pairs for environment variables (optional)

### Ingress Configuration

- **Ingress Path**: Path prefix for routing traffic to the component (e.g., /api for services, / for static sites)
- **CORS Allow Origins**: Origins allowed for Cross-Origin Resource Sharing (e.g., https://example.com)
- **CORS Allow Methods**: HTTP methods allowed for CORS requests (e.g., GET, POST, PUT)

### Database Configuration

- **Add Database**: Attach a database to the app
- **Database Component Name**: Name used to reference the database in env vars (e.g., ${db.DATABASE_URL})
- **Database Engine**: PostgreSQL, MySQL, Redis, or MongoDB
- **Database Version**: Engine version (e.g., 16 for PostgreSQL)
- **Use Managed Database**: Connect to an existing DigitalOcean Managed Database cluster instead of a dev database
- **Database Cluster Name**: Name of the existing managed database cluster (required for managed databases)
- **Database Name / User**: Optional database name and user for managed database connections

### VPC Configuration

- **VPC**: ID of the VPC to deploy into. Apps in a VPC can communicate with other resources over the private network.

## Output

Returns the created app including:
- **id**: The unique app ID
- **name**: The app name
- **default_ingress**: The default ingress URL
- **live_url**: The live URL for the app
- **region**: The region where the app is deployed
- **active_deployment**: Information about the active deployment

## Notes

- The app will be created with a single component of the selected type
- Deployments are asynchronous and may take several minutes to complete
- The component emits an output once the deployment reaches ACTIVE status
- If the deployment fails, the component will report the failure
- Dev databases are free and suitable for development; use managed databases for production
- Use bindable variables (e.g., ${db.DATABASE_URL}) to reference database connection details in environment variables`
}

func (c *CreateApp) Icon() string {
	return "rocket"
}

func (c *CreateApp) Color() string {
	return "blue"
}

func (c *CreateApp) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateApp) Configuration() []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name for the app",
			Placeholder: "my-app",
		},
		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a region",
			Description: "The region to deploy the app in",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "region",
				},
			},
		},
		{
			Name:        "componentType",
			Label:       "Component Type",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "service",
			Description: "The type of component to create",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Service", Value: "service"},
						{Label: "Static Site", Value: "static-site"},
						{Label: "Worker", Value: "worker"},
						{Label: "Job", Value: "job"},
					},
				},
			},
		},
		{
			Name:        "sourceProvider",
			Label:       "Source Provider",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "github",
			Description: "The source code provider where the repository is hosted. The selected provider must be connected to your DigitalOcean account.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GitHub", Value: "github"},
						{Label: "GitLab", Value: "gitlab"},
						{Label: "Bitbucket", Value: "bitbucket"},
					},
				},
			},
		},
		{
			Name:        "gitHubRepo",
			Label:       "GitHub Repository",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The GitHub repository in owner/repo format. Requires GitHub to be connected in your DigitalOcean account.",
			Placeholder: "owner/repository",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceProvider", Values: []string{"github"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceProvider", Values: []string{"github"}},
			},
		},
		{
			Name:        "gitHubBranch",
			Label:       "GitHub Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The branch to deploy from",
			Placeholder: "main",
			Default:     "main",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceProvider", Values: []string{"github"}},
			},
		},
		{
			Name:        "gitLabRepo",
			Label:       "GitLab Repository",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The GitLab repository in owner/repo format. Requires GitLab to be connected in your DigitalOcean account.",
			Placeholder: "owner/repository",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceProvider", Values: []string{"gitlab"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceProvider", Values: []string{"gitlab"}},
			},
		},
		{
			Name:        "gitLabBranch",
			Label:       "GitLab Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The branch to deploy from",
			Placeholder: "main",
			Default:     "main",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceProvider", Values: []string{"gitlab"}},
			},
		},
		{
			Name:        "bitbucketRepo",
			Label:       "Bitbucket Repository",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The Bitbucket repository in owner/repo format. Requires Bitbucket to be connected in your DigitalOcean account.",
			Placeholder: "owner/repository",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceProvider", Values: []string{"bitbucket"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "sourceProvider", Values: []string{"bitbucket"}},
			},
		},
		{
			Name:        "bitbucketBranch",
			Label:       "Bitbucket Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The branch to deploy from",
			Placeholder: "main",
			Default:     "main",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "sourceProvider", Values: []string{"bitbucket"}},
			},
		},
	}

	fields = append(fields, appConfigurationFields(&appFieldVisibility{
		ServiceWorkerJob: []configuration.VisibilityCondition{
			{Field: "componentType", Values: []string{"service", "worker", "job"}},
		},
		ServiceOnly: []configuration.VisibilityCondition{
			{Field: "componentType", Values: []string{"service"}},
		},
		StaticSite: []configuration.VisibilityCondition{
			{Field: "componentType", Values: []string{"static-site"}},
		},
		ServiceStaticSite: []configuration.VisibilityCondition{
			{Field: "componentType", Values: []string{"service", "static-site"}},
		},
	})...)

	return fields
}

func (c *CreateApp) Setup(ctx core.SetupContext) error {
	spec := CreateAppSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.Region == "" {
		return errors.New("region is required")
	}

	if spec.ComponentType == "" {
		return errors.New("componentType is required")
	}

	switch spec.ComponentType {
	case "service", "static-site", "worker", "job":
		// valid
	default:
		return fmt.Errorf("unsupported component type: %s", spec.ComponentType)
	}

	if spec.SourceProvider == "" {
		return errors.New("sourceProvider is required")
	}

	switch spec.SourceProvider {
	case "github":
		if spec.GitHubRepo == "" {
			return errors.New("gitHubRepo is required when using GitHub as source provider")
		}
	case "gitlab":
		if spec.GitLabRepo == "" {
			return errors.New("gitLabRepo is required when using GitLab as source provider")
		}
	case "bitbucket":
		if spec.BitbucketRepo == "" {
			return errors.New("bitbucketRepo is required when using Bitbucket as source provider")
		}
	default:
		return fmt.Errorf("unsupported source provider: %s", spec.SourceProvider)
	}

	// Validate database configuration
	return validateDatabaseConfig(spec.AddDatabase, spec.DatabaseEngine, spec.DatabaseName, spec.DatabaseClusterName, spec.DatabaseProduction)
}

func (c *CreateApp) Execute(ctx core.ExecutionContext) error {
	spec := CreateAppSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Build environment variables
	envVars := parseEnvVars(spec.EnvVars)

	// Resolve the source configuration
	github, gitlab, bitbucket := buildSourceConfig(spec)

	// Build the app spec based on component type
	appSpec := &AppSpec{
		Name:   spec.Name,
		Region: spec.Region,
	}

	componentType := spec.ComponentType
	if componentType == "" {
		componentType = "service"
	}

	switch componentType {
	case "service":
		service := &AppService{
			Name:            spec.Name,
			GitHub:          github,
			GitLab:          gitlab,
			Bitbucket:       bitbucket,
			EnvironmentSlug: spec.EnvironmentSlug,
			EnvVariables:    envVars,
			BuildCommand:    spec.BuildCommand,
			RunCommand:      spec.RunCommand,
			SourceDir:       spec.SourceDir,
			HTTPPort:        spec.HTTPPort,
		}

		if spec.InstanceSizeSlug != "" {
			service.InstanceSizeSlug = spec.InstanceSizeSlug
		}

		if spec.InstanceCount > 0 {
			service.InstanceCount = spec.InstanceCount
		} else {
			service.InstanceCount = 1
		}

		appSpec.Services = []*AppService{service}

	case "static-site":
		staticSite := &AppStaticSite{
			Name:             spec.Name,
			GitHub:           github,
			GitLab:           gitlab,
			Bitbucket:        bitbucket,
			EnvironmentSlug:  spec.EnvironmentSlug,
			EnvVariables:     envVars,
			BuildCommand:     spec.BuildCommand,
			SourceDir:        spec.SourceDir,
			OutputDir:        spec.OutputDir,
			IndexDocument:    spec.IndexDocument,
			ErrorDocument:    spec.ErrorDocument,
			CatchallDocument: spec.CatchallDocument,
		}

		appSpec.StaticSites = []*AppStaticSite{staticSite}

	case "worker":
		worker := &AppWorker{
			Name:            spec.Name,
			GitHub:          github,
			GitLab:          gitlab,
			Bitbucket:       bitbucket,
			EnvironmentSlug: spec.EnvironmentSlug,
			EnvVariables:    envVars,
			BuildCommand:    spec.BuildCommand,
			RunCommand:      spec.RunCommand,
			SourceDir:       spec.SourceDir,
		}

		if spec.InstanceSizeSlug != "" {
			worker.InstanceSizeSlug = spec.InstanceSizeSlug
		}

		if spec.InstanceCount > 0 {
			worker.InstanceCount = spec.InstanceCount
		} else {
			worker.InstanceCount = 1
		}

		appSpec.Workers = []*AppWorker{worker}

	case "job":
		job := &AppJob{
			Name:            spec.Name,
			GitHub:          github,
			GitLab:          gitlab,
			Bitbucket:       bitbucket,
			EnvironmentSlug: spec.EnvironmentSlug,
			EnvVariables:    envVars,
			BuildCommand:    spec.BuildCommand,
			RunCommand:      spec.RunCommand,
			SourceDir:       spec.SourceDir,
		}

		if spec.InstanceSizeSlug != "" {
			job.InstanceSizeSlug = spec.InstanceSizeSlug
		}

		if spec.InstanceCount > 0 {
			job.InstanceCount = spec.InstanceCount
		} else {
			job.InstanceCount = 1
		}

		appSpec.Jobs = []*AppJob{job}
	}

	// Build ingress rules if configured
	if ingress := buildIngressConfig(spec.Name, spec.IngressPath, spec.CORSAllowOrigins, spec.CORSAllowMethods); ingress != nil {
		appSpec.Ingress = ingress
	}

	// Build database configuration
	if db := buildDatabaseConfig(spec.AddDatabase, spec.DatabaseName, spec.DatabaseEngine, spec.DatabaseVersion, spec.DatabaseProduction, spec.DatabaseClusterName, spec.DatabaseDBName, spec.DatabaseDBUser); db != nil {
		appSpec.Databases = []*AppDatabase{db}
	}

	// Set VPC if configured
	if spec.VPCID != "" {
		appSpec.VPC = &AppVPC{ID: spec.VPCID}
	}

	// Create the app
	app, err := client.CreateApp(CreateAppRequest{Spec: appSpec})
	if err != nil {
		return fmt.Errorf("failed to create app: %v", err)
	}

	deploymentID := getDeploymentIDForPolling(app)
	if deploymentID == "" {
		// Defensive fallback in case the API response has no pending/in-progress deployment.
		return emitAppOutput(ctx.ExecutionState, "digitalocean.app.created", app)
	}

	// Store the app and deployment IDs for polling
	err = ctx.Metadata.Set(appDeploymentMetadata{
		AppID:        app.ID,
		DeploymentID: deploymentID,
	})
	if err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	// Schedule the first poll
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, appPollInterval)
}

// buildSourceConfig resolves the source provider configuration from the spec.
func buildSourceConfig(spec CreateAppSpec) (*GitHubSource, *GitLabSource, *BitbucketSource) {
	deployOnPush := true
	if spec.DeployOnPush != nil {
		deployOnPush = *spec.DeployOnPush
	}

	switch spec.SourceProvider {
	case "github":
		branch := spec.GitHubBranch
		if branch == "" {
			branch = "main"
		}
		return &GitHubSource{
			Repo:         spec.GitHubRepo,
			Branch:       branch,
			DeployOnPush: deployOnPush,
		}, nil, nil

	case "gitlab":
		branch := spec.GitLabBranch
		if branch == "" {
			branch = "main"
		}
		return nil, &GitLabSource{
			Repo:         spec.GitLabRepo,
			Branch:       branch,
			DeployOnPush: deployOnPush,
		}, nil

	case "bitbucket":
		branch := spec.BitbucketBranch
		if branch == "" {
			branch = "main"
		}
		return nil, nil, &BitbucketSource{
			Repo:         spec.BitbucketRepo,
			Branch:       branch,
			DeployOnPush: deployOnPush,
		}
	}

	return nil, nil, nil
}

func (c *CreateApp) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateApp) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateApp) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CreateApp) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return pollDeployment(ctx, "digitalocean.app.created")
}

func (c *CreateApp) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateApp) Cleanup(ctx core.SetupContext) error {
	return nil
}
