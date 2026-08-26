import type { FactoriesWorkOrder } from "@/api-client";
import { useCreateWorkOrder } from "@/hooks/useFactoryData";
import {
  useFactoryIntakes,
  useImportFactoryIntakeItem,
  useSearchFactoryIntakeItems,
} from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useEffect, useMemo, useState } from "react";

import { useBacklogIntakeItemCatalog } from "./useBacklogIntakeItemCatalog";
import {
  BACKLOG_CREATE_COPY,
  BACKLOG_SEARCH_MAX_ITEMS,
  BACKLOG_SEARCH_PAGE_SIZE,
  listBacklogIntakeSources,
  searchBacklogIntakeItems,
  type BacklogIntakeItem,
} from "./backlogIntakeItems";

const SEARCH_DEBOUNCE_MS = 300;

export function useBacklogCreateMenu(
  organizationId: string,
  factoryId: string,
  onImported?: (orderId: string, order?: FactoriesWorkOrder) => void,
) {
  const intakesQuery = useFactoryIntakes(organizationId, factoryId);
  const catalog = useBacklogIntakeItemCatalog();
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [focusedIntakeId, setFocusedIntakeId] = useState<string | null>(null);
  const [visibleCount, setVisibleCount] = useState(BACKLOG_SEARCH_PAGE_SIZE);
  const createWorkOrder = useCreateWorkOrder(organizationId, factoryId);
  const importIntakeItem = useImportFactoryIntakeItem(organizationId, factoryId);
  const intakes = useMemo(() => intakesQuery.data ?? [], [intakesQuery.data]);
  const hasCatalog = catalog.items.length > 0;

  useEffect(() => {
    const timeout = window.setTimeout(() => setDebouncedQuery(query), SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timeout);
  }, [query]);

  useEffect(() => {
    setVisibleCount(BACKLOG_SEARCH_PAGE_SIZE);
  }, [focusedIntakeId, debouncedQuery]);

  const searchQuery = useSearchFactoryIntakeItems({
    organizationId,
    factoryId,
    intakeId: focusedIntakeId,
    query: debouncedQuery,
    enabled: !hasCatalog && Boolean(focusedIntakeId),
    limit: visibleCount,
  });

  const sources = useMemo(() => listBacklogIntakeSources({ intakes, catalog }), [intakes, catalog]);

  const focusedIntakes = useMemo(
    () => intakes.filter((intake) => intake.id === focusedIntakeId),
    [intakes, focusedIntakeId],
  );

  const catalogItems = useMemo(
    () => searchBacklogIntakeItems({ intakes: focusedIntakes, catalog, query }).flatMap((group) => group.items),
    [focusedIntakes, catalog, query],
  );

  const liveItems = useMemo(
    () => liveBacklogItems(focusedIntakeId, searchQuery.data),
    [focusedIntakeId, searchQuery.data],
  );

  const items = (hasCatalog ? catalogItems : liveItems).slice(0, visibleCount);
  const { hasMore, isLoadingMore } = backlogSearchPaging({
    hasCatalog,
    catalogCount: catalogItems.length,
    liveCount: liveItems.length,
    visibleCount,
    isFetching: Boolean(focusedIntakeId) && !hasCatalog && searchQuery.isFetching,
    isError: searchQuery.isError,
  });

  const setFocusedIntake = (intakeId: string | null) => {
    if (intakeId !== focusedIntakeId) {
      setQuery("");
      setDebouncedQuery("");
      setVisibleCount(BACKLOG_SEARCH_PAGE_SIZE);
    }
    setFocusedIntakeId(intakeId);
  };

  const loadMore = () => {
    setVisibleCount((current) => Math.min(current + BACKLOG_SEARCH_PAGE_SIZE, BACKLOG_SEARCH_MAX_ITEMS));
  };

  const importItem = async (item: BacklogIntakeItem) => {
    try {
      if (hasCatalog) {
        await createWorkOrder.mutateAsync({ title: item.title, description: item.body });
        return;
      }

      const order = await importIntakeItem.mutateAsync({ intakeId: item.intakeId, itemId: item.id });
      if (order.id) {
        onImported?.(order.id, order);
      }
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "SuperPlane could not create the work order."));
    }
  };

  const isSearching = Boolean(focusedIntakeId) && !hasCatalog && (searchQuery.isLoading || query !== debouncedQuery);

  return {
    sources,
    items,
    query,
    setQuery,
    focusedIntakeId,
    setFocusedIntake,
    isLoading: intakesQuery.isLoading || (isSearching && items.length === 0),
    isLoadingMore,
    hasMore,
    loadMore,
    errorMessage: searchQuery.isError
      ? getApiErrorMessage(searchQuery.error, BACKLOG_CREATE_COPY.unconnected)
      : undefined,
    importItem,
  };
}

function liveBacklogItems(
  intakeId: string | null,
  items: { id?: string; key?: string; title?: string; body?: string }[] | undefined,
): BacklogIntakeItem[] {
  if (!intakeId) {
    return [];
  }
  return (items ?? []).map((item) => ({
    id: item.id ?? "",
    intakeId,
    key: item.key ?? "",
    title: item.title ?? "",
    body: item.body ?? "",
  }));
}

function backlogSearchPaging(args: {
  hasCatalog: boolean;
  catalogCount: number;
  liveCount: number;
  visibleCount: number;
  isFetching: boolean;
  isError: boolean;
}): { hasMore: boolean; isLoadingMore: boolean } {
  if (args.hasCatalog) {
    return { hasMore: args.catalogCount > args.visibleCount, isLoadingMore: false };
  }

  const waitingForLargerPage = args.isFetching && args.visibleCount > args.liveCount;
  const hasMore = waitingForLargerPage || canPageLiveItems(args.liveCount, args.visibleCount, args.isError);
  return {
    hasMore,
    isLoadingMore: waitingForLargerPage || (args.isFetching && args.liveCount > 0 && hasMore),
  };
}

function canPageLiveItems(itemCount: number, visibleCount: number, isError: boolean): boolean {
  return itemCount >= visibleCount && visibleCount < BACKLOG_SEARCH_MAX_ITEMS && !isError;
}
