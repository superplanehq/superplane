export function factoryListPath(organizationId: string) {
  return `/${organizationId}/workspaces`;
}

export function factoryDetailPath(organizationId: string, factoryId: string) {
  return `${factoryListPath(organizationId)}/${factoryId}`;
}

export function factoryOverviewPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/overview`;
}

export function factoryOnboardingPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/onboarding`;
}

export function workOrdersPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/work-orders`;
}

export function createWorkOrderPath(organizationId: string, factoryId: string) {
  return `${workOrdersPath(organizationId, factoryId)}/new`;
}

export function workOrderDetailPath(organizationId: string, factoryId: string, orderId: string) {
  return `${workOrdersPath(organizationId, factoryId)}/${orderId}`;
}

export function linesPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/lines`;
}

export function createFactoryLinePath(organizationId: string, factoryId: string) {
  return `${linesPath(organizationId, factoryId)}/new`;
}

export function factoryLineDetailPath(organizationId: string, factoryId: string, lineId: string) {
  return `${linesPath(organizationId, factoryId)}/${lineId}`;
}

export function editFactoryLinePath(organizationId: string, factoryId: string, lineId: string) {
  return `${linesPath(organizationId, factoryId)}/${lineId}/edit`;
}

export function automationsPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/automations`;
}

export function automationDetailPath(organizationId: string, factoryId: string, appId: string) {
  return `${automationsPath(organizationId, factoryId)}/${appId}`;
}

export type FactoryAppNavFrom = "automations" | "lines" | "work-order" | "overview";

export type FactoryAppNavOptions = {
  from?: FactoryAppNavFrom;
  lineId?: string;
  orderId?: string;
  runId?: string;
  /**
   * Open factory-shell Configure chrome. Uses `configure=1` (not `edit=1`) so
   * AppPage auto-edit cleanup does not tear down the Configure UI mid-bootstrap.
   */
  configure?: boolean;
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
  if (options.from) {
    params.set("from", options.from);
  }
  if (options.lineId) {
    params.set("lineId", options.lineId);
  }
  if (options.orderId) {
    params.set("orderId", options.orderId);
  }
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

export function factoryAppConfigurePath(
  organizationId: string,
  factoryId: string,
  appId: string,
  options?: Omit<FactoryAppNavOptions, "configure" | "runId">,
) {
  return factoryAppPath(organizationId, factoryId, appId, { ...options, configure: true });
}

export function factoryAppPath(
  organizationId: string,
  factoryId: string,
  appId: string,
  options?: FactoryAppNavOptions,
) {
  return `${factoryDetailPath(organizationId, factoryId)}/apps/${appId}${buildFactoryAppSearchParams(options)}`;
}

export function factoryAppRunPath(
  organizationId: string,
  factoryId: string,
  appId: string,
  runId: string,
  options?: Omit<FactoryAppNavOptions, "runId">,
) {
  return factoryAppPath(organizationId, factoryId, appId, { ...options, runId });
}

export function factorySettingsPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/settings`;
}

export function factorySettingsSectionPath(organizationId: string, factoryId: string, section: string) {
  return `${factorySettingsPath(organizationId, factoryId)}/${section}`;
}

export function factoryMissionsPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/missions`;
}

export function factoryWikiPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/wiki`;
}

export function factoryVelocityPath(organizationId: string, factoryId: string) {
  return `${factoryDetailPath(organizationId, factoryId)}/velocity`;
}
