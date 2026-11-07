package factories

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

type pullRequestImportGitHubSearch interface {
	ListPullRequests(context.Context, string, *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	SearchIssues(context.Context, string, *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error)
}

func attachExistingPullRequestImportTasks(ctx context.Context, factory *models.Factory, items []*pb.FactoryPullRequestImportItem) error {
	numbers := make([]int64, len(items))
	for i, item := range items {
		numbers[i] = item.GetNumber()
	}
	config := factory.OnboardingConfigValue()
	ordersByNumber, err := factory.ListWorkOrdersByPullRequestNumbers(database.DB(ctx), config.EffectiveVCSProvider(), config.AppRepository, numbers)
	if err != nil {
		return err
	}
	orders := make([]models.FactoryWorkOrder, 0, len(ordersByNumber))
	seen := make(map[uuid.UUID]bool, len(ordersByNumber))
	for _, order := range ordersByNumber {
		if seen[order.ID] {
			continue
		}
		seen[order.ID] = true
		orders = append(orders, order)
	}
	summaries, err := loadAndSerializeWorkOrders(ctx, factory, orders)
	if err != nil {
		return err
	}
	summariesByID := make(map[string]*pb.WorkOrderSummary, len(summaries))
	for _, summary := range summaries {
		summariesByID[summary.GetId()] = summary
	}
	for _, item := range items {
		if order, ok := ordersByNumber[item.GetNumber()]; ok {
			item.ExistingTask = summariesByID[order.ID.String()]
		}
	}
	return nil
}

func searchPullRequestImportItems(ctx context.Context, db *gorm.DB, deps IntakeDependencies, factory *models.Factory, query string, limit int) ([]*pb.FactoryPullRequestImportItem, error) {
	config := factory.OnboardingConfigValue()
	repository := config.AppRepository
	provider := config.EffectiveVCSProvider()
	number, numericErr := strconv.Atoi(strings.TrimPrefix(query, "#"))
	if strings.Contains(query, "://") {
		parsed, err := validateImportPullRequestURL(db, factory, query)
		if err != nil {
			return nil, err
		}
		number = parsed.Number
	}
	if number > 0 && (numericErr == nil || strings.Contains(query, "://")) {
		rawURL := query
		if numericErr == nil {
			var err error
			rawURL, err = importPullRequestURL(db, factory, int64(number))
			if err != nil {
				return nil, err
			}
		}
		pr, err := loadHandOffPullRequest(ctx, db, deps, factory, rawURL)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(pr.State, models.FactoryPullRequestStateOpen) {
			return []*pb.FactoryPullRequestImportItem{}, nil
		}
		return []*pb.FactoryPullRequestImportItem{{Url: pr.URL, Title: pr.Title, Description: pr.Description, Number: pr.Number, Draft: pr.Draft, Repository: pr.Repository, Provider: pr.Provider}}, nil
	}
	if provider == models.ProviderBitbucket {
		return searchBitbucketImportItems(db, deps, factory, query, limit)
	}
	if provider != models.ProviderGitHub {
		return nil, invalidArgument("This workspace provider does not support pull request import.")
	}
	client, err := newFactoryGitHubAPI(db, deps, factory)
	if err != nil {
		return nil, err
	}
	search, ok := client.(pullRequestImportGitHubSearch)
	if !ok {
		return nil, fmt.Errorf("GitHub connection does not support pull request search")
	}
	items := make([]*pb.FactoryPullRequestImportItem, 0, limit)
	if query == "" {
		prs, _, err := search.ListPullRequests(ctx, repository, &github.PullRequestListOptions{State: "open", Sort: "created", Direction: "desc", ListOptions: github.ListOptions{PerPage: limit}})
		if err != nil {
			return nil, err
		}
		for _, pr := range prs {
			if !strings.EqualFold(pr.GetState(), "open") {
				continue
			}
			if base := pr.GetBase().GetRepo().GetFullName(); base != "" && !strings.EqualFold(base, repository) {
				continue
			}
			items = append(items, &pb.FactoryPullRequestImportItem{Url: pr.GetHTMLURL(), Title: pr.GetTitle(), Description: pr.GetBody(), Number: int64(pr.GetNumber()), Draft: pr.GetDraft(), Repository: repository, Provider: provider})
		}
		return items, nil
	}
	// Quoting prevents user text from adding repository or state qualifiers.
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(query)
	results, _, err := search.SearchIssues(ctx, fmt.Sprintf(`repo:%s is:pr is:open in:title "%s"`, repository, escaped), &github.SearchOptions{Sort: "created", Order: "desc", ListOptions: github.ListOptions{PerPage: limit}})
	if err != nil {
		return nil, err
	}
	for _, issue := range results.Issues {
		parsed, err := validateImportPullRequestURL(db, factory, issue.GetHTMLURL())
		if err != nil || issue.GetPullRequestLinks() == nil || issue.GetState() != "open" {
			continue
		}
		items = append(items, &pb.FactoryPullRequestImportItem{Url: issue.GetHTMLURL(), Title: issue.GetTitle(), Description: issue.GetBody(), Number: int64(parsed.Number), Draft: issue.GetDraft(), Repository: repository, Provider: provider})
	}
	return items, nil
}

func importPullRequestURL(db *gorm.DB, factory *models.Factory, number int64) (string, error) {
	config := factory.OnboardingConfigValue()
	if config.EffectiveVCSProvider() == models.ProviderBitbucket {
		return fmt.Sprintf("https://bitbucket.org/%s/pull-requests/%d", config.AppRepository, number), nil
	}
	host, err := githubcommon.HostFromFactoryOnboarding(db, factory.OrganizationID, config)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/pull/%d", host.RepoURL(config.AppRepository), number), nil
}
