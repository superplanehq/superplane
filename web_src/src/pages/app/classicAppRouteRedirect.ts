import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactories } from "@/hooks/useFactoryData";
import { isFactoryApp } from "@/lib/canvasFlowDirection";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { factoryAppConfigurePath, factoryAppPath, factoryListPath } from "@/pages/factories/lib/factoryPagePaths";

export type ClassicAppRouteRedirect = { kind: "none" } | { kind: "wait" } | { kind: "redirect"; to: string };

export type ClassicAppPinnedSearch = {
  runId?: string | null;
  nodeId?: string | null;
  version?: string | null;
  edit?: boolean;
  sidebar?: boolean;
};

export function factoryKeyForId(
  factories: Array<{ id?: string; key?: string }>,
  factoryId?: string | null,
): string | undefined {
  if (!factoryId) {
    return undefined;
  }
  return factories.find((factory) => factory.id === factoryId)?.key;
}

export function pinnedSearchFromParams(searchParams: URLSearchParams): ClassicAppPinnedSearch {
  return {
    runId: searchParams.get("run"),
    nodeId: searchParams.get("node"),
    version: searchParams.get("version"),
    edit: searchParams.get("edit") === "1",
    sidebar: searchParams.get("sidebar") === "1",
  };
}

function firstNonEmpty(...values: Array<string | null | undefined>): string | undefined {
  for (const value of values) {
    const trimmed = value?.trim();
    if (trimmed) {
      return trimmed;
    }
  }
  return undefined;
}

function appendPreservedClassicSearch(path: string, pinned: ClassicAppPinnedSearch): string {
  const version = pinned.version?.trim();
  if (!version) {
    return path;
  }

  const queryStart = path.indexOf("?");
  const pathname = queryStart >= 0 ? path.slice(0, queryStart) : path;
  const params = new URLSearchParams(queryStart >= 0 ? path.slice(queryStart + 1) : "");
  params.set("version", version);
  return `${pathname}?${params.toString()}`;
}

function factoryOwnedWorkspaceAppPath(
  organizationId: string,
  factoryKey: string,
  appId: string,
  pinned: ClassicAppPinnedSearch,
): string {
  const runId = firstNonEmpty(pinned.runId);
  const nodeId = firstNonEmpty(pinned.nodeId);

  if (runId) {
    return appendPreservedClassicSearch(factoryAppPath(organizationId, factoryKey, appId, { runId, nodeId }), pinned);
  }

  return appendPreservedClassicSearch(factoryAppConfigurePath(organizationId, factoryKey, appId, { nodeId }), pinned);
}

/**
 * Where a classic /:org/apps/:id URL should go once canvas ownership and
 * FEATURE_FACTORIES are known.
 */
export function decideClassicAppRouteRedirect({
  featureLoading,
  factoriesEnabled,
  factoriesLoading,
  factoriesError = false,
  factoryOwnedApp,
  factoryKey,
  organizationId,
  appId,
  runId,
  pinned = {},
}: {
  featureLoading: boolean;
  factoriesEnabled: boolean;
  factoriesLoading: boolean;
  factoriesError?: boolean;
  factoryOwnedApp: boolean;
  factoryKey: string | undefined;
  organizationId: string;
  appId: string;
  runId: string | null;
  pinned?: ClassicAppPinnedSearch;
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
    if (factoriesLoading || factoriesError) {
      return { kind: "wait" };
    }
    if (!factoryKey) {
      return { kind: "redirect", to: factoryListPath(organizationId) };
    }
    return {
      kind: "redirect",
      to: factoryOwnedWorkspaceAppPath(organizationId, factoryKey, appId, { ...pinned, runId: pinned.runId ?? runId }),
    };
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
  searchParams,
}: {
  organizationId: string;
  appId: string;
  factoryId?: string | null;
  searchParams: URLSearchParams;
}): { factoryOwnedApp: boolean; classicSurface: ClassicAppRouteRedirect } {
  const factoryOwnedApp = isFactoryApp(factoryId);
  const { has, isLoading: featureLoading } = useExperimentalFeature(organizationId);
  const factoriesEnabled = has(FEATURE_FACTORIES);
  const factoriesQueryEnabled = Boolean(organizationId && factoriesEnabled && factoryOwnedApp);
  const {
    data: factories = [],
    isLoading: factoriesLoading,
    isError: factoriesError,
  } = useFactories(organizationId, factoriesQueryEnabled);
  const pinned = pinnedSearchFromParams(searchParams);

  return {
    factoryOwnedApp,
    classicSurface: decideClassicAppRouteRedirect({
      featureLoading,
      factoriesEnabled,
      factoriesLoading: factoriesQueryEnabled && factoriesLoading,
      factoriesError: factoriesQueryEnabled && factoriesError,
      factoryOwnedApp,
      factoryKey: factoryKeyForId(factories, factoryId),
      organizationId,
      appId,
      runId: pinned.runId ?? null,
      pinned,
    }),
  };
}
