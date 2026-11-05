package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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

// ProcessMergeConfidenceCheckRequest follows the admitted run through completion.
// The request has no execution owner, so cancellation cannot discard delivery.
func ProcessMergeConfidenceCheckRequest(ctx context.Context, deps IntakeDependencies, requestID uuid.UUID) error {
	publicationCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var publicationErr error
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		request, err := models.LockNodeRequest(tx, requestID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		publicationErr = reconcileMergeConfidenceCheck(publicationCtx, tx, deps, request)
		if publicationErr == nil {
			return nil
		}
		// Commit the retry even when GitHub rejects or loses the response.
		return request.Reschedule(tx, time.Now().Add(time.Minute))
	})
	if err != nil {
		return err
	}
	return publicationErr
}

func reconcileMergeConfidenceCheck(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, request *models.CanvasNodeRequest) error {
	spec := request.Spec.Data().MergeConfidenceCheck
	if request.Type != models.NodeRequestTypeMergeConfidenceCheck || spec == nil {
		return fmt.Errorf("invalid merge confidence check request")
	}
	run, err := models.FindCanvasRunInTransaction(tx, request.WorkflowID, spec.RunID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return request.Complete(tx)
	}
	if err != nil {
		return err
	}
	canvas, err := models.FindUnscopedCanvasInTransaction(tx, run.WorkflowID)
	if err != nil {
		return err
	}
	if canvas.FactoryID == nil {
		return request.Complete(tx)
	}
	factory, err := models.FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return err
	}
	pr, err := factory.FindPullRequest(tx, models.FactoryPullRequestLookup{ID: spec.PullRequestID})
	if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
		return request.Complete(tx)
	}
	if err != nil {
		return err
	}
	if pr.Provider != models.FactoryPullRequestProviderGitHub {
		return request.Complete(tx)
	}
	// Admission also locks the PR. Serialize the latest-run check and publication
	// so an older run cannot create a check after a newer run for the same head.
	if err := pr.LockForUpdate(tx); err != nil {
		return err
	}
	root, err := run.RootEvent(tx)
	if err != nil {
		return err
	}
	payload, _ := models.RootEventSourcePayload(root.Data.Data()).(map[string]any)
	head := strings.TrimSpace(nestedString(payload, "pull_request", "head", "sha"))
	if head == "" {
		return fmt.Errorf("merge confidence run %s has no reviewed commit", run.ID)
	}
	latest, _, err := pr.LatestMergeConfidenceAssessment(tx, head)
	if err != nil {
		return err
	}
	superseded := latest != nil && latest.ID != run.ID
	client, err := newFactoryGitHubAPI(tx, deps, factory)
	if err != nil {
		return err
	}
	publisher, ok := client.(mergeConfidenceCheckPublisher)
	if !ok {
		return fmt.Errorf("GitHub client cannot publish merge confidence checks")
	}
	check, err := findMergeConfidenceCheck(ctx, publisher, pr.Repository, head, run.ID)
	if err != nil {
		return err
	}
	if superseded && check == nil {
		return request.Complete(tx)
	}
	publication, err := mergeConfidenceRunPublication(tx, run, head)
	if err != nil {
		return err
	}
	if superseded || canvas.DeletedAt.Valid {
		publication = mergeConfidenceCheckPublication{"completed", "cancelled", "Merge confidence assessment was canceled."}
	}
	order, err := factory.FindWorkOrder(tx, pr.WorkOrderID)
	if err != nil {
		return err
	}
	detailsURL := blob.PublicBaseURL() + order.URLPath(factory.RouteSegment())
	if err := publishMergeConfidenceCheck(ctx, publisher, pr.Repository, run.ID, head, detailsURL, check, publication); err != nil {
		return err
	}
	if publication.status == "completed" {
		return request.Complete(tx)
	}
	return request.Reschedule(tx, time.Now().Add(10*time.Second))
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
