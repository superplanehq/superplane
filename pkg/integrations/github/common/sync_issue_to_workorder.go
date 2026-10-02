package common

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

// SyncGitHubIssueEdit finds the work order for a GitHub issue and updates its
// title and description from the current issue content. Returns true if a
// work order was updated, false if none was found.
func SyncGitHubIssueEdit(
	ctx context.Context,
	repository string,
	issueNumber int,
	title string,
	body string,
) (bool, error) {
	repository = strings.TrimSpace(repository)
	if repository == "" || issueNumber <= 0 || title == "" {
		return false, nil
	}

	db := database.DB(ctx)

	// Build the URL pattern to search for work orders
	urlPattern := fmt.Sprintf("github.com/%s/issues/%d", repository, issueNumber)

	var workOrders []models.FactoryWorkOrder
	if err := db.
		Where("origin_url ILIKE ?", "%"+urlPattern+"%").
		Find(&workOrders).
		Error; err != nil {
		log.WithError(err).WithField("issue", urlPattern).
			Warn("Failed to find work orders for GitHub issue edit")
		return false, nil // Don't fail the webhook on database errors
	}

	if len(workOrders) == 0 {
		return false, nil // No work order found, that's OK
	}

	// Update the first work order found (the original one from "opened" action)
	order := workOrders[0]
	if err := order.UpdateContent(db, &title, &body); err != nil {
		log.WithError(err).WithField("workOrderID", order.ID).
			Warn("Failed to update work order from GitHub issue edit")
		return false, nil // Don't fail the webhook on update errors
	}

	log.WithField("workOrderID", order.ID).
		WithField("issue", urlPattern).
		Info("Synced GitHub issue edit to work order")

	return true, nil
}

// SyncGitHubIssueComment finds the work order for a GitHub issue and adds a
// comment to it. Returns true if a comment was added, false if no work order
// was found.
func SyncGitHubIssueComment(
	ctx context.Context,
	repository string,
	issueNumber int,
	commentBody string,
	commenterLogin string,
) (bool, error) {
	repository = strings.TrimSpace(repository)
	if repository == "" || issueNumber <= 0 || commentBody == "" {
		return false, nil
	}

	db := database.DB(ctx)

	// Build the URL pattern to search for work orders
	urlPattern := fmt.Sprintf("github.com/%s/issues/%d", repository, issueNumber)

	var workOrders []models.FactoryWorkOrder
	if err := db.
		Where("origin_url ILIKE ?", "%"+urlPattern+"%").
		Find(&workOrders).
		Error; err != nil {
		log.WithError(err).WithField("issue", urlPattern).
			Warn("Failed to find work orders for GitHub issue comment")
		return false, nil // Don't fail the webhook on database errors
	}

	if len(workOrders) == 0 {
		return false, nil // No work order found, that's OK
	}

	// Add comment to the first work order found
	order := workOrders[0]

	params := models.FactoryWorkOrderCommentParams{
		Body: commentBody,
		Author: factory.WorkOrderCommentAuthor{
			Kind: "automation",
			Automation: &factory.AutomationAuthor{
				Source: "github",
				Details: map[string]string{
					"issue":     fmt.Sprintf("%s#%d", repository, issueNumber),
					"commenter": commenterLogin,
				},
			},
		},
	}

	if _, err := order.RecordCommentAdded(db, params); err != nil {
		log.WithError(err).WithField("workOrderID", order.ID).
			Warn("Failed to add comment to work order from GitHub issue")
		return false, nil // Don't fail the webhook on comment errors
	}

	log.WithField("workOrderID", order.ID).
		WithField("issue", urlPattern).
		Info("Synced GitHub issue comment to work order")

	return true, nil
}
