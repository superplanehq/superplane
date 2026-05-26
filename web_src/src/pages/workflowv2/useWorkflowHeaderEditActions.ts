import { useCallback, useEffect, useRef } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface WorkflowHeaderEditActionsConfig {
  isRunsMode: boolean;
  handleToggleEditMode: () => Promise<void>;
  setIsRunsMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setRunDetailNodeId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useWorkflowHeaderEditActions({
  isRunsMode,
  handleToggleEditMode,
  setIsRunsMode,
  setSelectedRunId,
  setRunDetailNodeId,
  setSearchParams,
}: WorkflowHeaderEditActionsConfig) {
  const handleEnterEditModeFromHeader = useCallback(async () => {
    if (isRunsMode) {
      setIsRunsMode(false);
      setSelectedRunId(null);
      setRunDetailNodeId(null);
      setSearchParams(clearRunsViewSearchParams, { replace: true });
    }

    await handleToggleEditMode();
  }, [handleToggleEditMode, isRunsMode, setIsRunsMode, setRunDetailNodeId, setSearchParams, setSelectedRunId]);

  const handleExitEditModeFromHeader = useCallback(async () => {
    await handleToggleEditMode();
  }, [handleToggleEditMode]);

  return { handleEnterEditModeFromHeader, handleExitEditModeFromHeader };
}

/**
 * Auto-enters edit mode when `?edit=1` is in the URL.
 * Removes the param after triggering to avoid re-entering on refresh.
 */
export function useAutoEnterEditMode(
  hasEditableVersion: boolean,
  canUpdateCanvas: boolean,
  versionsLoaded: boolean,
  handleToggleEditMode: () => Promise<void>,
  searchParams: URLSearchParams,
  setSearchParams: SetURLSearchParams,
) {
  const triggeredRef = useRef(false);

  useEffect(() => {
    if (triggeredRef.current) return;
    if (searchParams.get("edit") !== "1") return;
    if (!versionsLoaded) return;
    if (hasEditableVersion) return;
    if (!canUpdateCanvas) return;

    triggeredRef.current = true;

    void handleToggleEditMode().then(() => {
      const next = new URLSearchParams(searchParams);
      next.delete("edit");
      setSearchParams(next, { replace: true });
    });
  }, [searchParams, setSearchParams, hasEditableVersion, canUpdateCanvas, versionsLoaded, handleToggleEditMode]);
}

function clearRunsViewSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  next.delete("run");
  return next;
}

/**
 * After a blank canvas is created, automatically adds a placeholder "New Component" node.
 * Waits until edit mode is active and canvas is loaded.
 */
export function useAutoPlaceholderNode(
  hasEditableVersion: boolean,
  canvasHasSpec: boolean,
  canvasId: string | undefined,
  handlePlaceholderAdd?: (data: { position: { x: number; y: number } }) => Promise<string>,
) {
  const addedRef = useRef(false);

  useEffect(() => {
    if (addedRef.current) return;
    if (typeof window === "undefined" || !canvasId) return;
    if (sessionStorage.getItem("add-placeholder-node") !== canvasId) return;
    if (!hasEditableVersion || !canvasHasSpec || !handlePlaceholderAdd) return;

    addedRef.current = true;

    void handlePlaceholderAdd({ position: { x: 400, y: 300 } }).then(() => {
      sessionStorage.removeItem("add-placeholder-node");
    });
  }, [hasEditableVersion, canvasHasSpec, canvasId, handlePlaceholderAdd]);
}
