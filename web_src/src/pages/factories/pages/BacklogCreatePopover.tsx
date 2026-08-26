import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Loader2, Plus, SquarePen } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import {
  BACKLOG_CREATE_COPY,
  searchPlaceholderForIntake,
  type BacklogIntakeItem,
  type BacklogIntakeSource,
} from "./backlogIntakeItems";

function CreateTriggerButton({
  canAdd,
  atCapacity,
  onClick,
}: {
  canAdd: boolean;
  atCapacity: boolean;
  onClick?: () => void;
}) {
  return (
    <PermissionTooltip allowed={canAdd} message="You don't have permission to create work orders.">
      <button
        type="button"
        disabled={!canAdd}
        aria-label={BACKLOG_CREATE_COPY.createWorkOrder}
        title={atCapacity ? "The backlog is full." : BACKLOG_CREATE_COPY.createWorkOrder}
        data-testid="lines-backlog-create"
        onClick={onClick}
        className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:pointer-events-none disabled:opacity-60"
      >
        <Plus className="size-3.5" aria-hidden />
      </button>
    </PermissionTooltip>
  );
}

function GhostCardOutline() {
  return (
    <svg
      className="pointer-events-none absolute inset-0 size-full text-border transition-colors group-hover/create-ghost:text-foreground/40"
      aria-hidden
    >
      <rect
        x="1"
        y="1"
        width="calc(100% - 2px)"
        height="calc(100% - 2px)"
        rx="6"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeDasharray="12 8"
        strokeLinecap="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

function CreateGhostCard({
  canAdd,
  atCapacity,
  onClick,
}: {
  canAdd: boolean;
  atCapacity: boolean;
  onClick?: () => void;
}) {
  return (
    <PermissionTooltip allowed={canAdd} message="You don't have permission to create work orders." className="w-full">
      <button
        type="button"
        disabled={!canAdd}
        aria-label={BACKLOG_CREATE_COPY.createWorkOrder}
        title={atCapacity ? "The backlog is full." : BACKLOG_CREATE_COPY.createWorkOrder}
        data-testid="lines-backlog-create-ghost"
        onClick={onClick}
        className="group/create-ghost relative flex min-h-[4.5rem] w-full flex-col items-center justify-center gap-1.5 rounded-md bg-muted/40 px-2.5 py-5 text-[13px] font-medium tracking-[-0.01em] text-muted-foreground transition-colors hover:bg-muted/70 hover:text-foreground disabled:pointer-events-none disabled:opacity-60"
      >
        <GhostCardOutline />
        <Plus className="size-4" aria-hidden />
        {BACKLOG_CREATE_COPY.createWorkOrder}
      </button>
    </PermissionTooltip>
  );
}

function SearchLoadingStatus({ label, testId }: { label: string; testId: string }) {
  return (
    <div className="flex items-center gap-2 px-2 py-2 text-[13px] text-muted-foreground" data-testid={testId}>
      <Loader2 className="size-3.5 shrink-0 animate-spin" aria-hidden />
      {label}
    </div>
  );
}

function IntakeSourceIcon({ source }: { source: BacklogIntakeSource }) {
  if (source.iconSrc) {
    return <img src={source.iconSrc} alt="" className="size-3.5 shrink-0" />;
  }

  return (
    <span className="flex size-3.5 shrink-0 items-center justify-center rounded-sm bg-muted text-[9px] font-medium">
      {source.iconAlt.slice(0, 1)}
    </span>
  );
}

export function BacklogCreatePopover({
  canAdd,
  atCapacity = false,
  variant = "icon",
  sources,
  items,
  query,
  focusedIntakeId,
  onQueryChange,
  onFocusedIntakeChange,
  onCreateManually,
  onImportItem,
  isLoading = false,
  isLoadingMore = false,
  hasMore = false,
  onLoadMore,
  errorMessage,
}: {
  canAdd: boolean;
  atCapacity?: boolean;
  variant?: "icon" | "ghost";
  sources: BacklogIntakeSource[];
  items: BacklogIntakeItem[];
  query: string;
  focusedIntakeId: string | null;
  onQueryChange: (query: string) => void;
  onFocusedIntakeChange: (intakeId: string | null) => void;
  onCreateManually: () => void;
  onImportItem: (item: BacklogIntakeItem) => void;
  isLoading?: boolean;
  isLoadingMore?: boolean;
  hasMore?: boolean;
  onLoadMore?: () => void;
  errorMessage?: string;
}) {
  const [open, setOpen] = useState(false);
  const resultsRef = useRef<HTMLDivElement>(null);
  const loadMoreIfNeeded = useAutoLoadMoreOnScroll({
    hasMore,
    isLoading: isLoading || isLoadingMore,
    onLoadMore,
  });

  useEffect(() => {
    loadMoreIfNeeded(resultsRef.current);
  }, [items.length, hasMore, isLoading, isLoadingMore, loadMoreIfNeeded]);

  const close = () => {
    setOpen(false);
    onFocusedIntakeChange(null);
    onQueryChange("");
  };

  const Trigger = variant === "ghost" ? CreateGhostCard : CreateTriggerButton;

  if (!isLoading && sources.length === 0) {
    return <Trigger canAdd={canAdd} atCapacity={atCapacity} onClick={canAdd ? onCreateManually : undefined} />;
  }

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        if (!canAdd) {
          return;
        }
        setOpen(nextOpen);
        if (!nextOpen) {
          onFocusedIntakeChange(null);
          onQueryChange("");
        }
      }}
    >
      <PopoverTrigger asChild>
        <span className={variant === "ghost" ? "flex w-full" : "inline-flex"}>
          <Trigger canAdd={canAdd} atCapacity={atCapacity} />
        </span>
      </PopoverTrigger>
      <PopoverContent
        side="right"
        align="start"
        className="w-80 p-1.5"
        sideOffset={6}
        data-testid="lines-backlog-create-menu"
      >
        <button
          type="button"
          className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-accent"
          data-testid="lines-backlog-create-manually"
          onClick={() => {
            close();
            onCreateManually();
          }}
        >
          <SquarePen className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          {BACKLOG_CREATE_COPY.createManually}
        </button>
        {sources.map((source) => {
          const focused = focusedIntakeId === source.intakeId;
          return (
            <div key={source.intakeId} data-testid={`lines-backlog-create-source-${source.intakeId}`}>
              <label className="mt-0.5 flex items-center gap-2 px-2 py-1">
                <IntakeSourceIcon source={source} />
                <Input
                  value={focused ? query : ""}
                  onChange={(event) => onQueryChange(event.target.value)}
                  onFocus={() => onFocusedIntakeChange(source.intakeId)}
                  placeholder={searchPlaceholderForIntake(source.name)}
                  data-testid={`lines-backlog-create-search-${source.intakeId}`}
                  className="h-7 px-2 text-[13px]"
                />
              </label>
              {focused ? (
                <div
                  ref={resultsRef}
                  className="mt-0.5 mb-1 ml-7 flex max-h-36 flex-col overflow-y-auto"
                  data-testid={`lines-backlog-create-items-${source.intakeId}`}
                  onScroll={(event) => loadMoreIfNeeded(event.currentTarget)}
                >
                  {isLoading && items.length === 0 ? (
                    <SearchLoadingStatus label={BACKLOG_CREATE_COPY.loading} testId="lines-backlog-create-loading" />
                  ) : errorMessage ? (
                    <div className="px-2 py-2 text-[13px] text-muted-foreground">{errorMessage}</div>
                  ) : items.length === 0 ? (
                    <div className="px-2 py-2 text-[13px] text-muted-foreground">{BACKLOG_CREATE_COPY.empty}</div>
                  ) : (
                    <>
                      {items.map((item) => (
                        <button
                          key={item.id}
                          type="button"
                          className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-accent"
                          data-testid={`lines-backlog-create-item-${item.id}`}
                          onClick={() => {
                            close();
                            onImportItem(item);
                          }}
                        >
                          <span className="min-w-0 flex-1 truncate">{item.title}</span>
                          <span className="shrink-0 text-[12px] text-muted-foreground">{item.key}</span>
                        </button>
                      ))}
                      {isLoadingMore ? (
                        <SearchLoadingStatus
                          label={BACKLOG_CREATE_COPY.loadingMore}
                          testId="lines-backlog-create-loading-more"
                        />
                      ) : null}
                    </>
                  )}
                </div>
              ) : null}
            </div>
          );
        })}
      </PopoverContent>
    </Popover>
  );
}
