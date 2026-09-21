import type { FactoriesFactoryIntakeSettings, FactoryIntakeHealth, OrganizationsIntegration } from "@/api-client";

import { factoryIntakePath } from "../lib/factoryPagePaths";
import { repairActionLabel } from "../lib/brokenIntegrations";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export type IntakeConnectionBinding = {
  integrationId: string;
  resourceId: string;
};

export function intakeSourceAllowsRebind(sourceId: LineIntakeSourceId): boolean {
  return sourceId === "jira-issues" || sourceId === "sentry-exceptions" || sourceId === "productive-tasks";
}

export function intakeProviderAppName(sourceId: LineIntakeSourceId): string {
  if (sourceId === "jira-issues") {
    return "jira";
  }
  if (sourceId === "sentry-exceptions") {
    return "sentry";
  }
  return "productive";
}

export function intakeProviderDisplayName(sourceId: LineIntakeSourceId): string {
  if (sourceId === "jira-issues") {
    return "Jira";
  }
  if (sourceId === "sentry-exceptions") {
    return "Sentry";
  }
  return "Productive.io";
}

export const INTAKE_CONNECTION_COPY = {
  section: "Connection",
  chooseJira: "Choose the Jira site that SuperPlane will monitor.",
  chooseSentry: "Choose the Sentry organization that SuperPlane will monitor.",
  chooseProductive: "Choose the Productive.io account that SuperPlane will monitor.",
  project: "Project",
  connect: "Connect",
  connecting: "Connecting...",
  loadingConnections: "Loading connections...",
  loadingProjects: "Loading projects...",
  projectsError: "SuperPlane could not load projects.",
  projectsEmpty: "This connection has no available projects.",
  retry: "Try again",
  missing: "This intake has no live connection.",
  notReady: "This connection cannot receive items.",
  webhook: "SuperPlane is still registering the Jira webhook.",
  graph: "This automation cannot create tasks. Open the Automation tab to repair the steps.",
  connectError: "SuperPlane could not open the connection setup.",
} as const;

export function intakeChooseConnectionCopy(sourceId: LineIntakeSourceId): string {
  if (sourceId === "jira-issues") {
    return INTAKE_CONNECTION_COPY.chooseJira;
  }
  if (sourceId === "sentry-exceptions") {
    return INTAKE_CONNECTION_COPY.chooseSentry;
  }
  return INTAKE_CONNECTION_COPY.chooseProductive;
}

export function intakeHealthBanner(health: FactoryIntakeHealth | undefined): string | undefined {
  if (health === "HEALTH_MISSING_INTEGRATION") {
    return INTAKE_CONNECTION_COPY.missing;
  }
  if (health === "HEALTH_INTEGRATION_NOT_READY") {
    return INTAKE_CONNECTION_COPY.notReady;
  }
  if (health === "HEALTH_WEBHOOK_NOT_READY") {
    return INTAKE_CONNECTION_COPY.webhook;
  }
  if (health === "HEALTH_GRAPH_BROKEN") {
    return INTAKE_CONNECTION_COPY.graph;
  }
  return undefined;
}

export function intakeConnectionChanged(current: IntakeConnectionBinding, next: IntakeConnectionBinding): boolean {
  return current.integrationId !== next.integrationId || current.resourceId !== next.resourceId;
}

export function intakeConnectionComplete(binding: IntakeConnectionBinding): boolean {
  return binding.integrationId.trim().length > 0 && binding.resourceId.trim().length > 0;
}

export function filterIntakeConnections(
  integrations: OrganizationsIntegration[],
  sourceId: LineIntakeSourceId,
): OrganizationsIntegration[] {
  const appName = intakeProviderAppName(sourceId);
  return integrations.filter(
    (integration) => integration.metadata?.integrationName === appName && integration.metadata?.id,
  );
}

export function selectedIntakeIntegration(
  integrations: OrganizationsIntegration[],
  integrationId: string,
): OrganizationsIntegration | undefined {
  return integrations.find((integration) => integration.metadata?.id === integrationId);
}

export function intakeReconnectLabel(integration: OrganizationsIntegration | undefined): string {
  if (!integration || integration.status?.state === "ready") {
    return INTAKE_CONNECTION_COPY.connect;
  }
  return repairActionLabel(integration.status?.stateDescription, integration.metadata?.integrationName);
}

export function showIntakeConnectAction(integrationCount: number, integrationsLoading: boolean): boolean {
  return integrationCount === 0 && !integrationsLoading;
}

export function intakeConnectLabel(providerName: string): string {
  return `${INTAKE_CONNECTION_COPY.connect} ${providerName}`;
}

export function intakeConnectionFromSource(source: {
  integrationId?: string;
  resourceId?: string;
}): IntakeConnectionBinding {
  return {
    integrationId: source.integrationId?.trim() ?? "",
    resourceId: source.resourceId?.trim() ?? "",
  };
}

export function intakeConnectionReturnPath(
  organizationId: string,
  factoryKey: string,
  lineId: string | undefined,
  intakeId: string,
): string {
  return factoryIntakePath(organizationId, factoryKey, lineId, intakeId, "general");
}

export function canSaveIntakeConnection(
  allowsRebind: boolean,
  health: FactoryIntakeHealth | undefined,
  current: IntakeConnectionBinding,
  next: IntakeConnectionBinding,
): boolean {
  if (!allowsRebind) {
    return true;
  }
  if (intakeConnectionChanged(current, next) || health === "HEALTH_MISSING_INTEGRATION") {
    return intakeConnectionComplete(next);
  }
  return true;
}

export function factoryIntakeUpdateInput(
  intakeId: string,
  settings: FactoriesFactoryIntakeSettings,
  current: IntakeConnectionBinding,
  next: IntakeConnectionBinding,
  allowsRebind: boolean,
): {
  intakeId: string;
  settings: FactoriesFactoryIntakeSettings;
  integrationId?: string;
  resourceId?: string;
} {
  if (!allowsRebind || !intakeConnectionChanged(current, next) || !intakeConnectionComplete(next)) {
    return { intakeId, settings };
  }
  return {
    intakeId,
    settings,
    integrationId: next.integrationId,
    resourceId: next.resourceId,
  };
}
