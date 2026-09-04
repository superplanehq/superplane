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
  file?: string | null;
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
    file: searchParams.get("file"),
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
  const file = pinned.file?.trim();
  if (!version && !file) {
    return path;
  }

  const queryStart = path.indexOf("?");
  const pathname = queryStart >= 0 ? path.slice(0, queryStart) : path;
  const params = new URLSearchParams(queryStart >= 0 ? path.slice(queryStart + 1) : "");
  if (version) {
    params.set("version", version);
  }
  if (file) {
    params.set("file", file);
  }
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
  const wantsEditor = Boolean(pinned.edit || nodeId || pinned.sidebar || firstNonEmpty(pinned.file));

  if (runId && !wantsEditor) {
    return appendPreservedClassicSearch(factoryAppPath(organizationId, factoryKey, appId, { runId }), pinned);
  }

  if (runId && nodeId && !pinned.edit) {
    return appendPreservedClassicSearch(factoryAppPath(organizationId, factoryKey, appId, { runId, nodeId }), pinned);
  }

  return appendPreservedClassicSearch(
    factoryAppConfigurePath(organizationId, factoryKey, appId, { nodeId, runId }),
    pinned,
  );
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
  pinned = {},
}: {
  featureLoading: boolean;
  factoriesEnabled: boolean;
  factoriesLoading: boolean;
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
    if (factoriesLoading) {
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
  const { data: factories = [], isLoading: factoriesLoading } = useFactories(organizationId, factoriesQueryEnabled);
  const pinned = pinnedSearchFromParams(searchParams);

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
      runId: pinned.runId ?? null,
      pinned,
    }),
  };
}
