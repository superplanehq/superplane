package pulls

import (
	"context"
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func publishPullRequestCheckRun(ctx context.Context, client *common.Client, state *reviewCheckState) error {
	target := state.checkRunTarget()
	if target.ID == 0 {
		// GitHub may have accepted creation before the worker saved its receipt.
		opts := &github.ListCheckRunsOptions{CheckName: &state.Configuration.Context, Filter: github.Ptr("all"), ListOptions: github.ListOptions{PerPage: 100}}
		var match *github.CheckRun
		for {
			runs, response, err := client.ListCheckRunsForRef(ctx, state.Configuration.Repository, state.SHA, opts)
			if err != nil {
				return fmt.Errorf("find assessment check run: %w", err)
			}
			for _, run := range runs.CheckRuns {
				if len(state.Retired) > 0 && state.CheckRun != nil && run.GetID() == state.CheckRun.ID {
					continue
				}
				if run.GetExternalID() == target.ExternalID && run.GetHeadSHA() == state.SHA && run.GetName() == state.Configuration.Context {
					// Prefer a late running creation over an earlier completed recovery.
					if match == nil || run.GetStatus() != "completed" {
						match = run
					}
				}
			}
			if match != nil && match.GetStatus() != "completed" || response == nil || response.NextPage == 0 {
				break
			}
			opts.Page = response.NextPage
		}
		if match != nil {
			target.ID = match.GetID()
		}
	}
	status := "in_progress"
	var conclusion *string
	if state.State != "pending" {
		status = "completed"
		conclusion = github.Ptr("failure")
		if state.State == "success" {
			conclusion = github.Ptr("success")
		}
	}
	resultURL := state.resultURL()
	summary := state.Description
	if resultURL != "" {
		summary += "\n\n[View SuperPlane run](" + resultURL + ")"
	}
	output := &github.CheckRunOutput{Title: &state.Configuration.Context, Summary: &summary}
	if target.ID != 0 {
		_, _, err := client.UpdateCheckRun(ctx, state.Configuration.Repository, target.ID, github.UpdateCheckRunOptions{
			Name: state.Configuration.Context, Status: &status, Conclusion: conclusion, DetailsURL: &resultURL, Output: output,
		})
		if err != nil {
			return fmt.Errorf("update assessment check run: %w", err)
		}
		return nil
	}
	run, _, err := client.CreateCheckRun(ctx, state.Configuration.Repository, github.CreateCheckRunOptions{
		Name: state.Configuration.Context, HeadSHA: state.SHA, ExternalID: &target.ExternalID,
		Status: &status, Conclusion: conclusion, DetailsURL: &resultURL, Output: output,
	})
	if err != nil {
		return fmt.Errorf("create assessment check run: %w", err)
	}
	if run.GetID() == 0 {
		return fmt.Errorf("GitHub returned no check run ID")
	}
	target.ID = run.GetID()
	return nil
}
