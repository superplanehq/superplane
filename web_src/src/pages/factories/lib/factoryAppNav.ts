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
  factoryKey: string,
  options: {
    from?: string | null;
    appId?: string | null;
    appName?: string | null;
    lineId?: string | null;
    orderNumber?: string | null;
    /** Legacy app-canvas query param; used when `orderNumber` is absent. */
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
        href: automationDetailPath(organizationId, factoryKey, options.appId),
      };
    }
    return { label: "Automations", href: automationsPath(organizationId, factoryKey) };
  }

  if (from === "lines") {
    if (options.lineId) {
      return {
        label: options.lineName?.trim() || "All lines",
        href: factoryLineDetailPath(organizationId, factoryKey, options.lineId),
      };
    }
    return { label: "All lines", href: linesPath(organizationId, factoryKey) };
  }

  if (from === "work-order") {
    const orderRef = options.orderNumber || options.orderId;
    if (orderRef) {
      return {
        label: options.orderTitle?.trim() || "Work Orders",
        href: workOrderDetailPath(organizationId, factoryKey, orderRef),
      };
    }
    return { label: "Work Orders", href: workOrdersPath(organizationId, factoryKey) };
  }

  return { label: "Overview", href: factoryOverviewPath(organizationId, factoryKey) };
}
