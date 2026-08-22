import { useCallback, useEffect, useMemo, useState } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { buildNodeMap, buildRunPresentation, type RunStatusFilter } from "@/ui/Runs/runPresentation";
import { runMatchesStatusTriggerFilters } from "@/ui/Runs/runStatusTriggerFilter";
import { loadPersistedFilters, savePersistedFilters } from "./filterPersistence";
import type { TriggerOption } from "./RunFiltersPopover";

interface UseRunFiltersParams {
  runs: CanvasesCanvasRun[];
  workflowNodes: ComponentsNode[];
  componentIconMap: Record<string, string>;
  onStatusFiltersChange?: (filters: RunStatusFilter[]) => void;
  /** The run currently open in the inspector, which no filter may hide. */
  selectedRunId?: string | null;
}

export function useRunFilters({
  runs,
  workflowNodes,
  componentIconMap,
  onStatusFiltersChange,
  selectedRunId,
}: UseRunFiltersParams) {
  const [selectedTriggerIds, setSelectedTriggerIds] = useState<Set<string>>(() => loadPersistedFilters().triggerIds);
  const [selectedStatuses, setSelectedStatuses] = useState<Set<RunStatusFilter>>(() => loadPersistedFilters().statuses);
  const [searchQuery, setSearchQuery] = useState("");
  //
  // Replay runs are noise in the ordinary runs list — they are debugging
  // artifacts, not canvas traffic — so they stay hidden until asked for.
  // Deliberately not persisted: "hidden by default" has to hold on every load,
  // not just the first one.
  //
  const [showReplays, setShowReplays] = useState(false);

  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);

  const triggerOptions = useMemo<TriggerOption[]>(
    () =>
      workflowNodes
        .filter((node) => node.id && node.type === "TYPE_TRIGGER")
        .map((node) => ({
          id: node.id!,
          name: node.name || node.component || "Trigger",
          iconSrc: getHeaderIconSrc(node.component),
          iconSlug: node.component ? componentIconMap[node.component] : undefined,
        }))
        .sort((a, b) => a.name.localeCompare(b.name)),
    [workflowNodes, componentIconMap],
  );

  useEffect(() => {
    onStatusFiltersChange?.(Array.from(selectedStatuses));
    savePersistedFilters({ statuses: selectedStatuses, triggerIds: selectedTriggerIds });
  }, [selectedStatuses, selectedTriggerIds, onStatusFiltersChange]);

  useEffect(() => {
    if (triggerOptions.length === 0) return;
    const validTriggerIds = new Set(triggerOptions.map((option) => option.id));
    setSelectedTriggerIds((currentTriggerIds) => {
      const nextTriggerIds = new Set(Array.from(currentTriggerIds).filter((id) => validTriggerIds.has(id)));
      return nextTriggerIds.size === currentTriggerIds.size ? currentTriggerIds : nextTriggerIds;
    });
  }, [triggerOptions]);

  const decoratedRuns = useMemo(() => runs.map((run) => buildRunPresentation(run, nodeMap)), [runs, nodeMap]);

  const statusTriggerFilters = useMemo(
    () => ({
      statuses: selectedStatuses.size > 0 ? Array.from(selectedStatuses) : undefined,
      triggers: selectedTriggerIds.size > 0 ? Array.from(selectedTriggerIds) : undefined,
    }),
    [selectedStatuses, selectedTriggerIds],
  );

  const { filteredRuns, hiddenReplayCount } = useMemo(() => {
    const normalizedSearchQuery = searchQuery.trim().toLowerCase();
    const visibleRuns: typeof decoratedRuns = [];
    let hiddenReplays = 0;

    for (const decoratedRun of decoratedRuns) {
      const { run, haystack } = decoratedRun;
      // Shared matcher with console run datasources — keeps status/trigger
      // semantics identical across the sidebar and widget surfaces.
      if (!runMatchesStatusTriggerFilters(run, statusTriggerFilters)) continue;
      if (normalizedSearchQuery && !haystack.includes(normalizedSearchQuery)) continue;
      // The run being inspected always stays listed, even when it is a replay
      // and replays are hidden — otherwise opening one leaves the list showing
      // nothing that matches what the inspector is displaying.
      if (run.isReplay && !showReplays && run.id !== selectedRunId) {
        hiddenReplays += 1;
        continue;
      }
      visibleRuns.push(decoratedRun);
    }

    return { filteredRuns: visibleRuns, hiddenReplayCount: hiddenReplays };
  }, [decoratedRuns, searchQuery, statusTriggerFilters, showReplays, selectedRunId]);

  const orderedRuns = useMemo(
    () => ({
      active: filteredRuns.filter((run) => run.status === "running"),
      rest: filteredRuns.filter((run) => run.status !== "running"),
    }),
    [filteredRuns],
  );

  const hasSearchFilter = searchQuery.trim().length > 0;
  const hasAnyFilter = selectedTriggerIds.size > 0 || selectedStatuses.size > 0 || hasSearchFilter || showReplays;

  const clearFilters = useCallback(() => {
    setSelectedStatuses(new Set());
    setSelectedTriggerIds(new Set());
    setSearchQuery("");
    setShowReplays(false);
  }, []);

  const toggleShowReplays = useCallback(() => {
    setShowReplays((current) => !current);
  }, []);

  const toggleStatus = useCallback((status: RunStatusFilter) => {
    setSelectedStatuses((currentStatuses) => {
      const nextStatuses = new Set(currentStatuses);
      if (nextStatuses.has(status)) nextStatuses.delete(status);
      else nextStatuses.add(status);
      return nextStatuses;
    });
  }, []);

  const toggleTrigger = useCallback((triggerId: string) => {
    setSelectedTriggerIds((currentTriggerIds) => {
      const nextTriggerIds = new Set(currentTriggerIds);
      if (nextTriggerIds.has(triggerId)) nextTriggerIds.delete(triggerId);
      else nextTriggerIds.add(triggerId);
      return nextTriggerIds;
    });
  }, []);

  return {
    selectedStatuses,
    selectedTriggerIds,
    searchQuery,
    showReplays,
    toggleShowReplays,
    triggerOptions,
    filteredRuns,
    hiddenReplayCount,
    orderedRuns,
    hasSearchFilter,
    hasAnyFilter,
    setSearchQuery,
    clearFilters,
    toggleStatus,
    toggleTrigger,
    clearStatuses: () => setSelectedStatuses(new Set()),
    clearTriggers: () => setSelectedTriggerIds(new Set()),
  };
}

export type RunFiltersState = ReturnType<typeof useRunFilters>;
