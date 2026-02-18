package jsruntime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/superplanehq/superplane/pkg/core"
)

const defaultExecutionTimeout = 30 * time.Second

type Runtime struct {
	Timeout time.Duration
}

func NewRuntime(timeout time.Duration) *Runtime {
	if timeout == 0 {
		timeout = defaultExecutionTimeout
	}

	return &Runtime{Timeout: timeout}
}

// ParseDefinition executes a JS file in a temporary VM to extract the component definition
// (metadata, configuration, output channels) without running the execute/setup handlers.
func (r *Runtime) ParseDefinition(source string) (*ComponentDefinition, error) {
	vm := goja.New()

	var def *ComponentDefinition

	componentFn := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("superplane.component() requires a definition object"))
		}

		obj := call.Arguments[0].ToObject(vm)
		def = r.extractDefinition(vm, obj)
		return goja.Undefined()
	}

	spObj := vm.NewObject()
	if err := spObj.Set("component", componentFn); err != nil {
		return nil, fmt.Errorf("failed to set superplane.component: %w", err)
	}

	if err := vm.Set("superplane", spObj); err != nil {
		return nil, fmt.Errorf("failed to set superplane global: %w", err)
	}

	_, err := vm.RunString(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse component script: %w", err)
	}

	if def == nil {
		return nil, fmt.Errorf("script did not call superplane.component()")
	}

	if !def.HasExecute {
		return nil, fmt.Errorf("component definition must include an execute function")
	}

	return def, nil
}

// Execute runs the component's execute handler in a fresh VM with the given execution context.
func (r *Runtime) Execute(source string, ctx core.ExecutionContext) error {
	vm := goja.New()

	timer := time.AfterFunc(r.Timeout, func() {
		vm.Interrupt("execution timeout exceeded")
	})
	defer timer.Stop()

	r.injectExecuteSDK(vm, ctx)

	_, err := vm.RunString(source)
	if err != nil {
		return convertJSError(err)
	}

	return nil
}

// Setup runs the component's setup handler in a fresh VM with the given setup context.
func (r *Runtime) Setup(source string, ctx core.SetupContext) error {
	vm := goja.New()

	timer := time.AfterFunc(r.Timeout, func() {
		vm.Interrupt("setup timeout exceeded")
	})
	defer timer.Stop()

	r.injectSetupSDK(vm, ctx)

	_, err := vm.RunString(source)
	if err != nil {
		return convertJSError(err)
	}

	return nil
}

func (r *Runtime) injectExecuteSDK(vm *goja.Runtime, ctx core.ExecutionContext) {
	ctxObj := vm.NewObject()

	ctxObj.Set("id", ctx.ID.String())
	ctxObj.Set("workflowId", ctx.WorkflowID)
	ctxObj.Set("nodeId", ctx.NodeID)
	ctxObj.Set("input", ctx.Data)
	ctxObj.Set("configuration", ctx.Configuration)

	ctxObj.Set("emit", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.NewTypeError("ctx.emit() requires channel, type, and data arguments"))
		}

		channel := call.Arguments[0].String()
		payloadType := call.Arguments[1].String()
		data := call.Arguments[2].Export()

		err := ctx.ExecutionState.Emit(channel, payloadType, []any{data})
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return goja.Undefined()
	})

	ctxObj.Set("pass", func(call goja.FunctionCall) goja.Value {
		err := ctx.ExecutionState.Pass()
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return goja.Undefined()
	})

	ctxObj.Set("fail", func(call goja.FunctionCall) goja.Value {
		reason := ""
		message := ""

		if len(call.Arguments) > 0 {
			reason = call.Arguments[0].String()
		}

		if len(call.Arguments) > 1 {
			message = call.Arguments[1].String()
		}

		err := ctx.ExecutionState.Fail(reason, message)
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return goja.Undefined()
	})

	r.injectMetadata(vm, ctxObj, ctx.Metadata)
	r.injectHTTP(vm, ctxObj, ctx.HTTP)
	r.injectLogger(vm, ctxObj, ctx.Logger)

	if ctx.Secrets != nil {
		r.injectSecrets(vm, ctxObj, ctx.Secrets)
	}

	if ctx.ExpressionEnv != nil {
		ctxObj.Set("eval", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) < 1 {
				panic(vm.NewTypeError("ctx.eval() requires an expression argument"))
			}

			expression := call.Arguments[0].String()
			result, err := ctx.ExpressionEnv(expression)
			if err != nil {
				panic(vm.NewGoError(err))
			}

			return vm.ToValue(result)
		})
	}

	var executeCalled bool

	componentFn := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("superplane.component() requires a definition object"))
		}

		obj := call.Arguments[0].ToObject(vm)

		executeFn, ok := goja.AssertFunction(obj.Get("execute"))
		if !ok {
			panic(vm.NewTypeError("component definition must include an execute function"))
		}

		_, err := executeFn(goja.Undefined(), ctxObj)
		if err != nil {
			panic(err)
		}

		executeCalled = true
		return goja.Undefined()
	}

	spObj := vm.NewObject()
	spObj.Set("component", componentFn)
	vm.Set("superplane", spObj)

	_ = executeCalled
}

func (r *Runtime) injectSetupSDK(vm *goja.Runtime, ctx core.SetupContext) {
	ctxObj := vm.NewObject()
	ctxObj.Set("configuration", ctx.Configuration)

	r.injectMetadata(vm, ctxObj, ctx.Metadata)
	r.injectHTTP(vm, ctxObj, ctx.HTTP)

	if ctx.Logger != nil {
		r.injectLogger(vm, ctxObj, ctx.Logger)
	}

	componentFn := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("superplane.component() requires a definition object"))
		}

		obj := call.Arguments[0].ToObject(vm)
		setupVal := obj.Get("setup")

		if setupVal == nil || goja.IsUndefined(setupVal) || goja.IsNull(setupVal) {
			return goja.Undefined()
		}

		setupFn, ok := goja.AssertFunction(setupVal)
		if !ok {
			return goja.Undefined()
		}

		_, err := setupFn(goja.Undefined(), ctxObj)
		if err != nil {
			panic(err)
		}

		return goja.Undefined()
	}

	spObj := vm.NewObject()
	spObj.Set("component", componentFn)
	vm.Set("superplane", spObj)
}

func (r *Runtime) injectMetadata(vm *goja.Runtime, parent *goja.Object, metadata core.MetadataContext) {
	if metadata == nil {
		return
	}

	metaObj := vm.NewObject()

	metaObj.Set("get", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(metadata.Get())
	})

	metaObj.Set("set", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("ctx.metadata.set() requires a value argument"))
		}

		err := metadata.Set(call.Arguments[0].Export())
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return goja.Undefined()
	})

	parent.Set("metadata", metaObj)
}

func (r *Runtime) injectHTTP(vm *goja.Runtime, parent *goja.Object, httpCtx core.HTTPContext) {
	if httpCtx == nil {
		return
	}

	httpObj := vm.NewObject()

	httpObj.Set("request", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("ctx.http.request() requires method and url arguments"))
		}

		method := call.Arguments[0].String()
		url := call.Arguments[1].String()

		var bodyStr string
		var headers map[string]string
		var timeout time.Duration

		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
			opts := call.Arguments[2].ToObject(vm)

			if b := opts.Get("body"); b != nil && !goja.IsUndefined(b) {
				bodyStr = b.String()
			}

			if h := opts.Get("headers"); h != nil && !goja.IsUndefined(h) {
				headers = make(map[string]string)
				hObj := h.ToObject(vm)
				for _, key := range hObj.Keys() {
					headers[key] = hObj.Get(key).String()
				}
			}

			if t := opts.Get("timeout"); t != nil && !goja.IsUndefined(t) {
				timeout = time.Duration(t.ToInteger()) * time.Millisecond
			}
		}

		if timeout == 0 {
			timeout = 10 * time.Second
		}

		var body io.Reader
		if bodyStr != "" {
			body = strings.NewReader(bodyStr)
		}

		req, err := http.NewRequest(method, url, body)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("failed to create request: %w", err)))
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := httpCtx.Do(req)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("failed to read response body: %w", err)))
		}

		var parsedBody any
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &parsedBody); err != nil {
				parsedBody = string(respBody)
			}
		}

		respHeaders := make(map[string]string)
		for k := range resp.Header {
			respHeaders[strings.ToLower(k)] = resp.Header.Get(k)
		}

		result := vm.NewObject()
		result.Set("status", resp.StatusCode)
		result.Set("headers", respHeaders)
		result.Set("body", parsedBody)

		return result
	})

	parent.Set("http", httpObj)
}

func (r *Runtime) injectLogger(vm *goja.Runtime, parent *goja.Object, logger interface {
	Info(args ...any)
	Error(args ...any)
}) {
	logObj := vm.NewObject()

	logObj.Set("info", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			logger.Info(call.Arguments[0].String())
		}

		return goja.Undefined()
	})

	logObj.Set("error", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) > 0 {
			logger.Error(call.Arguments[0].String())
		}

		return goja.Undefined()
	})

	parent.Set("log", logObj)
}

func (r *Runtime) injectSecrets(vm *goja.Runtime, parent *goja.Object, secrets core.SecretsContext) {
	secretsObj := vm.NewObject()

	secretsObj.Set("getKey", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("ctx.secrets.getKey() requires secretName and keyName arguments"))
		}

		secretName := call.Arguments[0].String()
		keyName := call.Arguments[1].String()

		val, err := secrets.GetKey(secretName, keyName)
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return vm.ToValue(string(val))
	})

	parent.Set("secrets", secretsObj)
}

func (r *Runtime) extractDefinition(vm *goja.Runtime, obj *goja.Object) *ComponentDefinition {
	def := &ComponentDefinition{
		Icon:  "code",
		Color: "blue",
	}

	if v := obj.Get("name"); v != nil && !goja.IsUndefined(v) {
		def.Name = v.String()
	}

	if v := obj.Get("label"); v != nil && !goja.IsUndefined(v) {
		def.Label = v.String()
	}

	if v := obj.Get("description"); v != nil && !goja.IsUndefined(v) {
		def.Description = v.String()
	}

	if v := obj.Get("documentation"); v != nil && !goja.IsUndefined(v) {
		def.Documentation = v.String()
	}

	if v := obj.Get("icon"); v != nil && !goja.IsUndefined(v) {
		def.Icon = v.String()
	}

	if v := obj.Get("color"); v != nil && !goja.IsUndefined(v) {
		def.Color = v.String()
	}

	if v := obj.Get("execute"); v != nil && !goja.IsUndefined(v) {
		if _, ok := goja.AssertFunction(v); ok {
			def.HasExecute = true
		}
	}

	if v := obj.Get("setup"); v != nil && !goja.IsUndefined(v) {
		if _, ok := goja.AssertFunction(v); ok {
			def.HasSetup = true
		}
	}

	if v := obj.Get("configuration"); v != nil && !goja.IsUndefined(v) {
		def.RawConfiguration = v.Export()
	}

	if v := obj.Get("outputChannels"); v != nil && !goja.IsUndefined(v) {
		def.RawOutputChannels = v.Export()
	}

	return def
}

func convertJSError(err error) error {
	if jsErr, ok := err.(*goja.InterruptedError); ok {
		return fmt.Errorf("%s", jsErr.Value())
	}

	if jsErr, ok := err.(*goja.Exception); ok {
		return fmt.Errorf("%s", jsErr.Value().String())
	}

	return err
}
