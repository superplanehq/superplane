import { useEffect, useRef, useState } from "react";
import type { SetURLSearchParams } from "react-router-dom";

/**
 * Keeps runs/console view mode and selected run in sync with `view` and `run` search params.
 *
 * Public URL value is `view=console` for the Console tab. Internal state flags
 * (`isDashboardMode`, etc.) still use the legacy "dashboard" name.
 */
export function useWorkflowViewSearchParams(
  searchParams: URLSearchParams,
  setSearchParams: SetURLSearchParams,
  dashboardsFeatureEnabled: boolean,
) {
  const [isRunsMode, setIsRunsMode] = useState(() => searchParams.get("view") === "runs");
  const [isDashboardMode, setIsDashboardMode] = useState(() => searchParams.get("view") === "console");
  const [isMemoryMode, setIsMemoryMode] = useState(() => searchParams.get("view") === "memory");
  const [isDashboardAddPanelOpen, setIsDashboardAddPanelOpen] = useState(false);
  const [isDashboardYamlOpen, setIsDashboardYamlOpen] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(() => searchParams.get("run"));

  const viewParam = searchParams.get("view") ?? "";
  const runParam = searchParams.get("run") ?? "";

  const setSearchParamsRef = useRef(setSearchParams);
  setSearchParamsRef.current = setSearchParams;

  useEffect(() => {
    setIsRunsMode(viewParam === "runs");
    setIsMemoryMode(viewParam === "memory");
    if (viewParam === "console") {
      if (dashboardsFeatureEnabled) {
        setIsDashboardMode(true);
      } else {
        setIsDashboardMode(false);
        setIsDashboardAddPanelOpen(false);
        setIsDashboardYamlOpen(false);
        setSearchParamsRef.current(
          (current) => {
            const next = new URLSearchParams(current);
            if (next.get("view") !== "console") {
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
    if (viewParam !== "console") {
      setIsDashboardAddPanelOpen(false);
      setIsDashboardYamlOpen(false);
    }
  }, [viewParam, runParam, dashboardsFeatureEnabled]);

  return {
    isRunsMode,
    setIsRunsMode,
    isDashboardMode,
    setIsDashboardMode,
    isMemoryMode,
    setIsMemoryMode,
    isDashboardAddPanelOpen,
    setIsDashboardAddPanelOpen,
    isDashboardYamlOpen,
    setIsDashboardYamlOpen,
    selectedRunId,
    setSelectedRunId,
  };
}
