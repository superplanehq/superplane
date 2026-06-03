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
  handleExitRunsMode: () => void;
  handleToggleEditMode: () => Promise<void>;
  openStartEditingMenu?: () => void;
  setIsRunsMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setRunDetailNodeId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
  startup?: WorkflowStartupActionsConfig;
}

export function useWorkflowHeaderEditActions({
  isRunsMode,
  handleExitRunsMode,
  handleToggleEditMode,
  openStartEditingMenu,
  setIsRunsMode,
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
    }

    await handleToggleEditMode();
  }, [handleToggleEditMode, isRunsMode, setIsRunsMode, setRunDetailNodeId, setSearchParams, setSelectedRunId]);

  const handleExitEditModeFromHeader = useCallback(async () => {
    if (isRunsMode) {
      handleExitRunsMode();
      return;
    }
    await handleToggleEditMode();
  }, [handleExitRunsMode, handleToggleEditMode, isRunsMode]);

  useAutoEditMode(startup, openStartEditingMenu, setSearchParams);
  useAutoPlaceholderNode(startup);

  return { handleEnterEditModeFromHeader, handleExitEditModeFromHeader };
}

function useAutoEditMode(
  startup: WorkflowStartupActionsConfig | undefined,
  openStartEditingMenu: (() => void) | undefined,
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
    if (!openStartEditingMenu) return;

    triggeredRef.current = true;
    openStartEditingMenu();

    const next = new URLSearchParams(searchParams);
    next.delete("edit");
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams, hasEditableVersion, canUpdateCanvas, canvasLoaded, openStartEditingMenu]);
}

function clearRunsViewSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  next.delete("run");
  return next;
}

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
