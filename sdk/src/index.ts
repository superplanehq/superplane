import { PlaneletServer } from "./server.js";
import type {
  ActionDefinition,
  ActionExecutionContext,
  ActionManifest,
  CleanupTriggerRequest,
  CleanupTriggerResponse,
  ExecuteRequest,
  ExecuteResponse,
  ForwardedWebhookRequest,
  HandleTriggerWebhookRequest,
  HandleTriggerWebhookResponse,
  Manifest,
  ParameterDefinition,
  ParameterManifest,
  ParameterOption,
  PlaneletOptions,
  TriggerDefinition,
  TriggerCleanupContext,
  TriggerManifest,
  TriggerSetupContext,
  TriggerWebhookConfig,
  TriggerWebhookContext,
  TriggerWebhookResult,
  WebhookHttpResponse,
} from "./types.js";

export function createPlanelet(options: PlaneletOptions): PlaneletBuilder {
  return new PlaneletBuilder(options);
}

class PlaneletBuilder {
  private server: PlaneletServer;

  constructor(options: PlaneletOptions) {
    this.server = new PlaneletServer(options);
  }

  action<TParams = Record<string, unknown>>(
    id: string,
    definition: ActionDefinition<TParams>,
  ): this {
    this.server.addAction(id, definition as ActionDefinition);
    return this;
  }

  trigger<
    TParams = Record<string, unknown>,
    TMetadata = Record<string, unknown>,
  >(id: string, definition: TriggerDefinition<TParams, TMetadata>): this {
    this.server.addTrigger(id, definition as TriggerDefinition);
    return this;
  }

  listen(port: number, callback?: () => void): void {
    this.server.listen(port, callback);
  }
}

export type {
  ActionDefinition,
  ActionExecutionContext,
  ActionManifest,
  CleanupTriggerRequest,
  CleanupTriggerResponse,
  ExecuteRequest,
  ExecuteResponse,
  ForwardedWebhookRequest,
  HandleTriggerWebhookRequest,
  HandleTriggerWebhookResponse,
  Manifest,
  ParameterDefinition,
  ParameterManifest,
  ParameterOption,
  PlaneletOptions,
  TriggerDefinition,
  TriggerCleanupContext,
  TriggerManifest,
  TriggerSetupContext,
  TriggerWebhookConfig,
  TriggerWebhookContext,
  TriggerWebhookResult,
  WebhookHttpResponse,
};
