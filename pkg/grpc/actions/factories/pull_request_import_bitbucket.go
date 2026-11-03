package factories

import (
	"encoding/json"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func searchBitbucketImportItems(db *gorm.DB, deps IntakeDependencies, factory *models.Factory, query string, limit int) ([]*pb.FactoryPullRequestImportItem, error) {
	client, err := newFactoryBitbucketAPI(db, deps, factory)
	if err != nil {
		return nil, err
	}
	repository := factory.OnboardingConfigValue().AppRepository
	prs, err := client.ListPullRequests(repository, bitbucket.PullRequestListOptions{TitleContains: query, States: []string{bitbucket.PullRequestStateOpen}, Sort: "-created_on", Pagelen: limit})
	if err != nil {
		return nil, err
	}
	items := make([]*pb.FactoryPullRequestImportItem, 0, len(prs))
	for _, pr := range prs {
		// List responses contain nested destination metadata, unlike the merge model.
		var current struct {
			ID          int64  `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			State       string `json:"state"`
			Draft       bool   `json:"draft"`
			Destination struct {
				Repository struct {
					FullName string `json:"full_name"`
				} `json:"repository"`
			} `json:"destination"`
		}
		encoded, err := json.Marshal(pr)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(encoded, &current); err != nil {
			return nil, err
		}
		if current.ID <= 0 || !strings.EqualFold(current.State, bitbucket.PullRequestStateOpen) {
			continue
		}
		if base := current.Destination.Repository.FullName; base != "" && !strings.EqualFold(base, repository) {
			continue
		}
		rawURL, err := importPullRequestURL(db, factory, current.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, &pb.FactoryPullRequestImportItem{Url: rawURL, Title: current.Title, Description: current.Description, Number: current.ID, Draft: current.Draft, Repository: repository, Provider: models.ProviderBitbucket})
	}
	return items, nil
}
