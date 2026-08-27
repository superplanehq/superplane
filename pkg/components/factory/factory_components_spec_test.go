package factory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// fakeFactoryContext lets Execute-level tests drive `core.FactoryContext`
// without spinning up a database. Only UpdateWorkOrderStatus and
// FindWorkOrder are wired up; the rest of the interface returns zero values.
type fakeFactoryContext struct {
	statusCalls int
	nextChanged bool
	nextErr     error
	returnOrder *core.WorkOrder

	lastStatusParams core.UpdateWorkOrderStatusParams

	findCalls  int
	findParams core.FindWorkOrderParams
	findOrder  *core.WorkOrder
	findErr    error

	updateArtifactCalls  int
	updateArtifactParams core.UpdateWorkOrderArtifactParams
	updateArtifactResult *core.WorkOrderArtifact
	updateArtifactErr    error

	reportCheckCalls  int
	reportCheckParams core.ReportWorkOrderCheckParams
	reportCheckResult *core.WorkOrderCheck
	reportCheckErr    error

	setStatusNoteCalls  int
	setStatusNoteParams core.SetWorkOrderStatusNoteParams
	setStatusNoteResult *core.WorkOrderStatusNote
	setStatusNoteErr    error
}

func (f *fakeFactoryContext) CreateWorkOrder(_ core.WorkOrderParams) (*core.WorkOrder, error) {
	return nil, nil
}

func (f *fakeFactoryContext) FindWorkOrder(params core.FindWorkOrderParams) (*core.WorkOrder, error) {
	f.findCalls++
	f.findParams = params
	return f.findOrder, f.findErr
}

func (f *fakeFactoryContext) UpdateWorkOrderStatus(params core.UpdateWorkOrderStatusParams) (*core.WorkOrder, bool, error) {
	f.statusCalls++
	f.lastStatusParams = params
	return f.returnOrder, f.nextChanged, f.nextErr
}

func (f *fakeFactoryContext) AddWorkOrderComment(_ core.AddWorkOrderCommentParams) error {
	return nil
}

func (f *fakeFactoryContext) AddWorkOrderArtifact(_ core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	return nil, nil
}

func (f *fakeFactoryContext) UpdateWorkOrderArtifact(params core.UpdateWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	f.updateArtifactCalls++
	f.updateArtifactParams = params
	return f.updateArtifactResult, f.updateArtifactErr
}

func (f *fakeFactoryContext) ReportWorkOrderCheck(params core.ReportWorkOrderCheckParams) (*core.WorkOrderCheck, error) {
	f.reportCheckCalls++
	f.reportCheckParams = params
	return f.reportCheckResult, f.reportCheckErr
}

func (f *fakeFactoryContext) SetWorkOrderStatusNote(params core.SetWorkOrderStatusNoteParams) (*core.WorkOrderStatusNote, error) {
	f.setStatusNoteCalls++
	f.setStatusNoteParams = params
	return f.setStatusNoteResult, f.setStatusNoteErr
}

func TestUpdateWorkOrderStatus_Execute(t *testing.T) {
	component := &UpdateWorkOrderStatus{}
	workOrder := &core.WorkOrder{ID: "wo-1", Title: "t", State: "open"}

	t.Run("emits workOrder.statusUpdated on a real transition", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{nextChanged: true, returnOrder: workOrder}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"status": "open"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, factoryCtx.statusCalls)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, "workOrder.statusUpdated", stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)
	})

	// A re-run against an already-open order must not fan out a
	// phantom `workOrder.statusUpdated` — otherwise downstream nodes
	// re-fire on every replay.
	t.Run("passes silently when the transition is a no-op", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{nextChanged: false, returnOrder: workOrder}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"status": "open"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, factoryCtx.statusCalls)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Empty(t, stateCtx.Channel, "no-op must not emit on any channel")
		assert.Empty(t, stateCtx.Type)
		assert.Nil(t, stateCtx.Payloads)
	})
}

// Regression coverage for the github.onPullRequest -> close-work-order
// bug: a run with no factory_work_order_executions row must still be able
// to target a work order explicitly via `orderId`.
func TestUpdateWorkOrderStatus_Execute_PassesThroughOrderID(t *testing.T) {
	component := &UpdateWorkOrderStatus{}
	workOrder := &core.WorkOrder{ID: "wo-1", Title: "t", State: "closed"}
	factoryCtx := &fakeFactoryContext{nextChanged: true, returnOrder: workOrder}
	stateCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"orderId": "wo-1",
			"status":  "closed",
			"result":  "completed",
		},
		ExecutionState: stateCtx,
		Factory:        factoryCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, "wo-1", factoryCtx.lastStatusParams.OrderID)
}

func TestFindWorkOrder_Execute(t *testing.T) {
	component := &FindWorkOrder{}
	workOrder := &core.WorkOrder{ID: "wo-1", Title: "t", State: "open"}

	t.Run("emits workOrder.found on a match", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{findOrder: workOrder}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"by":      "id",
				"orderId": "wo-1",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, factoryCtx.findCalls)
		assert.Equal(t, "id", factoryCtx.findParams.By)
		assert.Equal(t, "wo-1", factoryCtx.findParams.OrderID)
		assert.Equal(t, FindWorkOrderChannelNameFound, stateCtx.Channel)
		assert.Equal(t, "workOrder.found", stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)
	})

	t.Run("passes through artifactKey lookups", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{findOrder: workOrder}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"by":          "artifactKey",
				"artifactKey": "https://github.com/example/repo/pull/1",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, "artifactKey", factoryCtx.findParams.By)
		assert.Equal(t, "https://github.com/example/repo/pull/1", factoryCtx.findParams.ArtifactKey)
	})

	// A PR merge (or similar) unrelated to any tracked order must not
	// red the run — the component emits on the notFound channel instead
	// of failing, so the flow can branch on it explicitly.
	t.Run("emits on the notFound channel when nothing matches", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{findErr: core.ErrWorkOrderNotFound}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"by": "id", "orderId": "missing"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, FindWorkOrderChannelNameNotFound, stateCtx.Channel)
		assert.Equal(t, "workOrder.notFound", stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)
	})

	t.Run("propagates real errors", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{findErr: errors.New("boom")}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"by": "id", "orderId": "wo-1"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.EqualError(t, err, "boom")
	})
}

func TestFindWorkOrder_ValidatesConfiguration(t *testing.T) {
	c := &FindWorkOrder{}
	fields := c.Configuration()

	t.Run("requires orderId when finding by id", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"by": "id",
		})
		if err == nil {
			t.Fatal("expected error for by=id without orderId")
		}
	})

	t.Run("requires artifactKey when finding by artifactKey", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"by": "artifactKey",
		})
		if err == nil {
			t.Fatal("expected error for by=artifactKey without artifactKey")
		}
	})

	t.Run("accepts by id with orderId", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"by":      "id",
			"orderId": "wo-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts by artifactKey with artifactKey", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"by":          "artifactKey",
			"artifactKey": "https://github.com/example/repo/pull/1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestUpdateWorkOrderStatus_ValidatesConfiguration(t *testing.T) {
	c := &UpdateWorkOrderStatus{}
	fields := c.Configuration()

	t.Run("rejects unknown status", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"status": "bogus",
		})
		if err == nil {
			t.Fatal("expected error for invalid status option")
		}
	})

	t.Run("requires result when closing", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId": "{{ order().id }}",
			"status":  "closed",
		})
		if err == nil {
			t.Fatal("expected error for closing without a result")
		}
	})

	t.Run("requires orderId", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"status": "open",
		})
		if err == nil {
			t.Fatal("expected error for missing orderId")
		}
	})

	t.Run("accepts draft status", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId": "{{ order().id }}",
			"status":  "draft",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts open status", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId": "{{ order().id }}",
			"status":  "open",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts closed with failed result", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId": "{{ order().id }}",
			"status":  "closed",
			"result":  "failed",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAddWorkOrderComment_ValidatesConfiguration(t *testing.T) {
	c := &AddWorkOrderComment{}
	fields := c.Configuration()

	t.Run("rejects missing body", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId": "{{ order().id }}",
		})
		if err == nil {
			t.Fatal("expected error for missing body")
		}
	})

	t.Run("rejects missing orderId", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"body": "hello",
		})
		if err == nil {
			t.Fatal("expected error for missing orderId")
		}
	})

	t.Run("accepts orderId and body", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId": "{{ order().id }}",
			"body":    "hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAddWorkOrderArtifact_ValidatesConfiguration(t *testing.T) {
	c := &AddWorkOrderArtifact{}
	fields := c.Configuration()

	t.Run("requires url for pr", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "pr",
		})
		if err == nil {
			t.Fatal("expected error for pr without url")
		}
	})

	t.Run("requires body for markdown", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "markdown",
		})
		if err == nil {
			t.Fatal("expected error for markdown without body")
		}
	})

	t.Run("requires name for branch", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "branch",
			"repository":   "example/repo",
		})
		if err == nil {
			t.Fatal("expected error for branch without name")
		}
	})

	t.Run("accepts branch with name and url without repository", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "branch",
			"name":         "feature/refund-retry",
			"url":          "https://github.com/example/repo/tree/feature/refund-retry",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects name-only branch on component validator", func(t *testing.T) {
		config := map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "branch",
			"name":         "feature/refund-retry",
		}
		err := configuration.ValidateConfiguration(fields, config)
		if err != nil {
			t.Fatalf("schema should still allow name-only branch: %v", err)
		}
		err = c.ValidateNodeConfiguration(config)
		if err == nil {
			t.Fatal("expected error for branch without url or repository")
		}
	})

	t.Run("component validator accepts url-only branch", func(t *testing.T) {
		err := c.ValidateNodeConfiguration(map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "branch",
			"name":         "feature/refund-retry",
			"url":          "https://github.com/example/repo/tree/feature/refund-retry",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("component validator accepts repository expression without url", func(t *testing.T) {
		err := c.ValidateNodeConfiguration(map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "branch",
			"name":         "{{ previous().result.branch }}",
			"repository":   "{{ install_params.appRepository }}",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("requires url for link", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "link",
		})
		if err == nil {
			t.Fatal("expected error for link without url")
		}
	})

	t.Run("requires orderId", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactType": "pr",
			"url":          "https://github.com/example/repo/pull/1",
		})
		if err == nil {
			t.Fatal("expected error for missing orderId")
		}
	})

	t.Run("accepts valid pr", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "pr",
			"url":          "https://github.com/example/repo/pull/1",
			"number":       "1",
			"title":        "Draft",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts markdown with body", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "markdown",
			"body":         "investigation notes",
			"title":        "Design notes",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts branch with name and repository", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "branch",
			"name":         "feature/refund-retry",
			"repository":   "example/repo",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts branch with name, repository, and url", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "branch",
			"name":         "feature/refund-retry",
			"repository":   "example/repo",
			"url":          "https://github.com/example/repo/tree/feature/refund-retry",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts valid link", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "link",
			"url":          "https://preview.example.com/pr-42",
			"title":        "Preview",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("repository field is visible only for branch", func(t *testing.T) {
		var repositoryField *configuration.Field
		for i := range fields {
			if fields[i].Name == "repository" {
				repositoryField = &fields[i]
				break
			}
		}
		if repositoryField == nil {
			t.Fatal("expected a repository field in configuration")
		}
		if len(repositoryField.RequiredConditions) != 0 {
			t.Fatalf("expected repository not to be required when URL is set, got %d required conditions", len(repositoryField.RequiredConditions))
		}
		if len(repositoryField.VisibilityConditions) != 1 {
			t.Fatalf("expected a single visibility condition for repository, got %d", len(repositoryField.VisibilityConditions))
		}
		condition := repositoryField.VisibilityConditions[0]
		if condition.Field != "artifactType" {
			t.Fatalf("expected visibility condition on artifactType, got %q", condition.Field)
		}
		if len(condition.Values) != 1 || condition.Values[0] != "branch" {
			t.Fatalf("expected repository visibility to be branch-only, got %v", condition.Values)
		}
	})

	t.Run("url field is visible for pr, branch, and link", func(t *testing.T) {
		var urlField *configuration.Field
		for i := range fields {
			if fields[i].Name == "url" {
				urlField = &fields[i]
				break
			}
		}
		if urlField == nil {
			t.Fatal("expected a url field in configuration")
		}
		if len(urlField.VisibilityConditions) != 1 {
			t.Fatalf("expected a single visibility condition for url, got %d", len(urlField.VisibilityConditions))
		}
		condition := urlField.VisibilityConditions[0]
		if condition.Field != "artifactType" {
			t.Fatalf("expected visibility condition on artifactType, got %q", condition.Field)
		}
		for _, want := range []string{"pr", "branch", "link"} {
			found := false
			for _, got := range condition.Values {
				if got == want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected url visibility to include %q, got %v", want, condition.Values)
			}
		}
	})

	t.Run("accepts pr with free-form data entries", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "pr",
			"url":          "https://github.com/example/repo/pull/1",
			"data": []any{
				map[string]any{"name": "provider", "value": "github"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestUpdateWorkOrderArtifact_Execute(t *testing.T) {
	component := &UpdateWorkOrderArtifact{}
	artifact := &core.WorkOrderArtifact{ID: "art-1", WorkOrderID: "wo-1", Type: "pr", Data: map[string]any{
		"url":   "https://github.com/example/repo/pull/1",
		"state": "merged",
	}}

	t.Run("merges state and title into the artifact resolved by key", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{updateArtifactResult: artifact}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":     "wo-1",
				"artifactKey": "https://github.com/example/repo/pull/1",
				"state":       "merged",
				"title":       "Retitled PR",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, factoryCtx.updateArtifactCalls)
		assert.Equal(t, "wo-1", factoryCtx.updateArtifactParams.OrderID)
		assert.Equal(t, "https://github.com/example/repo/pull/1", factoryCtx.updateArtifactParams.Key)
		assert.Equal(t, map[string]any{"state": "merged", "merged": true, "draft": false, "title": "Retitled PR"}, factoryCtx.updateArtifactParams.Data)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, "workOrder.artifactUpdated", stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)
	})

	t.Run("omits blank fields from the merge so they're left untouched", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{updateArtifactResult: artifact}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":     "wo-1",
				"artifactKey": "https://github.com/example/repo/pull/1",
				"state":       "draft",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"state": "draft", "merged": false, "draft": true}, factoryCtx.updateArtifactParams.Data)
	})

	t.Run("propagates errors from the factory context", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{updateArtifactErr: errors.New("boom")}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":     "wo-1",
				"artifactKey": "https://github.com/example/repo/pull/1",
				"state":       "open",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.Error(t, err)
	})

	// A `github.onPullRequest` merged event carries `{ state: "closed",
	// merged: true }`; the update must resolve to SuperPlane's `merged`
	// so the chip flips to purple without an if-node in the flow.
	t.Run("resolves merged=true (with state=closed) to SuperPlane state=merged", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{updateArtifactResult: artifact}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":     "wo-1",
				"artifactKey": "https://github.com/example/repo/pull/1",
				"state":       "closed",
				"merged":      true,
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"state": "merged", "merged": true, "draft": false}, factoryCtx.updateArtifactParams.Data)
	})

	// Flow templates resolve values to strings, so `merged: "true"` must
	// work the same as a native bool.
	t.Run("accepts a stringified merged flag from a templated input", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{updateArtifactResult: artifact}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":     "wo-1",
				"artifactKey": "https://github.com/example/repo/pull/1",
				"merged":      "true",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"state": "merged", "merged": true, "draft": false}, factoryCtx.updateArtifactParams.Data)
	})

	// Velocity relies on the artifact table's merged_at column. The model
	// falls back to now when the canvas only sends state, but a canvas
	// that has GitHub's real timestamp should pass it through.
	t.Run("forwards mergedAt and closedAt to the factory context", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{updateArtifactResult: artifact}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":     "wo-1",
				"artifactKey": "https://github.com/example/repo/pull/1",
				"state":       "merged",
				"mergedAt":    "2026-08-17T12:34:56Z",
				"closedAt":    "2026-08-17T12:00:00Z",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"state":    "merged",
			"merged":   true,
			"draft":    false,
			"mergedAt": "2026-08-17T12:34:56Z",
			"closedAt": "2026-08-17T12:00:00Z",
		}, factoryCtx.updateArtifactParams.Data)
	})
}

func TestUpdateWorkOrderArtifact_ValidatesConfiguration(t *testing.T) {
	c := &UpdateWorkOrderArtifact{}
	fields := c.Configuration()

	t.Run("requires orderId", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"artifactKey": "https://github.com/example/repo/pull/1",
		})
		if err == nil {
			t.Fatal("expected error for missing orderId")
		}
	})

	t.Run("requires artifactKey", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId": "{{ order().id }}",
		})
		if err == nil {
			t.Fatal("expected error for missing artifactKey")
		}
	})

	t.Run("accepts orderId, artifactKey, and state", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":     "{{ order().id }}",
			"artifactKey": "https://github.com/example/repo/pull/1",
			"state":       "merged",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestArtifactDataToMap_FlattensEntries(t *testing.T) {
	entries := []ArtifactDataEntry{
		{Name: "number", Value: "482"},
		{Name: "provider", Value: "github"},
		{Name: "", Value: "ignored"},
	}
	out := artifactDataToMap(entries)
	if got := out["number"]; got != "482" {
		t.Fatalf("expected number=482, got %v", got)
	}
	if got := out["provider"]; got != "github" {
		t.Fatalf("expected provider=github, got %v", got)
	}
	if len(out) != 2 {
		t.Fatalf("expected only two entries (blank names skipped), got %d", len(out))
	}
	if artifactDataToMap(nil) != nil {
		t.Fatal("expected nil map when no entries were provided")
	}
}

func TestBuildArtifactData_TypedFieldsWinOverFreeForm(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		Number:       "9",
		Title:        "Typed title",
		Data: []ArtifactDataEntry{
			{Name: "url", Value: "https://evil.example/typosquat"},
			{Name: "provider", Value: "github"},
		},
	})

	if got := data["url"]; got != "https://github.com/example/repo/pull/9" {
		t.Fatalf("expected typed url to win, got %v", got)
	}
	if got := data["provider"]; got != "github" {
		t.Fatalf("expected free-form provider to survive, got %v", got)
	}
	if got := data["number"]; got != "9" {
		t.Fatalf("expected typed number, got %v", got)
	}
	if got := data["title"]; got != "Typed title" {
		t.Fatalf("expected typed title, got %v", got)
	}
}

func TestBuildArtifactData_IncludesPrState(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		State:        "draft",
	})

	if got := data["state"]; got != "draft" {
		t.Fatalf("expected state=draft, got %v", got)
	}
	if got := data["merged"]; got != false {
		t.Fatalf("expected merged=false when state is draft, got %v", got)
	}
	if got := data["draft"]; got != true {
		t.Fatalf("expected draft=true when state is draft, got %v", got)
	}
}

func TestBuildArtifactData_IncludesMergedAndClosedAt(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		State:        "merged",
		MergedAt:     "2026-08-17T12:34:56Z",
		ClosedAt:     "2026-08-17T12:00:00Z",
	})

	if got := data["mergedAt"]; got != "2026-08-17T12:34:56Z" {
		t.Fatalf("expected mergedAt to pass through, got %v", got)
	}
	if got := data["closedAt"]; got != "2026-08-17T12:00:00Z" {
		t.Fatalf("expected closedAt to pass through, got %v", got)
	}
}

// A `github.onPullRequest` merged event carries `{ state: "closed",
// merged: true }`; the artifact must persist as merged so the chip
// renders purple, not red.
func TestBuildArtifactData_MergedFlagWinsOverStateClosed(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		State:        "closed",
		Merged:       true,
	})

	if got := data["state"]; got != "merged" {
		t.Fatalf("expected state=merged, got %v", got)
	}
}

// Flow templates resolve values to strings; the `Merged` field must
// accept "true" so a caller doesn't need a boolean cast in the expression.
func TestBuildArtifactData_MergedFlagAcceptsStringTrue(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		Merged:       "true",
	})

	if got := data["state"]; got != "merged" {
		t.Fatalf("expected state=merged when merged=\"true\", got %v", got)
	}
}

// GitHub draft PRs stay `state: "open"`; without picking up the `draft`
// flag the chip would render green.
func TestBuildArtifactData_DraftFlagRendersAsDraftWhenNotMerged(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		State:        "open",
		Draft:        true,
	})

	if got := data["state"]; got != "draft" {
		t.Fatalf("expected state=draft, got %v", got)
	}
}

// A merged PR that once was a draft must not flip back to draft on the
// next redisplay.
func TestBuildArtifactData_MergedBeatsDraft(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		Merged:       true,
		Draft:        true,
	})

	if got := data["state"]; got != "merged" {
		t.Fatalf("expected merged to beat draft, got %v", got)
	}
}

// resolvePrArtifactState is the small state-machine both the add and the
// update components rely on; test it directly so intent is unambiguous
// and doesn't drift from the frontend's `extractPrArtifactState`.
func TestResolvePrArtifactState_Precedence(t *testing.T) {
	cases := []struct {
		name   string
		state  any
		merged any
		draft  any
		want   string
	}{
		{"empty inputs stay empty", nil, nil, nil, ""},
		{"explicit open passes through", "open", nil, nil, "open"},
		{"case-insensitive", "MERGED", nil, nil, "merged"},
		{"trimmed whitespace", "  draft  ", nil, nil, "draft"},
		{"merged bool wins over closed state", "closed", true, nil, "merged"},
		{"merged bool wins over open state", "open", true, nil, "merged"},
		{"merged string wins", nil, "true", nil, "merged"},
		{"draft bool sets draft when not merged", "open", nil, true, "draft"},
		{"draft bool ignored when merged", nil, true, true, "merged"},
		{"non-string state is treated as absent", 42, nil, nil, ""},
		{"merged false vetoes leftover state merged", "merged", false, nil, ""},
		{"draft false vetoes leftover state draft", "draft", nil, false, ""},
		{"merged false keeps closed", "closed", false, nil, "closed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolvePrArtifactState(tc.state, tc.merged, tc.draft)
			if got != tc.want {
				t.Fatalf("resolvePrArtifactState(%v,%v,%v) = %q, want %q", tc.state, tc.merged, tc.draft, got, tc.want)
			}
		})
	}
}

func TestBuildArtifactData_IgnoresPrLifecycleOnNonPr(t *testing.T) {
	// Switching a node from PR to branch/markdown can leave the sticky
	// `state` default (and leftover merged/draft flags) in the config.
	// Those fields are PR-only and must not be written — or reject the attach.
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "branch",
		Name:         "feature/refund-retry",
		Repository:   "example/repo",
		State:        "open",
		Merged:       true,
		Draft:        true,
	})
	if _, ok := data["state"]; ok {
		t.Fatal("expected leftover PR state not to be stored on a branch artifact")
	}
	if _, ok := data["merged"]; ok {
		t.Fatal("expected leftover merged flag not to be stored on a branch artifact")
	}
	if _, ok := data["draft"]; ok {
		t.Fatal("expected leftover draft flag not to be stored on a branch artifact")
	}
}

func TestBuildArtifactData_DoesNotRejectInvalidStateOnMarkdown(t *testing.T) {
	data, err := buildArtifactData(AddWorkOrderArtifactConfiguration{
		ArtifactType: "markdown",
		Body:         "note body",
		State:        "in_review",
	})
	require.NoError(t, err)
	if _, ok := data["state"]; ok {
		t.Fatal("expected leftover PR state not to be stored on a markdown artifact")
	}
}

func TestBuildArtifactData_SkipsBlankTypedInputs(t *testing.T) {
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "markdown",
		Body:         "note body",
	})

	if got := data["body"]; got != "note body" {
		t.Fatalf("expected body to be set, got %v", got)
	}
	if _, ok := data["title"]; ok {
		t.Fatal("expected blank title to be skipped")
	}
	if _, ok := data["url"]; ok {
		t.Fatal("expected blank url to be skipped")
	}
	if _, ok := data["state"]; ok {
		t.Fatal("expected blank state to be skipped")
	}
}

func mustBuildArtifactData(t *testing.T, config AddWorkOrderArtifactConfiguration) map[string]any {
	t.Helper()
	data, err := buildArtifactData(config)
	require.NoError(t, err)
	return data
}

func TestBuildArtifactData_RejectsInvalidResolvedState(t *testing.T) {
	_, err := buildArtifactData(AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		State:        "in_review",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pull request state")
}

func TestBuildArtifactData_CanonicalFlagsOverwriteFreeFormMerged(t *testing.T) {
	// A leftover `merged: true` in free-form metadata must not outrank
	// an explicit SuperPlane `state: open` after resolve — otherwise the
	// chip stays purple on the next page load.
	data := mustBuildArtifactData(t, AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		State:        "open",
		Data:         []ArtifactDataEntry{{Name: "merged", Value: "true"}},
	})

	assert.Equal(t, "open", data["state"])
	assert.Equal(t, false, data["merged"])
	assert.Equal(t, false, data["draft"])
}

func TestPrArtifactStateUpdates_ClearsStaleMergedFlag(t *testing.T) {
	updates, err := prArtifactStateUpdates(nil, false, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"merged": false}, updates)
}

func TestPrArtifactStateUpdates_DoesNotPersistVetoedMergedState(t *testing.T) {
	// Incoming `state: merged` + `merged: false` must not write `state: merged`
	// back — that leftover is what kept the chip purple.
	updates, err := prArtifactStateUpdates("merged", false, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"merged": false}, updates)
}

func TestUpdateWorkOrderArtifact_Execute_RejectsInvalidState(t *testing.T) {
	component := &UpdateWorkOrderArtifact{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"orderId":     "wo-1",
			"artifactKey": "https://github.com/example/repo/pull/1",
			"state":       "in_review",
		},
		ExecutionState: &contexts.ExecutionStateContext{},
		Factory:        &fakeFactoryContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pull request state")
}

func TestPrArtifactLifecycleFields_SharedByAddAndUpdate(t *testing.T) {
	addNames := fieldNames((&AddWorkOrderArtifact{}).Configuration())
	updateNames := fieldNames((&UpdateWorkOrderArtifact{}).Configuration())
	for _, name := range []string{"state", "merged", "draft"} {
		assert.Contains(t, addNames, name)
		assert.Contains(t, updateNames, name)
	}
}

func fieldNames(fields []configuration.Field) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}
