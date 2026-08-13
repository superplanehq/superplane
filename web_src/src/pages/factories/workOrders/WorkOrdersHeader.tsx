import type { FactoriesFactoryLine } from "@/api-client";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Check, Columns3, Funnel, List, Plus, Search, Settings2, Table as TableIcon, X } from "lucide-react";
import { useEffect, useRef, type KeyboardEvent } from "react";
import { WORK_ORDER_DISPLAY_STATUSES, getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import {
  UNASSIGNED_FILTER_VALUE,
  WORK_ORDER_LAYOUTS,
  WORK_ORDER_ORDERINGS,
  WORK_ORDER_SCOPES,
  type WorkOrderLayoutId,
  type WorkOrderListEntry,
  type WorkOrderOrdering,
  type WorkOrderScope,
} from "../lib/workOrderListModel";
import type { WorkOrderFilterDimension, WorkOrderListState } from "../lib/useWorkOrderListState";

const LAYOUT_ICONS: Record<WorkOrderLayoutId, typeof Columns3> = {
  board: Columns3,
  list: List,
  table: TableIcon,
};

const TRIGGER_CLASSNAME = "h-8 gap-1.5 px-2.5 text-muted-foreground";
const MENU_ITEM_CLASSNAME = "cursor-pointer text-[13px]";
const MENU_LABEL_CLASSNAME = "text-[11px] font-medium tracking-[0.04em] text-muted-foreground";

interface WorkOrdersHeaderProps {
  state: WorkOrderListState;
  /** Every entry before scope/filters, used to build the assignee options. */
  entries: WorkOrderListEntry[];
  factoryLines: FactoriesFactoryLine[];
  createHref: string;
  canCreate: boolean;
  permissionsLoading: boolean;
}

/**
 * Title bar for the Work Orders page. Everything lives on one row: the
 * page title and scope pills on the left, and the Filter menu, collapsible
 * search, Display menu, and New button on the right. Layout and ordering
 * sit inside Display rather than on the bar so the row stays quiet.
 *
 * Keyboard shortcuts: `F` opens the Filter menu, `/` opens and focuses
 * search. Both bail out while the user types so we never eat the key.
 */
export function WorkOrdersHeader({
  state,
  entries,
  factoryLines,
  createHref,
  canCreate,
  permissionsLoading,
}: WorkOrdersHeaderProps) {
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    function handleShortcut(event: globalThis.KeyboardEvent) {
      if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }
      const target = event.target as HTMLElement | null;
      if (target && isEditableTarget(target)) {
        return;
      }
      if (event.key === "/") {
        event.preventDefault();
        state.openSearch();
        return;
      }
      if (event.key === "f" || event.key === "F") {
        event.preventDefault();
        state.setFilterMenuOpen(true);
      }
    }
    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, [state]);

  useEffect(() => {
    if (state.searchOpen) {
      searchRef.current?.focus();
    }
  }, [state.searchOpen]);

  return (
    <div className="flex w-full flex-col" data-testid="work-orders-header">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-wrap items-center gap-3">
          <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-foreground">Work Orders</h1>
          <ScopePills value={state.scope} onChange={state.setScope} />
        </div>

        <div className="flex items-center gap-1">
          <FilterMenu state={state} entries={entries} factoryLines={factoryLines} />
          <SearchField
            inputRef={searchRef}
            open={state.searchOpen}
            value={state.search}
            onOpen={state.openSearch}
            onChange={state.setSearch}
            onClose={state.closeSearch}
          />
          <DisplayMenu state={state} />
          <PermissionTooltip
            allowed={canCreate || permissionsLoading}
            message="You don't have permission to create work orders."
          >
            <Button
              type="button"
              size="sm"
              asChild
              disabled={!canCreate}
              className="h-8 shrink-0 gap-1.5 px-2.5"
              data-testid="work-order-list-create-button"
            >
              <Link href={canCreate ? createHref : "#"}>
                <Plus className="size-3.5" aria-hidden />
                New
              </Link>
            </Button>
          </PermissionTooltip>
        </div>
      </div>

      <WorkOrdersFilterChips state={state} entries={entries} factoryLines={factoryLines} />
    </div>
  );
}

function ScopePills({ value, onChange }: { value: WorkOrderScope; onChange: (scope: WorkOrderScope) => void }) {
  return (
    <div className="flex items-center rounded-md border border-border p-0.5" role="group" aria-label="Scope">
      {WORK_ORDER_SCOPES.map((entry) => {
        const active = entry.id === value;
        return (
          <Tooltip key={entry.id}>
            <TooltipTrigger asChild>
              <button
                type="button"
                aria-pressed={active}
                onClick={() => onChange(entry.id)}
                data-testid={`work-orders-scope-${entry.id}`}
                className={cn(
                  "inline-flex h-7 items-center rounded-[5px] px-2.5 text-[12px] font-medium transition-colors",
                  active ? "bg-accent text-foreground" : "text-muted-foreground hover:text-foreground",
                )}
              >
                {entry.label}
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom">{entry.tooltip}</TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}

function FilterMenu({
  state,
  entries,
  factoryLines,
}: {
  state: WorkOrderListState;
  entries: WorkOrderListEntry[];
  factoryLines: FactoriesFactoryLine[];
}) {
  const assignees = collectAssigneeOptions(entries);
  const lines = factoryLines.filter((line) => Boolean(line.id));

  return (
    <DropdownMenu open={state.filterMenuOpen} onOpenChange={state.setFilterMenuOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={TRIGGER_CLASSNAME}
          data-testid="work-orders-filter-trigger"
        >
          <Funnel className="size-3.5" aria-hidden />
          Filter
          {state.filterCount > 0 ? (
            <span className="ml-0.5 rounded bg-accent px-1 text-[10px] text-foreground">{state.filterCount}</span>
          ) : (
            <kbd className="ml-0.5 hidden rounded border border-border px-1 font-sans text-[10px] text-muted-foreground sm:inline">
              F
            </kbd>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Add filter</DropdownMenuLabel>

        <FilterSubMenu
          label="Status"
          resetLabel="Any status"
          dimension="statuses"
          state={state}
          options={WORK_ORDER_DISPLAY_STATUSES.map((status) => {
            const meta = getWorkOrderDisplayStatusMeta(status);
            return {
              value: status,
              label: meta.filterLabel,
              dot: meta.dotClassName,
            };
          })}
        />

        <FilterSubMenu
          label="Line"
          resetLabel="Any line"
          dimension="lineIds"
          state={state}
          emptyLabel="No lines yet"
          options={lines.map((line) => ({ value: line.id as string, label: line.name?.trim() || "Untitled line" }))}
        />

        <FilterSubMenu
          label="Assignee"
          resetLabel="Anyone"
          dimension="assigneeIds"
          state={state}
          options={[{ value: UNASSIGNED_FILTER_VALUE, label: "Unassigned" }, ...assignees]}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface FilterOption {
  value: string;
  label: string;
  dot?: string;
}

function FilterSubMenu({
  label,
  resetLabel,
  dimension,
  state,
  options,
  emptyLabel,
}: {
  label: string;
  resetLabel: string;
  dimension: WorkOrderFilterDimension;
  state: WorkOrderListState;
  options: FilterOption[];
  emptyLabel?: string;
}) {
  const selected: readonly string[] = state.filters[dimension];
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger className={MENU_ITEM_CLASSNAME} data-testid={`work-orders-filter-${dimension}`}>
        <span className="flex-1">{label}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent className="w-48">
          <DropdownMenuItem
            className={MENU_ITEM_CLASSNAME}
            onSelect={(event) => {
              event.preventDefault();
              state.clearFilterDimension(dimension);
            }}
          >
            <span className="flex-1">{resetLabel}</span>
            {selected.length === 0 ? <Check className="size-3.5" aria-hidden /> : null}
          </DropdownMenuItem>
          {options.length === 0 ? (
            <div className="px-2 py-1.5 text-[12px] text-muted-foreground">{emptyLabel ?? "Nothing to filter"}</div>
          ) : null}
          {options.map((option) => (
            <DropdownMenuItem
              key={option.value}
              className={MENU_ITEM_CLASSNAME}
              onSelect={(event) => {
                event.preventDefault();
                state.toggleFilter(dimension, option.value);
              }}
            >
              {option.dot ? <span className={cn("size-1.5 rounded-full", option.dot)} aria-hidden /> : null}
              <span className="flex-1 truncate">{option.label}</span>
              {selected.includes(option.value) ? <Check className="size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}

function SearchField({
  inputRef,
  open,
  value,
  onOpen,
  onChange,
  onClose,
}: {
  inputRef: React.RefObject<HTMLInputElement | null>;
  open: boolean;
  value: string;
  onOpen: () => void;
  onChange: (value: string) => void;
  onClose: () => void;
}) {
  if (!open) {
    return (
      <Button
        type="button"
        variant="ghost"
        size="icon"
        onClick={onOpen}
        aria-label="Search work orders"
        className="size-8 shrink-0 text-muted-foreground"
        data-testid="work-orders-search-trigger"
      >
        <Search className="size-3.5" aria-hidden />
      </Button>
    );
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      onClose();
    }
  };

  return (
    <div className="relative">
      <Search
        className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
        aria-hidden
      />
      <Input
        ref={inputRef}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Search…"
        aria-label="Search work orders"
        className="!h-8 w-[200px] pl-8 pr-8 text-[13px] shadow-none"
        data-testid="work-orders-search-input"
      />
      <button
        type="button"
        onClick={onClose}
        aria-label="Close search"
        className="absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
        data-testid="work-orders-search-close"
      >
        <X className="size-3.5" aria-hidden />
      </button>
    </div>
  );
}

function DisplayMenu({ state }: { state: WorkOrderListState }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={TRIGGER_CLASSNAME}
          data-testid="work-orders-display-trigger"
        >
          <Settings2 className="size-3.5" aria-hidden />
          Display
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Layout</DropdownMenuLabel>
        {WORK_ORDER_LAYOUTS.map((entry) => {
          const Icon = LAYOUT_ICONS[entry.id];
          return (
            <DropdownMenuItem
              key={entry.id}
              className={MENU_ITEM_CLASSNAME}
              onSelect={() => state.setLayout(entry.id)}
              data-testid={`work-orders-layout-${entry.id}`}
            >
              <Icon className="size-3.5" aria-hidden />
              <span className="flex-1">{entry.label}</span>
              {state.layout === entry.id ? <Check className="size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          );
        })}

        <DropdownMenuSeparator />

        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Ordering</DropdownMenuLabel>
        {WORK_ORDER_ORDERINGS.map((entry) => (
          <DropdownMenuItem
            key={entry.id}
            className={MENU_ITEM_CLASSNAME}
            onSelect={() => state.setOrdering(entry.id as WorkOrderOrdering)}
            data-testid={`work-orders-ordering-${entry.id}`}
          >
            <span className="flex-1">{entry.label}</span>
            {state.ordering === entry.id ? <Check className="size-3.5" aria-hidden /> : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** Applied filters, rendered under the title bar only when something is set. */
function WorkOrdersFilterChips({
  state,
  entries,
  factoryLines,
}: {
  state: WorkOrderListState;
  entries: WorkOrderListEntry[];
  factoryLines: FactoriesFactoryLine[];
}) {
  if (state.filterCount === 0) {
    return null;
  }

  const lineNames = new Map(factoryLines.map((line) => [line.id ?? "", line.name?.trim() || "Untitled line"]));
  const assigneeNames = new Map(collectAssigneeOptions(entries).map((option) => [option.value, option.label]));

  const chips: Array<{ dimension: WorkOrderFilterDimension; value: string; label: string }> = [
    ...state.filters.statuses.map((status) => ({
      dimension: "statuses" as const,
      value: status,
      label: `Status is ${getWorkOrderDisplayStatusMeta(status).filterLabel}`,
    })),
    ...state.filters.lineIds.map((lineId) => ({
      dimension: "lineIds" as const,
      value: lineId,
      label: `Line is ${lineNames.get(lineId) ?? "Unknown line"}`,
    })),
    ...state.filters.assigneeIds.map((assigneeId) => ({
      dimension: "assigneeIds" as const,
      value: assigneeId,
      label:
        assigneeId === UNASSIGNED_FILTER_VALUE
          ? "Assignee is unassigned"
          : `Assignee is ${assigneeNames.get(assigneeId) ?? "Unknown"}`,
    })),
  ];

  return (
    <div className="mt-3 flex flex-wrap items-center gap-1.5" data-testid="work-orders-filter-chips">
      {chips.map((chip) => (
        <span
          key={`${chip.dimension}:${chip.value}`}
          className="inline-flex h-7 items-center gap-1 rounded-md border border-border bg-background px-2 text-[12px] text-foreground"
        >
          {chip.label}
          <button
            type="button"
            onClick={() => state.removeFilter(chip.dimension, chip.value)}
            aria-label={`Remove filter ${chip.label}`}
            className="rounded p-0.5 text-muted-foreground hover:text-foreground"
          >
            <X className="size-3" aria-hidden />
          </button>
        </span>
      ))}
      <button
        type="button"
        onClick={state.clearFilters}
        className="px-1.5 text-[12px] text-muted-foreground hover:text-foreground"
        data-testid="work-orders-filter-clear"
      >
        Clear
      </button>
    </div>
  );
}

/** People who appear on at least one work order, sorted by name. */
function collectAssigneeOptions(entries: WorkOrderListEntry[]): FilterOption[] {
  const byId = new Map<string, string>();
  for (const entry of entries) {
    for (const assignee of entry.order.assignees ?? []) {
      if (assignee.id) {
        byId.set(assignee.id, assignee.name?.trim() || "Unknown");
      }
    }
  }
  return [...byId.entries()].map(([value, label]) => ({ value, label })).sort((a, b) => a.label.localeCompare(b.label));
}

function isEditableTarget(target: HTMLElement): boolean {
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable;
}
