import type { CustomAutomationColumn } from "./customAutomation";
import { factoryHomePath } from "./factoryPagePaths";

const CUSTOM_AUTOMATION_PARAM = "customAutomation";
const LEGACY_CUSTOM_AUTOMATION_PARAM = "customVerification";
const NEW_CUSTOM_AUTOMATION = "new";

export function factoryCustomAutomationSetupPath(
  organizationId: string,
  factoryKey: string,
  automationId?: string,
  columnKey?: CustomAutomationColumn,
) {
  const params = new URLSearchParams({
    [CUSTOM_AUTOMATION_PARAM]: automationId?.trim() || NEW_CUSTOM_AUTOMATION,
  });
  if (columnKey) params.set("automationColumn", columnKey);
  return `${factoryHomePath(organizationId, factoryKey)}?${params.toString()}`;
}

/** Open the automation modal on one run. */
export function factoryCustomAutomationRunPath(
  organizationId: string,
  factoryKey: string,
  automationId: string,
  runId: string,
  columnKey: CustomAutomationColumn = "backlog",
) {
  const params = new URLSearchParams({
    [CUSTOM_AUTOMATION_PARAM]: automationId,
    automationColumn: columnKey,
    automationTab: "runs",
    automationRun: runId,
  });
  return `${factoryHomePath(organizationId, factoryKey)}?${params.toString()}`;
}

export function customAutomationIdFromSearch(search: string): string | null | undefined {
  const params = new URLSearchParams(search);
  const value = params.get(CUSTOM_AUTOMATION_PARAM)?.trim() || params.get(LEGACY_CUSTOM_AUTOMATION_PARAM)?.trim();
  if (!value) {
    return undefined;
  }
  return value === NEW_CUSTOM_AUTOMATION ? null : value;
}
