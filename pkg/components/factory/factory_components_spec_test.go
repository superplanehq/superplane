package factory

import (
	"errors"
	"testing"
	"time"

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

	reportCheckCalls  int
	reportCheckParams core.ReportWorkOrderCheckParams
	reportCheckResult *core.WorkOrderCheck
	reportCheckErr    error

	setStatusNoteCalls  int
	setStatusNoteParams core.SetWorkOrderStatusNoteParams
	setStatusNoteResult *core.WorkOrderStatusNote
	setStatusNoteErr    error

	lastActivityParams core.AddPullRequestActivityParams
	activityResult     *core.PullRequestActivityResult
	activityErr        error

	lastUpdateParams core.UpdatePullRequestActivityParams
	updateResult     *core.PullRequestActivityResult
	updateErr        error
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

func (f *fakeFactoryContext) AddPullRequest(_ core.AddPullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *fakeFactoryContext) UpdatePullRequest(_ core.UpdatePullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *fakeFactoryContext) FindPullRequest(_ core.FindPullRequestParams) (*core.PullRequestMatch, error) {
	return nil, nil
}

func (f *fakeFactoryContext) AddPullRequestActivity(params core.AddPullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	f.lastActivityParams = params
	if f.activityErr != nil {
		return nil, f.activityErr
	}
	if f.activityResult != nil {
		return f.activityResult, nil
	}
	return &core.PullRequestActivityResult{
		PullRequest: &core.PullRequest{ID: params.PullRequestID, Number: 42},
		WorkOrder:   &core.WorkOrder{ID: "wo-1", Number: 123, Key: "SP-123"},
		Activity:    &core.PullRequestActivity{Description: params.Description, Access: core.PullRequestActivityAccessConcurrent, State: "active"},
		Outcome:     core.PullRequestActivityOutcomeReady,
	}, nil
}

func (f *fakeFactoryContext) UpdatePullRequestActivity(params core.UpdatePullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	f.lastUpdateParams = params
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.updateResult != nil {
		return f.updateResult, nil
	}
	description := ""
	if params.Description != nil {
		description = *params.Description
	}
	return &core.PullRequestActivityResult{
		PullRequest: &core.PullRequest{ID: "pr-1", Number: 42},
		WorkOrder:   &core.WorkOrder{ID: "wo-1", Number: 123, Key: "SP-123"},
		Activity:    &core.PullRequestActivity{Description: description, Access: params.Access, State: "active"},
		Outcome:     core.PullRequestActivityOutcomeReady,
	}, nil
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
			"artifactType": "markdown",
			"body":         "notes",
		})
		if err == nil {
			t.Fatal("expected error for missing orderId")
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

	t.Run("url field is visible for branch and link", func(t *testing.T) {
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
		for _, want := range []string{"branch", "link"} {
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

	t.Run("accepts link with free-form data entries", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "link",
			"url":          "https://preview.example.com/pr-1",
			"data": []any{
				map[string]any{"name": "provider", "value": "github"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

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

func TestPrArtifactLifecycleFields_SharedByAddAndUpdate(t *testing.T) {
	addNames := fieldNames((&AddPullRequest{}).Configuration())
	updateNames := fieldNames((&UpdatePullRequest{}).Configuration())
	for _, name := range []string{"state", "merged", "draft"} {
		assert.Contains(t, addNames, name)
		assert.Contains(t, updateNames, name)
	}
}

func TestAddPullRequestActivity_Execute(t *testing.T) {
	component := &AddPullRequestActivity{}

	t.Run("passes description to the factory context", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"pullRequestId": "pr-1",
				"description":   "Please add tests for the retry path.",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, "pr-1", factoryCtx.lastActivityParams.PullRequestID)
		assert.Equal(t, "Please add tests for the retry path.", factoryCtx.lastActivityParams.Description)
		assert.Equal(t, "pullRequest.activityAdded", stateCtx.Type)
		require.Len(t, stateCtx.Payloads, 1)
		payload, ok := stateCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Please add tests for the retry path.", data["description"])
	})

	t.Run("passes without output when another activity owns the revision", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{activityErr: core.ErrPullRequestActivityAlreadyActive}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"pullRequestId": "pr-1", "revision": "abc", "access": "concurrent"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Empty(t, stateCtx.Channel)
		assert.Empty(t, stateCtx.Payloads)
	})

	t.Run("waits when exclusive access is not available", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{activityResult: &core.PullRequestActivityResult{
			PullRequest: &core.PullRequest{ID: "pr-1"},
			WorkOrder:   &core.WorkOrder{ID: "wo-1"},
			Outcome:     core.PullRequestActivityOutcomeWaiting,
		}}
		stateCtx := &contexts.ExecutionStateContext{}
		requestCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"pullRequestId": "pr-1", "access": "exclusive"},
			ExecutionState: stateCtx,
			Requests:       requestCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.False(t, stateCtx.Finished)
		assert.Equal(t, acquireAccessHookName, requestCtx.Action)
		assert.GreaterOrEqual(t, requestCtx.Duration, 10*time.Second)
	})
}

func TestUpdatePullRequestActivity_Execute(t *testing.T) {
	component := &UpdatePullRequestActivity{}

	t.Run("updates the description", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"description": "Checks passed on d1209da"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, factoryCtx.lastUpdateParams.Description)
		assert.Equal(t, "Checks passed on d1209da", *factoryCtx.lastUpdateParams.Description)
		assert.Equal(t, "pullRequest.activityUpdated", stateCtx.Type)
	})

	t.Run("emits limitReached when the handler is at the attempt limit", func(t *testing.T) {
		limit := 3
		factoryCtx := &fakeFactoryContext{updateResult: &core.PullRequestActivityResult{
			PullRequest: &core.PullRequest{ID: "pr-1"},
			WorkOrder:   &core.WorkOrder{ID: "wo-1"},
			Activity:    &core.PullRequestActivity{AttemptLimit: &limit},
			Outcome:     core.PullRequestActivityOutcomeLimitReached,
		}}
		stateCtx := &contexts.ExecutionStateContext{}
		runs := &contexts.RunExecutionContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"access": "exclusive"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
			Runs:           runs,
		})
		require.NoError(t, err)
		assert.Equal(t, core.PullRequestActivityOutcomeLimitReached, stateCtx.Channel)
		assert.Equal(t, []string{"Automatic fixes paused after 3 attempts"}, runs.AddErrorCalls)
	})

	t.Run("emits default when only the description is updated after the limit", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{updateResult: &core.PullRequestActivityResult{
			PullRequest: &core.PullRequest{ID: "pr-1"},
			WorkOrder:   &core.WorkOrder{ID: "wo-1"},
			Outcome:     core.PullRequestActivityOutcomeLimitReached,
		}}
		stateCtx := &contexts.ExecutionStateContext{}
		runs := &contexts.RunExecutionContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"description": "Automatic fixes paused after 3 attempts"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
			Runs:           runs,
		})
		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Empty(t, runs.AddErrorCalls)
	})
}

func TestPullRequestActivity_OutputChannels(t *testing.T) {
	channels := pullRequestActivityChannels()
	names := make([]string, 0, len(channels))
	for _, channel := range channels {
		names = append(names, channel.Name)
	}
	assert.Equal(t, []string{core.DefaultOutputChannel.Name, core.PullRequestActivityOutcomeLimitReached}, names)
}

func TestAddPullRequestActivity_ValidatesConfiguration(t *testing.T) {
	fields := (&AddPullRequestActivity{}).Configuration()

	t.Run("requires pullRequestId", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{})
		require.Error(t, err)
	})

	t.Run("accepts an optional description", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"pullRequestId": "pr-1",
			"description":   "Please add tests for the retry path.",
		})
		require.NoError(t, err)
	})
}

func fieldNames(fields []configuration.Field) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}
