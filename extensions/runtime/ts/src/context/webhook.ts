import type { RuntimeValue } from "./runtime-value.js";

export interface WebhookContext {
  getID(): Promise<string> | string;
  getURL(): Promise<string> | string;
  getSecret(): Promise<Uint8Array> | Uint8Array;
  getMetadata(): Promise<RuntimeValue> | RuntimeValue;
  getConfiguration(): Promise<RuntimeValue> | RuntimeValue;
  setSecret(secret: Uint8Array): Promise<void> | void;
}

export interface NodeWebhookContext {
  setup(): Promise<string> | string;
  getSecret(): Promise<Uint8Array> | Uint8Array;
  setSecret(secret: Uint8Array): Promise<void> | void;
  resetSecret():
    | Promise<{ previous: Uint8Array; current: Uint8Array }>
    | { previous: Uint8Array; current: Uint8Array };
  getBaseURL(): Promise<string> | string;
}
