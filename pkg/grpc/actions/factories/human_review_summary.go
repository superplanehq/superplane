package factories

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func publishHumanReviewSummary(ctx context.Context, db *gorm.DB, client humanReviewCheckPublisher, pr *models.FactoryPullRequest, decision *models.FactoryHumanReviewDecision, publication humanReviewPublication) error {
	lines := []string{"## SuperPlane human review", publication.Reason}
	if decision.ReviewRequired && len(decision.SelectedReviewers) > 0 {
		reviewers := make([]string, 0, len(decision.SelectedReviewers))
		for _, reviewer := range decision.SelectedReviewers {
			reviewers = append(reviewers, "@"+reviewer.Login)
		}
		lines = append(lines, "Selected reviewers: "+strings.Join(reviewers, ", "))
	}
	lines = append(lines, "Reviewed commit: "+decision.AssessedCommit)
	if decision.CheckURL != "" {
		lines = append(lines, "[View review details]("+decision.CheckURL+")")
	}
	_, err := prbody.Publish(ctx, humanReviewBodyClient{client}, prbody.Configuration{
		Repository: pr.Repository, PullNumber: strconv.FormatInt(pr.Number, 10), SectionKey: "human-review",
		Body: strings.Join(lines, "\n\n"), ExpectedHeadSHA: decision.AssessedCommit,
	}, func() (bool, error) { return humanReviewPublicationCurrent(db, pr, decision) })
	if err != nil {
		return fmt.Errorf("publish Human review summary: %w", err)
	}
	return nil
}

type humanReviewBodyClient struct{ client humanReviewCheckPublisher }

func (c humanReviewBodyClient) Read(ctx context.Context, repository string, number int64) (prbody.Description, error) {
	pr, _, err := c.client.GetPullRequest(ctx, repository, int(number))
	if err != nil {
		return prbody.Description{}, err
	}
	return prbody.Description{Body: pr.GetBody(), Open: pr.GetState() == "open" && !pr.GetMerged(), HeadSHA: pr.GetHead().GetSHA()}, nil
}

func (c humanReviewBodyClient) Write(ctx context.Context, repository string, number int64, body string) error {
	_, _, err := c.client.EditPullRequest(ctx, repository, int(number), &github.PullRequest{Body: &body})
	return err
}
