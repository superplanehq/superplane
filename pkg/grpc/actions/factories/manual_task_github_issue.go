package factories

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	ghintegration "github.com/superplanehq/superplane/pkg/integrations/github"
	ghcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const manualTaskGitHubIssueTimeout = 10 * time.Second
const manualTaskGitHubIssueReconcileTimeout = 3 * time.Second

var errManualTaskGitHubIssueUnconfirmed = errors.New("github issue creation was not confirmed")

type manualTaskIssueTarget struct {
	intake      models.FactoryIntake
	integration *models.Integration
	repository  string
}

func attachManualTaskGitHubIssue(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	target *manualTaskIssueTarget,
	marker, title, description string,
) {
	opened, err := createManualTaskGitHubIssue(ctx, deps, db, target, title, description, marker)
	if err != nil {
		if errors.Is(err, errManualTaskGitHubIssueUnconfirmed) {
			log.WithError(err).Warnf(
				"factory %s: intake %s: GitHub did not confirm the issue for manual task %s",
				factory.ID,
				target.intake.ID,
				order.ID,
			)
			return
		}
		log.WithError(err).Warnf(
			"factory %s: intake %s: failed to create a GitHub issue in %s for a manual task",
			factory.ID,
			target.intake.ID,
			target.repository,
		)
		if clearErr := order.ClearPendingGitHubMarker(db, marker); clearErr != nil {
			log.WithError(clearErr).Warnf("factory %s: failed to clear the pending GitHub marker on task %s", factory.ID, order.ID)
		}
		return
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if lockErr := ghintegration.LockIssueWorkOrder(tx, factory, opened.URL); lockErr != nil {
			return lockErr
		}
		existing, findErr := ghintegration.FindIssueWorkOrder(tx, factory, opened.URL)
		if findErr != nil {
			return findErr
		}
		if existing != nil && existing.ID != order.ID {
			if claimErr := claimManualTaskDetails(tx, existing, order); claimErr != nil {
				return claimErr
			}
		}
		return order.SetOrigin(tx, opened)
	}); err != nil {
		log.WithError(err).Warnf(
			"factory %s: failed to store GitHub issue %s on manual task %s",
			factory.ID,
			opened.URL,
			order.ID,
		)
	}
}

func claimManualTaskDetails(tx *gorm.DB, existing, manual *models.FactoryWorkOrder) error {
	if manual.CreatedByID != nil {
		if err := existing.AssignCreator(tx, *manual.CreatedByID); err != nil {
			return err
		}
		if _, err := existing.AddAssignee(tx, *manual.CreatedByID, *manual.CreatedByID); err != nil {
			return err
		}
	}
	if strings.TrimSpace(manual.Description) != "" && strings.TrimSpace(existing.Description) == "" {
		description := manual.Description
		if err := existing.UpdateContent(tx, nil, &description); err != nil {
			return err
		}
	}
	log.Warnf(
		"factory %s: GitHub issue already has task %s; manual task %s keeps the creator and files",
		manual.FactoryID,
		existing.ID,
		manual.ID,
	)
	return nil
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
	title, description, marker string,
) (models.WorkOrderOrigin, error) {
	client, err := newIntakeGitHubClient(deps, db, target.integration)
	if err != nil {
		return models.WorkOrderOrigin{}, err
	}

	request := &github.IssueRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(ghintegration.AppendManualTaskMarker(description, marker)),
	}

	callCtx, cancel := context.WithTimeout(ctx, manualTaskGitHubIssueTimeout)
	defer cancel()
	issue, _, err := client.CreateIssue(callCtx, target.repository, request)
	if err != nil {
		if !githubIssueCreateOutcomeUnknown(err) {
			return models.WorkOrderOrigin{}, err
		}
		found, findErr := findCreatedManualTaskIssue(client, target.repository, marker)
		if findErr != nil || found == nil {
			if findErr != nil {
				err = fmt.Errorf("%w: %w", err, findErr)
			}
			return models.WorkOrderOrigin{}, fmt.Errorf("%w: %w", errManualTaskGitHubIssueUnconfirmed, err)
		}
		issue = found
	}
	return manualTaskIssueFromGitHub(issue)
}

func findCreatedManualTaskIssue(client *ghcommon.Client, repository, marker string) (*github.Issue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), manualTaskGitHubIssueReconcileTimeout)
	defer cancel()

	issues, _, err := client.ListRecentIssues(ctx, repository, 30)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		if issue == nil || issue.GetNumber() == 0 || issue.PullRequestLinks != nil {
			continue
		}
		if strings.Contains(issue.GetBody(), marker) && strings.TrimSpace(issue.GetHTMLURL()) != "" {
			return issue, nil
		}
	}
	return nil, nil
}

func manualTaskIssueFromGitHub(issue *github.Issue) (models.WorkOrderOrigin, error) {
	if issue == nil || strings.TrimSpace(issue.GetHTMLURL()) == "" || issue.GetNumber() == 0 {
		return models.WorkOrderOrigin{}, fmt.Errorf("github issue response is missing html_url or number")
	}

	issueURL := strings.TrimSpace(issue.GetHTMLURL())
	return models.WorkOrderOrigin{
		URL:   issueURL,
		Label: models.OriginLabelFromURL(issueURL),
	}, nil
}

func githubIssueCreateOutcomeUnknown(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
