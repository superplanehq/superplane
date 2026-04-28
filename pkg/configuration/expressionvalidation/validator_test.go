package expressionvalidation

import (
	"strings"
	"testing"
)

type exprCase struct {
	name       string
	raw        string
	knownNames []string
	wantErr    string // empty == valid; otherwise substring expected in err.Error()
}

func runExprCases(t *testing.T, group string, cases []exprCase) {
	t.Helper()
	t.Run(group, func(t *testing.T) {
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				known := make(map[string]struct{}, len(tc.knownNames))
				for _, n := range tc.knownNames {
					known[n] = struct{}{}
				}
				err := ValidateExpression(tc.raw, known)
				if tc.wantErr == "" {
					if err != nil {
						t.Fatalf("expected no error, got: %v", err)
					}
					return
				}
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
				}
			})
		}
	})
}

func TestValidateExpression_Valid(t *testing.T) {
	runExprCases(t, "valid", []exprCase{
		{name: "bracket node ref", raw: `$['Build'].artifact`, knownNames: []string{"Build"}},
		{name: "name with space and percent", raw: `$['Deploy 10%'].status`, knownNames: []string{"Deploy 10%"}},
		{name: "root call", raw: `root().data.ref`},
		{name: "previous no args", raw: `previous()`},
		{name: "previous with int", raw: `previous(2)`},
		{name: "memory find", raw: `memory.find('users', {id: 1})`},
		{name: "memory findFirst", raw: `memory.findFirst('users', {id: 1})`},
		{name: "string builtins chain", raw: `lower(trim($['Build'].name))`, knownNames: []string{"Build"}},
		{name: "now builtin", raw: `now()`},
		{name: "date builtin", raw: `date('2026-01-01')`},
		{name: "type conversion", raw: `int($['Build'].count) + 1`, knownNames: []string{"Build"}},
		{name: "config blueprint placeholder", raw: `config.foo.bar`},
		{name: "standalone dollar", raw: `$`},
	})
}

func TestValidateExpression_Syntax(t *testing.T) {
	runExprCases(t, "syntax", []exprCase{
		{name: "unclosed paren", raw: `root(`, wantErr: "syntax error"},
		{name: "unclosed bracket", raw: `$[`, wantErr: "syntax error"},
		{name: "trailing dot", raw: `$['Build'].`, knownNames: []string{"Build"}, wantErr: "syntax error"},
		{name: "incomplete binary", raw: `1 +`, wantErr: "syntax error"},
		{name: "unterminated string", raw: `'unterminated`, wantErr: "syntax error"},
		{name: "empty body", raw: `   `, wantErr: "expression body is empty"},
	})
}

func TestValidateExpression_UnknownNodeRef(t *testing.T) {
	runExprCases(t, "unknown_node_ref", []exprCase{
		{name: "bracket missing", raw: `$['Missing'].x`, knownNames: []string{"Build"}, wantErr: "unknown node reference 'Missing'"},
		{name: "one valid one missing", raw: `$['Build'].x + $['AlsoMissing'].y`, knownNames: []string{"Build"}, wantErr: "unknown node reference 'AlsoMissing'"},
	})
}

func TestValidateExpression_BadArity(t *testing.T) {
	runExprCases(t, "bad_arity", []exprCase{
		{name: "root with int", raw: `root(1)`, wantErr: "root() takes no arguments"},
		{name: "root with string", raw: `root('x')`, wantErr: "root() takes no arguments"},
		{name: "previous too many", raw: `previous(1, 2)`, wantErr: "previous() accepts zero or one argument"},
		{name: "previous string literal", raw: `previous('a')`, wantErr: "previous() depth must be an integer literal"},
		{name: "previous float literal", raw: `previous(1.5)`, wantErr: "previous() depth must be an integer literal"},
		{name: "memory.find missing matches", raw: `memory.find('ns')`, wantErr: "memory.find() requires a namespace and matches"},
		{name: "memory.find no args", raw: `memory.find()`, wantErr: "memory.find() requires a namespace and matches"},
		{name: "memory.findFirst no args", raw: `memory.findFirst()`, wantErr: "memory.findFirst() requires a namespace and matches"},
		{name: "memory.find too many", raw: `memory.find('ns', {}, 'extra')`, wantErr: "memory.find() requires a namespace and matches"},
	})
}

func TestValidateExpression_UnknownIdentifier(t *testing.T) {
	runExprCases(t, "unknown_identifier", []exprCase{
		{name: "bare identifier", raw: `unknownThing`, wantErr: "compile error"},
		{name: "wrong case memory", raw: `Memory.find('ns', {})`, wantErr: "compile error"},
	})
}
