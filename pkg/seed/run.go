package seed

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/git"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	canvasespb "github.com/superplanehq/superplane/pkg/protos/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type lineAppSpec struct {
	templateID string
	title      string
	entrypoint string
}

type eventAppSpec struct {
	templateID string
	title      string
}

var onboardingLineApps = []lineAppSpec{
	{templateID: "line-planning", title: "Plan", entrypoint: "onrun-create-plan"},
	{templateID: "line-implementation", title: "Implement", entrypoint: "onrun-implement"},
}

var onboardingEventApps = []eventAppSpec{
	{templateID: "pr-closure", title: "PR Closure"},
	{templateID: "create-with-agent", title: "Create with an Agent"},
}

// Dependencies are the collaborators seed needs. Tests replace GitHub and Claude.
type Dependencies struct {
	Config              Config
	DB                  *gorm.DB
	Registry            *registry.Registry
	Encryptor           crypto.Encryptor
	AuthService         authorization.Authorization
	GitProvider         gitprovider.Provider
	ListInstallations   func(ctx context.Context) ([]common.PendingInstallation, error)
	BindGitHub          func(ctx context.Context, integration *models.Integration, installationID string) error
	VerifyClaude        func(ctx context.Context, apiKey string) error
	ProvisionRepository func(ctx context.Context, repository models.Repository) error
}

// Result is the printed summary after a successful seed.
type Result struct {
	Email             string
	Password          string
	CreatedAccount    bool
	OrganizationSlug  string
	WorkspaceName     string
	WorkspaceKey      string
	GitHubOwner       string
	AppRepository     string
	BacklogRepository string
	LoginURL          string
}

// NewLiveDependencies builds production collaborators from the process environment.
func NewLiveDependencies() (*Dependencies, error) {
	cfg := LoadConfig()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	encryptor := encryptorFromEnv()
	reg, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	if err != nil {
		return nil, err
	}

	authService, err := authorization.NewAuthService()
	if err != nil {
		return nil, err
	}

	gitProvider, err := git.NewProvider()
	if err != nil {
		return nil, err
	}

	provisioner := workers.NewRepositoryProvisionerWorker("", gitProvider)

	return &Dependencies{
		Config:      cfg,
		DB:          database.Conn(),
		Registry:    reg,
		Encryptor:   encryptor,
		AuthService: authService,
		GitProvider: gitProvider,
		ListInstallations: func(ctx context.Context) ([]common.PendingInstallation, error) {
			return github.ListHostedAppInstallations(ctx, cfg.GitHubApp)
		},
		BindGitHub: func(_ context.Context, integration *models.Integration, installationID string) error {
			return bindHostedGitHub(reg, encryptor, integration, cfg.GitHubApp, installationID)
		},
		VerifyClaude:        verifyClaudeAPIKey,
		ProvisionRepository: provisioner.ProvisionRepository,
	}, nil
}

// Run creates or reuses a local owner, GitHub connection, Claude connection,
// and a finished workspace.
func Run(ctx context.Context, deps *Dependencies) (*Result, error) {
	if deps == nil {
		return nil, fmt.Errorf("seed dependencies are required")
	}
	if err := deps.Config.Validate(); err != nil {
		return nil, err
	}
	if deps.DB == nil {
		return nil, fmt.Errorf("database is required")
	}

	seeder := &seeder{deps: deps}
	return seeder.run(ctx)
}

type seeder struct {
	deps *Dependencies
}

func (s *seeder) run(ctx context.Context) (*Result, error) {
	account, user, org, createdAccount, err := s.ensureOwner()
	if err != nil {
		return nil, err
	}

	ctx = authentication.SetUserIdInMetadata(ctx, user.ID.String())

	githubIntegration, err := s.ensureGitHub(ctx, org, user)
	if err != nil {
		return nil, err
	}

	claudeIntegration, err := s.ensureClaude(ctx, org)
	if err != nil {
		return nil, err
	}

	factory, err := s.ensureFactory(org)
	if err != nil {
		return nil, err
	}

	appRepository, backlogRepository, owner, err := s.repositories(githubIntegration)
	if err != nil {
		return nil, err
	}

	if err := s.finishWorkspace(ctx, org, factory, githubIntegration, claudeIntegration, appRepository, backlogRepository); err != nil {
		return nil, err
	}

	factory, err = models.FindFactory(s.deps.DB, org.ID, factory.ID)
	if err != nil {
		return nil, err
	}

	return &Result{
		Email:             account.Email,
		Password:          s.passwordForSummary(createdAccount),
		CreatedAccount:    createdAccount,
		OrganizationSlug:  org.Slug,
		WorkspaceName:     factory.Name,
		WorkspaceKey:      factory.Key,
		GitHubOwner:       owner,
		AppRepository:     appRepository,
		BacklogRepository: backlogRepository,
		LoginURL:          s.deps.Config.BaseURL,
	}, nil
}

func (s *seeder) passwordForSummary(created bool) string {
	if created {
		return s.deps.Config.Password
	}
	return ""
}

func (s *seeder) ensureOwner() (*models.Account, *models.User, *models.Organization, bool, error) {
	cfg := s.deps.Config
	account, err := models.FindAccountByEmail(cfg.Email)
	if err == nil {
		org, user, err := s.orgAndUserForAccount(account)
		if err != nil {
			return nil, nil, nil, false, err
		}
		return account, user, org, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, false, err
	}

	var createdAccount *models.Account
	var createdUser *models.User
	var createdOrg *models.Organization

	err = s.deps.DB.Transaction(func(tx *gorm.DB) error {
		account, err := models.CreateAccountInTransaction(tx, cfg.Name, cfg.Email)
		if err != nil {
			return err
		}
		if err := tx.Model(account).Update("installation_admin", true).Error; err != nil {
			return err
		}
		account.InstallationAdmin = true

		passwordHash, err := crypto.HashPassword(cfg.Password)
		if err != nil {
			return err
		}
		if _, err := models.CreateAccountPasswordAuthInTransaction(tx, account.ID, passwordHash); err != nil {
			return err
		}

		org, err := models.CreateOrganizationInTransaction(tx, cfg.OrganizationName, "")
		if err != nil {
			return err
		}

		user, err := models.CreateUserInTransaction(tx, org.ID, account.ID, cfg.Email, cfg.Name)
		if err != nil {
			return err
		}

		if err := s.deps.AuthService.SetupOrganization(tx, org.ID.String(), user.ID.String()); err != nil {
			return err
		}
		if err := models.SetOrganizationCreatedByAccount(tx, org.ID, account.ID); err != nil {
			return err
		}

		createdAccount = account
		createdUser = user
		createdOrg = org
		return nil
	})
	if err != nil {
		return nil, nil, nil, false, err
	}

	return createdAccount, createdUser, createdOrg, true, nil
}

func (s *seeder) orgAndUserForAccount(account *models.Account) (*models.Organization, *models.User, error) {
	orgs, err := models.ListOrganizationsCreatedByAccount(s.deps.DB, account.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(orgs) == 0 {
		return nil, nil, fmt.Errorf("account %s has no organization", account.Email)
	}
	org := orgs[0]
	user, err := models.FindActiveHumanUserByAccountAndOrganization(s.deps.DB, org.ID, account.ID)
	if err != nil {
		return nil, nil, err
	}
	return &org, user, nil
}

func (s *seeder) ensureGitHub(ctx context.Context, org *models.Organization, user *models.User) (*models.Integration, error) {
	if existing := findReadyIntegration(s.deps.DB, org.ID, "github"); existing != nil {
		return existing, nil
	}
	existing := findIntegrationByApp(s.deps.DB, org.ID, "github")

	installations, err := s.deps.ListInstallations(ctx)
	if err != nil {
		return nil, err
	}
	installation, err := SelectInstallation(installations, s.deps.Config.InstallationID)
	if err != nil {
		if installationHelp := githubInstallHelp(s.deps.Config.GitHubApp.Slug, err); installationHelp != "" {
			return nil, fmt.Errorf("%w\n%s", err, installationHelp)
		}
		return nil, err
	}

	integration := existing
	if integration == nil {
		created, err := models.CreateIntegration(uuid.New(), org.ID, "github", defaultGitHubName, map[string]any{})
		if err != nil {
			return nil, err
		}
		integration = created
	}

	metadata := common.Metadata{
		HostedApp:       true,
		StartedByUserID: user.ID.String(),
		GitHubApp: common.GitHubAppMetadata{
			ID:   s.deps.Config.GitHubApp.ID,
			Slug: s.deps.Config.GitHubApp.Slug,
		},
	}
	if err := writeIntegrationMetadata(s.deps.DB, integration, metadata); err != nil {
		return nil, err
	}

	if err := s.deps.BindGitHub(ctx, integration, installation.ID); err != nil {
		return nil, err
	}

	updated, err := models.FindIntegrationInTransaction(s.deps.DB, org.ID, integration.ID)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *seeder) ensureClaude(ctx context.Context, org *models.Organization) (*models.Integration, error) {
	if existing := findReadyIntegration(s.deps.DB, org.ID, "claude"); existing != nil {
		return existing, nil
	}
	existing := findIntegrationByApp(s.deps.DB, org.ID, "claude")

	if err := s.deps.VerifyClaude(ctx, s.deps.Config.AnthropicAPIKey); err != nil {
		return nil, fmt.Errorf("Claude API key is not valid: %w", err)
	}

	integration := existing
	if integration == nil {
		created, err := models.CreateIntegration(
			uuid.New(),
			org.ID,
			"claude",
			defaultClaudeName,
			map[string]any{},
		)
		if err != nil {
			return nil, err
		}
		integration = created
	}

	encrypted, err := s.deps.Encryptor.Encrypt(ctx, []byte(s.deps.Config.AnthropicAPIKey), []byte(integration.ID.String()))
	if err != nil {
		return nil, err
	}
	integration.Configuration = datatypes.NewJSONType(map[string]any{
		"apiKey": base64.StdEncoding.EncodeToString(encrypted),
	})
	integration.State = models.IntegrationStateReady
	integration.StateDescription = ""
	if err := s.deps.DB.Save(integration).Error; err != nil {
		return nil, err
	}
	return integration, nil
}

func (s *seeder) ensureFactory(org *models.Organization) (*models.Factory, error) {
	factories, err := models.ListFactories(s.deps.DB, org.ID)
	if err != nil {
		return nil, err
	}
	for i := range factories {
		if factories[i].Name == s.deps.Config.WorkspaceName {
			return &factories[i], nil
		}
	}
	return models.CreateFactory(s.deps.DB, org.ID, s.deps.Config.WorkspaceName, "", "")
}

func (s *seeder) repositories(integration *models.Integration) (string, string, string, error) {
	var metadata common.Metadata
	raw, err := json.Marshal(integration.Metadata.Data())
	if err != nil {
		return "", "", "", err
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return "", "", "", err
	}

	appRepository, err := SelectRepository(metadata.Owner, metadata.Repositories, s.deps.Config.AppRepository)
	if err != nil {
		return "", "", "", err
	}
	backlogRepository := s.deps.Config.BacklogRepository
	if backlogRepository == "" {
		backlogRepository = appRepository
	} else {
		backlogRepository = repositoryFullName(metadata.Owner, backlogRepository)
	}
	return appRepository, backlogRepository, metadata.Owner, nil
}

func (s *seeder) finishWorkspace(
	ctx context.Context,
	org *models.Organization,
	factory *models.Factory,
	githubIntegration *models.Integration,
	claudeIntegration *models.Integration,
	appRepository, backlogRepository string,
) error {
	if factory.IsOnboardingComplete() {
		return nil
	}

	issuesSource := models.FactoryOnboardingIssuesSourceVCS
	agentHarness := models.FactoryOnboardingAgentHarnessClaudeCode
	githubID := githubIntegration.ID.String()
	claudeID := claudeIntegration.ID.String()
	defaultBranch := s.deps.Config.DefaultBranch

	if err := factory.UpdateOnboarding(s.deps.DB, models.FactoryOnboardingPatch{
		VCSIntegrationID:   &githubID,
		AgentIntegrationID: &claudeID,
		AppRepository:      &appRepository,
		BacklogRepository:  &backlogRepository,
		DefaultBranch:      &defaultBranch,
		IssuesSource:       &issuesSource,
		AgentHarness:       &agentHarness,
	}); err != nil {
		return err
	}

	lineApps := make([]*models.Canvas, 0, len(onboardingLineApps))
	for _, spec := range onboardingLineApps {
		canvas, err := s.ensureFactoryApp(ctx, org, factory, githubIntegration, claudeIntegration, spec.templateID, spec.title, appRepository, backlogRepository, defaultBranch)
		if err != nil {
			return err
		}
		lineApps = append(lineApps, canvas)
	}

	line, err := s.ensureLine(ctx, org, factory, lineApps)
	if err != nil {
		return err
	}

	for _, spec := range onboardingEventApps {
		if _, err := s.ensureFactoryApp(ctx, org, factory, githubIntegration, claudeIntegration, spec.templateID, spec.title, appRepository, backlogRepository, defaultBranch); err != nil {
			return err
		}
	}

	if err := s.ensureIntake(ctx, org, factory); err != nil {
		return err
	}
	if err := s.ensurePRFeedback(ctx, org, factory, appRepository); err != nil {
		return err
	}

	appID := lineApps[0].ID.String()
	lineID := line.ID.String()
	return factory.CompleteOnboarding(s.deps.DB, models.FactoryOnboardingPatch{
		ProvisionedAppID:  &appID,
		ProvisionedLineID: &lineID,
	})
}

func (s *seeder) ensureFactoryApp(
	ctx context.Context,
	org *models.Organization,
	factory *models.Factory,
	githubIntegration *models.Integration,
	claudeIntegration *models.Integration,
	templateID, title, appRepository, backlogRepository, defaultBranch string,
) (*models.Canvas, error) {
	existing, err := models.FindCanvasByName(s.deps.DB, org.ID, &factory.ID, title)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	created, err := canvases.CreateCanvas(
		ctx,
		s.deps.Registry,
		s.deps.Encryptor,
		s.deps.AuthService,
		s.deps.GitProvider,
		s.deps.Config.WebhooksBaseURL,
		org.ID,
		title,
		"",
		&factory.ID,
		nil,
	)
	if err != nil {
		return nil, err
	}

	canvasID, err := uuid.Parse(created.GetCanvas().GetMetadata().GetId())
	if err != nil {
		return nil, err
	}
	canvas, err := models.FindCanvas(org.ID, canvasID)
	if err != nil {
		return nil, err
	}

	if err := s.provisionCanvasRepo(ctx, canvas); err != nil {
		return nil, err
	}

	materialized, err := factories.MaterializeFactoryAppTemplate(ctx, org.ID.String(), &pb.MaterializeFactoryAppTemplateRequest{
		FactoryId:  factory.ID.String(),
		TemplateId: templateID,
		AppId:      canvas.ID.String(),
		InstallParams: map[string]string{
			"appRepository":     appRepository,
			"backlogRepository": backlogRepository,
			"defaultBranch":     defaultBranch,
		},
		Integrations: []*pb.FactoryAppTemplateIntegration{
			{Type: "github", Id: githubIntegration.ID.String(), Name: githubIntegration.InstallationName},
			{Type: "claude", Id: claudeIntegration.ID.String(), Name: claudeIntegration.InstallationName},
		},
		Agent: &pb.FactoryAppTemplateAgent{
			Component:                 "runnerClaudeCode",
			Model:                     defaultAgentModel,
			PlanningModel:             defaultPlanningModel,
			CredentialSource:          "integration",
			CredentialIntegrationName: claudeIntegration.InstallationName,
		},
	})
	if err != nil {
		return nil, err
	}

	if _, err := canvases.PutCanvasStaging(ctx, s.deps.DB, canvas, []*canvasespb.CanvasRepositoryFileOperation{
		{Path: canvases.CanvasYAMLRepositoryPath, Content: []byte(materialized.GetCanvasYaml())},
		{Path: canvases.ConsoleYAMLRepositoryPath, Content: []byte(materialized.GetConsoleYaml())},
	}); err != nil {
		return nil, err
	}

	if _, err := canvases.CommitCanvasStaging(
		ctx,
		s.deps.DB,
		s.deps.GitProvider,
		nil,
		s.deps.Encryptor,
		s.deps.Registry,
		canvas,
		"Install factory template",
		s.deps.Config.WebhooksBaseURL,
		s.deps.AuthService,
	); err != nil {
		return nil, err
	}

	return models.FindCanvas(org.ID, canvas.ID)
}

func (s *seeder) ensureLine(ctx context.Context, org *models.Organization, factory *models.Factory, lineApps []*models.Canvas) (*models.FactoryLine, error) {
	lines, err := factory.ListLines(s.deps.DB)
	if err != nil {
		return nil, err
	}
	for i := range lines {
		if lines[i].Name == defaultLineName {
			return &lines[i], nil
		}
	}

	steps := make([]*pb.FactoryLine_Step, 0, len(onboardingLineApps))
	for i, spec := range onboardingLineApps {
		steps = append(steps, &pb.FactoryLine_Step{
			Type: models.FactoryLineStepTypeRunApp,
			App: &pb.FactoryLine_AppStep{
				App:        lineApps[i].ID.String(),
				Entrypoint: spec.entrypoint,
			},
		})
	}

	response, err := factories.CreateFactoryLine(ctx, org.ID.String(), &pb.CreateFactoryLineRequest{
		FactoryId: factory.ID.String(),
		Name:      defaultLineName,
		Steps:     steps,
	})
	if err != nil {
		return nil, err
	}

	lineID, err := uuid.Parse(response.GetLine().GetId())
	if err != nil {
		return nil, err
	}
	return factory.FindLine(s.deps.DB, lineID)
}

func (s *seeder) ensureIntake(ctx context.Context, org *models.Organization, factory *models.Factory) error {
	intakes, err := factory.ListIntakes(s.deps.DB)
	if err != nil {
		return err
	}
	for _, intake := range intakes {
		if intake.Source == models.FactoryIntakeSourceGitHubIssues {
			return nil
		}
	}

	_, err = factories.CreateFactoryIntake(ctx, s.intakeDeps(), org.ID.String(), &pb.CreateFactoryIntakeRequest{
		FactoryId: factory.ID.String(),
		Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
	})
	if err != nil {
		return err
	}
	return s.provisionPendingRepos(ctx)
}

func (s *seeder) ensurePRFeedback(ctx context.Context, org *models.Organization, factory *models.Factory, repository string) error {
	handlers, err := factory.ListPRFeedbackHandlers(s.deps.DB)
	if err != nil {
		return err
	}
	if len(handlers) > 0 {
		return nil
	}

	_, err = factories.CreateFactoryPRFeedbackHandler(ctx, s.intakeDeps(), org.ID.String(), &pb.CreateFactoryPRFeedbackHandlerRequest{
		FactoryId: factory.ID.String(),
		Settings: &pb.FactoryPRFeedbackHandler_Settings{
			Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{Repository: repository},
		},
	})
	if err != nil {
		return err
	}
	return s.provisionPendingRepos(ctx)
}

func (s *seeder) intakeDeps() factories.IntakeDependencies {
	return factories.IntakeDependencies{
		Registry:       s.deps.Registry,
		Encryptor:      s.deps.Encryptor,
		AuthService:    s.deps.AuthService,
		GitProvider:    s.deps.GitProvider,
		WebhookBaseURL: s.deps.Config.WebhooksBaseURL,
	}
}

func (s *seeder) provisionCanvasRepo(ctx context.Context, canvas *models.Canvas) error {
	repository, err := models.FindRepositoryUnscoped(canvas.ID)
	if err != nil {
		return err
	}
	if repository.Status == models.RepositoryStatusReady {
		return nil
	}
	return s.deps.ProvisionRepository(ctx, *repository)
}

func (s *seeder) provisionPendingRepos(ctx context.Context) error {
	repositories, err := models.ListPendingRepositories(100)
	if err != nil {
		return err
	}
	for _, repository := range repositories {
		if err := s.deps.ProvisionRepository(ctx, repository); err != nil {
			return err
		}
	}
	return nil
}

func findReadyIntegration(db *gorm.DB, orgID uuid.UUID, appName string) *models.Integration {
	integration := findIntegrationByApp(db, orgID, appName)
	if integration == nil || integration.State != models.IntegrationStateReady {
		return nil
	}
	return integration
}

func findIntegrationByApp(db *gorm.DB, orgID uuid.UUID, appName string) *models.Integration {
	integrations, err := models.ListIntegrations(db, orgID)
	if err != nil {
		return nil
	}
	for i := range integrations {
		if integrations[i].AppName == appName {
			return &integrations[i]
		}
	}
	return nil
}

func writeIntegrationMetadata(db *gorm.DB, integration *models.Integration, metadata common.Metadata) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		return err
	}
	integration.Metadata = datatypes.NewJSONType(asMap)
	return db.Save(integration).Error
}

func bindHostedGitHub(
	reg *registry.Registry,
	encryptor crypto.Encryptor,
	integration *models.Integration,
	app common.HostedApp,
	installationID string,
) error {
	integrationCtx := contexts.NewIntegrationContext(database.Conn(), nil, integration, encryptor, reg, nil)
	metadata := common.Metadata{
		HostedApp: true,
		GitHubApp: common.GitHubAppMetadata{
			ID:   app.ID,
			Slug: app.Slug,
		},
	}
	raw, err := json.Marshal(integration.Metadata.Data())
	if err == nil {
		_ = json.Unmarshal(raw, &metadata)
	}

	g := &github.GitHub{}
	if err := g.BindHostedInstallation(integrationCtx, logrus.NewEntry(logrus.StandardLogger()), metadata, installationID); err != nil {
		return err
	}
	return integrationCtx.Persist()
}

func verifyClaudeAPIKey(ctx context.Context, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Claude rejected the API key (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func encryptorFromEnv() crypto.Encryptor {
	if os.Getenv("NO_ENCRYPTION") == "yes" {
		return crypto.NewNoOpEncryptor()
	}
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		return crypto.NewNoOpEncryptor()
	}
	return crypto.NewAESGCMEncryptor([]byte(key))
}

func githubInstallHelp(slug string, err error) string {
	if !errors.Is(err, ErrNoGitHubInstallations) {
		return ""
	}
	if slug == "" {
		return "Install the GitHub App, then run make seed again."
	}
	return fmt.Sprintf("Install the GitHub App at https://github.com/apps/%s/installations/new, then run make seed again.", slug)
}

// WriteSummary prints login details after a successful seed.
func WriteSummary(w io.Writer, result *Result) {
	fmt.Fprintln(w, "Local environment is ready.")
	fmt.Fprintf(w, "  URL:          %s\n", result.LoginURL)
	fmt.Fprintf(w, "  Email:        %s\n", result.Email)
	if result.CreatedAccount {
		fmt.Fprintf(w, "  Password:     %s\n", result.Password)
	} else {
		fmt.Fprintln(w, "  Password:     (unchanged; this account already existed)")
	}
	fmt.Fprintf(w, "  Organization: %s\n", result.OrganizationSlug)
	fmt.Fprintf(w, "  Workspace:    %s (%s)\n", result.WorkspaceName, result.WorkspaceKey)
	if result.GitHubOwner != "" {
		fmt.Fprintf(w, "  GitHub owner: %s\n", result.GitHubOwner)
	}
	fmt.Fprintf(w, "  App repo:     %s\n", result.AppRepository)
	fmt.Fprintf(w, "  Backlog repo: %s\n", result.BacklogRepository)
}
