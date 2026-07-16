import { useCallback, useEffect, useState } from "react";

type UseContinueEditingTransitionOptions = {
  isEditing: boolean;
  isRunInspectionMode: boolean;
  onEnterEditMode?: () => void | Promise<void>;
  onOpenPendingEditNode: (nodeId: string) => void;
  selectedNodeId: string | null;
};

export function useContinueEditingTransition({
  isEditing,
  isRunInspectionMode,
  onEnterEditMode,
  onOpenPendingEditNode,
  selectedNodeId,
}: UseContinueEditingTransitionOptions) {
  const [pendingRuntimeEditNodeId, setPendingRuntimeEditNodeId] = useState<string | null>(null);

  useEffect(() => {
    if (!pendingRuntimeEditNodeId || isRunInspectionMode || !isEditing) {
      return;
    }

    onOpenPendingEditNode(pendingRuntimeEditNodeId);
    setPendingRuntimeEditNodeId(null);
  }, [isEditing, isRunInspectionMode, onOpenPendingEditNode, pendingRuntimeEditNodeId]);

  const beginEditSessionForNode = useCallback(
    (nodeId: string) => {
      setPendingRuntimeEditNodeId(nodeId);
      void (async () => {
        try {
          await onEnterEditMode?.();
        } catch {
          setPendingRuntimeEditNodeId((currentNodeId) => (currentNodeId === nodeId ? null : currentNodeId));
        }
      })();
    },
    [onEnterEditMode],
  );

  const handleContinueEditing = useCallback(() => {
    if (!selectedNodeId || !onEnterEditMode) {
      return;
    }

    beginEditSessionForNode(selectedNodeId);
  }, [beginEditSessionForNode, onEnterEditMode, selectedNodeId]);

  return {
    beginEditSessionForNode,
    handleContinueEditing,
    pendingRuntimeEditNodeId,
  };
}
