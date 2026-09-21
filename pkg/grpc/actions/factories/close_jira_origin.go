package factories

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const jiraCompletionComment = "The SuperPlane task completed."

// JiraCloseContext is the HTTP and crypto stack the close-back worker uses
// after a work order completes. Jira calls stay outside any DB transaction.
type JiraCloseContext struct {
	HTTP      core.HTTPContext
	Encryptor crypto.Encryptor
	Registry  *registry.Registry
	BaseURL   string
}

type jiraCloseTarget struct {
	Integration    *models.Integration
	IssueKey       string
	MoveOnComplete bool
	Column         string
}

// CloseJiraOrigin moves the originating Jira issue after a work order
// completes. A missing origin, a disabled move, or a Jira error does not
// change the work order.
func CloseJiraOrigin(
	tx *gorm.DB,
	closeCtx JiraCloseContext,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
) error {
	if factory == nil || order == nil {
		return nil
	}

	origin := order.Origin()
	if origin == nil {
		return nil
	}

	target, err := resolveJiraCloseTarget(tx, factory, origin.URL)
	if err != nil {
		return err
	}
	if target == nil || !target.MoveOnComplete {
		return nil
	}

	return closeJiraIssue(tx, closeCtx, factory, order, target)
}

func resolveJiraCloseTarget(tx *gorm.DB, factory *models.Factory, originURL string) (*jiraCloseTarget, error) {
	originURL = strings.TrimSpace(originURL)
	if factory == nil || originURL == "" {
		return nil, nil
	}

	intakes, err := factory.ListIntakes(tx)
	if err != nil {
		return nil, err
	}

	var canvasIDs []uuid.UUID
	jiraIntakes := make([]models.FactoryIntake, 0, len(intakes))
	for i := range intakes {
		if intakes[i].Source != models.FactoryIntakeSourceJiraIssues {
			continue
		}
		jiraIntakes = append(jiraIntakes, intakes[i])
		canvasIDs = append(canvasIDs, intakes[i].CanvasID)
	}
	if len(jiraIntakes) == 0 {
		return nil, nil
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, canvasIDs)
	if err != nil {
		return nil, err
	}

	for i := range jiraIntakes {
		intake := &jiraIntakes[i]
		spec := specs[intake.CanvasID]
		graph := resolveIntakeGraph(intake.Source, spec)
		trigger := findIntakeNode(spec.Nodes, graph.TriggerNodeID)
		if trigger == nil {
			continue
		}

		projectKey := configurationString(trigger.Configuration["project"])
		integrationID := strings.TrimSpace(derefString(trigger.IntegrationID))
		if projectKey == "" || integrationID == "" {
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
		if integration.AppName != intakeJiraAppName || integration.State != models.IntegrationStateReady {
			continue
		}

		siteURL := jira.SiteURLFromMetadata(integration.Metadata.Data())
		if siteURL == "" {
			continue
		}

		issueKey, ok := parseJiraIssueURL(originURL, siteURL)
		if !ok || !strings.EqualFold(jiraIssueProjectKey(issueKey), projectKey) {
			continue
		}

		settings := jiraCompletionSettingsFromMetadata(trigger.Metadata, defaultJiraIntakeSettings())
		return &jiraCloseTarget{
			Integration:    integration,
			IssueKey:       issueKey,
			MoveOnComplete: settings.JiraMoveOnComplete,
			Column:         settings.JiraCompletionColumn,
		}, nil
	}

	return nil, nil
}

func closeJiraIssue(
	tx *gorm.DB,
	closeCtx JiraCloseContext,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	target *jiraCloseTarget,
) error {
	if closeCtx.HTTP == nil {
		closeCtx.HTTP = closeCtx.Registry.HTTPContext()
	}

	client, err := jira.NewClient(
		closeCtx.HTTP,
		contexts.NewIntegrationContext(tx, nil, target.Integration, closeCtx.Encryptor, closeCtx.Registry, nil),
	)
	if err != nil {
		return mapJiraCloseError(target.IssueKey, err)
	}

	issue, err := client.GetIssueWithOptions(target.IssueKey, jira.GetIssueOptions{Fields: "status"})
	if err != nil {
		return mapJiraCloseError(target.IssueKey, err)
	}
	if jira.IssueAlreadyInColumn(issue, target.Column) {
		return nil
	}

	err = jira.ApplyCompletionStatus(client, target.IssueKey, target.Column, jira.DoTransitionOptions{
		Comment: jiraCompletionCommentFor(closeCtx.BaseURL, factory, order),
	})
	if err != nil {
		return mapJiraCloseError(target.IssueKey, err)
	}

	return nil
}

func jiraCompletionCommentFor(baseURL string, factory *models.Factory, order *models.FactoryWorkOrder) string {
	if factory == nil || order == nil {
		return jiraCompletionComment
	}

	path := order.URLPath(factory.Key)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || path == "" {
		return jiraCompletionComment
	}

	return jiraCompletionComment + " " + baseURL + path
}

func mapJiraCloseError(issueKey string, err error) error {
	if jira.IsRetryableAPIError(err) {
		return fmt.Errorf("failed to close Jira issue %s: %w", issueKey, err)
	}

	log.WithError(err).Warnf("Skipping Jira close for issue %s", issueKey)
	return nil
}
