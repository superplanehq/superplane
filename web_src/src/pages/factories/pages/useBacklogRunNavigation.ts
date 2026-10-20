import { useState } from "react";
import { useSearchParams } from "react-router";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_RUN_HISTORY } from "@/lib/experimentalFeatures";

export type BacklogSettingsTab = "general" | "agent" | "automation" | "runs";
type HistoryScope = { organizationId?: string; factoryId?: string };

/** Preserve legacy tabs without the flag; share flagged tabs and runs through the URL. */
export function useBacklogRunNavigation({
  organizationId,
  factoryId,
  graph,
  agent,
  parameter,
  initialTab = "general",
  hasAgent,
  eligible = true,
}: {
  organizationId?: string;
  factoryId?: string;
  graph?: HistoryScope;
  agent?: HistoryScope;
  parameter: "settings" | "planningSettings";
  initialTab?: BacklogSettingsTab;
  hasAgent: boolean;
  eligible?: boolean;
}) {
  const scope = historyScope(organizationId, factoryId, graph, agent);
  const features = useExperimentalFeature(scope.organizationId);
  const enabled = eligible && features.has(FEATURE_FACTORY_RUN_HISTORY);
  const [params, setParams] = useSearchParams();
  const [legacyTab, setLegacyTab] = useState(initialTab);
  const selectedRunId = params.get("automationRun");
  const requested = enabled ? (params.get(parameter) ?? (selectedRunId ? "runs" : initialTab)) : legacyTab;
  let tab: BacklogSettingsTab = "general";
  if (requested === "agent" && hasAgent) tab = "agent";
  if (requested === "automation" || requested === "runs") tab = enabled ? "runs" : "automation";

  const setTab = (next: BacklogSettingsTab) => {
    if (!enabled) {
      setLegacyTab(next);
      return;
    }
    if (next === tab) return;
    setParams((current) => {
      const updated = new URLSearchParams(current);
      updated.set(parameter, next);
      return updated;
    });
  };
  const setSelectedRunId = (id: string | null) => {
    if (id === selectedRunId) return;
    setParams((current) => {
      const updated = new URLSearchParams(current);
      updated.set(parameter, "runs");
      if (id) updated.set("automationRun", id);
      else updated.delete("automationRun");
      return updated;
    });
  };
  return { ...scope, enabled, tab, setTab, selectedRunId, setSelectedRunId };
}

function historyScope(
  organizationId?: string,
  factoryId?: string,
  graph?: HistoryScope,
  agent?: HistoryScope,
): HistoryScope {
  return {
    organizationId: organizationId ?? graph?.organizationId ?? agent?.organizationId,
    factoryId: factoryId ?? graph?.factoryId ?? agent?.factoryId,
  };
}
