import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactories } from "@/hooks/useFactoryData";
import { isFactoryApp } from "@/lib/canvasFlowDirection";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { factoryAppConfigurePath, factoryAppPath, factoryListPath } from "@/pages/factories/lib/factoryPagePaths";

export type ClassicAppRouteRedirect = { kind: "none" } | { kind: "wait" } | { kind: "redirect"; to: string };

export function factoryKeyForId(
  factories: Array<{ id?: string; key?: string }>,
  factoryId?: string | null,
): string | undefined {
  if (!factoryId) {
    return undefined;
  }
  return factories.find((factory) => factory.id === factoryId)?.key;
}

/**
 * Where a classic /:org/apps/:id URL should go once canvas ownership and
 * FEATURE_FACTORIES are known.
 */
export function decideClassicAppRouteRedirect({
  featureLoading,
  factoriesEnabled,
  factoriesLoading,
  factoryOwnedApp,
  factoryKey,
  organizationId,
  appId,
  runId,
}: {
  featureLoading: boolean;
  factoriesEnabled: boolean;
  factoriesLoading: boolean;
  factoryOwnedApp: boolean;
  factoryKey: string | undefined;
  organizationId: string;
  appId: string;
  runId: string | null;
}): ClassicAppRouteRedirect {
  if (!organizationId || !appId) {
    return { kind: "none" };
  }

  if (featureLoading) {
    return { kind: "wait" };
  }

  if (factoryOwnedApp) {
    if (!factoriesEnabled) {
      return { kind: "redirect", to: `/${organizationId}` };
    }
    if (factoriesLoading) {
      return { kind: "wait" };
    }
    if (!factoryKey) {
      return { kind: "redirect", to: factoryListPath(organizationId) };
    }
    if (runId) {
      return { kind: "redirect", to: factoryAppPath(organizationId, factoryKey, appId, { runId }) };
    }
    return { kind: "redirect", to: factoryAppConfigurePath(organizationId, factoryKey, appId) };
  }

  if (factoriesEnabled) {
    return { kind: "redirect", to: factoryListPath(organizationId) };
  }

  return { kind: "none" };
}

export function useClassicAppRouteRedirect({
  organizationId,
  appId,
  factoryId,
  runId,
}: {
  organizationId: string;
  appId: string;
  factoryId?: string | null;
  runId: string | null;
}): { factoryOwnedApp: boolean; classicSurface: ClassicAppRouteRedirect } {
  const factoryOwnedApp = isFactoryApp(factoryId);
  const { has, isLoading: featureLoading } = useExperimentalFeature(organizationId);
  const factoriesEnabled = has(FEATURE_FACTORIES);
  const factoriesQueryEnabled = Boolean(organizationId && factoriesEnabled && factoryOwnedApp);
  const { data: factories = [], isLoading: factoriesLoading } = useFactories(organizationId, factoriesQueryEnabled);

  return {
    factoryOwnedApp,
    classicSurface: decideClassicAppRouteRedirect({
      featureLoading,
      factoriesEnabled,
      factoriesLoading: factoriesQueryEnabled && factoriesLoading,
      factoryOwnedApp,
      factoryKey: factoryKeyForId(factories, factoryId),
      organizationId,
      appId,
      runId,
    }),
  };
}
