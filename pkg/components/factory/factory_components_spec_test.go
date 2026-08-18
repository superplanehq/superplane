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
		})
		if err == nil {
			t.Fatal("expected error for branch without name")
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

	t.Run("accepts branch with name", func(t *testing.T) {
		err := configuration.ValidateConfiguration(fields, map[string]any{
			"orderId":      "{{ order().id }}",
			"artifactType": "branch",
			"name":         "feature/refund-retry",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("accepts branch with name and url", func(t *testing.T) {
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
		assert.Equal(t, map[string]any{"state": "merged", "title": "Retitled PR"}, factoryCtx.updateArtifactParams.Data)
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
		assert.Equal(t, map[string]any{"state": "draft"}, factoryCtx.updateArtifactParams.Data)
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

	// Velocity relies on the artifact table's merged_at column. The model
	// falls back to `now` when the canvas only sends `state`, but a
	// canvas that has GitHub's real timestamp should get to pass it
	// through so history is honest.
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
	data := buildArtifactData(AddWorkOrderArtifactConfiguration{
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
	data := buildArtifactData(AddWorkOrderArtifactConfiguration{
		ArtifactType: "pr",
		URL:          "https://github.com/example/repo/pull/9",
		State:        "draft",
	})

	if got := data["state"]; got != "draft" {
		t.Fatalf("expected state=draft, got %v", got)
	}
}

func TestBuildArtifactData_SkipsBlankTypedInputs(t *testing.T) {
	data := buildArtifactData(AddWorkOrderArtifactConfiguration{
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
