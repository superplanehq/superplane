declare module '@superplane/framework' {
  export type ActionContext<TProps, TConnection> = {
    props: TProps;
    connection: TConnection;
  };

  export type TriggerHookContext<TProps> = {
    props: TProps;
    body: unknown;
  };

  export function createConnection<TConnection>(config: Record<string, unknown>): {
    name?: string;
    description?: string;
    props?: Record<string, unknown>;
    [key: string]: unknown;
  };

  export function createAction<TProps, TOutput, TConnection>(config: {
    name: string;
    description?: string;
    props: Record<string, unknown>;
    connection: unknown;
    run: (ctx: ActionContext<TProps, TConnection>) => Promise<TOutput>;
    [key: string]: unknown;
  }): {
    name: string;
    description?: string;
    props: Record<string, unknown>;
    connection: unknown;
    run: (ctx: ActionContext<TProps, TConnection>) => Promise<TOutput>;
    [key: string]: unknown;
  };

  export function createTrigger<TProps, TOutput, TConnection>(config: {
    name: string;
    description?: string;
    props: Record<string, unknown>;
    connection: unknown;
    type: string;
    onWebhook: (ctx: TriggerHookContext<TProps>) => Promise<TOutput | undefined>;
    [key: string]: unknown;
  }): {
    name: string;
    description?: string;
    props: Record<string, unknown>;
    connection: unknown;
    type: string;
    onWebhook: (ctx: TriggerHookContext<TProps>) => Promise<TOutput | undefined>;
    [key: string]: unknown;
  };

  export function createIntegration(config: {
    name: string;
    description?: string;
    logoUrl?: string;
    connection: unknown;
    triggers?: unknown[];
    actions?: unknown[];
    [key: string]: unknown;
  }): Record<string, unknown>;
}
