import { factoriesListFactoryIntakeCatalog } from "@/api-client";
import {
  intakeSurfaceEntries,
  intakeSurfaceState,
  type IntakeCatalogItem,
  type IntakeSurfaceEntry,
  type IntakeSurfaceState,
} from "@/lib/intakeCatalog";
import type { IntakeSurface } from "@/lib/intakePresentation";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";

export function intakeCatalogKey(organizationId: string) {
  return ["factories", organizationId, "intake-catalog"] as const;
}

async function fetchIntakeCatalog(organizationId: string): Promise<IntakeCatalogItem[]> {
  const response = await factoriesListFactoryIntakeCatalog(withOrganizationHeader({ organizationId }));
  return (response.data?.entries ?? []).map((entry) => ({
    key: entry.key ?? "",
    name: entry.name ?? "",
    category: entry.category ?? "",
    status: entry.status ?? "",
    available: Boolean(entry.available),
  }));
}

export interface IntakeCatalogAvailability {
  catalog: IntakeCatalogItem[];
  /** True after the catalog loaded for the organization. */
  loaded: boolean;
  loading: boolean;
  error: boolean;
  /** State for one key. Undefined while the catalog loads. */
  stateOf: (key: string) => IntakeSurfaceState | undefined;
  /** Entries that a surface lists, in display order. Empty while the catalog loads. */
  entriesFor: (surface: IntakeSurface) => IntakeSurfaceEntry[];
}

export function useIntakeCatalogAvailability(organizationId: string): IntakeCatalogAvailability {
  const query = useQuery({
    queryKey: intakeCatalogKey(organizationId),
    queryFn: () => fetchIntakeCatalog(organizationId),
    enabled: Boolean(organizationId),
    staleTime: 60 * 1000,
  });

  const catalog = useMemo(() => query.data ?? [], [query.data]);
  const loaded = query.isSuccess;

  const stateOf = useCallback(
    (key: string) => (loaded ? intakeSurfaceState(catalog.find((item) => item.key === key)) : undefined),
    [catalog, loaded],
  );
  const entriesFor = useCallback(
    (surface: IntakeSurface) => (loaded ? intakeSurfaceEntries(surface, catalog) : []),
    [catalog, loaded],
  );

  return {
    catalog,
    loaded,
    loading: query.isLoading,
    error: query.isError,
    stateOf,
    entriesFor,
  };
}
