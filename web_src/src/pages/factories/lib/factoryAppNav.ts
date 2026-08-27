import { automationDetailPath, automationsPath, factoryHomePath, factoryLineDetailPath } from "./factoryPagePaths";

export type FactoryAppBackNav = {
  label: string;
  href: string;
};

/**
 * Resolves the route-aware back link for the factory-embedded canvas view.
 * Unknown / missing `from` falls back to the line board home.
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
        label: "Back",
        href: factoryLineDetailPath(organizationId, factoryKey, options.lineId),
      };
    }
    return { label: "Back", href: factoryHomePath(organizationId, factoryKey) };
  }

  if (from === "work-order") {
    return { label: "Back", href: factoryHomePath(organizationId, factoryKey, options.lineId) };
  }

  return { label: "Back", href: factoryHomePath(organizationId, factoryKey) };
}
