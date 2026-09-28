package factories

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type manualTaskIssueTarget struct {
	intake      models.FactoryIntake
	integration *models.Integration
	repository  string
}

func manualTaskGitHubOrigin(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	factory *models.Factory,
	title, description string,
) (models.WorkOrderOrigin, bool) {
	target, err := oldestManualTaskIssueIntake(db, factory)
	if err != nil {
		log.WithError(err).Warnf("factory %s: failed to resolve a GitHub issue for a manual task", factory.ID)
		return models.WorkOrderOrigin{}, false
	}
	if target == nil {
		return models.WorkOrderOrigin{}, false
	}

	origin, err := createManualTaskGitHubIssue(ctx, deps, db, target, title, description)
	if err != nil {
		log.WithError(err).Warnf(
			"factory %s: intake %s: failed to create a GitHub issue in %s for a manual task",
			factory.ID,
			target.intake.ID,
			target.repository,
		)
		return models.WorkOrderOrigin{}, false
	}

	return origin, true
}

func oldestManualTaskIssueIntake(tx *gorm.DB, factory *models.Factory) (*manualTaskIssueTarget, error) {
	intakes, err := factory.ListIntakes(tx)
	if err != nil {
		return nil, err
	}

	githubIntakes := make([]models.FactoryIntake, 0)
	canvasIDs := make([]uuid.UUID, 0)
	for i := range intakes {
		intake := intakes[i]
		if intake.Source != models.FactoryIntakeSourceGitHubIssues || intake.Paused() {
			continue
		}
		githubIntakes = append(githubIntakes, intake)
		canvasIDs = append(canvasIDs, intake.CanvasID)
	}
	if len(githubIntakes) == 0 {
		return nil, nil
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, canvasIDs)
	if err != nil {
		return nil, err
	}

	for i := range githubIntakes {
		intake := githubIntakes[i]
		spec, ok := specs[intake.CanvasID]
		if !ok {
			return nil, fmt.Errorf("intake %s has no live canvas", intake.ID)
		}

		graph := resolveIntakeGraph(intake.Source, spec)
		settings := intakeSettingsFromGraph(intake.Source, graph, spec)
		if !settings.GitHubCreateIssueForManualTasks {
			continue
		}

		trigger, integration, err := resolveLiveIntakeTrigger(tx, &intake)
		if err != nil {
			return nil, fmt.Errorf("intake %s: %w", intake.ID, err)
		}
		repository, _ := trigger.Configuration["repository"].(string)
		repository = strings.TrimSpace(repository)
		if repository == "" {
			return nil, fmt.Errorf("intake %s has no repository", intake.ID)
		}

		return &manualTaskIssueTarget{
			intake:      intake,
			integration: integration,
			repository:  repository,
		}, nil
	}

	return nil, nil
}

func createManualTaskGitHubIssue(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	target *manualTaskIssueTarget,
	title, description string,
) (models.WorkOrderOrigin, error) {
	client, err := newIntakeGitHubClient(deps, db, target.integration)
	if err != nil {
		return models.WorkOrderOrigin{}, err
	}

	request := &github.IssueRequest{Title: github.Ptr(title)}
	if strings.TrimSpace(description) != "" {
		request.Body = github.Ptr(description)
	}

	issue, _, err := client.CreateIssue(ctx, target.repository, request)
	if err != nil {
		return models.WorkOrderOrigin{}, err
	}
	if issue == nil || strings.TrimSpace(issue.GetHTMLURL()) == "" || issue.GetNumber() == 0 {
		return models.WorkOrderOrigin{}, fmt.Errorf("github issue response is missing html_url or number")
	}

	issueURL := strings.TrimSpace(issue.GetHTMLURL())
	return models.WorkOrderOrigin{
		URL:   issueURL,
		Label: models.OriginLabelFromURL(issueURL),
	}, nil
}
