import type { FactoriesWorkOrder } from "@/api-client";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useLayoutEffect, useRef, type ReactNode, type RefObject } from "react";

import { LineBoardColumnCardList } from "../pages/LineBoardOrderCard";
import type { MobileBoardColumn } from "./mobileBoardColumns";
import { MOBILE_BOARD_COPY } from "./mobileCopy";

export type ColumnPaging = {
  hasMore: boolean;
  isLoading: boolean;
  hasPageError: boolean;
  onLoadMore: () => void;
};

export function MobileColumn({
  column,
  hidden,
  cardsPending,
  lane,
  paging,
  renderCard,
}: {
  column: MobileBoardColumn;
  hidden: boolean;
  cardsPending: boolean;
  lane: { className?: string; surfaceClassName?: string };
  paging: ColumnPaging;
  renderCard: (order: FactoriesWorkOrder) => ReactNode;
}) {
  const listRef = useRef<HTMLUListElement>(null);
  const loadIfVisible = useVisibleColumnLoadMore(listRef, hidden, column.cards.length, paging);
  const showEmpty = !cardsPending && column.cards.length === 0 && !paging.hasMore;

  return (
    <section
      aria-label={column.title}
      aria-hidden={hidden || undefined}
      inert={hidden || undefined}
      data-testid={`mobile-board-column-${column.key}`}
      className={cn("flex h-full w-full shrink-0 snap-start flex-col p-3", lane.className, lane.surfaceClassName)}
    >
      {showEmpty ? (
        <p className="rounded-md border border-dashed border-border px-3 py-6 text-center text-[13px] text-muted-foreground">
          {column.emptyDescription}
        </p>
      ) : (
        <LineBoardColumnCardList
          ref={listRef}
          pending={cardsPending}
          className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto pb-4 [scrollbar-width:none]"
          testId={`mobile-board-column-scroll-${column.key}`}
          onScroll={loadIfVisible}
        >
          {column.cards.map((card) => (
            <li key={card.key}>{renderCard(card.order)}</li>
          ))}
        </LineBoardColumnCardList>
      )}
      {paging.isLoading ? <MobileColumnLoadingMore /> : null}
      {paging.hasPageError && !paging.isLoading ? <MobileColumnPageError onRetry={paging.onLoadMore} /> : null}
    </section>
  );
}

function MobileColumnLoadingMore() {
  return (
    <div
      role="status"
      aria-label={MOBILE_BOARD_COPY.loadingMore}
      className="mt-2 flex h-8 shrink-0 items-center justify-center"
    >
      <Loader2 className="size-4 animate-spin text-muted-foreground" aria-hidden />
    </div>
  );
}

function MobileColumnPageError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="mt-2 flex shrink-0 flex-col items-center gap-2 px-1">
      <p className="text-center text-[13px] text-destructive" role="alert">
        {MOBILE_BOARD_COPY.loadMoreError}
      </p>
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        {MOBILE_BOARD_COPY.loadMoreRetry}
      </Button>
    </div>
  );
}

function useVisibleColumnLoadMore(
  listRef: RefObject<HTMLUListElement | null>,
  hidden: boolean,
  cardCount: number,
  paging: ColumnPaging,
) {
  const pageErrorRef = useRef(paging.hasPageError);
  pageErrorRef.current = paging.hasPageError;
  const loadMoreIfNeeded = useAutoLoadMoreOnScroll({
    hasMore: paging.hasMore && !paging.hasPageError,
    isLoading: paging.isLoading,
    onLoadMore: paging.onLoadMore,
  });
  const hiddenRef = useRef(hidden);
  hiddenRef.current = hidden;
  const loadIfVisible = useCallback(
    (element: HTMLElement | null) => {
      if (hiddenRef.current || pageErrorRef.current) {
        return;
      }
      loadMoreIfNeeded(element);
    },
    [loadMoreIfNeeded],
  );

  useLayoutEffect(() => {
    loadIfVisible(listRef.current);
  }, [cardCount, hidden, paging.hasPageError, listRef, loadIfVisible]);

  useEffect(() => {
    const list = listRef.current;
    if (!list || hidden || paging.hasPageError) {
      return;
    }
    const observer = new ResizeObserver(() => {
      loadIfVisible(list);
    });
    observer.observe(list);
    return () => observer.disconnect();
  }, [cardCount, hidden, paging.hasPageError, listRef, loadIfVisible]);

  return loadIfVisible;
}
