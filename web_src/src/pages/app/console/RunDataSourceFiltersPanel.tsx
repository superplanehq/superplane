import { useMemo, useState } from "react";
import { ChevronDown, ListFilter } from "lucide-react";

import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunStatusFilterSection, RunTriggerFilterSection, type TriggerOption } from "@/ui/Runs/RunFilterSections";
import { type RunStatusFilter } from "@/ui/Runs/runPresentation";
import type { SuperplaneComponentsNode } from "@/api-client";

import { resolveConsoleTrigger, useConsoleContext } from "./ConsoleContext";

interface RunDataSourceFiltersPanelProps {
  statuses: readonly RunStatusFilter[] | undefined;
  triggers: readonly string[] | undefined;
  onStatusesChange: (next: RunStatusFilter[] | undefined) => void;
  onTriggersChange: (next: string[] | undefined) => void;
  /** Extra id string added to the collapsible testid, useful when multiple panels render on the same form. */
  testIdSuffix?: string;
}

/**
 * Optional per-datasource status + trigger filter editor shared by every
 * console surface that resolves runs (widget datasources and markdown / html
 * `kind: "run"` variables). Wrapped in a collapsible so the filter chrome
 * stays out of the way when authors haven't configured any filters yet, and
 * opens automatically when either dimension carries a selection so authors
 * can see the active filter at a glance.
 *
 * Empty selections in either dimension mean "all" — the panel forwards
 * `undefined` back to the caller when both would end up empty so persisted
 * YAML stays clean (matching the sidebar's Clear semantics).
 */
export function RunDataSourceFiltersPanel({
  statuses,
  triggers,
  onStatusesChange,
  onTriggersChange,
  testIdSuffix,
}: RunDataSourceFiltersPanelProps) {
  const ctx = useConsoleContext();
  const triggerOptions = useMemo(() => buildTriggerOptions(ctx?.nodes ?? []), [ctx?.nodes]);

  const selectedStatuses = useMemo(() => new Set<RunStatusFilter>(statuses ?? []), [statuses]);
  const selectedTriggerIds = useMemo(() => new Set<string>(resolveSelectedTriggerIds(triggers, ctx)), [triggers, ctx]);

  // Count persisted trigger refs (not just resolvable ones) so a badge /
  // Clear affordance still reflects stale YAML that no longer matches a
  // canvas node — otherwise authors can be stuck with an active filter
  // they cannot clear from the UI.
  const activeStatusCount = selectedStatuses.size;
  const activeTriggerCount = triggers?.length ?? 0;
  const activeFilterCount = activeStatusCount + activeTriggerCount;
  const hasActiveFilters = activeFilterCount > 0;
  const hasPersistedTriggerFilter = activeTriggerCount > 0;

  const [open, setOpen] = useState<boolean>(hasActiveFilters);

  const toggleStatus = (status: RunStatusFilter) => {
    const next = new Set(selectedStatuses);
    if (next.has(status)) next.delete(status);
    else next.add(status);
    onStatusesChange(next.size > 0 ? Array.from(next) : undefined);
  };

  const clearStatuses = () => onStatusesChange(undefined);

  const toggleTrigger = (triggerId: string) => {
    const next = new Set(selectedTriggerIds);
    if (next.has(triggerId)) next.delete(triggerId);
    else next.add(triggerId);
    onTriggersChange(next.size > 0 ? Array.from(next) : undefined);
  };

  const clearTriggers = () => onTriggersChange(undefined);

  const testIdBase = testIdSuffix ? `run-datasource-filters-${testIdSuffix}` : "run-datasource-filters";

  return (
    <Collapsible open={open} onOpenChange={setOpen} data-testid={testIdBase}>
      <CollapsibleTrigger
        className="group flex w-full items-center gap-2 rounded-md border border-slate-200 bg-white px-2 py-1.5 text-left text-xs text-slate-700 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-300 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300 dark:hover:bg-gray-800 dark:focus-visible:ring-gray-600"
        data-testid={`${testIdBase}-toggle`}
        aria-label="Toggle run filters"
      >
        <ChevronDown
          aria-hidden="true"
          className="size-3.5 shrink-0 text-slate-500 transition-transform duration-150 group-data-[state=closed]:-rotate-90 dark:text-gray-400"
        />
        <ListFilter className="size-3.5 shrink-0 text-slate-500 dark:text-gray-400" aria-hidden="true" />
        <span className="flex-1 truncate">Filters</span>
        <span
          className={cn(
            "shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium",
            hasActiveFilters
              ? "bg-sky-100 text-sky-700 dark:bg-indigo-900/60 dark:text-indigo-200"
              : "bg-slate-100 text-slate-500 dark:bg-gray-800 dark:text-gray-400",
          )}
        >
          {hasActiveFilters ? activeFilterCount : "All"}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div
          className="mt-1 overflow-hidden rounded-md border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-900"
          data-testid={`${testIdBase}-content`}
        >
          <RunStatusFilterSection
            selectedStatuses={selectedStatuses}
            onToggleStatus={toggleStatus}
            onClearStatuses={clearStatuses}
          />
          <RunTriggerFilterSection
            triggerOptions={triggerOptions}
            selectedTriggerIds={selectedTriggerIds}
            onToggleTrigger={toggleTrigger}
            onClearTriggers={clearTriggers}
            hasFilter={hasPersistedTriggerFilter}
            headerClassName="border-t border-slate-950/10 dark:border-gray-800/70"
          />
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

/**
 * Build sorted trigger options for the console filter list from the shared
 * console context. Mirrors the sidebar's transform so the two surfaces
 * render the same set of trigger nodes in the same order.
 */
function buildTriggerOptions(nodes: SuperplaneComponentsNode[]): TriggerOption[] {
  return nodes
    .filter((node) => node.id && node.type === "TYPE_TRIGGER")
    .map((node) => ({
      id: node.id!,
      name: node.name || node.component || "Trigger",
      iconSrc: getHeaderIconSrc(node.component),
    }))
    .sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * Resolve every persisted trigger reference (id-or-name) to its concrete
 * node id so the shared checkbox list can drive selection state off of
 * ids while YAML keeps the friendly name authors typed. Entries that no
 * longer match a canvas node are dropped so stale references quietly
 * fall out of the UI.
 */
function resolveSelectedTriggerIds(
  triggers: readonly string[] | undefined,
  ctx: ReturnType<typeof useConsoleContext>,
): string[] {
  if (!triggers || triggers.length === 0) return [];
  const out: string[] = [];
  const seen = new Set<string>();
  for (const reference of triggers) {
    const resolved = resolveConsoleTrigger(ctx, reference)?.node.id;
    if (!resolved || seen.has(resolved)) continue;
    seen.add(resolved);
    out.push(resolved);
  }
  return out;
}
