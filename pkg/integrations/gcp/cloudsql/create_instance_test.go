package cloudsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateInstance__Setup(t *testing.T) {
	c := &CreateInstance{}
	setup := func(cfg map[string]any) error {
		return c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing name -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"databaseVersion": "POSTGRES_16", "region": "us-central1", "tier": "db-f1-micro"}), "name is required")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"name": "i1", "databaseVersion": "POSTGRES_16", "tier": "db-f1-micro"}), "region is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"name": "i1", "databaseVersion": "POSTGRES_16", "region": "us-central1", "tier": "db-f1-micro"}))
	})
}

func Test__CreateInstance__Execute(t *testing.T) {
	c := &CreateInstance{}

	runnableInstance := []byte(`{"name":"my-instance","state":"RUNNABLE","databaseVersion":"POSTGRES_16","region":"us-central1","connectionName":"my-project:us-central1:my-instance","selfLink":"https://x/my-instance","settings":{"tier":"db-f1-micro","dataDiskSizeGb":"10","edition":"ENTERPRISE"},"ipAddresses":[{"type":"PRIMARY","ipAddress":"34.41.10.20"}]}`)
	creatingInstance := []byte(`{"name":"my-instance","state":"PENDING_CREATE"}`)

	t.Run("starts the instance and schedules a poll that emits on RUNNABLE", func(t *testing.T) {
		var postURL string
		var postBody map[string]any
		instanceState := creatingInstance
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postURL = url
				postBody, _ = body.(map[string]any)
				return []byte(`{"name":"op-123","status":"PENDING","targetId":"my-instance"}`), nil
			},
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return instanceState, nil
			},
		}
		withFactory(mc)

		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "my-instance", "databaseVersion": "POSTGRES_16",
				"region": "us-central1", "tier": "db-f1-micro", "diskSizeGb": 10, "edition": "ENTERPRISE",
			},
			Metadata:       metadata,
			Requests:       requests,
			ExecutionState: state,
		})
		require.NoError(t, err)
		// Execute starts the insert and schedules a poll rather than emitting.
		assert.Equal(t, pollHookName, requests.Action)
		assert.False(t, state.Passed)
		assert.Contains(t, postURL, "/projects/my-project/instances")
		assert.Equal(t, "my-instance", postBody["name"])

		// First poll: still creating -> re-schedules, nothing emitted yet.
		reqs := &contexts.RequestContext{}
		require.NoError(t, c.HandleHook(core.ActionHookContext{Name: pollHookName, Metadata: metadata, Requests: reqs, ExecutionState: state}))
		assert.Equal(t, pollHookName, reqs.Action)
		assert.Empty(t, state.Payloads)

		// Next poll: RUNNABLE -> emits the instance details.
		instanceState = runnableInstance
		require.NoError(t, c.HandleHook(core.ActionHookContext{Name: pollHookName, Metadata: metadata, Requests: &contexts.RequestContext{}, ExecutionState: state}))
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.cloudsql.instance", state.Type)
		data := firstData(t, state)
		assert.Equal(t, "RUNNABLE", data["state"])
		assert.Equal(t, "34.41.10.20", data["ipAddress"])
	})

	t.Run("fails the execution when the instance enters FAILED", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				return []byte(`{"name":"op-9","status":"PENDING","targetId":"my-instance"}`), nil
			},
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return []byte(`{"name":"my-instance","state":"FAILED"}`), nil
			},
		}
		withFactory(mc)

		metadata := &contexts.MetadataContext{}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "my-instance", "databaseVersion": "POSTGRES_16",
				"region": "us-central1", "tier": "db-f1-micro",
			},
			Metadata:       metadata,
			Requests:       &contexts.RequestContext{},
			ExecutionState: state,
		}))

		// The poll must fail the execution (not return an error, which would roll
		// back the request and leave the run in progress forever).
		err := c.HandleHook(core.ActionHookContext{Name: pollHookName, Metadata: metadata, Requests: &contexts.RequestContext{}, ExecutionState: state})
		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "entered state FAILED")
	})

	t.Run("clamps a sub-minimum disk size up to the minimum", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postBody, _ = body.(map[string]any)
				return []byte(`{"name":"op-1","status":"PENDING","targetId":"my-instance"}`), nil
			},
		}
		withFactory(mc)

		requests := &contexts.RequestContext{}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "my-instance", "databaseVersion": "POSTGRES_16",
				"region": "us-central1", "tier": "db-f1-micro", "diskSizeGb": 5,
			},
			Metadata:       &contexts.MetadataContext{},
			Requests:       requests,
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.Equal(t, pollHookName, requests.Action)
		// 5 GB is below Cloud SQL's 10 GB minimum, so it is clamped rather than
		// forwarded to the API.
		settings := postBody["settings"].(map[string]any)
		assert.Equal(t, "10", settings["dataDiskSizeGb"])
	})

	t.Run("maps the security and SSL settings into the request body", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postBody, _ = body.(map[string]any)
				return []byte(`{"name":"op-2","status":"PENDING","targetId":"my-instance"}`), nil
			},
		}
		withFactory(mc)

		requests := &contexts.RequestContext{}
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "my-instance", "databaseVersion": "POSTGRES_16",
				"region": "us-central1", "tier": "db-f1-micro",
				"dataDiskType": "PD_HDD", "availabilityType": "REGIONAL",
				"automatedBackups": true,
				"publicIp":         false, "sslMode": "ENCRYPTED_ONLY",
				"authorizedNetworks": []any{"203.0.113.0/24", "  ", "198.51.100.7/32"},
				"deletionProtection": true,
				"labels": []any{
					map[string]any{"key": "env", "value": "staging"},
					map[string]any{"key": "  ", "value": "ignored"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			Requests:       requests,
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.Equal(t, pollHookName, requests.Action)

		settings := postBody["settings"].(map[string]any)
		assert.Equal(t, true, settings["deletionProtectionEnabled"])
		assert.Equal(t, "PD_HDD", settings["dataDiskType"])
		assert.Equal(t, "REGIONAL", settings["availabilityType"])
		backups := settings["backupConfiguration"].(map[string]any)
		assert.Equal(t, true, backups["enabled"])
		// Blank-key labels are dropped; the rest become userLabels.
		labels := settings["userLabels"].(map[string]string)
		assert.Equal(t, map[string]string{"env": "staging"}, labels)
		ipConfig := settings["ipConfiguration"].(map[string]any)
		assert.Equal(t, false, ipConfig["ipv4Enabled"])
		assert.Equal(t, "ENCRYPTED_ONLY", ipConfig["sslMode"])
		// Blank CIDRs are dropped; the rest become ACL entries.
		nets := ipConfig["authorizedNetworks"].([]map[string]any)
		require.Len(t, nets, 2)
		assert.Equal(t, "203.0.113.0/24", nets[0]["value"])
		assert.Equal(t, "198.51.100.7/32", nets[1]["value"])
	})

	t.Run("defaults public IP on and omits unset security settings", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postBody, _ = body.(map[string]any)
				return []byte(`{"name":"op-3","status":"PENDING","targetId":"my-instance"}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "my-instance", "databaseVersion": "POSTGRES_16",
				"region": "us-central1", "tier": "db-f1-micro",
			},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
			ExecutionState: state,
		})
		require.NoError(t, err)
		settings := postBody["settings"].(map[string]any)
		ipConfig := settings["ipConfiguration"].(map[string]any)
		assert.Equal(t, true, ipConfig["ipv4Enabled"])
		_, hasSSL := ipConfig["sslMode"]
		assert.False(t, hasSSL)
		_, hasNets := ipConfig["authorizedNetworks"]
		assert.False(t, hasNets)
		_, hasDP := settings["deletionProtectionEnabled"]
		assert.False(t, hasDP)
		_, hasBackups := settings["backupConfiguration"]
		assert.False(t, hasBackups)
		_, hasLabels := settings["userLabels"]
		assert.False(t, hasLabels)
	})

	t.Run("missing name fails the execution", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"databaseVersion": "POSTGRES_16", "region": "us-central1", "tier": "db-f1-micro"},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "name is required")
	})
}
