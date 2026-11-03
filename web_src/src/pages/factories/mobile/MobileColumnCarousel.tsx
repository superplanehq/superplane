import type { FactoriesWorkOrder } from "@/api-client";
import { cn } from "@/lib/utils";
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { useLocation, useSearchParams } from "react-router";
import { BOARD_COLUMN_SEARCH_PARAM, boardColumnKeyFromSearch } from "../lib/boardColumnSearch";
import { PhaseGlyph } from "../pages/linePhaseGlyph";
import { MobileColumn, type ColumnPaging } from "./MobileBoardColumn";
import { activeColumnIndex, type MobileBoardColumn } from "./mobileBoardColumns";

/**
 * Horizontal snap carousel. Each column fills the screen width and scrolls
 * on its own. Tabs above mirror the swipe position and jump on tap.
 *
 * The active column is the `column` search param, so it survives opening a
 * task and Back to board. A key that is missing or not in `columns` shows
 * Backlog. The track scrolls to a restored key once, when that key first
 * appears in `columns`.
 */
export function MobileColumnCarousel({
  columns,
  cardsPending,
  laneClassName,
  paging,
  renderCard,
  renderColumnAction,
}: {
  renderColumnAction: (column: MobileBoardColumn) => ReactNode;
  columns: MobileBoardColumn[];
  cardsPending: boolean;
  laneClassName: (column: MobileBoardColumn) => { className?: string; surfaceClassName?: string };
  paging: Record<MobileBoardColumn["paging"], ColumnPaging>;
  renderCard: (column: MobileBoardColumn, order: FactoriesWorkOrder) => ReactNode;
}) {
  const trackRef = useRef<HTMLDivElement>(null);
  const { search } = useLocation();
  const [, setSearchParams] = useSearchParams();
  const requestedColumnKey = boardColumnKeyFromSearch(search);
  const [initialColumnKey] = useState(requestedColumnKey);
  const restoredRef = useRef(false);
  const requestedIndex = columns.findIndex((column) => column.key === requestedColumnKey);
  const activeIndex = requestedIndex >= 0 ? requestedIndex : 0;

  const selectColumn = useCallback(
    (index: number) => {
      const key = columns[index]?.key;
      if (!key) {
        return;
      }
      restoredRef.current = true;
      setSearchParams(
        (previous) => {
          const next = new URLSearchParams(previous);
          next.set(BOARD_COLUMN_SEARCH_PARAM, key);
          return next;
        },
        { replace: true },
      );
    },
    [columns, setSearchParams],
  );

  useLayoutEffect(() => {
    const track = trackRef.current;
    if (restoredRef.current || !track || !initialColumnKey) {
      return;
    }
    const index = columns.findIndex((column) => column.key === initialColumnKey);
    if (index < 0) {
      return;
    }
    restoredRef.current = true;
    track.scrollTo({ left: index * track.clientWidth, behavior: "auto" });
  }, [columns, initialColumnKey]);

  const syncActiveFromScroll = useCallback(() => {
    const track = trackRef.current;
    if (!track) {
      return;
    }
    const index = activeColumnIndex(track.scrollLeft, track.clientWidth, columns.length);
    if (index !== activeIndex) {
      selectColumn(index);
    }
  }, [activeIndex, columns.length, selectColumn]);

  const scrollToColumn = useCallback(
    (index: number) => {
      const track = trackRef.current;
      if (!track) {
        return;
      }
      track.scrollTo({ left: index * track.clientWidth, behavior: "smooth" });
      selectColumn(index);
    },
    [selectColumn],
  );

  return (
    <>
      <MobileColumnTabs columns={columns} activeIndex={activeIndex} onSelect={scrollToColumn} />
      <div
        ref={trackRef}
        onScroll={syncActiveFromScroll}
        className="flex min-h-0 flex-1 snap-x snap-mandatory overflow-x-auto overflow-y-hidden [scrollbar-width:none]"
        data-testid="mobile-board-track"
      >
        {columns.map((column, index) => (
          <MobileColumn
            key={column.key}
            column={column}
            action={renderColumnAction(column)}
            hidden={index !== activeIndex}
            cardsPending={cardsPending}
            lane={laneClassName(column)}
            paging={paging[column.paging]}
            renderCard={(order) => renderCard(column, order)}
          />
        ))}
      </div>
    </>
  );
}

function MobileColumnTabs({
  columns,
  activeIndex,
  onSelect,
}: {
  columns: MobileBoardColumn[];
  activeIndex: number;
  onSelect: (index: number) => void;
}) {
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);
  useEffect(() => {
    tabRefs.current[activeIndex]?.scrollIntoView({ inline: "center", block: "nearest", behavior: "smooth" });
  }, [activeIndex]);

  return (
    <div
      role="tablist"
      aria-label="Board columns"
      className="flex shrink-0 gap-1 overflow-x-auto border-b border-border px-4 [scrollbar-width:none]"
      data-testid="mobile-board-tabs"
    >
      {columns.map((column, index) => {
        const active = index === activeIndex;
        return (
          <button
            key={column.key}
            ref={(element) => {
              tabRefs.current[index] = element;
            }}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onSelect(index)}
            data-testid={`mobile-board-tab-${column.key}`}
            className={cn(
              "flex h-12 shrink-0 items-center gap-1.5 border-b-2 px-2.5 text-[13px] font-medium tracking-[-0.01em] transition-colors",
              active ? "border-foreground text-foreground" : "border-transparent text-muted-foreground",
            )}
          >
            {column.glyph ? <PhaseGlyph kind={column.glyph} className="size-3" /> : null}
            <span className="truncate">{column.title}</span>
            <span className="tabular-nums text-[12px] text-muted-foreground">
              {column.totalCount ?? column.cards.length}
            </span>
          </button>
        );
      })}
    </div>
  );
}
