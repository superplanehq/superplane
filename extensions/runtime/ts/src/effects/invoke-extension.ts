import type { RuntimeValue } from "../context/runtime-value.js";
import type { ExecutionStateEffects } from "./execution-state.js";
import type { IntegrationEffects } from "./integration.js";

export interface InvocationScheduledAction {
  actionName: string;
  parameters: Record<string, RuntimeValue>;
  intervalMs: number;
}

export interface InvocationEvent {
  payloadType: string;
  payload: RuntimeValue;
}

export interface InvocationWebhookEffects {
  url: string;
  baseURL: string;
  secret: RuntimeValue;
}

export interface InvocationEffects {
  metadata: RuntimeValue;
  requests: {
    scheduledActions: InvocationScheduledAction[];
  };
  events: InvocationEvent[];
  executionState: ExecutionStateEffects;
  integration: IntegrationEffects;
  webhook: InvocationWebhookEffects;
}
