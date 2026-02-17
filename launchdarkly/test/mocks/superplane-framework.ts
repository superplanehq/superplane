type AnyRecord = Record<string, unknown>;

export function createConnection(config: AnyRecord): AnyRecord {
  return config as AnyRecord;
}

export function createAction(config: AnyRecord): AnyRecord {
  return config as AnyRecord;
}

export function createTrigger(config: AnyRecord): AnyRecord {
  return config as AnyRecord;
}

export function createIntegration(config: AnyRecord): AnyRecord {
  return config as AnyRecord;
}

export type ActionContext<TProps, TConnection> = {
  props: TProps;
  connection: TConnection;
};

export type TriggerHookContext<TProps> = {
  props: TProps;
  body: unknown;
};
