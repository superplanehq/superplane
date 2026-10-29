package factories

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"gorm.io/gorm"
)

const (
	HandOffColumnImplement = "implement"
	HandOffColumnVerify    = "verify"

	lineImplementationTemplateID = "line-implementation"
	prClosureTemplateID          = "pr-closure"

	handOffPullRequestURLInvalid = "pull_request_url must be a GitHub or Bitbucket pull request URL"
)

var bitbucketHandOffPullRequestURLPattern = regexp.MustCompile(
	`(?i)^https?://(?:www\.)?bitbucket\.org/([^/]+)/([^/]+)/pull-requests/(\d+)(?:[/?#].*)?$`,
)

type handOffPullRequest struct {
	Provider       string
	ExternalID     string
	Repository     string
	Number         int64
	URL            string
	Title          string
	State          string
	HeadBranch     string
	HeadRepository string
}

type parsedHandOffPullRequestURL struct {
	Provider   string
	Repository string
	Number     int
}

type HandOffWorkOrderRequest struct {
	FactoryID      string
	Title          string
	Description    string
	Plan           string
	Column         string
	PullRequestURL string
	LineName       string
	MCPClientID    string
	MCPClientName  string
}

type HandOffWorkOrderResult struct {
	Order          *pb.WorkOrder
	Column         string
	PullRequestURL string
}

type handOffStage struct {
	Index            int
	Name             string
	AppID            uuid.UUID
	IsDone           bool
	IsClosure        bool
	IsImplementation bool
}

func HandOffWorkOrder(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req HandOffWorkOrderRequest,
) (*HandOffWorkOrderResult, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to hand off work order")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, factoryErrorToStatus(invalidArgument("title is required"), "failed to hand off work order")
	}
	plan := strings.TrimSpace(req.Plan)
	if plan == "" {
		return nil, factoryErrorToStatus(invalidArgument("plan is required"), "failed to hand off work order")
	}
	column := strings.ToLower(strings.TrimSpace(req.Column))
	if column != HandOffColumnImplement && column != HandOffColumnVerify {
		return nil, factoryErrorToStatus(invalidArgument("column must be implement or verify"), "failed to hand off work order")
	}
	if column == HandOffColumnVerify && strings.TrimSpace(req.PullRequestURL) == "" {
		return nil, factoryErrorToStatus(invalidArgument("pull_request_url is required"), "failed to hand off work order")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to hand off work order")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.FactoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to hand off work order")
	}
	if column == HandOffColumnVerify {
		if _, err := validateHandOffPullRequestURL(factory, req.PullRequestURL); err != nil {
			return nil, factoryErrorToStatus(err, "failed to hand off work order")
		}
	}

	line, err := resolveHandOffLine(db, factory, req.LineName)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to hand off work order")
	}
	if len(line.Steps) == 0 {
		return nil, factoryErrorToStatus(models.ErrFactoryLineHasNoSteps, "failed to hand off work order")
	}

	stages, err := classifyHandOffStages(db, factory, line)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to hand off work order")
	}

	var implementIndex int
	var verifyIndex int
	if column == HandOffColumnImplement {
		implementIndex, err = resolveImplementationStageIndex(stages)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to hand off work order")
		}
	} else {
		verifyIndex, err = resolveVerifyStageIndex(stages)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to hand off work order")
		}
	}

	assigneeIDs, err := factory.DefaultSubscriptionAssignees(db, &createdByID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to choose task owner")
	}
	deferred := factory.OnboardingConfigValue().CodexSubscription && len(assigneeIDs) == 0
	var pullRequest *handOffPullRequest
	if column == HandOffColumnVerify && !deferred {
		pullRequest, err = loadHandOffPullRequest(ctx, db, deps, factory, strings.TrimSpace(req.PullRequestURL))
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to hand off work order")
		}
	}

	actor := createdByID
	clientName := strings.TrimSpace(req.MCPClientName)
	if clientName == "" {
		clientName = models.DefaultMCPClientName
	}

	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	var trackedPR *models.FactoryPullRequest
	var dispatched *workOrderLineDispatchResult
	err = db.Transaction(func(tx *gorm.DB) error {
		if !deferred {
			var ownerErr error
			assigneeIDs, ownerErr = factory.DefaultSubscriptionAssignees(tx, &createdByID)
			if ownerErr != nil {
				return ownerErr
			}
			deferred = factory.OnboardingConfigValue().CodexSubscription && len(assigneeIDs) == 0
		}

		created, err := factory.CreateWorkOrder(tx, title, req.Description, &createdByID, assigneeIDs, nil)
		if err != nil {
			return err
		}
		order = created
		if err := order.SetMCPClient(tx, req.MCPClientID, clientName); err != nil {
			return err
		}
		if err := order.StorePlanningSpec(tx, plan); err != nil {
			return err
		}
		result, bindErr := storedfiles.BindDescriptionFiles(
			ctx,
			tx,
			blob.Current(),
			orgID,
			factory.ID,
			order.ID,
			order.Description,
		)
		bound = result
		if bindErr != nil {
			return bindErr
		}
		if deferred {
			return order.SetPendingHandOff(tx, &models.PendingHandOff{LineID: line.ID, Column: column, PullRequestURL: strings.TrimSpace(req.PullRequestURL)})
		}
		if column == HandOffColumnImplement {
			started, err := dispatchWorkOrderOnLineTx(tx, factory, order.ID, line, &actor, implementIndex, false, "", "")
			if err != nil {
				return err
			}
			dispatched = started
			factory = started.factory
			order = started.order
			return nil
		}
		return recordVerifyHandOff(tx, factory, order, line, &actor, pullRequest, verifyIndex, models.FactoryLineStartOptions{})
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factory.ID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after bind")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to hand off work order")
	}
	captureFactoryProductEvent(
		ctx,
		db,
		factory,
		order.ID,
		productanalytics.EventFactoryTaskCreated,
		productanalytics.StableEventID(productanalytics.EventFactoryTaskCreated, order.ID.String()),
		productanalytics.Properties{"creation_source": "handoff"},
	)

	if dispatched != nil {
		dispatchID := productanalytics.StableEventID(productanalytics.EventFactoryTaskStarted, order.ID.String(), order.UpdatedAt.UTC().String())
		if active, findErr := order.FindActiveLineDispatch(db); findErr == nil {
			dispatchID = productanalytics.StableEventID(productanalytics.EventFactoryTaskStarted, active.ID.String())
		}
		captureFactoryProductEvent(
			ctx,
			db,
			factory,
			order.ID,
			productanalytics.EventFactoryTaskStarted,
			dispatchID,
			productanalytics.Properties{"run_kind": "initial"},
		)
		publishDispatchedWorkOrder(
			dispatched.logger,
			dispatched.factory.OrganizationID,
			dispatched.factory.ID,
			dispatched.order,
			&actor,
			dispatched.fromState,
			dispatched.startedSteps,
		)
	} else if !deferred {
		trackedPR, err = publishVerificationHandOff(ctx, deps, db, factory, order)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to hand off work order")
		}
	} else {
		if err := messages.PublishFactoryWorkOrderUpdated(factory.ID.String(), order.ID.String(), factoryevents.EventTypeOrderUpdated); err != nil {
			log.WithError(err).Warn("Failed to publish pending handoff")
		}

	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to hand off work order")
	}

	result := &HandOffWorkOrderResult{Order: serialized, Column: column}
	if deferred {
		result.Column = "backlog"
		result.PullRequestURL = strings.TrimSpace(req.PullRequestURL)
	}
	if trackedPR != nil {
		result.PullRequestURL = trackedPR.URL
	}
	return result, nil
}

func recordVerifyHandOff(
	tx *gorm.DB,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	line *models.FactoryLine,
	actor *uuid.UUID,
	pullRequest *handOffPullRequest,
	verifyIndex int,
	options models.FactoryLineStartOptions,
) error {
	if pullRequest == nil {
		return invalidArgument("pull_request_url is required")
	}
	taskRepo := ""
	if order.Repository != nil {
		taskRepo = strings.TrimSpace(*order.Repository)
	}
	if taskRepo == "" {
		return invalidArgument("This workspace has no repository.")
	}
	if !strings.EqualFold(pullRequest.Repository, taskRepo) {
		return invalidArgument("The pull request repository does not match the task repository.")
	}
	if !strings.EqualFold(strings.TrimSpace(pullRequest.State), models.FactoryPullRequestStateOpen) {
		return grpcerrors.FailedPrecondition(errFactoryPullRequestNotOpen, "The pull request is not open.")
	}

	headRef := strings.TrimPrefix(strings.TrimSpace(pullRequest.HeadBranch), "refs/heads/")
	if headRef == "" {
		return invalidArgument("The pull request has no head branch.")
	}
	headRepo := strings.TrimSpace(pullRequest.HeadRepository)
	if headRepo == "" {
		headRepo = taskRepo
	}

	if _, err := order.CreatePullRequest(tx, models.FactoryPullRequestParams{
		Provider:   pullRequest.Provider,
		ExternalID: pullRequest.ExternalID,
		Repository: pullRequest.Repository,
		Number:     pullRequest.Number,
		URL:        pullRequest.URL,
		Title:      pullRequest.Title,
		State:      models.FactoryPullRequestStateOpen,
	}); err != nil {
		return err
	}

	if _, err := order.CreateArtifact(tx, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeBranch,
		Data: map[string]any{
			"name":       headRef,
			"repository": headRepo,
		},
	}); err != nil {
		return err
	}

	if err := order.TransitionOnDispatch(tx, actor); err != nil {
		return err
	}
	if _, err := order.ClearAutoStart(tx); err != nil {
		return err
	}

	currentLine, err := factory.FindLine(tx, line.ID)
	if err != nil {
		return err
	}
	_, _, err = currentLine.RecordPassedStageWithoutRun(tx, order, verifyIndex, options)
	return err
}

func loadHandOffPullRequest(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	rawURL string,
) (*handOffPullRequest, error) {
	parsed, err := validateHandOffPullRequestURL(factory, rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Provider == models.FactoryPullRequestProviderBitbucket {
		return loadBitbucketHandOffPullRequest(db, deps, factory, rawURL, parsed)
	}
	return loadGitHubHandOffPullRequest(ctx, db, deps, factory, rawURL, parsed)
}

func validateHandOffPullRequestURL(factory *models.Factory, rawURL string) (parsedHandOffPullRequestURL, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsedURL.Host == "" || parsedURL.User != nil ||
		(!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https")) {
		return parsedHandOffPullRequestURL{}, invalidArgument(handOffPullRequestURLInvalid)
	}
	parsed, ok := parseHandOffPullRequestURL(rawURL)
	if !ok {
		return parsedHandOffPullRequestURL{}, invalidArgument(handOffPullRequestURLInvalid)
	}
	workspaceProvider := factory.OnboardingConfigValue().EffectiveVCSProvider()
	if !strings.EqualFold(parsed.Provider, workspaceProvider) {
		return parsedHandOffPullRequestURL{}, invalidArgument(handOffPullRequestProviderMismatch(parsed.Provider, workspaceProvider))
	}
	return parsed, nil
}

func loadGitHubHandOffPullRequest(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	rawURL string,
	parsed parsedHandOffPullRequestURL,
) (*handOffPullRequest, error) {
	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		return nil, err
	}

	githubPR, _, err := client.GetPullRequest(ctx, parsed.Repository, parsed.Number)
	if err != nil {
		return nil, grpcerrors.FailedPrecondition(err, "SuperPlane could not load the pull request.")
	}
	if githubPR == nil {
		return nil, grpcerrors.NotFound(errFactoryPullRequestMissing, "The pull request was not found.")
	}

	repository := strings.TrimSpace(githubPR.GetBase().GetRepo().GetFullName())
	if repository == "" {
		repository = parsed.Repository
	}
	return &handOffPullRequest{
		Provider:       models.FactoryPullRequestProviderGitHub,
		ExternalID:     strconv.FormatInt(githubPR.GetID(), 10),
		Repository:     repository,
		Number:         int64(githubPR.GetNumber()),
		URL:            handOffPullRequestURL(githubPR.GetHTMLURL(), rawURL),
		Title:          githubPR.GetTitle(),
		State:          githubPR.GetState(),
		HeadBranch:     githubPR.GetHead().GetRef(),
		HeadRepository: githubPR.GetHead().GetRepo().GetFullName(),
	}, nil
}

func loadBitbucketHandOffPullRequest(
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	rawURL string,
	parsed parsedHandOffPullRequestURL,
) (*handOffPullRequest, error) {
	client, err := newFactoryBitbucketAPI(db, deps, factory)
	if err != nil {
		return nil, err
	}

	pullRequest, err := client.GetPullRequest(parsed.Repository, int64(parsed.Number))
	if err != nil {
		return nil, grpcerrors.FailedPrecondition(err, "SuperPlane could not load the pull request.")
	}
	if pullRequest == nil {
		return nil, grpcerrors.NotFound(errFactoryPullRequestMissing, "The pull request was not found.")
	}

	number := pullRequest.ID
	if number <= 0 {
		number = int64(parsed.Number)
	}
	return &handOffPullRequest{
		Provider:       models.FactoryPullRequestProviderBitbucket,
		Repository:     parsed.Repository,
		Number:         number,
		URL:            handOffPullRequestURL(pullRequest.HTMLURL, rawURL),
		Title:          pullRequest.Title,
		State:          pullRequest.State,
		HeadBranch:     pullRequest.SourceBr,
		HeadRepository: pullRequest.SourceRepository,
	}, nil
}

func handOffPullRequestURL(apiURL, rawURL string) string {
	if trimmed := strings.TrimSpace(apiURL); trimmed != "" {
		return trimmed
	}
	return normalizeHandOffPullRequestURL(rawURL)
}

func normalizeHandOffPullRequestURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return trimmed
	}
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func handOffPullRequestProviderMismatch(urlProvider, workspaceProvider string) string {
	return fmt.Sprintf(
		"The pull request URL is for %s, but this workspace uses %s. SuperPlane supports GitHub and Bitbucket pull request URLs.",
		handOffProviderLabel(urlProvider),
		handOffProviderLabel(workspaceProvider),
	)
}

func handOffProviderLabel(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case models.ProviderBitbucket:
		return "Bitbucket"
	case models.ProviderGitHub:
		return "GitHub"
	default:
		return provider
	}
}

func resolveHandOffLine(tx *gorm.DB, factory *models.Factory, lineName string) (*models.FactoryLine, error) {
	name := strings.TrimSpace(lineName)
	if name != "" {
		return factory.FindLineByName(tx, name)
	}

	lines, err := factory.ListLines(tx)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, invalidArgument("This workspace has no line.")
	}
	if len(lines) > 1 {
		names := make([]string, 0, len(lines))
		for i := range lines {
			names = append(names, lines[i].Name)
		}
		return nil, invalidArgument("This workspace has more than one line. Set line to one of: " + strings.Join(names, ", "))
	}
	return &lines[0], nil
}

func classifyHandOffStages(tx *gorm.DB, factory *models.Factory, line *models.FactoryLine) ([]handOffStage, error) {
	appIDs := make([]uuid.UUID, 0, len(line.Steps))
	for _, step := range line.Steps {
		appIDs = append(appIDs, step.AppID)
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, appIDs)
	if err != nil {
		return nil, err
	}

	stages := make([]handOffStage, 0, len(line.Steps))
	for i, step := range line.Steps {
		canvas, err := models.FindCanvasInTransaction(tx, factory.OrganizationID, step.AppID)
		if err != nil {
			return nil, err
		}
		templateID := models.FactoryAppTemplateID(specs[step.AppID].Nodes)
		columnKey := ""
		if canvas.ColumnKey != nil {
			columnKey = strings.TrimSpace(*canvas.ColumnKey)
		}
		name := strings.TrimSpace(canvas.Name)
		stage := handOffStage{
			Index:            i,
			Name:             name,
			AppID:            step.AppID,
			IsImplementation: templateID == lineImplementationTemplateID,
			IsClosure:        templateID == prClosureTemplateID,
			IsDone:           isBoardDoneStage(name, columnKey, step.AppID),
		}
		if stage.Name == "" {
			stage.Name = fmt.Sprintf("step %d", i+1)
		}
		stages = append(stages, stage)
	}
	return stages, nil
}

func isBoardDoneStage(name, columnKey string, appID uuid.UUID) bool {
	if strings.EqualFold(strings.TrimSpace(name), "done") {
		return true
	}
	if columnKey == models.CanvasColumnKeyDone {
		return true
	}
	return strings.Contains(strings.ToLower(appID.String()), "pr-closure")
}

func resolveImplementationStageIndex(stages []handOffStage) (int, error) {
	board := make([]handOffStage, 0, len(stages))
	runnable := make([]handOffStage, 0, len(stages))
	matches := make([]handOffStage, 0, len(stages))
	for _, stage := range stages {
		if stage.IsDone {
			continue
		}
		board = append(board, stage)
		if stage.IsClosure {
			continue
		}
		runnable = append(runnable, stage)
		if stage.IsImplementation {
			matches = append(matches, stage)
		}
	}
	if len(board) == 1 && len(runnable) == 1 {
		return runnable[0].Index, nil
	}
	if len(matches) == 1 {
		return matches[0].Index, nil
	}
	return 0, invalidArgument("Cannot choose an implementation stage. Stages: " + joinStageNames(stages))
}

func resolveVerifyStageIndex(stages []handOffStage) (int, error) {
	last := -1
	for _, stage := range stages {
		if stage.IsDone {
			continue
		}
		last = stage.Index
	}
	if last < 0 {
		return 0, invalidArgument("This line has no stage for Verify. Stages: " + joinStageNames(stages))
	}
	return last, nil
}

func joinStageNames(stages []handOffStage) string {
	names := make([]string, 0, len(stages))
	for _, stage := range stages {
		names = append(names, stage.Name)
	}
	return strings.Join(names, ", ")
}

func parseHandOffPullRequestURL(rawURL string) (parsedHandOffPullRequestURL, bool) {
	if repository, number, ok := parseGitHubPullRequestURL(rawURL); ok {
		return parsedHandOffPullRequestURL{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: repository,
			Number:     number,
		}, true
	}
	return parseBitbucketPullRequestURL(rawURL)
}

func parseBitbucketPullRequestURL(rawURL string) (parsedHandOffPullRequestURL, bool) {
	matches := bitbucketHandOffPullRequestURLPattern.FindStringSubmatch(strings.TrimSpace(rawURL))
	if len(matches) != 4 {
		return parsedHandOffPullRequestURL{}, false
	}
	number, err := strconv.Atoi(matches[3])
	if err != nil || number <= 0 {
		return parsedHandOffPullRequestURL{}, false
	}
	return parsedHandOffPullRequestURL{
		Provider:   models.FactoryPullRequestProviderBitbucket,
		Repository: matches[1] + "/" + matches[2],
		Number:     number,
	}, true
}

func parseGitHubPullRequestURL(rawURL string) (repository string, number int, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return "", 0, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[1] == "" || parts[2] != "pull" {
		return "", 0, false
	}

	number, err = strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return "", 0, false
	}
	return parts[0] + "/" + parts[1], number, true
}

func latestTrackedPullRequest(tx *gorm.DB, orderID uuid.UUID) (*models.FactoryPullRequest, error) {
	grouped, err := models.ListPullRequestsByWorkOrderIDs(tx, []uuid.UUID{orderID})
	if err != nil {
		return nil, err
	}
	prs := grouped[orderID]
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

func publishVerificationHandOff(ctx context.Context, deps IntakeDependencies, db *gorm.DB, factory *models.Factory, order *models.FactoryWorkOrder) (*models.FactoryPullRequest, error) {

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
	tracked, err := latestTrackedPullRequest(db, order.ID)
	if err != nil {
		return nil, err
	}
	trackedPR := tracked
	if trackedPR != nil && trackedPR.Provider == models.FactoryPullRequestProviderGitHub {
		ScheduleFactoryPullRequestMergeabilityRefresh(
			ctx,
			deps,
			factory.OrganizationID,
			factory.ID,
			trackedPR.ID,
		)
		StartPullRequestChecks(ctx, deps, trackedPR.ID)
		StartVisualEvidenceCapture(
			ctx,
			deps,
			factory.OrganizationID,
			factory.ID,
			trackedPR.ID,
		)
	}
	if trackedPR != nil {
		if err := StartAutomaticCustomAutomations(ctx, deps, factory, order); err != nil {
			log.WithError(err).Warnf("Failed to start custom automation for order %s", order.ID)
		}
	}
	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderPullRequestAdded,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}
	return trackedPR, nil
}
