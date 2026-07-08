import type { CanvasesCanvas, CanvasesCanvasVersion, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { getActiveNoteId, restoreActiveNoteFocus } from "@/ui/annotationComponent/noteFocus";
import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import type { CanvasSaveResult } from "./canvasSaveTypes";
import { resolveEditableWorkflowSnapshot } from "./lib/editable-workflow-snapshot";

function applyPositionUpdates(
  nodes: ComponentsNode[],
  positionUpdates: Map<string, { x: number; y: number }>,
): ComponentsNode[] {
  return nodes.map((node) => {
    if (!node.id) return node;

    const positionUpdate = positionUpdates.get(node.id);
    if (positionUpdate) {
      return {
        ...node,
        position: positionUpdate,
      };
    }
    return node;
  });
}

function mergePendingPositionUpdates(
  workflow: CanvasesCanvas,
  positionUpdates: Map<string, { x: number; y: number }>,
): CanvasesCanvas {
  if (!workflow.spec?.nodes) {
    return workflow;
  }

  return {
    ...workflow,
    spec: {
      ...workflow.spec,
      nodes: applyPositionUpdates(workflow.spec.nodes, positionUpdates),
    },
  };
}

export type RunPositionAutoSaveOptions = {
  setIsPositionAutoSaveQueued: (queued: boolean) => void;
  organizationId?: string;
  canvasId?: string;
  pendingPositionUpdatesRef: MutableRefObject<Map<string, { x: number; y: number }>>;
  isReadOnly: boolean;
  isEditing: boolean;
  canvasRef: MutableRefObject<CanvasesCanvas | null>;
  queryClient: QueryClient;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  activeCanvasVersionId: string;
  canvasContentVersionIdRef: MutableRefObject<string>;
  enqueueCanvasSave: (workflow: CanvasesCanvas, savingVersionId?: string) => Promise<CanvasSaveResult>;
  setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  applyLocalWorkflowUpdate: (workflow: CanvasesCanvas) => void;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
};

export async function runPositionAutoSave({
  setIsPositionAutoSaveQueued,
  organizationId,
  canvasId,
  pendingPositionUpdatesRef,
  isReadOnly,
  isEditing,
  canvasRef,
  queryClient,
  activeCanvasVersionIdRef,
  activeCanvasVersionId,
  canvasContentVersionIdRef,
  enqueueCanvasSave,
  setActiveCanvasVersion,
  applyLocalWorkflowUpdate,
  setLastSavedWorkflowSnapshot,
}: RunPositionAutoSaveOptions): Promise<void> {
  setIsPositionAutoSaveQueued(false);
  if (!organizationId || !canvasId) return;

  const positionUpdates = new Map(pendingPositionUpdatesRef.current);
  if (positionUpdates.size === 0) return;
  const focusedNoteId = getActiveNoteId();

  try {
    if (isReadOnly) {
      return;
    }

    const latestWorkflow = resolveEditableWorkflowSnapshot({
      isEditing,
      renderedWorkflow: canvasRef.current,
      detailWorkflow: queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)),
    });

    if (!latestWorkflow?.spec?.nodes) return;

    const updatedWorkflow = mergePendingPositionUpdates(latestWorkflow, positionUpdates);

    const savingVersionID = activeCanvasVersionIdRef.current || activeCanvasVersionId || undefined;
    if (!savingVersionID || canvasContentVersionIdRef.current !== savingVersionID) {
      return;
    }

    const saveResult = await enqueueCanvasSave(updatedWorkflow, savingVersionID);
    if (saveResult.status !== "saved") {
      return;
    }
    if (
      saveResult.response?.data?.staging?.spec &&
      savingVersionID &&
      activeCanvasVersionIdRef.current === savingVersionID
    ) {
      setActiveCanvasVersion((current) =>
        current?.metadata?.id === savingVersionID
          ? { ...current, spec: saveResult.response?.data?.staging?.spec }
          : current,
      );
    }
    if (activeCanvasVersionIdRef.current !== (savingVersionID || "")) {
      return;
    }

    applyLocalWorkflowUpdate(updatedWorkflow);
    setLastSavedWorkflowSnapshot(updatedWorkflow);

    positionUpdates.forEach((_, nodeId) => {
      if (pendingPositionUpdatesRef.current.get(nodeId) === positionUpdates.get(nodeId)) {
        pendingPositionUpdatesRef.current.delete(nodeId);
      }
    });

    const currentWorkflow = resolveEditableWorkflowSnapshot({
      isEditing,
      renderedWorkflow: canvasRef.current,
      detailWorkflow: queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)),
    });

    if (currentWorkflow?.spec?.nodes && pendingPositionUpdatesRef.current.size > 0) {
      applyLocalWorkflowUpdate(mergePendingPositionUpdates(currentWorkflow, pendingPositionUpdatesRef.current));
    }
  } catch (error) {
    console.error("Failed to auto-save", error);
  } finally {
    if (focusedNoteId) {
      requestAnimationFrame(() => {
        restoreActiveNoteFocus();
      });
    }
  }
}
