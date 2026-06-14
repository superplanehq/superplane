import { FilesView } from "./FilesView";
import type { AppFile } from "./types";

export type { AppFile } from "./types";

interface FilesOverlayLayerProps {
  isFilesMode: boolean;
  isEditing?: boolean;
  canvasId?: string;
  versionId?: string;
  canWrite?: boolean;
  files: AppFile[];
  headerActionsSlotId?: string;
  stagingResetNonce?: number;
  suspendRepositoryFileStaging?: boolean;
  hasCanvasSpecDiffVersusLive?: boolean;
  hasConsoleSpecDiffVersusLive?: boolean;
  onSpecFileChange?: (path: string, content: string) => void;
  onLocalFilesStagingChange?: (hasStaging: boolean) => void;
  onFlushRepositoryFileStagingReady?: (flush: (() => Promise<void>) | null) => void;
}

export function FilesOverlayLayer({
  isFilesMode,
  isEditing = false,
  canvasId,
  versionId,
  canWrite = false,
  files,
  headerActionsSlotId,
  stagingResetNonce,
  suspendRepositoryFileStaging,
  hasCanvasSpecDiffVersusLive,
  hasConsoleSpecDiffVersusLive,
  onSpecFileChange,
  onLocalFilesStagingChange,
  onFlushRepositoryFileStagingReady,
}: FilesOverlayLayerProps) {
  if (!isFilesMode) return null;

  return (
    <FilesView
      canvasId={canvasId}
      versionId={versionId}
      isEditing={isEditing}
      canWrite={canWrite}
      files={files}
      headerActionsSlotId={headerActionsSlotId}
      stagingResetNonce={stagingResetNonce}
      suspendRepositoryFileStaging={suspendRepositoryFileStaging}
      hasCanvasSpecDiffVersusLive={hasCanvasSpecDiffVersusLive}
      hasConsoleSpecDiffVersusLive={hasConsoleSpecDiffVersusLive}
      onSpecFileChange={onSpecFileChange}
      onLocalFilesStagingChange={onLocalFilesStagingChange}
      onFlushRepositoryFileStagingReady={onFlushRepositoryFileStagingReady}
    />
  );
}
