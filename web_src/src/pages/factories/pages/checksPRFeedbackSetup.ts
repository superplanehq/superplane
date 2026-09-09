import type { OrganizationsIntegrationResourceRef } from "@/api-client";

export type ChecksPRFeedbackSetupStep = "checks" | "tools";

export const CHECKS_HANDLER_CI_INTEGRATIONS = new Set(["semaphore", "circleci", "harness", "cloudflare", "cloudsmith"]);

export function isChecksHandlerCIIntegration(name: string | undefined): boolean {
  return CHECKS_HANDLER_CI_INTEGRATIONS.has(name?.trim().toLowerCase() ?? "");
}

export function catalogStatusCheckNames(catalog: OrganizationsIntegrationResourceRef[]): string[] {
  return catalog.flatMap((check) => (check.name?.trim() ? [check.name.trim()] : []));
}

export function suggestIntegrationFromCheckURL(rawURL: string | undefined): string {
  const value = rawURL?.trim();
  if (!value) {
    return "";
  }
  let host = "";
  try {
    host = new URL(value).hostname.toLowerCase();
  } catch {
    return "";
  }
  if (!host) {
    return "";
  }
  if (hostHasSuffix(host, "semaphoreci.com") || hostHasSuffix(host, "semaphore.com")) {
    return "semaphore";
  }
  if (hostHasSuffix(host, "circleci.com")) {
    return "circleci";
  }
  if (hostHasSuffix(host, "harness.io")) {
    return "harness";
  }
  return "";
}

function hostHasSuffix(host: string, suffix: string): boolean {
  return host === suffix || host.endsWith(`.${suffix}`);
}

export function suggestedIntegrationsForChecks(
  catalog: OrganizationsIntegrationResourceRef[],
  selectedNames: string[],
): string[] {
  const selected = new Set(selectedNames.map((name) => name.toLowerCase()));
  const names = new Set<string>();
  for (const check of catalog) {
    if (!check.name || !selected.has(check.name.toLowerCase())) {
      continue;
    }
    const integration = suggestIntegrationFromCheckURL(check.url);
    if (!isChecksHandlerCIIntegration(integration)) {
      continue;
    }
    names.add(integration);
  }
  return [...names];
}

export type ChecksHandlerIntegrationRow = {
  type: string;
  displayName: string;
  instanceId?: string;
};

export function readyChecksHandlerIntegrationIds(
  connected: Array<{ status?: { state?: string }; metadata?: { id?: string; integrationName?: string } }>,
): string[] {
  return connected.flatMap((integration) => {
    const id = integration.metadata?.id;
    if (
      !id ||
      integration.status?.state !== "ready" ||
      !isChecksHandlerCIIntegration(integration.metadata?.integrationName)
    ) {
      return [];
    }
    return [id];
  });
}

export function checksHandlerIntegrationRows(
  definitions: Array<{ name?: string; label?: string }>,
  ready: Array<{ metadata?: { id?: string; name?: string; integrationName?: string } }>,
): ChecksHandlerIntegrationRow[] {
  const rows: ChecksHandlerIntegrationRow[] = [];
  for (const definition of definitions) {
    const type = definition.name?.trim();
    if (!type) {
      continue;
    }
    const label = definition.label?.trim() || type;
    const instances = ready.filter((item) => item.metadata?.integrationName === type && item.metadata?.id);
    if (instances.length === 0) {
      rows.push({ type, displayName: label });
      continue;
    }
    for (const instance of instances) {
      const instanceId = instance.metadata?.id;
      if (!instanceId) {
        continue;
      }
      rows.push({
        type,
        displayName: instance.metadata?.name?.trim() || label,
        instanceId,
      });
    }
  }
  return rows;
}

export function hasSelectedSuggestedIntegration(
  suggestedNames: string[],
  selectedIds: string[],
  ready: Array<{ metadata?: { id?: string; integrationName?: string } }>,
): boolean {
  if (suggestedNames.length === 0) {
    return false;
  }
  const selected = new Set(selectedIds);
  return ready.some((item) => {
    const id = item.metadata?.id;
    const type = item.metadata?.integrationName?.trim().toLowerCase();
    return Boolean(id && type && selected.has(id) && suggestedNames.includes(type));
  });
}
