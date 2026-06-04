import { WorkflowFilesCanvasView } from "./WorkflowFilesCanvasView";
import type { CanvasBranchStagingState } from "./useCanvasBranchStaging";
import type { WorkflowFile, WorkflowFilesHeaderActionsState } from "./workflow-files-types";

export type { WorkflowFile, WorkflowFilesHeaderActionsState } from "./workflow-files-types";

interface WorkflowFilesOverlayLayerProps {
  isFilesMode: boolean;
  isEditing?: boolean;
  canvasId?: string;
  canWrite?: boolean;
  activeBranch?: string | null;
  branchTipSha?: string;
  branchStaging?: CanvasBranchStagingState;
  files: WorkflowFile[];
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: WorkflowFilesHeaderActionsState | null) => void;
}

export function WorkflowFilesOverlayLayer({
  isFilesMode,
  isEditing = false,
  canvasId,
  canWrite = false,
  activeBranch,
  branchTipSha,
  branchStaging,
  files,
  headerActionsSlotId,
  onHeaderActionsChange,
}: WorkflowFilesOverlayLayerProps) {
  if (!isFilesMode) return null;

  return (
    <WorkflowFilesCanvasView
      canvasId={canvasId}
      isEditing={isEditing}
      canWrite={canWrite}
      activeBranch={activeBranch}
      branchTipSha={branchTipSha}
      branchStaging={branchStaging}
      files={files}
      headerActionsSlotId={headerActionsSlotId}
      onHeaderActionsChange={onHeaderActionsChange}
    />
  );
}
