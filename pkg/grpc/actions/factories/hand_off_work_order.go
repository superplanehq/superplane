package factories

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"gorm.io/gorm"
)

const (
	HandOffColumnImplement = "implement"
	HandOffColumnVerify    = "verify"

	lineImplementationTemplateID = "line-implementation"
	prClosureTemplateID          = "pr-closure"
)

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

	var githubPR *github.PullRequest
	var prRepository string
	if column == HandOffColumnVerify {
		githubPR, prRepository, err = loadHandOffPullRequest(ctx, db, deps, factory, strings.TrimSpace(req.PullRequestURL))
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to hand off work order")
		}
	}

	actor := createdByID
	assigneeIDs := []uuid.UUID{createdByID}
	clientName := strings.TrimSpace(req.MCPClientName)
	if clientName == "" {
		clientName = models.DefaultMCPClientName
	}

	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	var trackedPR *models.FactoryPullRequest
	var dispatched *workOrderLineDispatchResult
	err = db.Transaction(func(tx *gorm.DB) error {
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
		return recordVerifyHandOff(tx, factory, order, line, &actor, githubPR, prRepository, verifyIndex)
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factory.ID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after bind")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to hand off work order")
	}

	if dispatched != nil {
		publishDispatchedWorkOrder(
			dispatched.logger,
			dispatched.factory.OrganizationID,
			dispatched.factory.ID,
			dispatched.order,
			&actor,
			dispatched.fromState,
			dispatched.startedSteps,
		)
	} else {
		if err := messages.PublishFactoryWorkOrderUpdated(
			factory.ID.String(),
			order.ID.String(),
			factoryevents.EventTypeOrderStatusUpdated,
		); err != nil {
			log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
		}
		tracked, err := latestTrackedPullRequest(db, order.ID)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to hand off work order")
		}
		trackedPR = tracked
		if trackedPR != nil && trackedPR.Provider == models.FactoryPullRequestProviderGitHub {
			ScheduleFactoryPullRequestMergeabilityRefresh(
				ctx,
				deps,
				factory.OrganizationID,
				factory.ID,
				trackedPR.ID,
			)
			StartVisualEvidenceCapture(
				ctx,
				deps,
				factory.OrganizationID,
				factory.ID,
				trackedPR.ID,
			)
		}
		if err := messages.PublishFactoryWorkOrderUpdated(
			factory.ID.String(),
			order.ID.String(),
			factoryevents.EventTypeOrderPullRequestAdded,
		); err != nil {
			log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
		}
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to hand off work order")
	}

	result := &HandOffWorkOrderResult{Order: serialized, Column: column}
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
	githubPR *github.PullRequest,
	prRepository string,
	verifyIndex int,
) error {
	if githubPR == nil {
		return invalidArgument("pull_request_url is required")
	}
	taskRepo := ""
	if order.Repository != nil {
		taskRepo = strings.TrimSpace(*order.Repository)
	}
	if taskRepo == "" {
		return invalidArgument("This workspace has no repository.")
	}
	if !strings.EqualFold(prRepository, taskRepo) {
		return invalidArgument("The pull request repository does not match the task repository.")
	}
	if !strings.EqualFold(strings.TrimSpace(githubPR.GetState()), models.FactoryPullRequestStateOpen) {
		return grpcerrors.FailedPrecondition(errFactoryPullRequestNotOpen, "The pull request is not open.")
	}

	headRef := strings.TrimPrefix(strings.TrimSpace(githubPR.GetHead().GetRef()), "refs/heads/")
	if headRef == "" {
		return invalidArgument("The pull request has no head branch.")
	}
	headRepo := pullRequestHeadRepository(githubPR, taskRepo)

	if _, err := order.CreatePullRequest(tx, models.FactoryPullRequestParams{
		Provider:   models.FactoryPullRequestProviderGitHub,
		ExternalID: strconv.FormatInt(githubPR.GetID(), 10),
		Repository: prRepository,
		Number:     int64(githubPR.GetNumber()),
		URL:        githubPR.GetHTMLURL(),
		Title:      githubPR.GetTitle(),
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
	_, _, err = currentLine.RecordPassedStageWithoutRun(tx, order, verifyIndex)
	return err
}

func loadHandOffPullRequest(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	rawURL string,
) (*github.PullRequest, string, error) {
	repository, number, ok := parseGitHubPullRequestURL(rawURL)
	if !ok {
		return nil, "", invalidArgument("pull_request_url must be a GitHub pull request URL")
	}

	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		return nil, "", err
	}

	githubPR, _, err := client.GetPullRequest(ctx, repository, number)
	if err != nil {
		return nil, "", grpcerrors.FailedPrecondition(err, "SuperPlane could not load the pull request.")
	}
	if githubPR == nil {
		return nil, "", grpcerrors.NotFound(errFactoryPullRequestMissing, "The pull request was not found.")
	}

	prRepository := strings.TrimSpace(githubPR.GetBase().GetRepo().GetFullName())
	if prRepository == "" {
		prRepository = repository
	}
	return githubPR, prRepository, nil
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

func pullRequestHeadRepository(githubPR *github.PullRequest, taskRepo string) string {
	if githubPR == nil {
		return taskRepo
	}
	headRepo := strings.TrimSpace(githubPR.GetHead().GetRepo().GetFullName())
	if headRepo != "" {
		return headRepo
	}
	return taskRepo
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
