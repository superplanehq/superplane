export type RuntimeOperation =
  | "component.execute"
  | "component.setup"
  | "integration.sync"
  | "integration.cleanup";

export type ComponentExecutionRequest = {
  operation: RuntimeOperation;
  component: string;
  context: {
    executionId: string;
    workflowId: string;
    organizationId: string;
    nodeId: string;
    sourceNodeId: string;
    configuration: unknown;
    integrationConfiguration?: Record<string, unknown>;
    data: unknown;
    metadata?: Record<string, unknown>;
    nodeMetadata?: Record<string, unknown>;
  };
};

export type ComponentExecutionOutcome = "pass" | "fail" | "noop";

export type ComponentOutput = {
  channel: string;
  payloadType: string;
  payload: unknown;
};

export type ComponentExecutionKV = {
  key: string;
  value: string;
};

export type ComponentExecutionResponse = {
  outcome: ComponentExecutionOutcome;
  errorReason?: string;
  error?: string;
  outputs?: ComponentOutput[];
  metadata?: Record<string, unknown>;
  nodeMetadata?: Record<string, unknown>;
  kvs?: ComponentExecutionKV[];
};

export type SetupContext = {
  configuration: unknown;
  integrationConfiguration?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  nodeMetadata?: Record<string, unknown>;
};

export type ExecuteContext = ComponentExecutionRequest["context"] & {
  logger: RuntimeLogger;
};

export type RuntimeLogger = {
  debug(message: string, fields?: Record<string, unknown>): void;
  info(message: string, fields?: Record<string, unknown>): void;
  warn(message: string, fields?: Record<string, unknown>): void;
  error(message: string, fields?: Record<string, unknown>): void;
};

export type ComponentImplementation = {
  setup?(ctx: SetupContext): Promise<void> | void;
  execute(ctx: ExecuteContext): Promise<ComponentExecutionResponse> | ComponentExecutionResponse;
};

export type IntegrationContext = {
  configuration?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  organizationId?: string;
  baseUrl?: string;
  webhooksBaseUrl?: string;
};

export type IntegrationExecutionOutcome = "pass" | "fail" | "noop";

export type IntegrationResource = {
  type: string;
  name: string;
  id: string;
};

export type IntegrationHTTPResponse = {
  statusCode: number;
  headers?: Record<string, string[]>;
  body?: number[];
};

export type IntegrationExecutionResponse = {
  outcome: IntegrationExecutionOutcome;
  errorReason?: string;
  error?: string;
  metadata?: Record<string, unknown>;
  state?: string;
  stateDescription?: string;
  resources?: IntegrationResource[];
  http?: IntegrationHTTPResponse;
};

export type IntegrationImplementation = {
  sync?(ctx: IntegrationContext): Promise<IntegrationExecutionResponse> | IntegrationExecutionResponse;
  cleanup?(ctx: IntegrationContext): Promise<IntegrationExecutionResponse> | IntegrationExecutionResponse;
};

export type TriggerSetupContext = {
  configuration?: unknown;
  metadata?: Record<string, unknown>;
};

export type TriggerImplementation = {
  setup?(ctx: TriggerSetupContext): Promise<void> | void;
};
