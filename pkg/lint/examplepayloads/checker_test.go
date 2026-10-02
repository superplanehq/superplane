package examplepayloads

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	issues, err := Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, issue := range issues {
		t.Errorf("%s", issue.String())
	}
}

// emitSpecsFor parses a single-file package and returns the Emit specs
// collected for the action it declares.
func emitSpecsFor(t *testing.T, body string) []emitSpec {
	t.Helper()

	source := `package example

type Widget struct{}

func (w *Widget) Name() string { return "widget" }

func (w *Widget) ExampleOutput() map[string]any { return nil }

func (w *Widget) Execute(ctx Context) error {
` + body + `
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "widget.go", source, parser.AllErrors)
	require.NoError(t, err)

	analyzer := newPackageAnalyzer(&loadedPackage{
		ImportPath: "example",
		Dir:        ".",
		Fset:       fset,
		Syntax:     []*ast.File{file},
	})

	issues, specs, _ := analyzer.collectEmitSpecs()
	require.Empty(t, issues)

	return specs[exampleKey(nodeKindAction, "widget")]
}

// emitSpecsForSource is emitSpecsFor for sources that need declarations outside
// the component method, such as package-level emitting helpers.
func emitSpecsForSource(t *testing.T, source string) ([]emitSpec, []emitSpec) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "widget.go", source, parser.AllErrors)
	require.NoError(t, err)

	analyzer := newPackageAnalyzer(&loadedPackage{
		ImportPath: "example",
		Dir:        ".",
		Fset:       fset,
		Syntax:     []*ast.File{file},
	})

	issues, specs, shared := analyzer.collectEmitSpecs()
	require.Empty(t, issues)

	return specs[exampleKey(nodeKindAction, "widget")], shared
}

const widgetPreamble = `package example

type Widget struct{}

func (w *Widget) Name() string { return "widget" }

func (w *Widget) ExampleOutput() map[string]any { return nil }
`

// An Emit written as `err = ctx.Emit(...)` is an assignment, not an expression
// or a return. Missing that shape once made the checker judge a component by a
// fraction of its Emit calls and report real fields as phantom.
func TestEmitAssignedToAVariableIsCollected(t *testing.T) {
	specs := emitSpecsFor(t, `
	var err error
	payload := map[string]any{"fromAssignment": 1}
	err = ctx.Emit("widget.done", payload)
	return err`)

	require.Len(t, specs, 1)
	assert.Contains(t, specs[0].Data.Fields, "fromAssignment")
}

func TestEmitInsideAClosureStepsDownExactness(t *testing.T) {
	specs, _ := emitSpecsForSource(t, widgetPreamble+`
func (w *Widget) Execute(ctx Context) error {
	ctx.Later(func() {
		_ = ctx.Emit("widget.done", map[string]any{"fromClosure": 1})
	})
	return ctx.Emit("widget.done", map[string]any{"id": 1})
}
`)

	require.NotEmpty(t, specs)
	for _, spec := range specs {
		assert.False(t, spec.Data.Exact, "an Emit the walker cannot attribute must disable phantom reporting")
	}
}

func TestEmitInAPackageLevelHelperIsCollectedAsShared(t *testing.T) {
	own, shared := emitSpecsForSource(t, widgetPreamble+`
func (w *Widget) Execute(ctx Context) error {
	return ctx.Emit("widget.done", map[string]any{"id": 1})
}

func emitFailure(ctx Context) error {
	return ctx.Emit("widget.done", map[string]any{"failure": 1})
}
`)

	require.Len(t, own, 1)
	require.Len(t, shared, 1)
	assert.Contains(t, shared[0].Data.Fields, "failure")
	assert.NotContains(t, own[0].Data.Fields, "failure",
		"a helper Emit is not owned by the component; it only widens the shape")
}

// A helper Emit widens the known shape, so a field only that helper sends must
// not be reported against a component that shares the payload type.
func TestSharedHelperFieldIsNotPhantom(t *testing.T) {
	example := exampleRecord{
		Name: "widget",
		Kind: nodeKindAction,
		Payload: map[string]any{
			"type": "widget.done",
			"data": map[string]any{"id": 1, "failure": "boom"},
		},
	}

	own := []emitSpec{{
		PayloadType: "widget.done",
		Data:        schema{Kind: schemaObject, Exact: true, Fields: map[string]schema{"id": {Kind: schemaNumber}}},
	}}

	shared := []emitSpec{{
		PayloadType: "widget.done",
		Data:        schema{Kind: schemaObject, Exact: true, Fields: map[string]schema{"failure": {Kind: schemaString}}},
	}}

	assert.Empty(t, validateExampleAgainstSpecs(example, own, shared))
}

// Shared specs must not widen the set of payload types a component declares, or
// unrelated components in the same package get accused of the wrong type.
func TestSharedSpecsDoNotDefineAComponentsPayloadTypes(t *testing.T) {
	example := exampleRecord{
		Name:    "widget",
		Kind:    nodeKindAction,
		Payload: map[string]any{"type": "widget.done", "data": map[string]any{}},
	}

	shared := []emitSpec{{PayloadType: "other.thing"}}

	assert.Empty(t, validateExampleAgainstSpecs(example, nil, shared),
		"a component with no Emit of its own stays unchecked")
}

func TestMergeEmitSchemasHandlesNoSpecs(t *testing.T) {
	assert.Equal(t, schemaUnknown, mergeEmitSchemas(nil).Kind)
}

func TestExemptNodeIsNotChecked(t *testing.T) {
	key := exampleKey(nodeKindAction, "widget")
	dataCheckExemptions[key] = true
	defer delete(dataCheckExemptions, key)

	example := exampleRecord{
		Name: "widget",
		Kind: nodeKindAction,
		Payload: map[string]any{
			"type": "widget.done",
			"data": map[string]any{"documentedElsewhere": true},
		},
	}

	data := schema{
		Kind:   schemaObject,
		Exact:  true,
		Fields: map[string]schema{"id": {Kind: schemaNumber}},
	}

	assert.Empty(t, validateExampleData(example, data, token.Position{}))
}

func TestKeysAddedInsideAConditionalStayInTheSchema(t *testing.T) {
	specs := emitSpecsFor(t, `
	payload := map[string]any{"pipeline": 1}
	if ctx.Ready {
		payload["workflow"] = 2
	}
	return ctx.Emit("widget.done", payload)`)

	require.Len(t, specs, 1)
	assert.True(t, specs[0].Data.Exact, "a fully readable payload stays exact")
	assert.Contains(t, specs[0].Data.Fields, "pipeline")
	assert.Contains(t, specs[0].Data.Fields, "workflow", "the branch key must survive the per-branch env clone")
}

func TestPayloadHandedToAHelperIsNotExact(t *testing.T) {
	specs := emitSpecsFor(t, `
	payload := map[string]any{"id": 1}
	enrich(ctx, payload)
	return ctx.Emit("widget.done", payload)`)

	require.Len(t, specs, 1)
	assert.False(t, specs[0].Data.Exact, "a helper may add keys the analyzer cannot see")
}

func TestPayloadWithAComputedKeyIsNotExact(t *testing.T) {
	specs := emitSpecsFor(t, `
	payload := map[string]any{"id": 1}
	payload[ctx.Key] = 2
	return ctx.Emit("widget.done", payload)`)

	require.Len(t, specs, 1)
	assert.False(t, specs[0].Data.Exact, "a computed key can add anything")
}

func TestInlinePayloadLiteralIsExact(t *testing.T) {
	specs := emitSpecsFor(t, `
	return ctx.Emit("widget.done", map[string]any{"id": 1})`)

	require.Len(t, specs, 1)
	assert.True(t, specs[0].Data.Exact)
	assert.Contains(t, specs[0].Data.Fields, "id")
}

func TestMergeEmitSchemasUnionsFields(t *testing.T) {
	merged := mergeEmitSchemas([]emitSpec{
		{Data: schema{Kind: schemaObject, Exact: true, Fields: map[string]schema{"a": {Kind: schemaNumber}}}},
		{Data: schema{Kind: schemaObject, Exact: true, Fields: map[string]schema{"b": {Kind: schemaNumber}}}},
	})

	assert.True(t, merged.Exact)
	assert.Contains(t, merged.Fields, "a")
	assert.Contains(t, merged.Fields, "b")
}

func TestMergeEmitSchemasDropsExactnessOnUnreadableCall(t *testing.T) {
	merged := mergeEmitSchemas([]emitSpec{
		{Data: schema{Kind: schemaObject, Exact: true, Fields: map[string]schema{"a": {Kind: schemaNumber}}}},
		{Data: schema{Kind: schemaUnknown}},
	})

	assert.False(t, merged.Exact, "an unreadable Emit says nothing about the full key set")
}

func TestValidateExampleDataReportsPhantomFields(t *testing.T) {
	example := exampleRecord{
		Name: "widget",
		Kind: nodeKindAction,
		Payload: map[string]any{
			"type": "widget.done",
			"data": map[string]any{"id": 1, "region": "us-east-1"},
		},
	}

	data := schema{
		Kind:   schemaObject,
		Exact:  true,
		Fields: map[string]schema{"id": {Kind: schemaNumber}},
	}

	issues := validateExampleData(example, data, token.Position{})

	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "data.region")
}

func TestValidateExampleDataAcceptsAMatchingExample(t *testing.T) {
	example := exampleRecord{
		Name: "widget",
		Kind: nodeKindAction,
		Payload: map[string]any{
			"type": "widget.done",
			"data": map[string]any{"id": 1},
		},
	}

	data := schema{
		Kind:   schemaObject,
		Exact:  true,
		Fields: map[string]schema{"id": {Kind: schemaNumber}},
	}

	assert.Empty(t, validateExampleData(example, data, token.Position{}))
}

func TestValidateExampleDataStaysSilentWhenTheSchemaIsNotExact(t *testing.T) {
	example := exampleRecord{
		Name: "widget",
		Kind: nodeKindAction,
		Payload: map[string]any{
			"type": "widget.done",
			"data": map[string]any{"anything": true},
		},
	}

	data := schema{
		Kind:   schemaObject,
		Exact:  false,
		Fields: map[string]schema{"id": {Kind: schemaNumber}},
	}

	assert.Empty(t, validateExampleData(example, data, token.Position{}),
		"an inexact schema must never accuse a field of being phantom")
}

func TestValidateExampleAgainstSpecsMergesSitesSharingAType(t *testing.T) {
	example := exampleRecord{
		Name: "widget",
		Kind: nodeKindAction,
		Payload: map[string]any{
			"type": "widget.done",
			"data": map[string]any{"passed": true, "failed": true},
		},
	}

	specs := []emitSpec{
		{
			PayloadType: "widget.done",
			Data:        schema{Kind: schemaObject, Exact: true, Fields: map[string]schema{"passed": {Kind: schemaBool}}},
		},
		{
			PayloadType: "widget.done",
			Data:        schema{Kind: schemaObject, Exact: true, Fields: map[string]schema{"failed": {Kind: schemaBool}}},
		},
	}

	assert.Empty(t, validateExampleAgainstSpecs(example, specs, nil),
		"a field emitted by any site sharing the type is not phantom")
}

func TestValidateExampleAgainstSpecsReportsAnUnknownType(t *testing.T) {
	example := exampleRecord{
		Name: "widget",
		Kind: nodeKindAction,
		Payload: map[string]any{
			"type": "widget.missing",
			"data": map[string]any{},
		},
	}

	specs := []emitSpec{{PayloadType: "widget.done"}}

	issues := validateExampleAgainstSpecs(example, specs, nil)

	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "does not match any Emit(...) payload type")
}
