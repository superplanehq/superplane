export type ShouldReactToCanvasVersionUpdatedInput = {
  versionId?: string;
  activeCanvasVersionId: string;
  isEditing: boolean;
  editSessionActive: boolean;
};

// Other tabs on the same canvas only need version-list/detail refreshes when
// they are editing, previewing that version, or have the versions UI open.
// Live-view tabs can ignore remote draft churn; publish still flows through
// canvas_updated, and the agent sidebar listens to canvas:version-updated.
export function shouldReactToCanvasVersionUpdated({
  versionId,
  activeCanvasVersionId,
  isEditing,
  editSessionActive,
}: ShouldReactToCanvasVersionUpdatedInput): boolean {
  if (!versionId) {
    return editSessionActive;
  }

  if (isEditing && activeCanvasVersionId === versionId) {
    return true;
  }

  if (activeCanvasVersionId === versionId) {
    return true;
  }

  return editSessionActive;
}
