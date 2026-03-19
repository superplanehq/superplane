import type { HTTPContext } from "../../context/http.js";
import type { ExecutionStateContext } from "../../context/execution.js";
import type { RuntimeValue } from "../../context/runtime-value.js";
import type { ExecuteCodeEffects } from "../../effects/execute-code.js";

export interface ExecuteCodeJobContext {
  metadata?: RuntimeValue;
}

export interface ExecuteCodeJob {
  type: "execute-code";
  context?: ExecuteCodeJobContext;
}

export interface ExecuteCodeContext {
  http: HTTPContext;
  executionState: ExecutionStateContext;
}

export interface ExecuteCodeModule {
  default(context: ExecuteCodeContext): Promise<void> | void;
}

export interface ExecuteCodeResult {
  effects: ExecuteCodeEffects;
}
