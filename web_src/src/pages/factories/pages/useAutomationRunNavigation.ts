import { useSearchParams } from "react-router";

export type AutomationConfigTab = "automation" | "runs";

/** Keep automation tabs and run details in browser history, including board popups. */
export function useAutomationRunNavigation(showRunsTab = true, showAutomationTab = true) {
  const [params, setParams] = useSearchParams();
  const selectedRunId = showRunsTab ? params.get("automationRun") : null;
  const requestedTab = params.get("automationTab");
  const tab = automationConfigTab(showRunsTab, showAutomationTab, requestedTab, selectedRunId);

  const setTab = (next: AutomationConfigTab) => {
    if (next === tab || (!showAutomationTab && next === "automation")) return;
    setParams((current) => {
      const nextParams = new URLSearchParams(current);
      if (next === "runs" || selectedRunId) nextParams.set("automationTab", next);
      else nextParams.delete("automationTab");
      return nextParams;
    });
  };
  const setSelectedRunId = (id: string | null, options?: { replace?: boolean }) => {
    if (id === selectedRunId) return;
    setParams((current) => {
      const nextParams = new URLSearchParams(current);
      nextParams.set("automationTab", "runs");
      if (id) nextParams.set("automationRun", id);
      else nextParams.delete("automationRun");
      return nextParams;
    }, options);
  };
  return { tab, setTab, selectedRunId, setSelectedRunId };
}

function automationConfigTab(
  showRunsTab: boolean,
  showAutomationTab: boolean,
  requestedTab: string | null,
  selectedRunId: string | null,
): AutomationConfigTab {
  if (showRunsTab && !showAutomationTab) return "runs";
  if (showRunsTab && (requestedTab === "runs" || (!requestedTab && selectedRunId))) return "runs";
  return "automation";
}
