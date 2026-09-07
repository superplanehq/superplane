package approval

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/autoapprove"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// docsPayload is an inert, low-risk change: documentation only.
func docsPayload() map[string]any {
	return map[string]any{
		"files": []any{"README.md", "docs/guide.md"},
		"sha":   "abc123",
	}
}

// codePayload is application code: mid risk, must go to a human.
func codePayload() map[string]any {
	return map[string]any{
		"files": []any{"pkg/service/handler.go"},
		"sha":   "def456",
	}
}

// gateConfig always keeps a human approver ("anyone") so that when a change is
// not auto-approved, the gate falls through to a pending human decision rather
// than completing on its own.
func gateConfig(inert bool, when string) map[string]any {
	return map[string]any{
		"items": []any{map[string]any{"type": "anyone"}},
		"autoApprove": map[string]any{
			"inertChanges": inert,
			"when":         when,
		},
	}
}

func baseCtx(data any, config map[string]any, state *contexts.ExecutionStateContext) core.ExecutionContext {
	return core.ExecutionContext{
		Data:           data,
		Configuration:  config,
		NodeMetadata:   &contexts.MetadataContext{},
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: state,
		Auth:           &contexts.AuthContext{},
	}
}

func TestExecute_AutoApprovesInertChange_WhenEnabled(t *testing.T) {
	t.Setenv(autoApproveInstallEnabledEnv, "yes")
	state := &contexts.ExecutionStateContext{}

	require.NoError(t, (&Approval{}).Execute(baseCtx(docsPayload(), gateConfig(true, ""), state)))

	assert.True(t, state.Finished, "inert change should finish the gate")
	assert.Equal(t, ChannelApproved, state.Channel, "inert change should clear on the approved channel")
}

func TestExecute_EscalatesCode_EvenWhenEnabled(t *testing.T) {
	t.Setenv(autoApproveInstallEnabledEnv, "yes")
	state := &contexts.ExecutionStateContext{}

	require.NoError(t, (&Approval{}).Execute(baseCtx(codePayload(), gateConfig(true, ""), state)))

	assert.False(t, state.Finished, "code change must not be auto-approved; it falls through to a human")
}

func TestExecute_DoesNotAutoApprove_WhenInstallSwitchOff(t *testing.T) {
	// AUTO_APPROVE_INERT_ENABLED is unset: the instance switch is off.
	state := &contexts.ExecutionStateContext{}

	require.NoError(t, (&Approval{}).Execute(baseCtx(docsPayload(), gateConfig(true, ""), state)))

	assert.False(t, state.Finished, "with the install switch off, even inert changes need a human")
}

func TestExecute_GuardExpressionBlocksAutoApproval(t *testing.T) {
	t.Setenv(autoApproveInstallEnabledEnv, "yes")
	state := &contexts.ExecutionStateContext{}

	ctx := baseCtx(docsPayload(), gateConfig(true, "false"), state)
	ctx.Expressions = &contexts.ExpressionContext{Output: false}

	require.NoError(t, (&Approval{}).Execute(ctx))

	assert.False(t, state.Finished, "a guard that evaluates false must block auto-approval")
}

func TestExecute_NoPolicy_LeavesGateManual(t *testing.T) {
	state := &contexts.ExecutionStateContext{}

	config := map[string]any{"items": []any{map[string]any{"type": "anyone"}}}
	require.NoError(t, (&Approval{}).Execute(baseCtx(docsPayload(), config, state)))

	assert.False(t, state.Finished, "a gate with an approver and no policy still waits for a person")
}

// fakeCanvasMemory captures ledger writes so a test can assert what was recorded.
type fakeCanvasMemory struct {
	added []ledgerWrite
}

type ledgerWrite struct {
	namespace string
	value     any
}

func (f *fakeCanvasMemory) Add(namespace string, values any) error {
	f.added = append(f.added, ledgerWrite{namespace, values})
	return nil
}

func (f *fakeCanvasMemory) Find(string, map[string]any) ([]any, error)    { return nil, nil }
func (f *fakeCanvasMemory) FindFirst(string, map[string]any) (any, error) { return nil, nil }

func TestExecute_WritesLedgerAndAutoRecord_OnAutoApproval(t *testing.T) {
	t.Setenv(autoApproveInstallEnabledEnv, "yes")
	state := &contexts.ExecutionStateContext{}
	mem := &fakeCanvasMemory{}
	ctx := baseCtx(docsPayload(), gateConfig(true, ""), state)
	ctx.CanvasMemory = mem

	require.NoError(t, (&Approval{}).Execute(ctx))

	require.True(t, state.Finished)
	require.Equal(t, ChannelApproved, state.Channel)

	require.Len(t, state.Payloads, 1)
	wrapped, ok := state.Payloads[0].(map[string]any)
	require.True(t, ok, "expected a wrapped payload")
	md, ok := wrapped["data"].(*Metadata)
	require.True(t, ok, "expected a *Metadata payload")
	require.Len(t, md.Records, 1)
	assert.Equal(t, ItemTypeAuto, md.Records[0].Type)
	assert.Equal(t, StateApproved, md.Records[0].State)
	require.NotNil(t, md.Records[0].Approval)
	assert.NotEmpty(t, md.Records[0].Approval.Comment)

	require.Len(t, mem.added, 1, "exactly one decision should be recorded")
	assert.Equal(t, autoapprove.LedgerNamespace, mem.added[0].namespace)
	rec, ok := mem.added[0].value.(autoapprove.DecisionRecord)
	require.True(t, ok, "expected a DecisionRecord in the ledger")
	assert.Equal(t, autoapprove.OutcomeAuto, rec.Decision)
	assert.Equal(t, string(autoapprove.CategoryDocs), rec.Category)
	assert.Equal(t, "abc123", rec.Correlation)
}

func TestExecute_RecordsEscalationInLedger(t *testing.T) {
	t.Setenv(autoApproveInstallEnabledEnv, "yes")
	state := &contexts.ExecutionStateContext{}
	mem := &fakeCanvasMemory{}
	ctx := baseCtx(codePayload(), gateConfig(true, ""), state)
	ctx.CanvasMemory = mem

	require.NoError(t, (&Approval{}).Execute(ctx))

	assert.False(t, state.Finished, "code escalates to a human")
	require.Len(t, mem.added, 1)
	rec, ok := mem.added[0].value.(autoapprove.DecisionRecord)
	require.True(t, ok)
	assert.Equal(t, autoapprove.OutcomeHumanRequired, rec.Decision)
}

func TestExecute_NonBooleanGuardEscalates_DoesNotError(t *testing.T) {
	t.Setenv(autoApproveInstallEnabledEnv, "yes")
	state := &contexts.ExecutionStateContext{}
	ctx := baseCtx(docsPayload(), gateConfig(true, "someExpr"), state)
	ctx.Expressions = &contexts.ExpressionContext{Output: "not a bool"}

	require.NoError(t, (&Approval{}).Execute(ctx), "a non-boolean guard must not error the node")
	assert.False(t, state.Finished, "a non-boolean guard escalates to a human")
}

func TestExecute_GuardErrorEscalates_DoesNotError(t *testing.T) {
	t.Setenv(autoApproveInstallEnabledEnv, "yes")
	state := &contexts.ExecutionStateContext{}
	ctx := baseCtx(docsPayload(), gateConfig(true, "someExpr"), state)
	ctx.Expressions = &contexts.ExpressionContext{Error: errors.New("boom")}

	require.NoError(t, (&Approval{}).Execute(ctx), "a guard eval error must not error the node")
	assert.False(t, state.Finished, "a guard error escalates to a human")
}
