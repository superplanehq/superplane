export interface ParameterOption {
  label: string;
  value: string;
}

export interface ParameterDefinition {
  label: string;
  type: "string" | "text" | "number" | "bool" | "select" | "object";
  description?: string;
  required?: boolean;
  default?: unknown;
  options?: ParameterOption[];
}

export interface ActionExecutionContext<TParams = Record<string, unknown>> {
  parameters: TParams;
  input?: unknown;
}

export interface ActionDefinition<TParams = Record<string, unknown>> {
  label: string;
  description?: string;
  icon?: string;
  iconUrl?: string;
  parameters?: Record<string, ParameterDefinition>;
  execute: (
    ctx: ActionExecutionContext<TParams>,
  ) => Promise<Record<string, unknown>> | Record<string, unknown>;
}

export interface TriggerWebhookConfig {
  url: string;
  secret?: string;
}

export interface TriggerSetupContext<TParams = Record<string, unknown>> {
  parameters: TParams;
  webhook: TriggerWebhookConfig;
}

export interface TriggerCleanupContext<
  TParams = Record<string, unknown>,
  TMetadata = Record<string, unknown>,
> {
  parameters: TParams;
  metadata?: TMetadata;
}

export interface ForwardedWebhookRequest {
  method: string;
  headers: Record<string, string[]>;
  query?: Record<string, string[]>;
  rawBodyBase64: string;
}

export interface TriggerWebhookContext<
  TParams = Record<string, unknown>,
  TMetadata = Record<string, unknown>,
> {
  parameters: TParams;
  metadata?: TMetadata;
  request: ForwardedWebhookRequest;
}

export interface WebhookHttpResponse {
  status?: number;
  headers?: Record<string, string>;
  body?: string;
}

export type TriggerWebhookResult =
  | {
      emit?: true;
      eventType?: string;
      payload: unknown;
      response?: WebhookHttpResponse;
    }
  | {
      emit: false;
      reason?: string;
      response?: WebhookHttpResponse;
    };

export interface TriggerDefinition<
  TParams = Record<string, unknown>,
  TMetadata = Record<string, unknown>,
> {
  label: string;
  description?: string;
  icon?: string;
  iconUrl?: string;
  parameters?: Record<string, ParameterDefinition>;
  setup: (
    ctx: TriggerSetupContext<TParams>,
  ) => Promise<TMetadata | void> | TMetadata | void;
  cleanup?: (
    ctx: TriggerCleanupContext<TParams, TMetadata>,
  ) => Promise<void> | void;
  handleWebhook: (
    ctx: TriggerWebhookContext<TParams, TMetadata>,
  ) => Promise<TriggerWebhookResult> | TriggerWebhookResult;
}

export interface ParameterManifest extends ParameterDefinition {
  id: string;
  required: boolean;
}

export interface ActionManifest {
  id: string;
  label: string;
  icon?: string;
  iconUrl?: string;
  description?: string;
  parameters: ParameterManifest[];
}

export interface TriggerManifest {
  id: string;
  label: string;
  icon?: string;
  iconUrl?: string;
  description?: string;
  parameters: ParameterManifest[];
}

export interface Manifest {
  id: string;
  label: string;
  icon?: string;
  iconUrl?: string;
  description?: string;
  actions: ActionManifest[];
  triggers: TriggerManifest[];
}

export interface PlaneletOptions {
  id: string;
  label?: string;
  icon?: string;
  iconUrl?: string;
  description?: string;
}

export interface ExecuteRequest {
  parameters?: Record<string, unknown>;
  input?: unknown;
}

export interface ExecuteResponse {
  success: boolean;
  data?: Record<string, unknown>;
  error?: string;
}

export interface SetupTriggerRequest {
  parameters?: Record<string, unknown>;
  webhook: TriggerWebhookConfig;
}

export interface SetupTriggerResponse {
  success: boolean;
  metadata?: Record<string, unknown>;
  error?: string;
}

export interface CleanupTriggerRequest {
  parameters?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
}

export interface CleanupTriggerResponse {
  success: boolean;
  error?: string;
}

export interface HandleTriggerWebhookRequest {
  parameters?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  request: ForwardedWebhookRequest;
}

export interface HandleTriggerWebhookResponse {
  success: boolean;
  emit?: boolean;
  eventType?: string;
  payload?: unknown;
  reason?: string;
  response?: WebhookHttpResponse;
  error?: string;
  status?: number;
}
