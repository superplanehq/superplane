package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullRequestsCloseResult(t *testing.T) {
	pullRequests := func(states ...string) []FactoryPullRequest {
		result := make([]FactoryPullRequest, 0, len(states))
		for _, state := range states {
			result = append(result, FactoryPullRequest{State: state})
		}
		return result
	}

	cases := []struct {
		name   string
		states []string
		result string
		done   bool
	}{
		{name: "no pull requests", done: false},
		{name: "one open", states: []string{FactoryPullRequestStateMerged, FactoryPullRequestStateOpen}, done: false},
		{name: "one draft", states: []string{FactoryPullRequestStateClosed, FactoryPullRequestStateDraft}, done: false},
		{name: "all merged", states: []string{FactoryPullRequestStateMerged, FactoryPullRequestStateMerged}, result: FactoryWorkOrderResultCompleted, done: true},
		{name: "merged and closed", states: []string{FactoryPullRequestStateClosed, FactoryPullRequestStateMerged}, result: FactoryWorkOrderResultCompleted, done: true},
		{name: "all closed", states: []string{FactoryPullRequestStateClosed, FactoryPullRequestStateClosed}, result: FactoryWorkOrderResultPRClosed, done: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, done := PullRequestsCloseResult(pullRequests(tc.states...))
			assert.Equal(t, tc.done, done)
			assert.Equal(t, tc.result, result)
		})
	}
}
