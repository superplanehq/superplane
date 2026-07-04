package monitoring

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func snoozeJSON(name, displayName string, policies []string, start, end string) []byte {
	b, _ := json.Marshal(map[string]any{
		"name":        name,
		"displayName": displayName,
		"criteria":    map[string]any{"policies": policies},
		"interval":    map[string]any{"startTime": start, "endTime": end},
	})
	return b
}

func Test__ParseSnoozeName(t *testing.T) {
	t.Run("relative name", func(t *testing.T) {
		name, err := parseSnoozeName("projects/my-project/snoozes/abc")
		require.NoError(t, err)
		assert.Equal(t, "projects/my-project/snoozes/abc", name)
	})
	t.Run("full URL", func(t *testing.T) {
		name, err := parseSnoozeName("https://monitoring.googleapis.com/v3/projects/elffie/snoozes/9")
		require.NoError(t, err)
		assert.Equal(t, "projects/elffie/snoozes/9", name)
	})
	t.Run("empty rejected", func(t *testing.T) {
		_, err := parseSnoozeName("")
		require.Error(t, err)
	})
	t.Run("non-snooze name rejected", func(t *testing.T) {
		_, err := parseSnoozeName("projects/my-project/alertPolicies/1")
		require.Error(t, err)
	})
}

func Test__resolveSnoozeName(t *testing.T) {
	t.Run("same project resolves", func(t *testing.T) {
		name, err := resolveSnoozeName("projects/elffie/snoozes/9", "elffie")
		require.NoError(t, err)
		assert.Equal(t, "projects/elffie/snoozes/9", name)
	})
	t.Run("cross-project rejected", func(t *testing.T) {
		_, err := resolveSnoozeName("projects/other/snoozes/9", "elffie")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cross-project")
	})
	t.Run("empty bound project does not block", func(t *testing.T) {
		name, err := resolveSnoozeName("projects/elffie/snoozes/9", "")
		require.NoError(t, err)
		assert.Equal(t, "projects/elffie/snoozes/9", name)
	})
}

func Test__CreateSnooze__Execute(t *testing.T) {
	c := &CreateSnooze{}

	t.Run("creates snooze with computed interval and policies", func(t *testing.T) {
		var postURL string
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postURL = url
				postBody, _ = body.(map[string]any)
				return snoozeJSON("projects/my-project/snoozes/55", "Deploy window",
					[]string{"projects/my-project/alertPolicies/1"}, "2025-01-01T00:00:00Z", "2025-01-01T01:00:00Z"), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"displayName": "Deploy window",
				"policies":    []any{"projects/my-project/alertPolicies/1"},
				"duration":    "1h",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.monitoring.snooze.created", state.Type)
		assert.True(t, strings.HasSuffix(postURL, "/projects/my-project/snoozes"))
		assert.Equal(t, "Deploy window", postBody["displayName"])
		criteria := postBody["criteria"].(map[string]any)
		assert.Equal(t, []string{"projects/my-project/alertPolicies/1"}, criteria["policies"])
		interval := postBody["interval"].(map[string]any)
		// Start and end are RFC3339 timestamps an hour apart.
		assert.NotEmpty(t, interval["startTime"])
		assert.NotEmpty(t, interval["endTime"])
		assert.NotEqual(t, interval["startTime"], interval["endTime"])

		data := firstData(t, state)
		assert.Equal(t, "55", data["id"])
		assert.Equal(t, 1, data["policiesCount"])
	})

	t.Run("requires at least one policy", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"displayName": "x", "duration": "1h", "policies": []any{}},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "at least one alerting policy")
	})

	t.Run("rejects an invalid duration", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"displayName": "x", "duration": "99h", "policies": []any{"projects/my-project/alertPolicies/1"}},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "invalid or missing duration")
	})
}

func Test__GetSnooze__Execute(t *testing.T) {
	g := &GetSnooze{}
	var getURL string
	mc := &mockClient{
		projectID: "my-project",
		getFunc: func(ctx context.Context, url string) ([]byte, error) {
			getURL = url
			return snoozeJSON("projects/my-project/snoozes/55", "Deploy window",
				[]string{"projects/my-project/alertPolicies/1"}, "2025-01-01T00:00:00Z", "2025-01-01T01:00:00Z"), nil
		},
	}
	withFactory(mc)

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	require.NoError(t, g.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"snooze": "projects/my-project/snoozes/55"},
		ExecutionState: state,
	}))
	assert.True(t, state.Passed)
	assert.Equal(t, "gcp.monitoring.snooze.fetched", state.Type)
	assert.True(t, strings.HasSuffix(getURL, "/projects/my-project/snoozes/55"))
	data := firstData(t, state)
	assert.Equal(t, "Deploy window", data["displayName"])
}

func Test__GetSnooze__RejectsCrossProject(t *testing.T) {
	withFactory(&mockClient{projectID: "my-project"})
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	require.NoError(t, (&GetSnooze{}).Execute(core.ExecutionContext{
		Configuration:  map[string]any{"snooze": "projects/other-project/snoozes/55"},
		ExecutionState: state,
	}))
	assert.False(t, state.Passed)
	assert.Contains(t, state.FailureMessage, "cross-project")
}

func Test__ExpireSnooze__Execute(t *testing.T) {
	e := &ExpireSnooze{}

	newExpireClient := func(start string) (*mockClient, *string, *map[string]any) {
		var patchURL string
		var patchBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return snoozeJSON("projects/my-project/snoozes/55", "Deploy window",
					[]string{"projects/my-project/alertPolicies/1"}, start, "2999-01-01T00:00:00Z"), nil
			},
			patchFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				patchURL = url
				patchBody, _ = body.(map[string]any)
				return snoozeJSON("projects/my-project/snoozes/55", "Deploy window",
					[]string{"projects/my-project/alertPolicies/1"}, start, start), nil
			},
		}
		return mc, &patchURL, &patchBody
	}

	t.Run("cancels a snooze younger than a minute with a zero-length interval", func(t *testing.T) {
		start := time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339)
		mc, patchURL, patchBody := newExpireClient(start)
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, e.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"snooze": "projects/my-project/snoozes/55"},
			ExecutionState: state,
		}))
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.monitoring.snooze.expired", state.Type)
		assert.Contains(t, *patchURL, "updateMask=interval.endTime")
		interval := (*patchBody)["interval"].(map[string]any)
		// Collapses to length 0 so GCP accepts the cancellation.
		assert.Equal(t, start, interval["endTime"])
	})

	t.Run("ends an older snooze at now", func(t *testing.T) {
		start := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
		mc, patchURL, patchBody := newExpireClient(start)
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, e.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"snooze": "projects/my-project/snoozes/55"},
			ExecutionState: state,
		}))
		assert.True(t, state.Passed)
		assert.Contains(t, *patchURL, "updateMask=interval.endTime")
		interval := (*patchBody)["interval"].(map[string]any)
		assert.NotEqual(t, start, interval["endTime"])
		ended, err := time.Parse(time.RFC3339, interval["endTime"].(string))
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().UTC(), ended, time.Minute)
	})
}

func Test__expireEndTime(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("young snooze collapses to its start time", func(t *testing.T) {
		start := now.Add(-30 * time.Second).Format(time.RFC3339)
		s := &snooze{Interval: &struct {
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
		}{StartTime: start}}
		assert.Equal(t, start, expireEndTime(s, now))
	})

	t.Run("older snooze ends at now", func(t *testing.T) {
		start := now.Add(-10 * time.Minute).Format(time.RFC3339)
		s := &snooze{Interval: &struct {
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
		}{StartTime: start}}
		assert.Equal(t, now.Format(time.RFC3339), expireEndTime(s, now))
	})

	t.Run("missing interval ends at now", func(t *testing.T) {
		assert.Equal(t, now.Format(time.RFC3339), expireEndTime(&snooze{}, now))
	})

	t.Run("unparseable start time collapses to that value (never sub-minute now)", func(t *testing.T) {
		s := &snooze{Interval: &struct {
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
		}{StartTime: "not-a-timestamp"}}
		assert.Equal(t, "not-a-timestamp", expireEndTime(s, now))
	})
}

func firstData(t *testing.T, state *contexts.ExecutionStateContext) map[string]any {
	t.Helper()
	require.NotEmpty(t, state.Payloads)
	wrapped, ok := state.Payloads[0].(map[string]any)
	require.True(t, ok, "payload should be the {type, timestamp, data} envelope")
	data, ok := wrapped["data"].(map[string]any)
	require.True(t, ok, "payload data should be a map")
	return data
}
