package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type closedGitHubIssue struct {
	repository string
	number     int
}

type githubIssueWebhookPayload struct {
	Action     string                   `json:"action"`
	Issue      *githubIssueWebhookIssue `json:"issue"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type githubIssueWebhookIssue struct {
	Number      int             `json:"number"`
	HTMLURL     string          `json:"html_url"`
	PullRequest json.RawMessage `json:"pull_request"`
}

func ArchiveDraftWorkOrdersFromGitHubIssueClosed(
	ctx context.Context,
	encryptor crypto.Encryptor,
	webhook *models.Webhook,
	eventType string,
	headers http.Header,
	body []byte,
) (int, error) {
	issue, ok := closedGitHubIssueFromDelivery(eventType, body)
	if !ok {
		return http.StatusOK, nil
	}

	code, err := VerifyGitHubFactoryMergeabilitySignature(ctx, encryptor, webhook, headers, body)
	if err != nil {
		return code, err
	}

	archiveDraftWorkOrdersForClosedGitHubIssue(ctx, webhook, issue)
	return http.StatusOK, nil
}

func closedGitHubIssueFromDelivery(eventType string, body []byte) (closedGitHubIssue, bool) {
	if !strings.EqualFold(strings.TrimSpace(eventType), "issues") {
		return closedGitHubIssue{}, false
	}

	var payload githubIssueWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return closedGitHubIssue{}, false
	}
	if !strings.EqualFold(strings.TrimSpace(payload.Action), "closed") {
		return closedGitHubIssue{}, false
	}
	if payload.Issue == nil || payload.Issue.isPullRequest() {
		return closedGitHubIssue{}, false
	}

	repository := strings.TrimSpace(payload.Repository.FullName)
	number := payload.Issue.Number
	if repository == "" || number <= 0 {
		return closedGitHubIssue{}, false
	}

	htmlRepository, htmlNumber, htmlOK := parseGitHubIssueURL(payload.Issue.HTMLURL)
	if htmlOK && (htmlNumber != number || !strings.EqualFold(htmlRepository, repository)) {
		return closedGitHubIssue{}, false
	}

	return closedGitHubIssue{repository: repository, number: number}, true
}

func (issue *githubIssueWebhookIssue) isPullRequest() bool {
	if issue == nil || len(issue.PullRequest) == 0 {
		return false
	}
	return strings.TrimSpace(string(issue.PullRequest)) != "null"
}

func archiveDraftWorkOrdersForClosedGitHubIssue(
	ctx context.Context,
	webhook *models.Webhook,
	issue closedGitHubIssue,
) {
	if webhook == nil {
		return
	}

	db := database.DB(ctx)
	nodes, err := models.FindActiveWebhookNodesInTransaction(db, webhook.ID)
	if err != nil {
		log.WithError(err).Warnf(
			"factory intake: failed to load nodes for closed GitHub issue %s#%d",
			issue.repository,
			issue.number,
		)
		return
	}

	seenCanvas := map[uuid.UUID]struct{}{}
	seenFactory := map[uuid.UUID]struct{}{}
	for i := range nodes {
		canvasID := nodes[i].WorkflowID
		if _, seen := seenCanvas[canvasID]; seen {
			continue
		}
		seenCanvas[canvasID] = struct{}{}

		intake, err := models.FindFactoryIntakeByCanvasID(db, canvasID)
		if errors.Is(err, models.ErrFactoryIntakeNotFound) {
			continue
		}
		if err != nil {
			log.WithError(err).Warnf("factory intake: failed to load intake for canvas %s", canvasID)
			continue
		}
		if _, seen := seenFactory[intake.FactoryID]; seen {
			continue
		}
		seenFactory[intake.FactoryID] = struct{}{}

		factory, err := models.FindFactory(db, intake.OrganizationID, intake.FactoryID)
		if err != nil {
			log.WithError(err).Warnf("factory intake: factory %s not found", intake.FactoryID)
			continue
		}
		if err := archiveFactoryDraftsForClosedGitHubIssue(db, factory, issue); err != nil {
			log.WithError(err).Warnf(
				"factory intake: failed to archive drafts for closed GitHub issue %s#%d",
				issue.repository,
				issue.number,
			)
		}
	}
}

func archiveFactoryDraftsForClosedGitHubIssue(
	db *gorm.DB,
	factory *models.Factory,
	issue closedGitHubIssue,
) error {
	orders, err := factory.ListWorkOrdersByOriginURLFragment(db, fmt.Sprintf("/issues/%d", issue.number))
	if err != nil {
		return err
	}

	for i := range orders {
		order := &orders[i]
		if !workOrderMatchesClosedGitHubIssue(order, issue) {
			continue
		}

		closed, err := closeDraftWorkOrderIfCurrent(db, factory, order.ID, nil)
		if err != nil {
			log.WithError(err).Warnf("factory intake: failed to archive draft %s", order.ID)
			continue
		}
		if closed == nil {
			continue
		}

		publishWorkOrderClosed(
			factory.OrganizationID,
			factory,
			closed,
			nil,
			models.FactoryWorkOrderStateDraft,
			models.FactoryWorkOrderResultRejected,
			false,
		)
	}
	return nil
}

func workOrderMatchesClosedGitHubIssue(order *models.FactoryWorkOrder, issue closedGitHubIssue) bool {
	if order == nil || order.OriginURL == nil {
		return false
	}

	repository, number, ok := parseGitHubIssueURL(*order.OriginURL)
	if !ok {
		return false
	}
	return number == issue.number && strings.EqualFold(repository, issue.repository)
}

func closeDraftWorkOrderIfCurrent(
	db *gorm.DB,
	factory *models.Factory,
	orderID uuid.UUID,
	actor *uuid.UUID,
) (*models.FactoryWorkOrder, error) {
	var closed *models.FactoryWorkOrder
	err := db.Transaction(func(tx *gorm.DB) error {
		current, err := factory.FindWorkOrder(tx.Clauses(clause.Locking{Strength: "UPDATE"}), orderID)
		if err != nil {
			return err
		}
		if current.State != models.FactoryWorkOrderStateDraft {
			return nil
		}

		closed, err = current.Close(tx, models.FactoryWorkOrderResultRejected, actor)
		if errors.Is(err, models.ErrFactoryWorkOrderInvalidState) {
			closed = nil
			return nil
		}
		return err
	})
	return closed, err
}
