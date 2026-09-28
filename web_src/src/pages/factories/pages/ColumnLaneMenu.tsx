import { Check, ListFilter, MoreHorizontal, Pencil, Plus, RefreshCw, SlidersHorizontal, XIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { cn } from "@/lib/utils";
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
import { DropdownMenuValueSub } from "@/ui/dropdownMenu/DropdownMenuValueSub";

import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";
import { DEFAULT_LINE_STEP_PARALLELISM, setParallelismLabel } from "../lib/factoryLineFormShared";
import type { WorkOrderFilterOption } from "../lib/workOrderFilterOptions";
import {
  DEFAULT_BACKLOG_COLUMN_QUERY,
  isDefaultBacklogColumnQuery,
  type BacklogColumnAge,
  type BacklogColumnQuery,
  type BacklogColumnSort,
  type BacklogColumnSortDirection,
} from "../lib/workOrderListPagination";
import { BACKLOG_REFRESH_COPY } from "./backlogRefresh";
import { LINE_BOARD_COLUMN_COLORS, type LineBoardColumnColorId } from "./lineBoardColumnColors";

interface ColumnLaneMenuProps {
  title: string;
  testId: string;
  /** Opens the automation canvas. Ignored when onEdit is set. */
  editHref?: string | null;
  /** Opens the primary edit surface without route navigation. */
  onEdit?: () => void;
  /** Menu item copy. Defaults to Edit. */
  editLabel?: string;
  /** Opens the parallelism modal for canvas-backed phases. */
  onSetParallelism?: () => void;
  parallelism?: number;
  /** Opens the Add intake picker. Leads the menu when set. */
  onAddIntake?: () => void;
  /** Opens the Add automation picker. */
  onAddAutomation?: () => void;
  /** Refreshes backlog tasks from readable intake sources. Hidden when unset. */
  onRefreshBacklog?: () => void;
  refreshBacklogPending?: boolean;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
}

/**
 * Column header menu: Edit (optional) and a single row of colour circles.
 */
export function ColumnLaneMenu({
  title,
  testId,
  editHref,
  onEdit,
  editLabel = "Edit",
  onSetParallelism,
  parallelism = DEFAULT_LINE_STEP_PARALLELISM,
  onAddIntake,
  onAddAutomation,
  onRefreshBacklog,
  refreshBacklogPending = false,
  colorId,
  onColorChange,
}: ColumnLaneMenuProps) {
  const navigate = useNavigate();
  const canEdit = Boolean(onEdit || editHref);
  const hasActions = canEdit || Boolean(onSetParallelism || onAddIntake || onAddAutomation || onRefreshBacklog);

  const handleEdit = () => {
    if (onEdit) {
      onEdit();
      return;
    }
    if (editHref) {
      navigate(editHref);
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`${title} menu`}
          className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid={testId}
        >
          <MoreHorizontal className="size-3.5" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-32 w-max p-0" data-testid={`${testId}-content`}>
        {hasActions ? (
          <>
            <ColumnLaneMenuActions
              testId={testId}
              editLabel={editLabel}
              canEdit={canEdit}
              onEdit={handleEdit}
              onSetParallelism={onSetParallelism}
              parallelism={parallelism}
              onAddIntake={onAddIntake}
              onAddAutomation={onAddAutomation}
              onRefreshBacklog={onRefreshBacklog}
              refreshBacklogPending={refreshBacklogPending}
            />
            <DropdownMenuSeparator className="my-0" />
          </>
        ) : null}
        <ColumnLaneColorPicker title={title} testId={testId} colorId={colorId} onColorChange={onColorChange} />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

const SORT_OPTIONS: Array<{ value: BacklogColumnSort; label: string }> = [
  { value: "updated", label: "Updated" },
  { value: "confidence", label: "Confidence" },
  { value: "source", label: "Source" },
  { value: "created", label: "Created" },
];

const AGE_OPTIONS: Array<{ value: BacklogColumnAge; label: string }> = [
  { value: "any", label: "Any age" },
  { value: "last7", label: "Last 7 days" },
  { value: "last30", label: "Last 30 days" },
  { value: "last90", label: "Last 90 days" },
  { value: "older90", label: "Older than 90 days" },
];

const CONFIDENCE_OPTIONS = [
  { value: "any", label: "Any score" },
  { value: "missing", label: "No score" },
  { value: "1", label: "1 or higher" },
  { value: "2", label: "2 or higher" },
  { value: "3", label: "3 or higher" },
  { value: "4", label: "4 or higher" },
  { value: "5", label: "5" },
];

/**
 * Backlog column sort and filter. Choices apply to the whole column and clear on reload.
 */
export function BacklogColumnSortMenu({
  title,
  query,
  sourceOptions,
  onChange,
}: {
  title: string;
  query: BacklogColumnQuery;
  sourceOptions: WorkOrderFilterOption[];
  onChange: (query: BacklogColumnQuery) => void;
}) {
  const active = !isDefaultBacklogColumnQuery(query);
  const confidenceValue = query.confidenceMissing
    ? "missing"
    : query.minConfidence == null
      ? "any"
      : String(query.minConfidence);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Sort and filter ${title}`}
          className={cn(
            "relative flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
            active && "text-foreground",
          )}
          data-testid="lines-backlog-sort-filter"
        >
          <ListFilter className="size-3.5" aria-hidden />
          {active ? <span className="absolute top-0.5 right-0.5 size-1.5 rounded-full bg-foreground" /> : null}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56" data-testid="lines-backlog-sort-filter-content">
        <DropdownMenuLabel className="text-[11px] font-medium tracking-[0.04em] text-muted-foreground">
          Sort and filter
        </DropdownMenuLabel>
        <DropdownMenuValueSub
          label="Sort"
          value={query.sort}
          testId="lines-backlog-sort"
          options={SORT_OPTIONS.map((option) => ({
            ...option,
            testId: `lines-backlog-sort-${option.value}`,
          }))}
          onValueChange={(value) => onChange(withSort(query, value as BacklogColumnSort))}
        />
        <DropdownMenuValueSub
          label="Order"
          value={query.sortDirection}
          testId="lines-backlog-direction"
          options={directionOptions(query.sort)}
          onValueChange={(value) => onChange({ ...query, sortDirection: value as BacklogColumnSortDirection })}
        />
        <BacklogSourceSubmenu
          options={sourceOptions}
          selected={query.sources}
          onChange={(sources) => onChange({ ...query, sources })}
        />
        <DropdownMenuValueSub
          label="Confidence"
          value={confidenceValue}
          testId="lines-backlog-confidence"
          options={CONFIDENCE_OPTIONS.map((option) => ({
            ...option,
            testId: `lines-backlog-confidence-${option.value}`,
          }))}
          onValueChange={(value) => onChange(withConfidence(query, value))}
        />
        <DropdownMenuValueSub
          label="Age"
          value={query.age}
          testId="lines-backlog-age"
          options={AGE_OPTIONS.map((option) => ({
            ...option,
            testId: `lines-backlog-age-${option.value}`,
          }))}
          onValueChange={(value) => onChange({ ...query, age: value as BacklogColumnAge })}
        />
        {active ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              data-testid="lines-backlog-sort-filter-clear"
              onSelect={() => onChange(DEFAULT_BACKLOG_COLUMN_QUERY)}
            >
              Clear sort and filter
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function withSort(query: BacklogColumnQuery, sort: BacklogColumnSort): BacklogColumnQuery {
  return {
    ...query,
    sort,
    sortDirection: sort === "source" ? "asc" : "desc",
  };
}

function withConfidence(query: BacklogColumnQuery, value: string): BacklogColumnQuery {
  if (value === "missing") {
    return { ...query, confidenceMissing: true, minConfidence: undefined };
  }
  if (value === "any") {
    return { ...query, confidenceMissing: false, minConfidence: undefined };
  }
  return { ...query, confidenceMissing: false, minConfidence: Number(value) };
}

function directionOptions(
  sort: BacklogColumnSort,
): Array<{ value: BacklogColumnSortDirection; label: string; testId: string }> {
  if (sort === "confidence") {
    return [
      { value: "desc", label: "Highest first", testId: "lines-backlog-direction-desc" },
      { value: "asc", label: "Lowest first", testId: "lines-backlog-direction-asc" },
    ];
  }
  if (sort === "source") {
    return [
      { value: "asc", label: "A to Z", testId: "lines-backlog-direction-asc" },
      { value: "desc", label: "Z to A", testId: "lines-backlog-direction-desc" },
    ];
  }
  return [
    { value: "desc", label: "Newest first", testId: "lines-backlog-direction-desc" },
    { value: "asc", label: "Oldest first", testId: "lines-backlog-direction-asc" },
  ];
}

function BacklogSourceSubmenu({
  options,
  selected,
  onChange,
}: {
  options: WorkOrderFilterOption[];
  selected: string[];
  onChange: (sources: string[]) => void;
}) {
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger className="cursor-pointer text-[13px]" data-testid="lines-backlog-source">
        Source
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent className="w-52">
          <DropdownMenuItem
            data-testid="lines-backlog-source-any"
            onSelect={(event) => {
              event.preventDefault();
              onChange([]);
            }}
          >
            <span className="flex-1">Any source</span>
            {selected.length === 0 ? <Check className="size-3.5" aria-hidden /> : null}
          </DropdownMenuItem>
          {options.map((option) => {
            const checked = selected.includes(option.value);
            return (
              <DropdownMenuItem
                key={option.value}
                data-testid={`lines-backlog-source-${option.value}`}
                onSelect={(event) => {
                  event.preventDefault();
                  onChange(checked ? selected.filter((value) => value !== option.value) : [...selected, option.value]);
                }}
              >
                <span className="flex-1">{option.label}</span>
                {checked ? <Check className="size-3.5" aria-hidden /> : null}
              </DropdownMenuItem>
            );
          })}
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}

function ColumnLaneMenuActions({
  testId,
  editLabel,
  canEdit,
  onEdit,
  onSetParallelism,
  parallelism,
  onAddIntake,
  onAddAutomation,
  onRefreshBacklog,
  refreshBacklogPending,
}: {
  testId: string;
  editLabel: string;
  canEdit: boolean;
  onEdit: () => void;
  onSetParallelism?: () => void;
  parallelism: number;
  onAddIntake?: () => void;
  onAddAutomation?: () => void;
  onRefreshBacklog?: () => void;
  refreshBacklogPending: boolean;
}) {
  return (
    <div className="p-1">
      {onAddAutomation ? (
        <DropdownMenuItem onSelect={onAddAutomation} data-testid={`${testId}-add-automation`}>
          <Plus className="h-3.5 w-3.5" aria-hidden />
          {COLUMN_AUTOMATIONS_COPY.addLabel}
        </DropdownMenuItem>
      ) : null}
      {onAddIntake ? (
        <DropdownMenuItem onSelect={onAddIntake} data-testid={`${testId}-add-intake`}>
          <Plus className="h-3.5 w-3.5" aria-hidden />
          Add intake
        </DropdownMenuItem>
      ) : null}
      {canEdit ? (
        <DropdownMenuItem onSelect={onEdit} data-testid={`${testId}-edit`}>
          <Pencil className="h-3.5 w-3.5" aria-hidden />
          {editLabel}
        </DropdownMenuItem>
      ) : null}
      {onSetParallelism ? (
        <DropdownMenuItem onSelect={onSetParallelism} data-testid={`${testId}-parallelism`}>
          <SlidersHorizontal className="h-3.5 w-3.5" aria-hidden />
          {setParallelismLabel(parallelism)}
        </DropdownMenuItem>
      ) : null}
      {onRefreshBacklog ? (
        <DropdownMenuItem
          onSelect={onRefreshBacklog}
          disabled={refreshBacklogPending}
          data-testid={`${testId}-refresh-backlog`}
        >
          <RefreshCw className="h-3.5 w-3.5" aria-hidden />
          {BACKLOG_REFRESH_COPY.menu}
        </DropdownMenuItem>
      ) : null}
    </div>
  );
}

function ColumnLaneColorPicker({
  title,
  testId,
  colorId,
  onColorChange,
}: {
  title: string;
  testId: string;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
}) {
  return (
    <div className="px-2 pb-2 pt-2" data-testid={`${testId}-color-picker`}>
      <DropdownMenuLabel className="px-0 pb-1.5 pt-0 text-[12px] font-medium text-muted-foreground">
        Set color
      </DropdownMenuLabel>
      <div className="flex items-center gap-1" role="listbox" aria-label={`Color for ${title}`}>
        {LINE_BOARD_COLUMN_COLORS.map((color) => {
          const selected = colorId === color.id;
          return (
            <button
              key={color.id}
              type="button"
              role="option"
              aria-selected={selected}
              aria-label={color.label}
              title={color.label}
              data-testid={`${testId}-color-${color.id}`}
              onClick={() => onColorChange(color.id)}
              className={cn(
                "relative flex size-5 shrink-0 items-center justify-center rounded-full ring-1 ring-inset ring-black/10 transition-transform hover:z-10 hover:scale-125 hover:ring-2 hover:ring-foreground/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring dark:ring-white/15",
                color.className,
                selected && "ring-2 ring-foreground",
              )}
            >
              {selected ? <Check className="size-2.5 text-foreground" aria-hidden strokeWidth={3} /> : null}
            </button>
          );
        })}
      </div>
      {colorId ? (
        <button
          type="button"
          data-testid={`${testId}-color-remove`}
          onClick={() => onColorChange(null)}
          className="mt-2 flex w-full items-center justify-center gap-1 rounded-md border border-border px-1.5 py-1 text-[12px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <XIcon className="size-3" aria-hidden />
          Remove color
        </button>
      ) : null}
    </div>
  );
}
