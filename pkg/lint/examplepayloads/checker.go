package examplepayloads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	_ "github.com/superplanehq/superplane/pkg/server" // register all components and triggers
)

type nodeKind string

const (
	nodeKindComponent nodeKind = "component"
	nodeKindTrigger   nodeKind = "trigger"
)

type schemaKind string

const (
	schemaUnknown schemaKind = "unknown"
	schemaNull    schemaKind = "null"
	schemaBool    schemaKind = "bool"
	schemaNumber  schemaKind = "number"
	schemaString  schemaKind = "string"
	schemaObject  schemaKind = "object"
	schemaArray   schemaKind = "array"
)

type schema struct {
	Kind   schemaKind
	Fields map[string]schema
	Elem   *schema
	Exact  bool
}

type Issue struct {
	Name    string
	Kind    nodeKind
	Message string
	Path    string
	Line    int
}

func (i Issue) String() string {
	location := i.Path
	if location == "" {
		location = "<unknown>"
	}

	if i.Line > 0 {
		return fmt.Sprintf("%s:%d: %s %q: %s", location, i.Line, i.Kind, i.Name, i.Message)
	}

	return fmt.Sprintf("%s: %s %q: %s", location, i.Kind, i.Name, i.Message)
}

// coreTriggerNames lists triggers that use their own payload shape
// and are exempt from the standard {data, type, timestamp} envelope.
var coreTriggerNames = map[string]bool{
	"start": true,
}

type exampleRecord struct {
	Name    string
	Kind    nodeKind
	Payload map[string]any
}

type typeMeta struct {
	Kind     nodeKind
	Name     string
	RecvType string
}

type emitSpec struct {
	PayloadType string
	Data        schema
	Position    token.Position
}

type summary struct {
	Returns []schema
}

type listedPackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
}

type loadedPackage struct {
	ImportPath string
	Dir        string
	Fset       *token.FileSet
	Syntax     []*ast.File
}

type packageAnalyzer struct {
	pkg          *loadedPackage
	funcDecls    map[string]*ast.FuncDecl
	typeDecls    map[string]ast.Expr
	constStrings map[string]string
	typeMetas    map[string]typeMeta
	summaryCache map[string]summary
	summarizing  map[string]bool
}

func Run() ([]Issue, error) {
	examples, err := loadExamples()
	if err != nil {
		return nil, err
	}

	pkgs, err := loadTargetPackages(
		"./pkg/components/...",
		"./pkg/triggers/...",
		"./pkg/integrations/...",
	)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	var issues []Issue
	for _, pkg := range pkgs {
		analyzer := newPackageAnalyzer(pkg)
		typeIssues, specs := analyzer.collectEmitSpecs()
		issues = append(issues, typeIssues...)

		for _, meta := range analyzer.typeMetas {
			key := exampleKey(meta.Kind, meta.Name)
			example, ok := examples[key]
			if !ok {
				issues = append(issues, Issue{
					Name:    meta.Name,
					Kind:    meta.Kind,
					Message: "missing example payload",
				})
				continue
			}

			// Core triggers use their own payload shape and don't
			// follow the standard {data, type, timestamp} envelope.
			if meta.Kind == nodeKindTrigger && coreTriggerNames[meta.Name] {
				continue
			}

			issues = append(issues, validateExampleEnvelope(example)...)
			issues = append(issues, validateExampleAgainstSpecs(example, specs[key])...)
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		if issues[i].Kind != issues[j].Kind {
			return issues[i].Kind < issues[j].Kind
		}
		return issues[i].Name < issues[j].Name
	})

	return issues, nil
}

func loadExamples() (map[string]exampleRecord, error) {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		return nil, fmt.Errorf("create registry: %w", err)
	}

	examples := map[string]exampleRecord{}
	add := func(kind nodeKind, name string, payload map[string]any) {
		examples[exampleKey(kind, name)] = exampleRecord{
			Name:    name,
			Kind:    kind,
			Payload: payload,
		}
	}

	for _, component := range reg.ListComponents() {
		add(nodeKindComponent, component.Name(), cloneMap(component.ExampleOutput()))
	}

	for _, trigger := range reg.ListTriggers() {
		add(nodeKindTrigger, trigger.Name(), cloneMap(trigger.ExampleData()))
	}

	for _, integration := range reg.ListIntegrations() {
		for _, component := range integration.Components() {
			add(nodeKindComponent, component.Name(), cloneMap(component.ExampleOutput()))
		}
		for _, trigger := range integration.Triggers() {
			add(nodeKindTrigger, trigger.Name(), cloneMap(trigger.ExampleData()))
		}
	}

	return examples, nil
}

func loadTargetPackages(patterns ...string) ([]*loadedPackage, error) {
	targetListed, err := goList(patterns...)
	if err != nil {
		return nil, err
	}

	targetPkgs := make([]*loadedPackage, 0, len(targetListed))
	for _, listed := range targetListed {
		if listed.Dir == "" || len(listed.GoFiles) == 0 {
			continue
		}

		pkg, err := parsePackage(listed)
		if err != nil {
			return nil, err
		}

		targetPkgs = append(targetPkgs, pkg)
	}

	return targetPkgs, nil
}

func goList(args ...string) ([]listedPackage, error) {
	baseArgs := append([]string{"list", "-json"}, args...)
	cmd := exec.Command("go", baseArgs...)
	cmd.Dir = repoRoot()
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %s: %w", strings.Join(baseArgs, " "), err)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	var out []listedPackage
	for decoder.More() {
		var listed listedPackage
		if err := decoder.Decode(&listed); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		out = append(out, listed)
	}

	return out, nil
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func parsePackage(listed listedPackage) (*loadedPackage, error) {
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(listed.GoFiles))

	for _, name := range listed.GoFiles {
		path := filepath.Join(listed.Dir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		files = append(files, file)
	}

	return &loadedPackage{
		ImportPath: listed.ImportPath,
		Dir:        listed.Dir,
		Fset:       fset,
		Syntax:     files,
	}, nil
}

func newPackageAnalyzer(pkg *loadedPackage) *packageAnalyzer {
	analyzer := &packageAnalyzer{
		pkg:          pkg,
		funcDecls:    map[string]*ast.FuncDecl{},
		typeDecls:    map[string]ast.Expr{},
		constStrings: map[string]string{},
		typeMetas:    map[string]typeMeta{},
		summaryCache: map[string]summary{},
		summarizing:  map[string]bool{},
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if ok {
				analyzer.collectGenDecl(genDecl)
			}

			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}

			if fn.Recv == nil {
				analyzer.funcDecls[fn.Name.Name] = fn
				continue
			}

			recvType := receiverTypeName(fn.Recv.List[0].Type)
			if recvType == "" {
				continue
			}

			switch fn.Name.Name {
			case "ExampleOutput":
				meta := analyzer.typeMetas[recvType]
				meta.Kind = nodeKindComponent
				meta.RecvType = recvType
				analyzer.typeMetas[recvType] = meta
			case "ExampleData":
				meta := analyzer.typeMetas[recvType]
				meta.Kind = nodeKindTrigger
				meta.RecvType = recvType
				analyzer.typeMetas[recvType] = meta
			}
		}
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv == nil || fn.Name.Name != "Name" {
				continue
			}

			recvType := receiverTypeName(fn.Recv.List[0].Type)
			meta, ok := analyzer.typeMetas[recvType]
			if !ok {
				continue
			}

			if name, ok := analyzer.constantStringFromFunc(fn); ok {
				meta.Name = name
				analyzer.typeMetas[recvType] = meta
			}
		}
	}

	return analyzer
}

func (a *packageAnalyzer) collectGenDecl(genDecl *ast.GenDecl) {
	switch genDecl.Tok {
	case token.CONST:
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, name := range valueSpec.Names {
				if len(valueSpec.Values) <= i {
					continue
				}

				if value, ok := a.constantString(valueSpec.Values[i]); ok {
					a.constStrings[name.Name] = value
				}
			}
		}
	case token.TYPE:
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			a.typeDecls[typeSpec.Name.Name] = typeSpec.Type
		}
	}
}

func (a *packageAnalyzer) collectEmitSpecs() ([]Issue, map[string][]emitSpec) {
	specs := map[string][]emitSpec{}
	var issues []Issue

	for recvType, meta := range a.typeMetas {
		if meta.Name == "" {
			issues = append(issues, Issue{
				Kind:    meta.Kind,
				Message: fmt.Sprintf("could not resolve Name() for receiver type %s in package %s", recvType, a.pkg.ImportPath),
			})
			continue
		}

		for _, file := range a.pkg.Syntax {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || fn.Recv == nil {
					continue
				}

				if receiverTypeName(fn.Recv.List[0].Type) != recvType {
					continue
				}

				methodSpecs := a.collectEmitSpecsFromBlock(fn.Body, map[string]schema{})
				key := exampleKey(meta.Kind, meta.Name)
				specs[key] = append(specs[key], methodSpecs...)
			}
		}
	}

	return issues, specs
}

func (a *packageAnalyzer) collectEmitSpecsFromBlock(block *ast.BlockStmt, env map[string]schema) []emitSpec {
	if block == nil {
		return nil
	}

	var out []emitSpec

	for _, stmt := range block.List {
		switch s := stmt.(type) {
		case *ast.BlockStmt:
			out = append(out, a.collectEmitSpecsFromBlock(s, cloneEnv(env))...)
		case *ast.AssignStmt:
			a.applyAssign(env, s)
			out = append(out, a.collectEmitSpecsFromStmt(stmt, env)...)
		case *ast.DeclStmt:
			a.applyDecl(env, s)
			out = append(out, a.collectEmitSpecsFromStmt(stmt, env)...)
		case *ast.IfStmt:
			out = append(out, a.collectEmitSpecsFromStmt(stmt, env)...)
		case *ast.ForStmt:
			out = append(out, a.collectEmitSpecsFromStmt(stmt, env)...)
		case *ast.RangeStmt:
			out = append(out, a.collectEmitSpecsFromStmt(stmt, env)...)
		default:
			out = append(out, a.collectEmitSpecsFromStmt(stmt, env)...)
		}
	}

	return out
}

func (a *packageAnalyzer) collectEmitSpecsFromStmt(stmt ast.Stmt, env map[string]schema) []emitSpec {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		if spec, ok := a.emitSpecFromCall(s.X, env); ok {
			return []emitSpec{spec}
		}
	case *ast.ReturnStmt:
		var specs []emitSpec
		for _, result := range s.Results {
			if spec, ok := a.emitSpecFromCall(result, env); ok {
				specs = append(specs, spec)
			}
		}
		return specs
	case *ast.IfStmt:
		var specs []emitSpec
		specs = append(specs, a.collectEmitSpecsFromBlock(s.Body, cloneEnv(env))...)
		if elseBlock, ok := s.Else.(*ast.BlockStmt); ok {
			specs = append(specs, a.collectEmitSpecsFromBlock(elseBlock, cloneEnv(env))...)
		}
		if elseIf, ok := s.Else.(*ast.IfStmt); ok {
			specs = append(specs, a.collectEmitSpecsFromStmt(elseIf, cloneEnv(env))...)
		}
		return specs
	case *ast.ForStmt:
		return a.collectEmitSpecsFromBlock(s.Body, cloneEnv(env))
	case *ast.RangeStmt:
		return a.collectEmitSpecsFromBlock(s.Body, cloneEnv(env))
	case *ast.SwitchStmt:
		var specs []emitSpec
		for _, stmt := range s.Body.List {
			clause, ok := stmt.(*ast.CaseClause)
			if !ok {
				continue
			}
			for _, caseStmt := range clause.Body {
				specs = append(specs, a.collectEmitSpecsFromStmt(caseStmt, cloneEnv(env))...)
			}
		}
		return specs
	}

	return nil
}

func (a *packageAnalyzer) emitSpecFromCall(expr ast.Expr, env map[string]schema) (emitSpec, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return emitSpec{}, false
	}

	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Emit" {
		return emitSpec{}, false
	}

	switch len(call.Args) {
	case 2:
		payloadType, ok := a.constantString(call.Args[0])
		if !ok {
			return emitSpec{}, false
		}

		return emitSpec{
			PayloadType: payloadType,
			Data:        a.inferExpr(call.Args[1], env),
			Position:    a.pkg.Fset.Position(call.Pos()),
		}, true
	case 3:
		payloadType, ok := a.constantString(call.Args[1])
		if !ok {
			return emitSpec{}, false
		}

		return emitSpec{
			PayloadType: payloadType,
			Data:        a.inferPayloadList(call.Args[2], env),
			Position:    a.pkg.Fset.Position(call.Pos()),
		}, true
	}

	return emitSpec{}, false
}

func (a *packageAnalyzer) inferPayloadList(expr ast.Expr, env map[string]schema) schema {
	switch payloads := expr.(type) {
	case *ast.CompositeLit:
		result := schema{Kind: schemaUnknown}
		for _, elt := range payloads.Elts {
			result = mergeSchema(result, a.inferExpr(elt, env))
		}
		if result.Kind == schemaUnknown {
			return schema{Kind: schemaUnknown}
		}
		return result
	default:
		inferred := a.inferExpr(expr, env)
		if inferred.Kind == schemaArray && inferred.Elem != nil {
			return *inferred.Elem
		}
		return inferred
	}
}

func (a *packageAnalyzer) applyDecl(env map[string]schema, stmt *ast.DeclStmt) {
	genDecl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok {
		return
	}

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for i, name := range valueSpec.Names {
			if len(valueSpec.Values) > i {
				env[name.Name] = a.inferExpr(valueSpec.Values[i], env)
				continue
			}

			if valueSpec.Type != nil {
				env[name.Name] = a.schemaFromTypeExpr(valueSpec.Type, map[string]bool{})
			}
		}
	}
}

func (a *packageAnalyzer) applyAssign(env map[string]schema, stmt *ast.AssignStmt) {
	if len(stmt.Rhs) == 1 && len(stmt.Lhs) > 1 {
		results := a.inferCallResults(stmt.Rhs[0], env)
		for i, lhs := range stmt.Lhs {
			switch l := lhs.(type) {
			case *ast.Ident:
				if l.Name == "_" {
					continue
				}
				if len(results) > i {
					env[l.Name] = results[i]
					continue
				}
				env[l.Name] = schema{Kind: schemaUnknown}
			case *ast.IndexExpr:
				a.applyIndexedAssign(env, l, nil, env)
			}
		}
		return
	}

	for i, lhs := range stmt.Lhs {
		var rhs ast.Expr
		if len(stmt.Rhs) > i {
			rhs = stmt.Rhs[i]
		} else if len(stmt.Rhs) == 1 {
			rhs = stmt.Rhs[0]
		}

		switch l := lhs.(type) {
		case *ast.Ident:
			if l.Name == "_" {
				continue
			}
			if rhs == nil {
				env[l.Name] = schema{Kind: schemaUnknown}
				continue
			}
			env[l.Name] = a.inferExpr(rhs, env)
		case *ast.IndexExpr:
			a.applyIndexedAssign(env, l, rhs, env)
		}
	}
}

func (a *packageAnalyzer) applyIndexedAssign(env map[string]schema, indexExpr *ast.IndexExpr, rhs ast.Expr, current map[string]schema) {
	base, ok := indexExpr.X.(*ast.Ident)
	if !ok {
		return
	}

	key, ok := a.stringLiteral(indexExpr.Index)
	if !ok {
		return
	}

	objectSchema, ok := env[base.Name]
	if !ok || objectSchema.Kind != schemaObject {
		objectSchema = schema{
			Kind:   schemaObject,
			Fields: map[string]schema{},
			Exact:  false,
		}
	}

	if rhs != nil {
		objectSchema.Fields[key] = a.inferExpr(rhs, current)
	}
	objectSchema.Exact = false
	env[base.Name] = objectSchema
}

func (a *packageAnalyzer) inferCallResults(expr ast.Expr, env map[string]schema) []schema {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		if builtinResult := builtinCallSchemas(fun.Name, call, a, env); builtinResult != nil {
			return builtinResult
		}

		fn := a.funcDecls[fun.Name]
		if fn == nil {
			return nil
		}

		sum := a.summarizeFunc(fn)
		if len(sum.Returns) > 0 {
			return sum.Returns
		}
	}

	return nil
}

func (a *packageAnalyzer) summarizeFunc(fn *ast.FuncDecl) summary {
	if fn == nil {
		return summary{}
	}

	if cached, ok := a.summaryCache[fn.Name.Name]; ok {
		return cached
	}
	if a.summarizing[fn.Name.Name] {
		return summary{}
	}

	a.summarizing[fn.Name.Name] = true
	defer delete(a.summarizing, fn.Name.Name)

	result := summary{}
	env := map[string]schema{}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			paramSchema := a.schemaFromTypeExpr(field.Type, map[string]bool{})
			for _, name := range field.Names {
				env[name.Name] = paramSchema
			}
		}
	}

	result.Returns = a.collectReturns(fn.Body, env)
	a.summaryCache[fn.Name.Name] = result
	return result
}

func (a *packageAnalyzer) collectReturns(block *ast.BlockStmt, env map[string]schema) []schema {
	if block == nil {
		return nil
	}

	var returns [][]schema

	for _, stmt := range block.List {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			a.applyAssign(env, s)
		case *ast.DeclStmt:
			a.applyDecl(env, s)
		case *ast.ReturnStmt:
			var result []schema
			for _, expr := range s.Results {
				result = append(result, a.inferExpr(expr, env))
			}
			returns = append(returns, result)
		case *ast.IfStmt:
			returns = append(returns, a.collectReturns(s.Body, cloneEnv(env)))
			if elseBlock, ok := s.Else.(*ast.BlockStmt); ok {
				returns = append(returns, a.collectReturns(elseBlock, cloneEnv(env)))
			}
			if elseIf, ok := s.Else.(*ast.IfStmt); ok {
				returns = append(returns, a.collectReturns(&ast.BlockStmt{List: []ast.Stmt{elseIf}}, cloneEnv(env)))
			}
		case *ast.SwitchStmt:
			for _, stmt := range s.Body.List {
				clause, ok := stmt.(*ast.CaseClause)
				if !ok {
					continue
				}
				returns = append(returns, a.collectReturns(&ast.BlockStmt{List: clause.Body}, cloneEnv(env)))
			}
		case *ast.ForStmt:
			returns = append(returns, a.collectReturns(s.Body, cloneEnv(env)))
		case *ast.RangeStmt:
			returns = append(returns, a.collectReturns(s.Body, cloneEnv(env)))
		}
	}

	return mergeReturnSets(returns)
}

func (a *packageAnalyzer) inferExpr(expr ast.Expr, env map[string]schema) schema {
	switch e := expr.(type) {
	case nil:
		return schema{Kind: schemaUnknown}
	case *ast.BasicLit:
		switch e.Kind {
		case token.STRING:
			return schema{Kind: schemaString}
		case token.INT, token.FLOAT:
			return schema{Kind: schemaNumber}
		default:
			return schema{Kind: schemaUnknown}
		}
	case *ast.ParenExpr:
		return a.inferExpr(e.X, env)
	case *ast.UnaryExpr:
		return a.inferExpr(e.X, env)
	case *ast.CompositeLit:
		return a.schemaFromCompositeLit(e, env)
	case *ast.Ident:
		switch e.Name {
		case "nil":
			return schema{Kind: schemaNull}
		case "true", "false":
			return schema{Kind: schemaBool}
		}

		if value, ok := env[e.Name]; ok {
			return value
		}
		return schema{Kind: schemaUnknown}
	case *ast.CallExpr:
		if results := a.inferCallResults(e, env); len(results) > 0 {
			return results[0]
		}
		return schema{Kind: schemaUnknown}
	default:
		return schema{Kind: schemaUnknown}
	}
}

func (a *packageAnalyzer) schemaFromCompositeLit(lit *ast.CompositeLit, env map[string]schema) schema {
	switch t := lit.Type.(type) {
	case *ast.MapType:
		fields := map[string]schema{}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				return schema{Kind: schemaObject, Fields: fields, Exact: false}
			}
			key, ok := a.stringLiteral(kv.Key)
			if !ok {
				return schema{Kind: schemaObject, Fields: fields, Exact: false}
			}
			fields[key] = a.inferExpr(kv.Value, env)
		}
		return schema{Kind: schemaObject, Fields: fields, Exact: true}
	case *ast.ArrayType:
		result := schema{Kind: schemaUnknown}
		for _, elt := range lit.Elts {
			result = mergeSchema(result, a.inferExpr(elt, env))
		}
		if result.Kind == schemaUnknown {
			result = schema{Kind: schemaUnknown}
		}
		return schema{Kind: schemaArray, Elem: &result, Exact: true}
	default:
		if lit.Type != nil {
			return a.schemaFromTypeExpr(t, map[string]bool{})
		}
		return schema{Kind: schemaUnknown}
	}
}

func (a *packageAnalyzer) constantStringFromFunc(fn *ast.FuncDecl) (string, bool) {
	if fn.Body == nil {
		return "", false
	}
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		return a.constantString(ret.Results[0])
	}
	return "", false
}

func (a *packageAnalyzer) constantString(expr ast.Expr) (string, bool) {
	switch value := expr.(type) {
	case *ast.BasicLit:
		return a.stringLiteral(value)
	case *ast.Ident:
		s, ok := a.constStrings[value.Name]
		return s, ok
	case *ast.BinaryExpr:
		if value.Op != token.ADD {
			return "", false
		}
		left, ok := a.constantString(value.X)
		if !ok {
			return "", false
		}
		right, ok := a.constantString(value.Y)
		if !ok {
			return "", false
		}
		return left + right, true
	case *ast.ParenExpr:
		return a.constantString(value.X)
	default:
		return "", false
	}
}

func (a *packageAnalyzer) stringLiteral(expr ast.Expr) (string, bool) {
	switch value := expr.(type) {
	case *ast.BasicLit:
		if value.Kind != token.STRING {
			return "", false
		}
		unquoted, err := strconv.Unquote(value.Value)
		if err != nil {
			return "", false
		}
		return unquoted, true
	case *ast.Ident:
		if resolved, ok := a.constStrings[value.Name]; ok {
			return resolved, true
		}
	}
	return "", false
}

func builtinCallSchemas(name string, call *ast.CallExpr, analyzer *packageAnalyzer, env map[string]schema) []schema {
	switch name {
	case "append":
		if len(call.Args) == 0 {
			return []schema{{Kind: schemaUnknown}}
		}
		base := analyzer.inferExpr(call.Args[0], env)
		elem := schema{Kind: schemaUnknown}
		for _, arg := range call.Args[1:] {
			elem = mergeSchema(elem, analyzer.inferExpr(arg, env))
		}

		if base.Kind == schemaArray && base.Elem != nil {
			elem = mergeSchema(*base.Elem, elem)
		}

		return []schema{{
			Kind:  schemaArray,
			Elem:  &elem,
			Exact: false,
		}}
	case "make":
		if len(call.Args) == 0 {
			return []schema{{Kind: schemaUnknown}}
		}
		return []schema{analyzer.schemaFromTypeExpr(call.Args[0], map[string]bool{})}
	default:
		return nil
	}
}


func (a *packageAnalyzer) schemaFromTypeExpr(expr ast.Expr, seen map[string]bool) schema {
	switch t := expr.(type) {
	case nil:
		return schema{Kind: schemaUnknown}
	case *ast.Ident:
		switch t.Name {
		case "string":
			return schema{Kind: schemaString}
		case "bool":
			return schema{Kind: schemaBool}
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
			return schema{Kind: schemaNumber}
		case "any", "interface{}", "error":
			return schema{Kind: schemaUnknown}
		}

		if seen[t.Name] {
			return schema{Kind: schemaUnknown}
		}

		typeExpr, ok := a.typeDecls[t.Name]
		if !ok {
			return schema{Kind: schemaUnknown}
		}

		nextSeen := cloneSeen(seen)
		nextSeen[t.Name] = true
		return a.schemaFromTypeExpr(typeExpr, nextSeen)
	case *ast.StarExpr:
		result := a.schemaFromTypeExpr(t.X, seen)
		result.Exact = false
		return result
	case *ast.MapType:
		return schema{
			Kind:   schemaObject,
			Fields: map[string]schema{},
			Exact:  false,
		}
	case *ast.ArrayType:
		elem := a.schemaFromTypeExpr(t.Elt, seen)
		return schema{
			Kind:  schemaArray,
			Elem:  &elem,
			Exact: false,
		}
	case *ast.StructType:
		fields := map[string]schema{}
		for _, field := range t.Fields.List {
			name := embeddedFieldName(field.Type)
			if len(field.Names) > 0 {
				name = field.Names[0].Name
			}
			if name == "" {
				continue
			}

			if field.Tag != nil {
				unquoted, err := strconv.Unquote(field.Tag.Value)
				if err == nil {
					jsonTag := reflect.StructTag(unquoted).Get("json")
					if jsonTag != "" {
						parts := strings.Split(jsonTag, ",")
						if parts[0] == "-" {
							continue
						}
						if parts[0] != "" {
							name = parts[0]
						}
					}
				}
			}

			fields[name] = a.schemaFromTypeExpr(field.Type, cloneSeen(seen))
		}
		return schema{
			Kind:   schemaObject,
			Fields: fields,
			Exact:  false,
		}
	case *ast.InterfaceType:
		return schema{Kind: schemaUnknown}
	default:
		return schema{Kind: schemaUnknown}
	}
}

func validateExampleEnvelope(example exampleRecord) []Issue {
	var issues []Issue

	if example.Payload == nil || len(example.Payload) == 0 {
		return []Issue{{
			Name:    example.Name,
			Kind:    example.Kind,
			Message: "example payload is empty",
		}}
	}

	requiredKeys := []string{"data", "timestamp", "type"}
	for _, key := range requiredKeys {
		if _, ok := example.Payload[key]; !ok {
			issues = append(issues, Issue{
				Name:    example.Name,
				Kind:    example.Kind,
				Message: fmt.Sprintf("example payload must include top-level %q", key),
			})
		}
	}

	rawType, ok := example.Payload["type"]
	if !ok {
		return issues
	}

	payloadType, ok := rawType.(string)
	if !ok || strings.TrimSpace(payloadType) == "" {
		issues = append(issues, Issue{
			Name:    example.Name,
			Kind:    example.Kind,
			Message: "example payload type must be a non-empty string",
		})
	}

	rawTimestamp, ok := example.Payload["timestamp"]
	if !ok {
		return issues
	}

	timestamp, ok := rawTimestamp.(string)
	if !ok || strings.TrimSpace(timestamp) == "" {
		issues = append(issues, Issue{
			Name:    example.Name,
			Kind:    example.Kind,
			Message: "example payload timestamp must be a non-empty string",
		})
		return issues
	}

	if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
		issues = append(issues, Issue{
			Name:    example.Name,
			Kind:    example.Kind,
			Message: fmt.Sprintf("example payload timestamp must be RFC3339/RFC3339Nano: %v", err),
		})
	}

	return issues
}

func validateExampleAgainstSpecs(example exampleRecord, specs []emitSpec) []Issue {
	if len(specs) == 0 {
		return nil
	}

	var payloadTypes []string
	typeMatched := false
	exampleType, _ := example.Payload["type"].(string)
	for _, spec := range specs {
		payloadTypes = append(payloadTypes, spec.PayloadType)
		if spec.PayloadType != exampleType {
			continue
		}

		typeMatched = true
		break
	}

	if !typeMatched {
		return []Issue{{
			Name:    example.Name,
			Kind:    example.Kind,
			Message: fmt.Sprintf("example payload type %q does not match any Emit(...) payload type: %s", exampleType, strings.Join(uniqueStrings(payloadTypes), ", ")),
		}}
	}

	return nil
}


func mergeSchema(left, right schema) schema {
	if left.Kind == schemaUnknown {
		return right
	}
	if right.Kind == schemaUnknown {
		return left
	}
	if left.Kind != right.Kind {
		return schema{Kind: schemaUnknown}
	}

	switch left.Kind {
	case schemaObject:
		fields := map[string]schema{}
		for key, value := range left.Fields {
			fields[key] = value
		}
		for key, value := range right.Fields {
			if current, ok := fields[key]; ok {
				fields[key] = mergeSchema(current, value)
				continue
			}
			fields[key] = value
		}
		return schema{
			Kind:   schemaObject,
			Fields: fields,
			Exact:  left.Exact && right.Exact,
		}
	case schemaArray:
		var elem schema
		switch {
		case left.Elem == nil && right.Elem == nil:
			elem = schema{Kind: schemaUnknown}
		case left.Elem == nil:
			elem = *right.Elem
		case right.Elem == nil:
			elem = *left.Elem
		default:
			elem = mergeSchema(*left.Elem, *right.Elem)
		}
		return schema{
			Kind:  schemaArray,
			Elem:  &elem,
			Exact: left.Exact && right.Exact,
		}
	default:
		return left
	}
}

func mergeReturnSets(sets [][]schema) []schema {
	if len(sets) == 0 {
		return nil
	}

	maxLen := 0
	for _, set := range sets {
		if len(set) > maxLen {
			maxLen = len(set)
		}
	}

	out := make([]schema, maxLen)
	for i := 0; i < maxLen; i++ {
		out[i] = schema{Kind: schemaUnknown}
	}

	for _, set := range sets {
		for i, item := range set {
			out[i] = mergeSchema(out[i], item)
		}
	}

	return out
}

func exampleKey(kind nodeKind, name string) string {
	return string(kind) + ":" + name
}

func receiverTypeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return receiverTypeName(typed.X)
	default:
		return ""
	}
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	raw, err := json.Marshal(input)
	if err != nil {
		return input
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return input
	}

	return out
}

func cloneEnv(env map[string]schema) map[string]schema {
	out := make(map[string]schema, len(env))
	for key, value := range env {
		out[key] = value
	}
	return out
}

func cloneSeen(seen map[string]bool) map[string]bool {
	out := make(map[string]bool, len(seen))
	for key, value := range seen {
		out[key] = value
	}
	return out
}

func embeddedFieldName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return embeddedFieldName(typed.X)
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
