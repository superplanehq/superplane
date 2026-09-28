package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const manualTaskGitHubIssueTimeout = 10 * time.Second

type manualTaskIssueTarget struct {
	intake      models.FactoryIntake
	integration *models.Integration
	repository  string
}

type manualTaskIssue struct {
	origin models.WorkOrderOrigin
	target *manualTaskIssueTarget
	number int
}

func manualTaskGitHubOrigin(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	factory *models.Factory,
	title, description string,
) (manualTaskIssue, bool) {
	target, err := oldestManualTaskIssueIntake(db, factory)
	if err != nil {
		log.WithError(err).Warnf("factory %s: failed to resolve a GitHub issue for a manual task", factory.ID)
		return manualTaskIssue{}, false
	}
	if target == nil {
		return manualTaskIssue{}, false
	}

	opened, err := createManualTaskGitHubIssue(ctx, deps, db, target, title, description)
	if err != nil {
		log.WithError(err).Warnf(
			"factory %s: intake %s: failed to create a GitHub issue in %s for a manual task",
			factory.ID,
			target.intake.ID,
			target.repository,
		)
		return manualTaskIssue{}, false
	}

	return opened, true
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
			log.Warnf("factory %s: intake %s has no live canvas; skipping GitHub issue creation", factory.ID, intake.ID)
			continue
		}

		graph := resolveIntakeGraph(intake.Source, spec)
		settings := intakeSettingsFromGraph(intake.Source, graph, spec)
		if !settings.GitHubCreateIssueForManualTasks {
			continue
		}

		trigger, integration, err := resolveLiveIntakeTrigger(tx, &intake)
		if err != nil {
			if errors.Is(err, errIntakeNotConnected) {
				log.WithError(err).Warnf(
					"factory %s: intake %s is not connected; skipping GitHub issue creation",
					factory.ID,
					intake.ID,
				)
				continue
			}
			return nil, fmt.Errorf("intake %s: %w", intake.ID, err)
		}
		repository, _ := trigger.Configuration["repository"].(string)
		repository = strings.TrimSpace(repository)
		if repository == "" {
			log.Warnf("factory %s: intake %s has no repository; skipping GitHub issue creation", factory.ID, intake.ID)
			continue
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
) (manualTaskIssue, error) {
	client, err := newIntakeGitHubClient(deps, db, target.integration)
	if err != nil {
		return manualTaskIssue{}, err
	}

	request := &github.IssueRequest{Title: github.Ptr(title)}
	if strings.TrimSpace(description) != "" {
		request.Body = github.Ptr(description)
	}

	callCtx, cancel := context.WithTimeout(ctx, manualTaskGitHubIssueTimeout)
	defer cancel()
	issue, _, err := client.CreateIssue(callCtx, target.repository, request)
	if err != nil {
		return manualTaskIssue{}, err
	}
	if issue == nil || strings.TrimSpace(issue.GetHTMLURL()) == "" || issue.GetNumber() == 0 {
		return manualTaskIssue{}, fmt.Errorf("github issue response is missing html_url or number")
	}

	issueURL := strings.TrimSpace(issue.GetHTMLURL())
	return manualTaskIssue{
		origin: models.WorkOrderOrigin{
			URL:   issueURL,
			Label: models.OriginLabelFromURL(issueURL),
		},
		target: target,
		number: issue.GetNumber(),
	}, nil
}

func closeManualTaskGitHubIssue(deps IntakeDependencies, db *gorm.DB, opened manualTaskIssue) {
	if opened.target == nil || opened.number == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), manualTaskGitHubIssueTimeout)
	defer cancel()

	client, err := newIntakeGitHubClient(deps, db, opened.target.integration)
	if err != nil {
		log.WithError(err).Warnf(
			"factory %s: intake %s: failed to close GitHub issue %d after the task save failed",
			opened.target.intake.FactoryID,
			opened.target.intake.ID,
			opened.number,
		)
		return
	}

	state := "closed"
	reason := "not_planned"
	_, _, err = client.EditIssue(ctx, opened.target.repository, opened.number, &github.IssueRequest{
		State:       &state,
		StateReason: &reason,
	})
	if err != nil {
		log.WithError(err).Warnf(
			"factory %s: intake %s: failed to close GitHub issue %d after the task save failed",
			opened.target.intake.FactoryID,
			opened.target.intake.ID,
			opened.number,
		)
	}
}
