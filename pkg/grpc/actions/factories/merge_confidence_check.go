package factories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const mergeConfidenceCheckName = "SuperPlane merge confidence"

type mergeConfidenceCheckPublisher interface {
	CreateCheckRun(context.Context, string, github.CreateCheckRunOptions) (*github.CheckRun, *github.Response, error)
	UpdateCheckRun(context.Context, string, int64, github.UpdateCheckRunOptions) (*github.CheckRun, *github.Response, error)
	ListCheckRunsForRef(context.Context, string, string, *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error)
}

type mergeConfidenceCheckPublication struct {
	status     string
	conclusion string
	summary    string
}

type mergeConfidenceCheckState struct {
	request     *models.CanvasNodeRequest
	run         *models.CanvasRun
	factory     *models.Factory
	pullRequest *models.FactoryPullRequest
	order       *models.FactoryWorkOrder
	head        string
	superseded  bool
	retired     bool
	publication mergeConfidenceCheckPublication
}

// ProcessMergeConfidenceCheckRequest follows the admitted run through completion.
// The request has no execution owner, so cancellation cannot discard delivery.
func ProcessMergeConfidenceCheckRequest(ctx context.Context, deps IntakeDependencies, requestID uuid.UUID) error {
	publicationCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db := database.DB(publicationCtx)
	state, err := readMergeConfidenceCheckState(db, requestID)
	if err != nil || state == nil {
		return err
	}
	return state.pullRequest.WithMergeConfidencePublicationLock(db, func(conn *gorm.DB) error {
		// A publisher can finish between the first read and advisory acquisition.
		state, err := readMergeConfidenceCheckState(conn, requestID)
		if err != nil || state == nil {
			return err
		}
		delivered, publicationErr := deliverMergeConfidenceCheck(publicationCtx, conn, deps, state)
		if delivered == nil {
			return publicationErr
		}
		return finalizeMergeConfidenceCheck(publicationCtx, conn, delivered, publicationErr)
	})
}

func readMergeConfidenceCheckState(db *gorm.DB, requestID uuid.UUID) (*mergeConfidenceCheckState, error) {
	var state *mergeConfidenceCheckState
	var readErr error
	err := db.Transaction(func(tx *gorm.DB) error {
		request, err := models.LockNodeRequest(tx, requestID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		state, readErr = loadMergeConfidenceCheckState(tx, request)
		if readErr != nil {
			return request.Reschedule(tx, time.Now().Add(time.Minute))
		}
		return nil
	})
	return state, errors.Join(readErr, err)
}

func loadMergeConfidenceCheckState(tx *gorm.DB, request *models.CanvasNodeRequest) (*mergeConfidenceCheckState, error) {
	spec := request.Spec.Data().MergeConfidenceCheck
	if request.Type != models.NodeRequestTypeMergeConfidenceCheck || spec == nil {
		return nil, fmt.Errorf("invalid merge confidence check request")
	}
	run, err := models.FindCanvasRunInTransaction(tx, request.WorkflowID, spec.RunID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, request.Complete(tx)
	}
	if err != nil {
		return nil, err
	}
	canvas, err := models.FindUnscopedCanvasInTransaction(tx, run.WorkflowID)
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil {
		return nil, request.Complete(tx)
	}
	// Pending publications must finish before deleted workspaces can be cleaned up.
	factory, err := models.FindFactory(tx.Unscoped(), canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	pr, err := factory.FindPullRequest(tx, models.FactoryPullRequestLookup{ID: spec.PullRequestID})
	if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
		return nil, request.Complete(tx)
	}
	if err != nil {
		return nil, err
	}
	if pr.Provider != models.FactoryPullRequestProviderGitHub {
		return nil, request.Complete(tx)
	}
	order, err := factory.FindWorkOrder(tx, pr.WorkOrderID)
	if err != nil {
		return nil, err
	}
	// Match the task-before-PR order used by review startup and PR closure.
	if err := order.LockForUpdate(tx); err != nil {
		return nil, err
	}
	if err := pr.LockForUpdate(tx); err != nil {
		return nil, err
	}
	// A row-lock wait can overlap completion or deletion. Read their current
	// state only after taking the task and PR locks.
	run, err = models.FindCanvasRunInTransaction(tx, request.WorkflowID, spec.RunID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, request.Complete(tx)
	}
	if err != nil {
		return nil, err
	}
	canvas, err = models.FindUnscopedCanvasInTransaction(tx, run.WorkflowID)
	if err != nil {
		return nil, err
	}
	root, err := run.RootEvent(tx)
	if err != nil {
		return nil, err
	}
	payload, _ := models.RootEventSourcePayload(root.Data.Data()).(map[string]any)
	head := strings.TrimSpace(nestedString(payload, "pull_request", "head", "sha"))
	if head == "" {
		return nil, fmt.Errorf("merge confidence run %s has no reviewed commit", run.ID)
	}
	latest, _, err := pr.LatestMergeConfidenceAssessment(tx, head)
	if err != nil {
		return nil, err
	}
	superseded := latest != nil && latest.ID != run.ID
	retired := superseded || canvas.DeletedAt.Valid
	publication := mergeConfidenceCheckPublication{"completed", "cancelled", "Merge confidence assessment was canceled."}
	if !retired {
		publication, err = mergeConfidenceRunPublication(tx, run, head)
		if err != nil {
			return nil, err
		}
	}
	return &mergeConfidenceCheckState{
		request: request, run: run, factory: factory, pullRequest: pr, order: order,
		head: head, superseded: superseded, retired: retired, publication: publication,
	}, nil
}

func deliverMergeConfidenceCheck(ctx context.Context, db *gorm.DB, deps IntakeDependencies, state *mergeConfidenceCheckState) (*mergeConfidenceCheckState, error) {
	// Client setup can refresh GitHub credentials and must run outside a transaction.
	client, err := newFactoryGitHubAPI(db, deps, state.factory)
	if err != nil {
		return state, err
	}
	publisher, ok := client.(mergeConfidenceCheckPublisher)
	if !ok {
		return state, fmt.Errorf("GitHub client cannot publish merge confidence checks")
	}
	check, err := findMergeConfidenceCheck(ctx, publisher, state.pullRequest.Repository, state.head, state.run.ID)
	if err != nil {
		return state, err
	}
	if err := ctx.Err(); err != nil {
		return state, err
	}
	// Admission and deletion remain free to proceed during GitHub reads.
	current, err := readMergeConfidenceCheckState(db, state.request.ID)
	if err != nil {
		return state, err
	}
	if current == nil {
		return nil, nil
	}
	state = current
	if state.superseded && check == nil {
		return state, nil
	}
	if err := ctx.Err(); err != nil {
		return state, err
	}
	detailsURL := blob.PublicBaseURL() + state.order.URLPath(state.factory.RouteSegment())
	err = publishMergeConfidenceCheck(ctx, publisher, state.pullRequest.Repository, state.run.ID, state.head, detailsURL, check, state.publication)
	return state, err
}

func finalizeMergeConfidenceCheck(ctx context.Context, db *gorm.DB, delivered *mergeConfidenceCheckState, publicationErr error) error {
	// An interrupted HTTP request still needs a durable retry and lock release.
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var changedOrder *models.FactoryWorkOrder
	var readErr error
	finalize := func(tx *gorm.DB) error {
		request := &models.CanvasNodeRequest{ID: delivered.request.ID}
		err := request.LockForUpdate(tx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		current, err := loadMergeConfidenceCheckState(tx, request)
		if err != nil {
			readErr = err
			return request.Reschedule(tx, time.Now().Add(time.Minute))
		}
		if current == nil {
			return nil
		}
		previousNotes := string(current.order.StatusNote)
		if current.retired || publicationErr == nil {
			err = current.order.ClearStatusNote(tx, "merge-confidence-check:"+current.run.ID.String())
		} else {
			err = recordMergeConfidenceCheckPermissionError(tx, current.order, current.run, publicationErr)
		}
		if err != nil {
			return err
		}
		if string(current.order.StatusNote) != previousNotes {
			changedOrder = current.order
		}
		if publicationErr != nil {
			return request.Reschedule(tx, time.Now().Add(time.Minute))
		}
		// A newer assessment, deletion, or run completion can change the required
		// result while GitHub accepts delivery. Keep it pending for reconciliation.
		if current.publication != delivered.publication || current.retired != delivered.retired {
			return request.Reschedule(tx, time.Now().Add(10*time.Second))
		}
		if current.publication.status == "completed" {
			return request.Complete(tx)
		}
		return request.Reschedule(tx, time.Now().Add(10*time.Second))
	}
	err := db.WithContext(finalizeCtx).Transaction(finalize)
	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, driver.ErrBadConn) {
		// Cancellation can close or discard the dedicated connection.
		// Retry finalization from the pool while its cleanup finishes.
		changedOrder = nil
		readErr = nil
		err = database.DB(finalizeCtx).Transaction(finalize)
	}
	if err == nil && changedOrder != nil {
		if err := messages.PublishFactoryWorkOrderUpdated(changedOrder.FactoryID.String(), changedOrder.ID.String(), factoryevents.EventTypeOrderStatusNoteUpdated); err != nil {
			log.WithError(err).Warn("Merge confidence: failed to publish task status note update")
		}
	}
	return errors.Join(publicationErr, readErr, err)
}

func recordMergeConfidenceCheckPermissionError(tx *gorm.DB, order *models.FactoryWorkOrder, run *models.CanvasRun, publicationErr error) error {
	var githubError *github.ErrorResponse
	if !order.IsOpen() || !errors.As(publicationErr, &githubError) || githubError.Response == nil ||
		githubError.Response.StatusCode != http.StatusForbidden || githubError.Message != "Resource not accessible by integration" {
		return nil
	}
	_, err := order.SetStatusNote(tx, models.FactoryWorkOrderStatusNoteParams{
		Key:        "merge-confidence-check:" + run.ID.String(),
		Headline:   "GitHub check needs permission",
		Body:       "SuperPlane could not publish the merge confidence check. Ask the GitHub App owner to grant Checks: Read and write. Then ask the installation owner to accept the permission update. SuperPlane retries automatically.",
		Automation: &factoryevents.AutomationRef{AppID: run.WorkflowID, AppName: "Merge confidence", NodeID: run.NodeID},
		Run:        &factoryevents.RunRef{ID: run.ID, State: run.State},
	})
	return err
}

func mergeConfidenceRunPublication(tx *gorm.DB, run *models.CanvasRun, head string) (mergeConfidenceCheckPublication, error) {
	if run.State == models.CanvasRunStateCancelling || run.Result == models.CanvasRunResultCancelled {
		return mergeConfidenceCheckPublication{"completed", "cancelled", "Merge confidence assessment was canceled."}, nil
	}
	if run.State != models.CanvasRunStateFinished {
		return mergeConfidenceCheckPublication{"in_progress", "", "Merge confidence assessment is running."}, nil
	}
	if run.Result != models.CanvasRunResultPassed {
		return mergeConfidenceCheckPublication{"completed", "failure", "Merge confidence assessment failed. View the task for details."}, nil
	}
	report, err := run.MergeConfidenceAssessment(tx)
	if err != nil {
		return mergeConfidenceCheckPublication{}, err
	}
	if report["complete"] != true || report["reviewedCommit"] != head {
		return mergeConfidenceCheckPublication{"completed", "failure", "Merge confidence assessment did not report all enabled checks."}, nil
	}
	// Scores describe risk. This check reports assessment delivery, not a merge gate.
	return mergeConfidenceCheckPublication{"completed", "success", "Merge confidence assessment completed. View the task for results."}, nil
}

func findMergeConfidenceCheck(ctx context.Context, client mergeConfidenceCheckPublisher, repository, head string, runID uuid.UUID) (*github.CheckRun, error) {
	options := &github.ListCheckRunsOptions{CheckName: github.Ptr(mergeConfidenceCheckName), Filter: github.Ptr("all"), ListOptions: github.ListOptions{PerPage: 100}}
	for {
		checks, response, err := client.ListCheckRunsForRef(ctx, repository, head, options)
		if err != nil {
			return nil, err
		}
		if checks != nil {
			for _, check := range checks.CheckRuns {
				if check.GetExternalID() == runID.String() {
					return check, nil
				}
			}
		}
		if response == nil || response.NextPage == 0 {
			return nil, nil
		}
		options.Page = response.NextPage
	}
}

func publishMergeConfidenceCheck(ctx context.Context, client mergeConfidenceCheckPublisher, repository string, runID uuid.UUID, head, detailsURL string, check *github.CheckRun, publication mergeConfidenceCheckPublication) error {
	output := &github.CheckRunOutput{Title: github.Ptr("Merge confidence"), Summary: &publication.summary}
	if check != nil && check.GetStatus() == publication.status && check.GetConclusion() == publication.conclusion && check.GetOutput().GetSummary() == publication.summary && check.GetDetailsURL() == detailsURL {
		return nil
	}
	var conclusion *string
	if publication.conclusion != "" {
		conclusion = &publication.conclusion
	}
	var result *github.CheckRun
	var err error
	if check == nil {
		result, _, err = client.CreateCheckRun(ctx, repository, github.CreateCheckRunOptions{
			Name: mergeConfidenceCheckName, HeadSHA: head, ExternalID: github.Ptr(runID.String()),
			Status: &publication.status, Conclusion: conclusion, DetailsURL: &detailsURL, Output: output,
		})
	} else {
		result, _, err = client.UpdateCheckRun(ctx, repository, check.GetID(), github.UpdateCheckRunOptions{
			Name: mergeConfidenceCheckName, ExternalID: github.Ptr(runID.String()),
			Status: &publication.status, Conclusion: conclusion, DetailsURL: &detailsURL, Output: output,
		})
	}
	if err != nil {
		return fmt.Errorf("publish merge confidence check: %w", err)
	}
	if result == nil || result.GetID() == 0 {
		return fmt.Errorf("GitHub did not return a merge confidence check ID")
	}
	return nil
}
