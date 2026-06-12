import { useCallback, useEffect, useRef } from "react";
import type { SetURLSearchParams } from "react-router-dom";
import type { CanvasesCanvas } from "@/api-client";
import {
  abandonPendingPlaceholderBoot,
  markAgentBootReady,
  PLACEHOLDER_NODE_CONTEXT_KEY,
} from "@/lib/agentBootContext";

type PlaceholderAddHandler = (data: { position: { x: number; y: number } }) => Promise<string>;

interface WorkflowStartupActionsConfig {
  hasEditableVersion: boolean;
  canUpdateCanvas: boolean;
  canvas: CanvasesCanvas | null | undefined;
  handlePlaceholderAdd?: PlaceholderAddHandler;
  searchParams: URLSearchParams;
}

interface WorkflowHeaderEditActionsConfig {
  isRunsMode: boolean;
  isVersionsMode: boolean;
  handleExitRunsMode: () => void;
  handleExitVersionsMode: () => void;
  handleToggleEditMode: () => Promise<void>;
  setIsRunsMode: (value: boolean) => void;
  setIsVersionsMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setRunDetailNodeId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
  startup?: WorkflowStartupActionsConfig;
}

export function useWorkflowHeaderEditActions({
  isRunsMode,
  isVersionsMode,
  handleExitRunsMode,
  handleExitVersionsMode,
  handleToggleEditMode,
  setIsRunsMode,
  setIsVersionsMode,
  setSelectedRunId,
  setRunDetailNodeId,
  setSearchParams,
  startup,
}: WorkflowHeaderEditActionsConfig) {
  const handleEnterEditModeFromHeader = useCallback(async () => {
    if (isRunsMode) {
      setIsRunsMode(false);
      setSelectedRunId(null);
      setRunDetailNodeId(null);
      setSearchParams(clearRunsViewSearchParams, { replace: true });
    } else if (isVersionsMode) {
      setIsVersionsMode(false);
      setSearchParams(clearVersionsViewSearchParams, { replace: true });
    }

    await handleToggleEditMode();
  }, [
    handleToggleEditMode,
    isRunsMode,
    isVersionsMode,
    setIsRunsMode,
    setIsVersionsMode,
    setRunDetailNodeId,
    setSearchParams,
    setSelectedRunId,
  ]);

  const handleExitEditModeFromHeader = useCallback(async () => {
    if (isRunsMode) {
      handleExitRunsMode();
      return;
    }
    if (isVersionsMode) {
      handleExitVersionsMode();
      return;
    }
    await handleToggleEditMode();
  }, [handleExitRunsMode, handleExitVersionsMode, handleToggleEditMode, isRunsMode, isVersionsMode]);

  useAutoEditMode(startup, handleToggleEditMode, setSearchParams);
  useAutoPlaceholderNode(startup);

  return { handleEnterEditModeFromHeader, handleExitEditModeFromHeader };
}

function useAutoEditMode(
  startup: WorkflowStartupActionsConfig | undefined,
  handleToggleEditMode: () => Promise<void>,
  setSearchParams: SetURLSearchParams,
) {
  const triggeredRef = useRef(false);
  const hasEditableVersion = startup?.hasEditableVersion ?? false;
  const canUpdateCanvas = startup?.canUpdateCanvas ?? false;
  const canvasLoaded = Boolean(startup?.canvas);
  const searchParams = startup?.searchParams;

  useEffect(() => {
    if (triggeredRef.current) return;
    if (!searchParams || searchParams.get("edit") !== "1") return;
    if (!canvasLoaded) return;
    if (hasEditableVersion) return;
    if (!canUpdateCanvas) return;

    triggeredRef.current = true;

    void handleToggleEditMode().then(() => {
      setSearchParams(
        (current) => {
          const next = new URLSearchParams(current);
          next.delete("edit");
          return next;
        },
        { replace: true },
      );
    });
  }, [searchParams, setSearchParams, hasEditableVersion, canUpdateCanvas, canvasLoaded, handleToggleEditMode]);
}

function clearVersionsViewSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  return next;
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
function useAutoPlaceholderNode(startup: WorkflowStartupActionsConfig | undefined) {
  const addedRef = useRef(false);
  const hasEditableVersion = startup?.hasEditableVersion ?? false;
  const canvasHasSpec = Boolean(startup?.canvas?.spec);
  const canvasId = startup?.canvas?.metadata?.id;
  const handlePlaceholderAdd = startup?.handlePlaceholderAdd;

  useEffect(() => {
    if (addedRef.current) return;
    if (typeof window === "undefined" || !canvasId) return;
    if (sessionStorage.getItem(PLACEHOLDER_NODE_CONTEXT_KEY) !== canvasId) return;
    if (!hasEditableVersion || !canvasHasSpec || !handlePlaceholderAdd) return;

    addedRef.current = true;

    void handlePlaceholderAdd({ position: { x: 400, y: 300 } })
      .then((placeholderId) => {
        if (!placeholderId) {
          abandonPendingPlaceholderBoot(canvasId);
          return;
        }

        markAgentBootReady(canvasId);
      })
      .catch(() => {
        abandonPendingPlaceholderBoot(canvasId);
      });
  }, [hasEditableVersion, canvasHasSpec, canvasId, handlePlaceholderAdd]);
}
