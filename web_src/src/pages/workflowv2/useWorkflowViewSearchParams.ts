import { useEffect, useRef, useState } from "react";
import type { SetURLSearchParams } from "react-router-dom";

/**
 * Keeps runs/dashboard view mode and selected run in sync with `view` and `run` search params.
 */
export function useWorkflowViewSearchParams(
  searchParams: URLSearchParams,
  setSearchParams: SetURLSearchParams,
  dashboardsFeatureEnabled: boolean,
) {
  const [isRunsMode, setIsRunsMode] = useState(() => searchParams.get("view") === "runs");
  const [isDashboardMode, setIsDashboardMode] = useState(() => searchParams.get("view") === "dashboard");
  const [isDashboardAddPanelOpen, setIsDashboardAddPanelOpen] = useState(false);
  const [isDashboardYamlOpen, setIsDashboardYamlOpen] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(() => searchParams.get("run"));

  const viewParam = searchParams.get("view") ?? "";
  const runParam = searchParams.get("run") ?? "";

  const setSearchParamsRef = useRef(setSearchParams);
  setSearchParamsRef.current = setSearchParams;

  useEffect(() => {
    setIsRunsMode(viewParam === "runs");
    if (viewParam === "dashboard") {
      if (dashboardsFeatureEnabled) {
        setIsDashboardMode(true);
      } else {
        setIsDashboardMode(false);
        setIsDashboardAddPanelOpen(false);
        setIsDashboardYamlOpen(false);
        setSearchParamsRef.current(
          (current) => {
            const next = new URLSearchParams(current);
            if (next.get("view") !== "dashboard") {
              return current;
            }
            next.delete("view");
            return next;
          },
          { replace: true },
        );
      }
    } else {
      setIsDashboardMode(false);
    }
    setSelectedRunId(runParam || null);
    if (viewParam !== "dashboard") {
      setIsDashboardAddPanelOpen(false);
      setIsDashboardYamlOpen(false);
    }
  }, [viewParam, runParam, dashboardsFeatureEnabled]);

  return {
    isRunsMode,
    setIsRunsMode,
    isDashboardMode,
    setIsDashboardMode,
    isDashboardAddPanelOpen,
    setIsDashboardAddPanelOpen,
    isDashboardYamlOpen,
    setIsDashboardYamlOpen,
    selectedRunId,
    setSelectedRunId,
  };
}
