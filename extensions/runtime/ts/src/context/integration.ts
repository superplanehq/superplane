import type { RuntimeValue } from "./runtime-value.js";

export interface BrowserAction {
  description: string;
  url: string;
  method: string;
  formFields?: Record<string, string>;
}

export interface IntegrationSecret {
  name: string;
  value: Uint8Array;
}

export interface IntegrationSubscription {
  configuration(): Promise<RuntimeValue> | RuntimeValue;
  sendMessage(message: RuntimeValue): Promise<void> | void;
}

export interface IntegrationContext {
  id(): Promise<string> | string;
  getMetadata(): Promise<RuntimeValue> | RuntimeValue;
  setMetadata(value: RuntimeValue): Promise<void> | void;
  getConfig(name: string): Promise<Uint8Array> | Uint8Array;
  ready(): Promise<void> | void;
  error(message: string): Promise<void> | void;
  newBrowserAction(action: BrowserAction): Promise<void> | void;
  removeBrowserAction(): Promise<void> | void;
  setSecret(name: string, value: Uint8Array): Promise<void> | void;
  getSecrets(): Promise<IntegrationSecret[]> | IntegrationSecret[];
  requestWebhook(configuration: RuntimeValue): Promise<void> | void;
  subscribe(configuration: RuntimeValue): Promise<string> | string;
  scheduleResync(intervalMs: number): Promise<void> | void;
  scheduleActionCall(
    actionName: string,
    parameters: RuntimeValue,
    intervalMs: number,
  ): Promise<void> | void;
  listSubscriptions():
    | Promise<IntegrationSubscription[]>
    | IntegrationSubscription[];
}
