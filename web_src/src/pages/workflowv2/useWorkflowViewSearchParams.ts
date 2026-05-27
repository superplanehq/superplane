import { useEffect, useState } from "react";

/**
 * Keeps runs/dashboard/memory/files view mode and selected run in sync with `view` and `run` search params.
 */
export function useWorkflowViewSearchParams(searchParams: URLSearchParams) {
  const [isRunsMode, setIsRunsMode] = useState(() => searchParams.get("view") === "runs");
  const [isDashboardMode, setIsDashboardMode] = useState(() => searchParams.get("view") === "dashboard");
  const [isMemoryMode, setIsMemoryMode] = useState(() => searchParams.get("view") === "memory");
  const [isFilesMode, setIsFilesMode] = useState(() => searchParams.get("view") === "files");
  const [isDashboardAddPanelOpen, setIsDashboardAddPanelOpen] = useState(false);
  const [isDashboardYamlOpen, setIsDashboardYamlOpen] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(() => searchParams.get("run"));

  const viewParam = searchParams.get("view") ?? "";
  const runParam = searchParams.get("run") ?? "";

  useEffect(() => {
    setIsRunsMode(viewParam === "runs");
    setIsMemoryMode(viewParam === "memory");
    setIsFilesMode(viewParam === "files");
    if (viewParam === "dashboard") {
      setIsDashboardMode(true);
    } else {
      setIsDashboardMode(false);
    }
    setSelectedRunId(runParam || null);
    if (viewParam !== "dashboard") {
      setIsDashboardAddPanelOpen(false);
      setIsDashboardYamlOpen(false);
    }
  }, [viewParam, runParam]);

  return {
    isRunsMode,
    setIsRunsMode,
    isDashboardMode,
    setIsDashboardMode,
    isMemoryMode,
    setIsMemoryMode,
    isFilesMode,
    setIsFilesMode,
    isDashboardAddPanelOpen,
    setIsDashboardAddPanelOpen,
    isDashboardYamlOpen,
    setIsDashboardYamlOpen,
    selectedRunId,
    setSelectedRunId,
  };
}
