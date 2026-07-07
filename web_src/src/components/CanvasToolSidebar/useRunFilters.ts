import { useCallback, useEffect, useMemo, useState } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { buildNodeMap, buildRunPresentation, type RunStatusFilter } from "@/ui/Runs/runPresentation";
import { loadPersistedFilters, savePersistedFilters } from "./filterPersistence";
import type { TriggerOption } from "./RunFiltersPopover";

interface UseRunFiltersParams {
  runs: CanvasesCanvasRun[];
  workflowNodes: ComponentsNode[];
  componentIconMap: Record<string, string>;
  onStatusFiltersChange?: (filters: RunStatusFilter[]) => void;
}

export function useRunFilters({ runs, workflowNodes, componentIconMap, onStatusFiltersChange }: UseRunFiltersParams) {
  const [selectedTriggerIds, setSelectedTriggerIds] = useState<Set<string>>(() => loadPersistedFilters().triggerIds);
  const [selectedStatuses, setSelectedStatuses] = useState<Set<RunStatusFilter>>(() => loadPersistedFilters().statuses);
  const [searchQuery, setSearchQuery] = useState("");

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

  const filteredRuns = useMemo(() => {
    const normalizedSearchQuery = searchQuery.trim().toLowerCase();

    return decoratedRuns.filter(({ run, status, haystack }) => {
      if (selectedStatuses.size > 0 && (status === "unknown" || !selectedStatuses.has(status))) return false;

      if (selectedTriggerIds.size > 0) {
        const triggerNodeId = run.rootEvent?.nodeId;
        if (!triggerNodeId || !selectedTriggerIds.has(triggerNodeId)) return false;
      }

      if (normalizedSearchQuery && !haystack.includes(normalizedSearchQuery)) return false;

      return true;
    });
  }, [decoratedRuns, searchQuery, selectedStatuses, selectedTriggerIds]);

  const orderedRuns = useMemo(
    () => ({
      active: filteredRuns.filter((run) => run.status === "running"),
      rest: filteredRuns.filter((run) => run.status !== "running"),
    }),
    [filteredRuns],
  );

  const hasSearchFilter = searchQuery.trim().length > 0;
  const hasAnyFilter = selectedTriggerIds.size > 0 || selectedStatuses.size > 0 || hasSearchFilter;

  const clearFilters = useCallback(() => {
    setSelectedStatuses(new Set());
    setSelectedTriggerIds(new Set());
    setSearchQuery("");
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
    triggerOptions,
    filteredRuns,
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
