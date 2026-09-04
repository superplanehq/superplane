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

/** First line name on a factory, used to Start a new task. */
export function firstFactoryLineName(
  factory: { lines?: Array<{ name?: string }> | null } | null | undefined,
): string | undefined {
  return factory?.lines?.find((line) => Boolean(line.name?.trim()))?.name?.trim();
}

/**
 * Workspace home: the first line board when a line id is present.
 * Without a line id, the workspace index — which redirects to that board.
 */
export function factoryHomePath(organizationId: string, factoryKey: string, lineId?: string | null) {
  if (lineId) {
    return factoryLineDetailPath(organizationId, factoryKey, lineId);
  }
  return factoryDetailPath(organizationId, factoryKey);
}

/** Opens the line board with the Intake drawer beside the columns. */
export const INTAKE_SEARCH_PARAM = "intake";
/**
 * Selects one intake when the drawer opens. A workspace can run several
 * intakes on the same source, so the identifier is the intake, not the source.
 */
export const INTAKE_ID_SEARCH_PARAM = "intakeId";
/** Opens intake settings on a tab: general, runs, or automation. */
export const INTAKE_SETTINGS_SEARCH_PARAM = "settings";

export function factoryIntakePath(
  organizationId: string,
  factoryKey: string,
  lineId?: string | null,
  intakeId?: string,
  settingsTab?: string,
) {
  const path = `${factoryHomePath(organizationId, factoryKey, lineId)}?${INTAKE_SEARCH_PARAM}=1`;
  const intakeQuery = intakeId ? `&${INTAKE_ID_SEARCH_PARAM}=${encodeURIComponent(intakeId)}` : "";
  const settingsQuery = settingsTab ? `&${INTAKE_SETTINGS_SEARCH_PARAM}=${encodeURIComponent(settingsTab)}` : "";
  return `${path}${intakeQuery}${settingsQuery}`;
}

export function isIntakeSearchOpen(search: string): boolean {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(INTAKE_SEARCH_PARAM) === "1";
}

export function intakeIdFromSearch(search: string): string | null {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(INTAKE_ID_SEARCH_PARAM);
}

export function intakeSettingsTabFromSearch(search: string): string | null {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(INTAKE_SETTINGS_SEARCH_PARAM);
}

export const PR_FEEDBACK_SEARCH_PARAM = "prFeedback";
/** Opens PR feedback settings on a tab: general or automation. */
export const PR_FEEDBACK_SETTINGS_SEARCH_PARAM = "prFeedbackSettings";
export const PR_FEEDBACK_HANDLER_SEARCH_PARAM = "prFeedbackHandler";

export function factoryPRFeedbackPath(
  organizationId: string,
  factoryKey: string,
  lineId?: string | null,
  settingsTab?: string,
  handlerId?: string,
) {
  const params = new URLSearchParams();
  params.set(PR_FEEDBACK_SEARCH_PARAM, "1");
  if (handlerId) {
    params.set(PR_FEEDBACK_HANDLER_SEARCH_PARAM, handlerId);
  }
  if (settingsTab) {
    params.set(PR_FEEDBACK_SETTINGS_SEARCH_PARAM, settingsTab);
  }
  return `${factoryHomePath(organizationId, factoryKey, lineId)}?${params.toString()}`;
}

export function isPRFeedbackSearchOpen(search: string): boolean {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(PR_FEEDBACK_SEARCH_PARAM) === "1";
}

export function prFeedbackSettingsTabFromSearch(search: string): string | null {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(PR_FEEDBACK_SETTINGS_SEARCH_PARAM);
}

export function prFeedbackHandlerIdFromSearch(search: string): string | null {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(PR_FEEDBACK_HANDLER_SEARCH_PARAM);
}

export function factorySetupPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/setup`;
}

export function workOrdersPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/tasks`;
}

export function createWorkOrderPath(organizationId: string, factoryKey: string) {
  return `${workOrdersPath(organizationId, factoryKey)}/new`;
}

/**
 * Canonical task permalink: `/{organizationId}/workspaces/{factoryKey}/task/{orderNumber}`
 * (singular segment, sibling of the plural `tasks` list). `orderNumber` is the
 * factory-scoped sequence number (`FactoriesWorkOrder.number`), not the
 * database id — see `legacyWorkOrderDetailPath` for the old id-based shape.
 */
export function workOrderDetailPath(
  organizationId: string,
  factoryKey: string,
  orderNumber: string | number,
  lineId?: string | null,
) {
  const path = `${factoryDetailPath(organizationId, factoryKey)}/task/${orderNumber}`;
  const boardLineId = lineId?.trim();
  if (!boardLineId) {
    return path;
  }
  return `${path}?${WORK_ORDER_LINE_SEARCH_PARAM}=${encodeURIComponent(boardLineId)}`;
}

/** Line id carried on a task permalink when the popup opened from a board. */
export const WORK_ORDER_LINE_SEARCH_PARAM = "lineId";

export function workOrderBoardLineIdFromSearch(search: string): string | null {
  const query = search.startsWith("?") ? search.slice(1) : search;
  return new URLSearchParams(query).get(WORK_ORDER_LINE_SEARCH_PARAM);
}

/**
 * Opens a task at its canonical permalink. Falls back to the line
 * board when the order has no number yet.
 */
export function workOrderOpenPath(
  organizationId: string,
  factoryKey: string,
  orderNumber: string | number | null | undefined,
  fallbackLineId?: string | null,
) {
  if (orderNumber === undefined || orderNumber === null || String(orderNumber).trim() === "") {
    return factoryHomePath(organizationId, factoryKey, fallbackLineId);
  }
  return workOrderDetailPath(organizationId, factoryKey, orderNumber);
}

/**
 * Old id-based task URL shape (`.../work-orders/{orderId}`), kept around only
 * so the legacy redirect route can compare against it / build test fixtures.
 * New code should always call `workOrderDetailPath`.
 */
export function legacyWorkOrderDetailPath(organizationId: string, factoryKey: string, orderId: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/work-orders/${orderId}`;
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

export type FactoryAppNavFrom = "automations" | "lines" | "task" | "overview";

/**
 * Parses the `from` nav hint, accepting the legacy `work-order` value (from
 * links generated before the task rename) and normalizing it to `task`.
 */
export function parseFactoryAppNavFrom(value: string | null): FactoryAppNavFrom | undefined {
  if (value === "work-order") {
    return "task";
  }
  if (value === "automations" || value === "lines" || value === "task" || value === "overview") {
    return value;
  }
  return undefined;
}

export type FactoryAppNavOptions = {
  from?: FactoryAppNavFrom;
  lineId?: string;
  /** Task `number` (route identifier), not the database id. */
  orderNumber?: string;
  runId?: string;
  /**
   * Open factory-shell Configure chrome. Uses `configure=1` (not `edit=1`) so
   * AppPage auto-edit cleanup does not tear down the Configure UI mid-bootstrap.
   */
  configure?: boolean;
  /** Open the agent sidebar in factory edit mode (`agent=1`). */
  agent?: boolean;
  /** Open the components panel in factory edit mode (`blocks=1`). */
  blocks?: boolean;
  /** Open the component sidebar on this node (`sidebar=1&node=`). */
  nodeId?: string;
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
  if (options.agent) {
    params.set("agent", "1");
  }
  if (options.blocks) {
    params.set("blocks", "1");
  }
  if (options.nodeId) {
    params.set("sidebar", "1");
    params.set("node", options.nodeId);
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
    agent: options?.agent ?? true,
    // Component edit is not run inspection. A leftover `run` param would hide
    // the editor sidebar and strip `node` while Configure starts.
    runId: options?.nodeId ? undefined : options?.runId,
  });
}

/** Factory canvas view URL. `runId` opens the dedicated run inspector. */
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

export function factoryAppSplitRunPath(
  organizationId: string,
  factoryKey: string,
  appId: string,
  options?: Omit<FactoryAppNavOptions, "configure" | "blocks"> & { canvas?: string },
) {
  const search = new URLSearchParams();
  if (options?.runId) {
    search.set("run", options.runId);
  }
  if (options?.from) {
    search.set("from", options.from);
  }
  if (options?.lineId) {
    search.set("lineId", options.lineId);
  }
  if (options?.orderNumber) {
    search.set("orderNumber", options.orderNumber);
  }
  if (options?.canvas) {
    search.set("canvas", options.canvas);
  }
  const qs = search.toString();
  return `${factoryDetailPath(organizationId, factoryKey)}/apps/${appId}/split-run${qs ? `?${qs}` : ""}`;
}

export function factorySettingsPath(organizationId: string, factoryKey: string) {
  return `${factoryDetailPath(organizationId, factoryKey)}/settings`;
}

export type FactorySettingsScope = "account" | "workspace" | "organization";

export function factorySettingsSectionPath(
  organizationId: string,
  factoryKey: string,
  scope: FactorySettingsScope,
  section: string,
) {
  return `${factorySettingsPath(organizationId, factoryKey)}/${scope}/${section}`;
}

export function factorySettingsWorkspaceGeneralPath(organizationId: string, factoryKey: string) {
  return factorySettingsSectionPath(organizationId, factoryKey, "workspace", "general");
}

export type OrganizationSettingsLocationState = {
  fromFactoryKey?: string;
};

export function organizationSettingsPath(organizationId: string) {
  return `/${organizationId}/organization`;
}

export function organizationSettingsSectionPath(organizationId: string, section: string) {
  return `${organizationSettingsPath(organizationId)}/${section}`;
}

export function organizationSettingsBackPath(organizationId: string, fromFactoryKey?: string) {
  if (fromFactoryKey) {
    return factoryDetailPath(organizationId, fromFactoryKey);
  }
  return factoryListPath(organizationId);
}

/**
 * Swaps the `/:organizationId` route segment, leaving the rest of the path
 * untouched so settings (and other workspace pages) stay in context.
 */
export function replaceOrganizationSegment(
  pathname: string,
  currentOrganizationId: string,
  nextOrganizationId: string,
): string {
  const prefix = `/${currentOrganizationId}`;
  if (pathname !== prefix && !pathname.startsWith(`${prefix}/`)) {
    return factoryListPath(nextOrganizationId);
  }
  return `/${nextOrganizationId}${pathname.slice(prefix.length)}`;
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
  return factorySettingsWorkspaceGeneralPath(organizationId, nextKey);
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
