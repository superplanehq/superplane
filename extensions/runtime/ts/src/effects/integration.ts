import type { RuntimeValue } from "../context/runtime-value.js";

export interface IntegrationEffectsSecret {
  name: string;
  value: RuntimeValue;
}

export interface IntegrationEffectsSubscription {
  id: string;
  configuration: RuntimeValue;
  messages: RuntimeValue[];
}

export interface IntegrationEffects {
  id: string;
  ready: boolean;
  error?: string;
  metadata: RuntimeValue;
  browserAction: RuntimeValue;
  requestedWebhooks: RuntimeValue[];
  scheduledResyncIntervalMs: number | null;
  scheduledActions: Array<{
    actionName: string;
    parameters: RuntimeValue;
    intervalMs: number;
  }>;
  secrets: IntegrationEffectsSecret[];
  subscriptions: IntegrationEffectsSubscription[];
}
