import { FilesView } from "./FilesView";
import type { AppFile, FilesHeaderActionsState } from "./types";

export type { AppFile, FilesHeaderActionsState } from "./types";

interface FilesOverlayLayerProps {
  isFilesMode: boolean;
  isEditing?: boolean;
  canvasId?: string;
  canWrite?: boolean;
  files: AppFile[];
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: FilesHeaderActionsState | null) => void;
}

export function FilesOverlayLayer({
  isFilesMode,
  isEditing = false,
  canvasId,
  canWrite = false,
  files,
  headerActionsSlotId,
  onHeaderActionsChange,
}: FilesOverlayLayerProps) {
  if (!isFilesMode) return null;

  return (
    <FilesView
      canvasId={canvasId}
      isEditing={isEditing}
      canWrite={canWrite}
      files={files}
      headerActionsSlotId={headerActionsSlotId}
      onHeaderActionsChange={onHeaderActionsChange}
    />
  );
}
