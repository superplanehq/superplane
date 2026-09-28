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
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	ghintegration "github.com/superplanehq/superplane/pkg/integrations/github"
	ghcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const manualTaskGitHubIssueTimeout = 10 * time.Second
const manualTaskGitHubIssueReconcileTimeout = 8 * time.Second
const manualTaskGitHubActorTimeout = 3 * time.Second
const manualTaskGitHubIssueListLimit = 100

var manualTaskGitHubIssueReconcileAttempts = 4
var manualTaskGitHubIssueReconcileDelay = 200 * time.Millisecond
var runManualTaskIssueBackgroundReconcile = true
var manualTaskIssueBackgroundAttempts = 10
var manualTaskIssueBackgroundDelay = 2 * time.Second

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
	client, err := newIntakeGitHubClient(deps, db, target.integration)
	if err != nil {
		log.WithError(err).Warnf(
			"factory %s: intake %s: failed to create a GitHub issue in %s for a manual task",
			factory.ID,
			target.intake.ID,
			target.repository,
		)
		clearPendingManualTaskMarker(db, factory, order, marker)
		return
	}

	actor := rememberManualTaskGitHubActor(ctx, client, db, factory, order, target, marker)
	opened, err := createManualTaskGitHubIssue(ctx, client, target, title, description, marker, actor, order.CreatedAt)
	if err != nil {
		if errors.Is(err, errManualTaskGitHubIssueUnconfirmed) {
			log.WithError(err).Warnf(
				"factory %s: intake %s: GitHub did not confirm the issue for manual task %s",
				factory.ID,
				target.intake.ID,
				order.ID,
			)
			scheduleManualTaskIssueReconcile(deps, factory, order, target, marker, actor, order.CreatedAt)
			return
		}
		log.WithError(err).Warnf(
			"factory %s: intake %s: failed to create a GitHub issue in %s for a manual task",
			factory.ID,
			target.intake.ID,
			target.repository,
		)
		clearPendingManualTaskMarker(db, factory, order, marker)
		return
	}

	if err := storeManualTaskGitHubOrigin(db, factory, order, opened); err != nil {
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
	client *ghcommon.Client,
	target *manualTaskIssueTarget,
	title, description, marker, actor string,
	since time.Time,
) (models.WorkOrderOrigin, error) {
	request := &github.IssueRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(ghintegration.AppendManualTaskMarker(description, marker)),
	}

	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), manualTaskGitHubIssueTimeout)
	defer cancel()
	issue, _, err := client.CreateIssue(callCtx, target.repository, request)
	if err != nil {
		if !githubIssueCreateOutcomeUnknown(err) {
			return models.WorkOrderOrigin{}, err
		}
		found, findErr := findCreatedManualTaskIssue(client, target.repository, marker, actor, since)
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

func reconcileManualTaskGitHubIssue(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	target *manualTaskIssueTarget,
	marker string,
) {
	if order == nil || order.Origin() != nil {
		return
	}
	client, err := newIntakeGitHubClient(deps, db, target.integration)
	if err != nil {
		log.WithError(err).Warnf("factory %s: failed to reconcile the GitHub issue for task %s", factory.ID, order.ID)
		return
	}
	actor := ""
	if order.OriginLabel != nil {
		actor = ghintegration.ManualTaskActorFromLabel(*order.OriginLabel)
	}
	if actor == "" {
		actor = rememberManualTaskGitHubActor(ctx, client, db, factory, order, target, marker)
	}
	found, err := findCreatedManualTaskIssue(client, target.repository, marker, actor, order.CreatedAt)
	if err != nil || found == nil {
		if err != nil {
			log.WithError(err).Warnf("factory %s: failed to list GitHub issues for task %s", factory.ID, order.ID)
		}
		return
	}
	opened, err := manualTaskIssueFromGitHub(found)
	if err != nil {
		log.WithError(err).Warnf("factory %s: GitHub issue for task %s has no URL", factory.ID, order.ID)
		return
	}
	if err := storeManualTaskGitHubOrigin(db, factory, order, opened); err != nil {
		log.WithError(err).Warnf("factory %s: failed to store GitHub issue %s on task %s", factory.ID, opened.URL, order.ID)
	}
}

func scheduleManualTaskIssueReconcile(
	deps IntakeDependencies,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	target *manualTaskIssueTarget,
	marker, actor string,
	since time.Time,
) {
	if !runManualTaskIssueBackgroundReconcile || factory == nil || order == nil || target == nil {
		return
	}
	targetCopy := *target
	factoryID := factory.ID
	orderID := order.ID
	actorLogin := actor
	go func() {
		for attempt := 0; attempt < manualTaskIssueBackgroundAttempts; attempt++ {
			time.Sleep(manualTaskIssueBackgroundDelay)
			db := database.DB(context.Background())
			current, err := factory.FindWorkOrder(db, orderID)
			if err != nil || current == nil || current.Origin() != nil {
				return
			}
			client, clientErr := newIntakeGitHubClient(deps, db, targetCopy.integration)
			if clientErr != nil {
				log.WithError(clientErr).Warnf("factory %s: failed to reconcile the GitHub issue for task %s", factoryID, orderID)
				return
			}
			login := actorLogin
			if login == "" && current.OriginLabel != nil {
				login = ghintegration.ManualTaskActorFromLabel(*current.OriginLabel)
			}
			found, findErr := listCreatedManualTaskIssue(client, targetCopy.repository, marker, login, since)
			if findErr != nil || found == nil {
				continue
			}
			opened, originErr := manualTaskIssueFromGitHub(found)
			if originErr != nil {
				continue
			}
			if storeErr := storeManualTaskGitHubOrigin(db, factory, current, opened); storeErr != nil {
				log.WithError(storeErr).Warnf("factory %s: failed to store GitHub issue %s on task %s", factoryID, opened.URL, orderID)
				return
			}
			if publishErr := messages.PublishFactoryWorkOrderUpdated(
				factoryID.String(),
				orderID.String(),
				factoryevents.EventTypeOrderUpdated,
			); publishErr != nil {
				log.WithError(publishErr).Warnf("factory %s: failed to publish the GitHub origin for task %s", factoryID, orderID)
			}
			return
		}
	}()
}

func rememberManualTaskGitHubActor(
	ctx context.Context,
	client *ghcommon.Client,
	db *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	target *manualTaskIssueTarget,
	marker string,
) string {
	actor, err := manualTaskGitHubActor(ctx, client, target.integration)
	if err != nil {
		log.WithError(err).Warnf("factory %s: failed to read the GitHub login for manual task %s", factory.ID, order.ID)
		return ""
	}
	if actor == "" {
		return ""
	}
	if err := order.SetPendingGitHubMarker(db, ghintegration.PendingManualTaskLabel(marker, actor)); err != nil {
		log.WithError(err).Warnf("factory %s: failed to store the GitHub login on task %s", factory.ID, order.ID)
		return actor
	}
	return actor
}

func manualTaskGitHubActor(ctx context.Context, client *ghcommon.Client, integration *models.Integration) (string, error) {
	if login := githubAppBotLogin(integration); login != "" {
		return login, nil
	}
	lookupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), manualTaskGitHubActorTimeout)
	defer cancel()
	return client.AuthenticatedLogin(lookupCtx)
}

func githubAppBotLogin(integration *models.Integration) string {
	if integration == nil || integrationProperty(integration, ghcommon.PropertyAuthMethod) != ghcommon.AuthMethodApp {
		return ""
	}
	slug := integrationProperty(integration, ghcommon.PropertyAppSlug)
	if slug == "" {
		return ""
	}
	return slug + "[bot]"
}

func integrationProperty(integration *models.Integration, name string) string {
	for _, property := range integration.Properties {
		if property.Name != name {
			continue
		}
		value, _ := property.Value.(string)
		return strings.TrimSpace(value)
	}
	return ""
}

func clearPendingManualTaskMarker(db *gorm.DB, factory *models.Factory, order *models.FactoryWorkOrder, marker string) {
	label := marker
	if order.OriginLabel != nil && strings.TrimSpace(*order.OriginLabel) != "" {
		label = *order.OriginLabel
	}
	if clearErr := order.ClearPendingGitHubMarker(db, label); clearErr != nil {
		log.WithError(clearErr).Warnf("factory %s: failed to clear the pending GitHub marker on task %s", factory.ID, order.ID)
	}
}

func storeManualTaskGitHubOrigin(
	db *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	opened models.WorkOrderOrigin,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
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
	})
}

func findCreatedManualTaskIssue(
	client *ghcommon.Client,
	repository, marker, actor string,
	since time.Time,
) (*github.Issue, error) {
	if strings.TrimSpace(actor) == "" {
		return nil, nil
	}
	var lastErr error
	for attempt := 0; attempt < manualTaskGitHubIssueReconcileAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(manualTaskGitHubIssueReconcileDelay)
		}
		issue, err := listCreatedManualTaskIssue(client, repository, marker, actor, since)
		if err != nil {
			lastErr = err
			continue
		}
		if issue != nil {
			return issue, nil
		}
	}
	return nil, lastErr
}

func listCreatedManualTaskIssue(
	client *ghcommon.Client,
	repository, marker, actor string,
	since time.Time,
) (*github.Issue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), manualTaskGitHubIssueReconcileTimeout)
	defer cancel()

	listSince := since
	if !listSince.IsZero() {
		listSince = listSince.Add(-time.Minute)
	}
	issues, _, err := client.ListIssuesCreatedSince(ctx, repository, actor, listSince, manualTaskGitHubIssueListLimit)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		if manualTaskIssueMatches(issue, marker, actor) {
			return issue, nil
		}
	}
	return nil, nil
}

func manualTaskIssueMatches(issue *github.Issue, marker, actor string) bool {
	if issue == nil || issue.GetNumber() == 0 || issue.PullRequestLinks != nil {
		return false
	}
	if strings.TrimSpace(issue.GetHTMLURL()) == "" || strings.TrimSpace(actor) == "" {
		return false
	}
	if !strings.Contains(issue.GetBody(), marker) {
		return false
	}
	return strings.EqualFold(issue.GetUser().GetLogin(), actor)
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
