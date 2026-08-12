import {
  automationDetailPath,
  automationsPath,
  factoryLineDetailPath,
  factoryOverviewPath,
  linesPath,
  workOrderDetailPath,
  workOrdersPath,
} from "./factoryPagePaths";

export type FactoryAppBackNav = {
  label: string;
  href: string;
};

/**
 * Resolves the route-aware back link for the factory-embedded canvas view.
 * Unknown / missing `from` falls back to Overview.
 */
export function resolveFactoryAppBackNav(
  organizationId: string,
  factoryId: string,
  options: {
    from?: string | null;
    appId?: string | null;
    appName?: string | null;
    lineId?: string | null;
    orderId?: string | null;
    lineName?: string | null;
    orderTitle?: string | null;
  },
): FactoryAppBackNav {
  const from = options.from;

  if (from === "automations") {
    if (options.appId) {
      return {
        label: options.appName?.trim() || "Automations",
        href: automationDetailPath(organizationId, factoryId, options.appId),
      };
    }
    return { label: "Automations", href: automationsPath(organizationId, factoryId) };
  }

  if (from === "lines") {
    if (options.lineId) {
      return {
        label: options.lineName?.trim() || "All lines",
        href: factoryLineDetailPath(organizationId, factoryId, options.lineId),
      };
    }
    return { label: "All lines", href: linesPath(organizationId, factoryId) };
  }

  if (from === "work-order") {
    if (options.orderId) {
      return {
        label: options.orderTitle?.trim() || "Work Orders",
        href: workOrderDetailPath(organizationId, factoryId, options.orderId),
      };
    }
    return { label: "Work Orders", href: workOrdersPath(organizationId, factoryId) };
  }

  return { label: "Overview", href: factoryOverviewPath(organizationId, factoryId) };
}
