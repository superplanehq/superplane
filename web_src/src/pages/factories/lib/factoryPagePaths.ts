export function factoryListPath(organizationId: string) {
  return `/${organizationId}/workspaces`;
}

/** Setup wizard for a workspace that does not exist yet. */
export function newFactoryPath(organizationId: string) {
  return `${factoryListPath(organizationId)}/new`;
}

export function factoryDetailPath(organizationId: string, factoryKey: string) {
  return `${factoryListPath(organizationId)}/${factoryKey}`;
}

export function factoryOverviewPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/overview`;
}

/** First line id on a factory, when the factory has at least one line. */
export function firstFactoryLineId(
  factory: { lines?: Array<{ id?: string }> | null } | null | undefined,
): string | undefined {
  return factory?.lines?.find((line) => Boolean(line.id))?.id;
}

/**
 * Workspace home: the first line board when a line exists, otherwise the
 * lines list (empty state).
 */
export function factoryHomePath(organizationId: string, factoryKey: string, lineId?: string | null) {
  if (lineId) {
    return factoryLineDetailPath(organizationId, factoryKey, lineId);
  }
  return linesPath(organizationId, factoryKey);
}

export function factorySetupPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/setup`;
}

export function workOrdersPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/work-orders`;
}

export function createWorkOrderPath(organizationId: string, factoryKey: string) {
  return `${workOrdersPath(organizationId, factoryKey)}/new`;
}

/**
 * Canonical work order permalink: `/{organizationId}/workspaces/{factoryKey}/work-order/{orderNumber}`
 * (singular segment, sibling of `work-orders`). `orderNumber` is the
 * factory-scoped sequence number (`FactoriesWorkOrder.number`), not the
 * database id — see `legacyWorkOrderDetailPath` for the old id-based shape.
 */
export function workOrderDetailPath(organizationId: string, factoryKey: string, orderNumber: string | number) {
  return `${factoryDetailPath(organizationId, factoryKey)}/work-order/${orderNumber}`;
}

/**
 * Old id-based work order URL shape, kept around only so the legacy
 * redirect route can compare against it / build test fixtures. New code
 * should always call `workOrderDetailPath`.
 */
export function legacyWorkOrderDetailPath(organizationId: string, factoryKey: string, orderId: string) {
  return `${workOrdersPath(organizationId, factoryKey)}/${orderId}`;
}

export function linesPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/lines`;
}

export function createFactoryLinePath(organizationId: string, factoryKey: string) {
  return `${linesPath(organizationId, factoryKey)}/new`;
}

export function factoryLineDetailPath(organizationId: string, factoryKey: string, lineId: string) {
  return `${linesPath(organizationId, factoryKey)}/${lineId}`;
}

export function editFactoryLinePath(organizationId: string, factoryKey: string, lineId: string) {
  return `${linesPath(organizationId, factoryKey)}/${lineId}/edit`;
}

export function automationsPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/automations`;
}

export function automationDetailPath(organizationId: string, factoryKey: string, appId: string) {
  return `${automationsPath(organizationId, factoryKey)}/${appId}`;
}

export type FactoryAppNavFrom = "automations" | "lines" | "work-order" | "overview";

export function parseFactoryAppNavFrom(value: string | null): FactoryAppNavFrom | undefined {
  if (value === "automations" || value === "lines" || value === "work-order" || value === "overview") {
    return value;
  }
  return undefined;
}

export type FactoryAppNavOptions = {
  from?: FactoryAppNavFrom;
  lineId?: string;
  /** Work order `number` (route identifier), not the database id. */
  orderNumber?: string;
  runId?: string;
  /**
   * Open factory-shell Configure chrome. Uses `configure=1` (not `edit=1`) so
   * AppPage auto-edit cleanup does not tear down the Configure UI mid-bootstrap.
   */
  configure?: boolean;
  /** Open the components panel in factory edit mode (`blocks=1`). */
  blocks?: boolean;
};

function buildFactoryAppSearchParams(options?: FactoryAppNavOptions): string {
  if (!options) {
    return "";
  }
  const params = new URLSearchParams();
  if (options.runId) {
    params.set("run", options.runId);
  }
  if (options.configure) {
    params.set("configure", "1");
  }
  if (options.blocks) {
    params.set("blocks", "1");
  }
  if (options.from) {
    params.set("from", options.from);
  }
  if (options.lineId) {
    params.set("lineId", options.lineId);
  }
  if (options.orderNumber) {
    params.set("orderNumber", options.orderNumber);
  }
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export function factoryAppConfigurePath(
  organizationId: string,
  factoryKey: string,
  appId: string,
  options?: Omit<FactoryAppNavOptions, "configure">,
) {
  return factoryAppPath(organizationId, factoryKey, appId, {
    ...options,
    configure: true,
  });
}

/** Factory canvas view URL. Keeps run/from context and omits edit chrome. */
export function factoryAppViewPath(
  organizationId: string,
  factoryKey: string,
  appId: string,
  options?: Pick<FactoryAppNavOptions, "from" | "lineId" | "orderNumber" | "runId">,
) {
  return factoryAppPath(organizationId, factoryKey, appId, options);
}

export function factoryAppPath(
  organizationId: string,
  factoryKey: string,
  appId: string,
  options?: FactoryAppNavOptions,
) {
  return `${factoryDetailPath(organizationId, factoryKey)}/apps/${appId}${buildFactoryAppSearchParams(options)}`;
}

export function factoryAppRunPath(
  organizationId: string,
  factoryKey: string,
  appId: string,
  runId: string,
  options?: Omit<FactoryAppNavOptions, "runId">,
) {
  return factoryAppPath(organizationId, factoryKey, appId, { ...options, runId });
}

export function factorySettingsPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/settings`;
}

export function factorySettingsSectionPath(organizationId: string, factoryKey: string, section: string) {
  return `${factorySettingsPath(organizationId, factoryKey)}/${section}`;
}

export function organizationSettingsPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/organization`;
}

export function organizationSettingsSectionPath(organizationId: string, factoryKey: string, section: string) {
  return `${organizationSettingsPath(organizationId, factoryKey)}/${section}`;
}

/** Settings General URL after a workspace key change, or `null` when the key did not change. */
export function factorySettingsGeneralPathAfterKeyChange(
  organizationId: string,
  previousKey: string,
  nextKey: string,
): string | null {
  if (!nextKey || nextKey === previousKey) {
    return null;
  }
  return factorySettingsSectionPath(organizationId, nextKey, "general");
}

export function factoryMissionsPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/missions`;
}

export function factoryWikiPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/wiki`;
}

export function factoryVelocityPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/velocity`;
}
