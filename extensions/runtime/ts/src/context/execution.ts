import type { RuntimeValue } from "./runtime-value.js";

export interface MetadataContext {
  get(): Promise<RuntimeValue> | RuntimeValue;
  set(value: RuntimeValue): Promise<void> | void;
}

export interface RequestContext {
  scheduleActionCall(
    actionName: string,
    parameters: Record<string, RuntimeValue>,
    intervalMs: number,
  ): Promise<void> | void;
}

export interface EventContext {
  emit(payloadType: string, payload: RuntimeValue): Promise<void> | void;
}

export interface ExecutionStateContext {
  isFinished(): Promise<boolean> | boolean;
  setKV(key: string, value: string): Promise<void> | void;
  emit(
    channel: string,
    payloadType: string,
    payloads: RuntimeValue[],
  ): Promise<void> | void;
  pass(): Promise<void> | void;
  fail(reason: string, message: string): Promise<void> | void;
}
