package factories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	ghissue "github.com/superplanehq/superplane/pkg/integrations/github/issue"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	intakeGitHubReplayEventLimit = 200
	intakeGitHubReplayIssueLimit = 20
	intakeGitHubReplayWindow     = 30 * 24 * time.Hour
	intakeGitHubReplayBudget     = 15 * time.Second
)

func replaySkippedGitHubIssues(ctx context.Context, deps IntakeDependencies, db *gorm.DB, intake *models.FactoryIntake) error {
	ctx, cancel := context.WithTimeout(ctx, intakeGitHubReplayBudget)
	defer cancel()
	reads := db.WithContext(ctx)
	binding, err := liveIntakeBinding(reads, intake)
	if err != nil {
		return err
	}
	if binding == nil || binding.installation() == nil {
		return fmt.Errorf("GitHub intake is not bound")
	}
	client, err := newIntakeGitHubClient(deps, reads, binding.installation())
	if err != nil {
		return err
	}
	repository, _ := binding.Configuration["repository"].(string)
	payloads, err := skippedGitHubIssueEvents(ctx, reads, intake, client, repository)
	if err != nil {
		return err
	}
	// Reads can exhaust their budget. Persist the readable subset with the
	// request context, in a separate transaction against the committed graph.
	return db.Transaction(func(tx *gorm.DB) error {
		return emitIntakeEvents(tx, intake.CanvasID, ghissue.PayloadType, payloads)
	})
}

func skippedGitHubIssueEvents(ctx context.Context, reads *gorm.DB, intake *models.FactoryIntake, client *common.Client, repository string) ([]map[string]any, error) {
	factory, err := models.FindFactory(reads, intake.OrganizationID, intake.FactoryID)
	if err != nil {
		return nil, err
	}
	events, err := models.ListRecentCanvasRootEvents(reads, intake.CanvasID, ghissue.PayloadType, time.Now().Add(-intakeGitHubReplayWindow), intakeGitHubReplayEventLimit)
	if err != nil {
		return nil, err
	}
	seen := map[ghissue.Ref]bool{}
	payloads := []map[string]any{}
	attempts := 0
	for _, event := range events {
		if attempts >= intakeGitHubReplayIssueLimit || ctx.Err() != nil {
			break
		}
		ref, ok := ghissue.RefFromEventData(event.Data.Data())
		if !ok || seen[ref] || !strings.EqualFold(ref.Repository, strings.TrimSpace(repository)) || !client.Host().IsWebHostname(ref.Host) {
			continue
		}
		seen[ref] = true
		eligible, err := gitHubIssueRunWasSkipped(reads, event)
		if err != nil {
			return nil, err
		}
		if !eligible {
			continue
		}
		order, err := ghissue.FindActiveWorkOrder(reads, factory, ref)
		if err != nil {
			return nil, err
		}
		if order != nil {
			continue
		}
		attempts++
		issue, _, err := client.GetIssue(ctx, repository, ref.Number)
		if err != nil {
			log.WithError(err).WithField("intake_id", intake.ID).WithField("issue", ref.Number).Warn("failed to refresh skipped GitHub issue")
			continue
		}
		if issue == nil || issue.GetState() != "open" || issue.IsPullRequest() {
			continue
		}
		refreshedRef, ok := ghissue.RefFromURL(issue.GetHTMLURL())
		if !ok || refreshedRef != ref {
			continue
		}
		payload, err := gitHubIssueEvent(issue, repository)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func gitHubIssueRunWasSkipped(db *gorm.DB, event models.CanvasEvent) (bool, error) {
	run, err := models.FindCanvasRunInTransaction(db, event.WorkflowID, event.RunID)
	if err != nil {
		return false, err
	}
	if run.State != models.CanvasRunStateFinished || len(run.Errors) > 0 || run.Result == models.CanvasRunResultCancelled {
		return false, nil
	}
	version, err := models.FindCanvasVersionInTransaction(db, event.WorkflowID, run.VersionID)
	if err != nil {
		return false, err
	}
	spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
	graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
	if event.NodeID != graph.TriggerNodeID {
		return false, nil
	}
	executions, err := models.ListExecutionsForRunsInTransaction(db, event.WorkflowID, []uuid.UUID{run.ID})
	if err != nil {
		return false, err
	}
	ids := []uuid.UUID{}
	var failedPermissionID *uuid.UUID
	for _, execution := range executions {
		if execution.State != models.CanvasNodeExecutionStateFinished || execution.NodeID == graph.CreateNodeID || execution.Result == models.CanvasNodeExecutionResultCancelled {
			return false, nil
		}
		if execution.Result == models.CanvasNodeExecutionResultFailed {
			if execution.NodeID != graph.AuthorPermissionNodeID {
				return false, nil
			}
			failedPermissionID = &execution.ID
		}
		if execution.NodeID == graph.AuthorFilterNodeID || execution.NodeID == graph.FilterNodeID {
			// Legacy scoring thresholds are not issue-filter rejections.
			node := findIntakeNode(spec.Nodes, execution.NodeID)
			expression, _ := node.Configuration["expression"].(string)
			if _, scoring := intakeConfidenceFromExpression(expression); !scoring {
				ids = append(ids, execution.ID)
			}
		}
	}
	if failedPermissionID != nil {
		for _, execution := range executions {
			if execution.PreviousExecutionID != nil && *execution.PreviousExecutionID == *failedPermissionID {
				return false, nil
			}
		}
		return true, nil
	}
	if run.Result != models.CanvasRunResultPassed {
		return false, nil
	}
	outputs, err := models.ListExecutionOutputChannels(db, event.WorkflowID, ids)
	if err != nil {
		return false, err
	}
	for _, output := range outputs {
		if output.Channel == "false" {
			for _, execution := range executions {
				if execution.PreviousExecutionID != nil && *execution.PreviousExecutionID == output.ExecutionID {
					return false, nil
				}
			}
			return true, nil
		}
	}
	return false, nil
}
