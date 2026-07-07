import { useCallback, useEffect, useRef } from "react";
import type { SetURLSearchParams } from "react-router-dom";
import type { CanvasesCanvas } from "@/api-client";
import {
  abandonPendingPlaceholderBoot,
  markAgentBootReady,
  PLACEHOLDER_NODE_CONTEXT_KEY,
} from "@/lib/agentBootContext";
import { clearRunInspectionSearchParams } from "./viewState";

type PlaceholderAddHandler = (data: { position: { x: number; y: number } }) => Promise<string>;

interface WorkflowStartupActionsConfig {
  hasEditableVersion: boolean;
  canUpdateCanvas: boolean;
  canvas: CanvasesCanvas | null | undefined;
  liveVersionLoading?: boolean;
  handlePlaceholderAdd?: PlaceholderAddHandler;
  searchParams: URLSearchParams;
}

interface WorkflowHeaderEditActionsConfig {
  isRunInspectionMode: boolean;
  handleClearRunInspection: () => void;
  handleToggleEditMode: () => Promise<void>;
  setRunDetailNodeId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
  startup?: WorkflowStartupActionsConfig;
}

export function useWorkflowHeaderEditActions({
  isRunInspectionMode,
  handleClearRunInspection,
  handleToggleEditMode,
  setRunDetailNodeId,
  setSearchParams,
  startup,
}: WorkflowHeaderEditActionsConfig) {
  const handleEnterEditModeFromHeader = useCallback(async () => {
    if (isRunInspectionMode) {
      handleClearRunInspection();
      await Promise.resolve();
    }

    await handleToggleEditMode();

    if (isRunInspectionMode) {
      handleClearRunInspection();
    }
  }, [handleClearRunInspection, handleToggleEditMode, isRunInspectionMode]);

  const handleExitEditModeFromHeader = useCallback(async () => {
    if (isRunInspectionMode) {
      handleClearRunInspection();
    }
    await handleToggleEditMode();
  }, [handleClearRunInspection, handleToggleEditMode, isRunInspectionMode]);

  useAutoEditMode(startup, handleToggleEditMode, setRunDetailNodeId, setSearchParams);
  useAutoPlaceholderNode(startup);

  const clearRunInspectionForEdit = useCallback(() => {
    if (!isRunInspectionMode) {
      return;
    }

    handleClearRunInspection();
  }, [handleClearRunInspection, isRunInspectionMode]);

  return { handleEnterEditModeFromHeader, handleExitEditModeFromHeader, clearRunInspectionForEdit };
}

function useAutoEditMode(
  startup: WorkflowStartupActionsConfig | undefined,
  handleToggleEditMode: () => Promise<void>,
  setRunDetailNodeId: (value: string | null) => void,
  setSearchParams: SetURLSearchParams,
) {
  const triggeredRef = useRef(false);
  const hasEditableVersion = startup?.hasEditableVersion ?? false;
  const canUpdateCanvas = startup?.canUpdateCanvas ?? false;
  const canvasLoaded = Boolean(startup?.canvas);
  const liveVersionLoading = startup?.liveVersionLoading ?? false;
  const searchParams = startup?.searchParams;

  useEffect(() => {
    if (triggeredRef.current) return;
    if (!searchParams || searchParams.get("edit") !== "1") return;
    if (!canvasLoaded) return;
    if (hasEditableVersion) return;
    if (!canUpdateCanvas) return;
    if (liveVersionLoading) return;

    triggeredRef.current = true;

    void (async () => {
      if (searchParams.get("run")) {
        setRunDetailNodeId(null);
        setSearchParams(clearRunInspectionSearchParams, { replace: true });
        await Promise.resolve();
      }

      await handleToggleEditMode();
      if (searchParams.get("run")) {
        setRunDetailNodeId(null);
        setSearchParams(clearRunInspectionSearchParams, { replace: true });
      }
      setSearchParams(
        (current) => {
          const next = new URLSearchParams(current);
          next.delete("edit");
          return next;
        },
        { replace: true },
      );
    })();
  }, [
    searchParams,
    setSearchParams,
    setRunDetailNodeId,
    hasEditableVersion,
    canUpdateCanvas,
    canvasLoaded,
    liveVersionLoading,
    handleToggleEditMode,
  ]);
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
