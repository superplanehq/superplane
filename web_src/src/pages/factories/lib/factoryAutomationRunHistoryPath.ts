import { factoryDetailPath, type FactoryAppNavOptions } from "./factoryPagePaths";

/** Read-only run history and details, without the automation canvas. */
export function factoryAutomationRunHistoryPath(
  organizationId: string,
  factoryKey: string,
  appId: string,
  runId?: string | null,
  options?: Pick<FactoryAppNavOptions, "from" | "lineId" | "orderNumber">,
) {
  const params = new URLSearchParams();
  if (options?.from) params.set("from", options.from);
  if (options?.lineId) params.set("lineId", options.lineId);
  if (options?.orderNumber) params.set("orderNumber", options.orderNumber);
  const path = `${factoryDetailPath(organizationId, factoryKey)}/automations/${appId}/runs`;
  const search = params.toString();
  return `${path}${runId ? `/${runId}` : ""}${search ? `?${search}` : ""}`;
}
