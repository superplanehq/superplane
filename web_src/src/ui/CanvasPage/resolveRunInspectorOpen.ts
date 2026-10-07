/** Whether the right-hand run inspector panel should mount. */
export function resolveRunInspectorOpen({
  isRunInspectionMode,
  isEditing,
  hasCanvasId,
  hasRunOrLoading,
  factoryEmbed = false,
  selectedNodeId,
}: {
  isRunInspectionMode?: boolean;
  isEditing?: boolean;
  hasCanvasId: boolean;
  hasRunOrLoading: boolean;
  factoryEmbed?: boolean;
  selectedNodeId?: string | null;
}): boolean {
  if (!isRunInspectionMode || isEditing || !hasCanvasId || !hasRunOrLoading) {
    return false;
  }
  // Factory: keep canvas clear until a node is selected (no empty "Select a node…" panel).
  if (factoryEmbed && !selectedNodeId) {
    return false;
  }
  return true;
}
