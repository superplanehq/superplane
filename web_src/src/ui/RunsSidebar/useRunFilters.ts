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
  const [search, setSearch] = useState("");
  const [selectedTriggerIds, setSelectedTriggerIds] = useState<Set<string>>(() => loadPersistedFilters().triggerIds);
  const [selectedStatuses, setSelectedStatuses] = useState<Set<RunStatusFilter>>(() => loadPersistedFilters().statuses);

  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);

  const triggerOptions = useMemo<TriggerOption[]>(() => {
    return workflowNodes
      .filter((node) => node.id && node.type === "TYPE_TRIGGER")
      .map((node) => ({
        id: node.id!,
        name: node.name || node.component || "Trigger",
        iconSrc: getHeaderIconSrc(node.component),
        iconSlug: node.component ? componentIconMap[node.component] : undefined,
      }))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [workflowNodes, componentIconMap]);

  useEffect(() => {
    onStatusFiltersChange?.(Array.from(selectedStatuses));
    savePersistedFilters({ statuses: selectedStatuses, triggerIds: selectedTriggerIds });
  }, [selectedStatuses, selectedTriggerIds, onStatusFiltersChange]);

  useEffect(() => {
    if (triggerOptions.length === 0) return;
    const valid = new Set(triggerOptions.map((option) => option.id));
    setSelectedTriggerIds((prev) => {
      const next = new Set(Array.from(prev).filter((id) => valid.has(id)));
      return next.size === prev.size ? prev : next;
    });
  }, [triggerOptions]);

  const decoratedRuns = useMemo(() => runs.map((run) => buildRunPresentation(run, nodeMap)), [runs, nodeMap]);

  const filteredRuns = useMemo(() => {
    const query = search.trim().toLowerCase();
    return decoratedRuns.filter(({ run, status, haystack }) => {
      if (query && !haystack.includes(query)) return false;
      if (selectedStatuses.size > 0) {
        if (status === "unknown" || !selectedStatuses.has(status)) return false;
      }
      if (selectedTriggerIds.size > 0) {
        const triggerNodeId = run.rootEvent?.nodeId;
        if (!triggerNodeId || !selectedTriggerIds.has(triggerNodeId)) return false;
      }
      return true;
    });
  }, [decoratedRuns, search, selectedStatuses, selectedTriggerIds]);

  const orderedRuns = useMemo(() => {
    const active = filteredRuns.filter((run) => run.status === "running");
    const rest = filteredRuns.filter((run) => run.status !== "running");
    return { active, rest };
  }, [filteredRuns]);

  const hasSearch = search.trim().length > 0;
  const hasTriggerFilter = selectedTriggerIds.size > 0;
  const hasStatusFilter = selectedStatuses.size > 0;
  const hasAnyFilter = hasSearch || hasTriggerFilter || hasStatusFilter;

  const clearFilters = useCallback(() => {
    setSearch("");
    setSelectedStatuses(new Set());
    setSelectedTriggerIds(new Set());
  }, []);

  const toggleStatus = useCallback((status: RunStatusFilter) => {
    setSelectedStatuses((prev) => {
      const next = new Set(prev);
      if (next.has(status)) next.delete(status);
      else next.add(status);
      return next;
    });
  }, []);

  const toggleTrigger = useCallback((triggerId: string) => {
    setSelectedTriggerIds((prev) => {
      const next = new Set(prev);
      if (next.has(triggerId)) next.delete(triggerId);
      else next.add(triggerId);
      return next;
    });
  }, []);

  const clearStatuses = useCallback(() => setSelectedStatuses(new Set()), []);
  const clearTriggers = useCallback(() => setSelectedTriggerIds(new Set()), []);

  return {
    search,
    setSearch,
    selectedStatuses,
    selectedTriggerIds,
    triggerOptions,
    filteredRuns,
    orderedRuns,
    hasAnyFilter,
    clearFilters,
    toggleStatus,
    toggleTrigger,
    clearStatuses,
    clearTriggers,
  };
}
