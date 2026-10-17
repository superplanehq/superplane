package factories

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

// LinearCloseContext is the HTTP and crypto stack the close-back worker uses
// after a work order completes. Linear calls stay outside any DB transaction.
type LinearCloseContext = JiraCloseContext

type linearCloseTarget struct {
	Integration    *models.Integration
	Issue          linear.Issue
	MoveOnComplete bool
	State          string
}

// CloseLinearOrigin moves the originating Linear issue after a work order
// completes. A missing origin, a disabled move, or a Linear error does not
// change the work order.
func CloseLinearOrigin(
	tx *gorm.DB,
	closeCtx LinearCloseContext,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
) error {
	if factory == nil || order == nil {
		return nil
	}

	target, err := resolveLinearCloseTarget(tx, closeCtx, factory, order)
	if err != nil {
		return err
	}
	if target == nil || !target.MoveOnComplete {
		return nil
	}

	return closeLinearIssue(tx, closeCtx, target)
}

func resolveLinearCloseTarget(
	tx *gorm.DB,
	closeCtx LinearCloseContext,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
) (*linearCloseTarget, error) {
	if factory == nil || order == nil {
		return nil, nil
	}

	origin := order.Origin()
	if origin == nil {
		return nil, nil
	}
	ref, ok := linear.IssueRefFromURL(strings.TrimSpace(origin.URL))
	if !ok {
		return nil, nil
	}

	intakes, err := factory.ListIntakes(tx)
	if err != nil {
		return nil, err
	}

	var canvasIDs []uuid.UUID
	linearIntakes := make([]models.FactoryIntake, 0, len(intakes))
	for i := range intakes {
		if intakes[i].Source != models.FactoryIntakeSourceLinearIssues {
			continue
		}
		linearIntakes = append(linearIntakes, intakes[i])
		canvasIDs = append(canvasIDs, intakes[i].CanvasID)
	}
	if len(linearIntakes) == 0 {
		return nil, nil
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, canvasIDs)
	if err != nil {
		return nil, err
	}

	preferredCanvasID, err := canvasIDFromSourceRun(tx, order)
	if err != nil {
		return nil, err
	}

	var fallback *linearCloseTarget
	var resolveErr error
	for i := range linearIntakes {
		intake := &linearIntakes[i]
		spec := specs[intake.CanvasID]
		graph := resolveIntakeGraph(intake.Source, spec)
		trigger := findIntakeNode(spec.Nodes, graph.TriggerNodeID)
		if trigger == nil {
			continue
		}

		integrationID := strings.TrimSpace(derefString(trigger.IntegrationID))
		if integrationID == "" {
			continue
		}

		id, err := uuid.Parse(integrationID)
		if err != nil {
			continue
		}
		integration, err := models.FindUnscopedIntegrationInTransaction(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if integration.AppName != intakeLinearAppName || integration.State != models.IntegrationStateReady {
			continue
		}

		client, err := linear.NewClient(
			closeCtx.HTTP,
			contexts.NewIntegrationContext(tx, nil, integration, closeCtx.Encryptor, closeCtx.Registry, nil),
		)
		if err != nil {
			if resolveErr == nil {
				resolveErr = err
			}
			continue
		}

		issue, err := client.GetIssue(ref.Identifier)
		if err != nil {
			// The issue lives in another workspace, or it is gone.
			// Another intake may still match, so keep looking.
			if resolveErr == nil {
				resolveErr = err
			}
			continue
		}

		projectIDs := configurationStrings(trigger.Configuration["projects"])
		if len(projectIDs) > 0 && (issue.Project == nil || !slices.Contains(projectIDs, issue.Project.ID)) {
			continue
		}

		settings := linearCompletionSettingsFromMetadata(trigger.Metadata, defaultLinearIntakeSettings())
		target := &linearCloseTarget{
			Integration:    integration,
			Issue:          *issue,
			MoveOnComplete: settings.LinearMoveOnComplete,
			State:          settings.LinearCompletionState,
		}
		if preferredCanvasID != uuid.Nil && intake.CanvasID == preferredCanvasID {
			return target, nil
		}
		if fallback == nil {
			fallback = target
		}
	}

	if fallback != nil {
		return fallback, nil
	}
	if resolveErr != nil {
		return nil, mapLinearCloseError(ref.Identifier, resolveErr)
	}
	return nil, nil
}

func closeLinearIssue(tx *gorm.DB, closeCtx LinearCloseContext, target *linearCloseTarget) error {
	if target.Issue.Team == nil || strings.TrimSpace(target.Issue.Team.ID) == "" {
		log.Warnf("Skipping Linear close for issue %s: issue has no team", target.Issue.Identifier)
		return nil
	}
	if target.Issue.State != nil &&
		(target.Issue.State.Type == "completed" || target.Issue.State.Type == "canceled") {
		return nil
	}

	client, err := linear.NewClient(
		closeCtx.HTTP,
		contexts.NewIntegrationContext(tx, nil, target.Integration, closeCtx.Encryptor, closeCtx.Registry, nil),
	)
	if err != nil {
		return mapLinearCloseError(target.Issue.Identifier, err)
	}

	states, err := client.ListWorkflowStates(target.Issue.Team.ID)
	if err != nil {
		return mapLinearCloseError(target.Issue.Identifier, err)
	}

	stateID, ok := linearCompletionStateID(states, target.State)
	if !ok {
		log.Warnf("Skipping Linear close for issue %s: no matching state", target.Issue.Identifier)
		return nil
	}
	if target.Issue.State != nil && target.Issue.State.ID == stateID {
		return nil
	}

	if _, err := client.UpdateIssue(target.Issue.ID, map[string]any{"stateId": stateID}); err != nil {
		return mapLinearCloseError(target.Issue.Identifier, err)
	}

	return nil
}

// linearCompletionStateID resolves the configured state name inside the
// issue's team. An empty name means the team's Done state: prefer a
// completed state named Done, else the first completed state.
func linearCompletionStateID(states []linear.WorkflowState, wanted string) (string, bool) {
	if wanted = strings.TrimSpace(wanted); wanted != "" {
		for _, state := range states {
			if strings.EqualFold(strings.TrimSpace(state.Name), wanted) {
				return state.ID, true
			}
		}
		return "", false
	}

	for _, state := range states {
		if state.Type == "completed" && strings.EqualFold(strings.TrimSpace(state.Name), "done") {
			return state.ID, true
		}
	}
	for _, state := range states {
		if state.Type == "completed" {
			return state.ID, true
		}
	}
	return "", false
}

var linearCloseStatusPattern = regexp.MustCompile(`request got (\d+)`)

// mapLinearCloseError decides whether a failed close retries. Rate limits,
// timeouts, server errors, and transport failures return an error so Tackle
// redelivers. Semantic Linear errors and client errors skip with a log, and
// the work order is unchanged either way.
func mapLinearCloseError(identifier string, err error) error {
	if linearCloseRetryable(err) {
		return fmt.Errorf("failed to close Linear issue %s: %w", identifier, err)
	}

	log.WithError(err).Warnf("Skipping Linear close for issue %s", identifier)
	return nil
}

func linearCloseRetryable(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()
	if strings.Contains(message, "linear API error:") {
		return false
	}

	match := linearCloseStatusPattern.FindStringSubmatch(message)
	if match == nil {
		return true
	}
	status, convErr := strconv.Atoi(match[1])
	if convErr != nil {
		return true
	}

	return status == 429 || status == 408 || status >= 500
}
