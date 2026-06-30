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
  /** True once branch metadata (or live version id) is available for entering edit. */
  editEntryReady?: boolean;
}

interface WorkflowHeaderEditActionsConfig {
  isRunInspectionMode: boolean;
  handleClearRunInspection: () => void;
  handleToggleEditMode: () => Promise<boolean>;
  setRunDetailNodeId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
  startup?: WorkflowStartupActionsConfig;
}

function clearRunInspectionSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("run");
  return next;
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
      setRunDetailNodeId(null);
      setSearchParams(clearRunInspectionSearchParams, { replace: true });
      await Promise.resolve();
    }

    await handleToggleEditMode();
  }, [handleToggleEditMode, isRunInspectionMode, setRunDetailNodeId, setSearchParams]);

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

    setRunDetailNodeId(null);
    setSearchParams(clearRunInspectionSearchParams, { replace: true });
  }, [isRunInspectionMode, setRunDetailNodeId, setSearchParams]);

  return { handleEnterEditModeFromHeader, handleExitEditModeFromHeader, clearRunInspectionForEdit };
}

function useAutoEditMode(
  startup: WorkflowStartupActionsConfig | undefined,
  handleToggleEditMode: () => Promise<boolean>,
  setRunDetailNodeId: (value: string | null) => void,
  setSearchParams: SetURLSearchParams,
) {
  const triggeredRef = useRef(false);
  const canvasId = startup?.canvas?.metadata?.id;
  const hasEditableVersion = startup?.hasEditableVersion ?? false;
  const canUpdateCanvas = startup?.canUpdateCanvas ?? false;
  const canvasLoaded = Boolean(startup?.canvas);
  const editEntryReady = startup?.editEntryReady ?? true;
  const searchParams = startup?.searchParams;

  useEffect(() => {
    triggeredRef.current = false;
  }, [canvasId]);

  useEffect(() => {
    if (triggeredRef.current) return;
    if (!searchParams || searchParams.get("edit") !== "1") return;
    if (!canvasLoaded) return;
    if (!editEntryReady) return;
    if (hasEditableVersion) return;
    if (!canUpdateCanvas) return;

    void (async () => {
      if (searchParams.get("run")) {
        setRunDetailNodeId(null);
        setSearchParams(clearRunInspectionSearchParams, { replace: true });
        await Promise.resolve();
      }

      const enteredEditMode = await handleToggleEditMode();
      if (!enteredEditMode) {
        return;
      }

      triggeredRef.current = true;
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
    editEntryReady,
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
