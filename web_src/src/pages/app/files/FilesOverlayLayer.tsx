import { FilesCanvasView } from "./FilesCanvasView";
import type { CanvasFile, FilesHeaderActionsState } from "./types";

export type { CanvasFile, FilesHeaderActionsState } from "./types";

interface FilesOverlayLayerProps {
  isFilesMode: boolean;
  isEditing?: boolean;
  canvasId?: string;
  canWrite?: boolean;
  files: CanvasFile[];
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
    <FilesCanvasView
      canvasId={canvasId}
      isEditing={isEditing}
      canWrite={canWrite}
      files={files}
      headerActionsSlotId={headerActionsSlotId}
      onHeaderActionsChange={onHeaderActionsChange}
    />
  );
}
