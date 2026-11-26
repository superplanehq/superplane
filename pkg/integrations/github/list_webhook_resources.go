package github

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func (*GitHub) listRepositoryWebhookResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repository := strings.TrimSpace(ctx.Parameters["repository"])
	if repository == "" {
		return nil, fmt.Errorf("repository is required")
	}
	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, err
	}
	readCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	options := &github.ListOptions{PerPage: 100}
	resources := []core.IntegrationResource{}
	for {
		hooks, response, err := client.ListHooks(readCtx, repository, options)
		if err != nil {
			return nil, err
		}
		for _, hook := range hooks {
			event := ctx.Parameters["event"]
			if event != "" && !slices.Contains(hook.Events, event) && !slices.Contains(hook.Events, "*") {
				continue
			}
			// Hook URL paths can also contain credentials. Only show the host.
			destination := "Unknown destination"
			if parsed, err := url.Parse(hook.GetConfig().GetURL()); err == nil && parsed.Host != "" {
				destination = parsed.Host
			}
			id := strconv.FormatInt(hook.GetID(), 10)
			resources = append(resources, core.IntegrationResource{
				Type: "repository_webhook", ID: id,
				Name: fmt.Sprintf("#%s · %s · %s", id, destination, strings.Join(hook.Events, ", ")),
				URL:  common.HostFromIntegration(ctx.Integration).RepoURL(repository) + "/settings/hooks/" + id,
			})
		}
		if response == nil || response.NextPage == 0 {
			return resources, nil
		}
		options.Page = response.NextPage
	}
}
