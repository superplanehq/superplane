import { factoryLineDetailPath } from "./factoryPagePaths";

const CUSTOM_AUTOMATION_PARAM = "customAutomation";
const LEGACY_CUSTOM_AUTOMATION_PARAM = "customVerification";
const NEW_CUSTOM_AUTOMATION = "new";

export function factoryCustomAutomationSetupPath(
  organizationId: string,
  factoryKey: string,
  lineId: string,
  automationId?: string,
  columnKey?: "backlog" | "verify",
) {
  const params = new URLSearchParams({
    [CUSTOM_AUTOMATION_PARAM]: automationId?.trim() || NEW_CUSTOM_AUTOMATION,
  });
  if (columnKey) params.set("automationColumn", columnKey);
  return `${factoryLineDetailPath(organizationId, factoryKey, lineId)}?${params.toString()}`;
}

export function customAutomationIdFromSearch(search: string): string | null | undefined {
  const params = new URLSearchParams(search);
  const value = params.get(CUSTOM_AUTOMATION_PARAM)?.trim() || params.get(LEGACY_CUSTOM_AUTOMATION_PARAM)?.trim();
  if (!value) {
    return undefined;
  }
  return value === NEW_CUSTOM_AUTOMATION ? null : value;
}
