import { FilesView } from "./FilesView";
import type { AppFile } from "./types";

export type { AppFile } from "./types";

interface FilesOverlayLayerProps {
  isFilesMode: boolean;
  isEditing?: boolean;
  canvasId?: string;
  organizationId?: string;
  versionId?: string;
  canWrite?: boolean;
  files: AppFile[];
  headerActionsSlotId?: string;
  stagingResetNonce?: number;
  suspendRepositoryFileStaging?: boolean;
  onSpecFileChange?: (path: string, content: string) => void;
  onLocalFilesStagingChange?: (hasStaging: boolean) => void;
  onFlushRepositoryFileStagingReady?: (flush: (() => Promise<void>) | null) => void;
}

export function FilesOverlayLayer({
  isFilesMode,
  isEditing = false,
  canvasId,
  organizationId,
  versionId,
  canWrite = false,
  files,
  headerActionsSlotId,
  stagingResetNonce,
  suspendRepositoryFileStaging,
  onSpecFileChange,
  onLocalFilesStagingChange,
  onFlushRepositoryFileStagingReady,
}: FilesOverlayLayerProps) {
  if (!isFilesMode) return null;

  return (
    <FilesView
      canvasId={canvasId}
      organizationId={organizationId}
      versionId={versionId}
      isEditing={isEditing}
      canWrite={canWrite}
      files={files}
      headerActionsSlotId={headerActionsSlotId}
      stagingResetNonce={stagingResetNonce}
      suspendRepositoryFileStaging={suspendRepositoryFileStaging}
      onSpecFileChange={onSpecFileChange}
      onLocalFilesStagingChange={onLocalFilesStagingChange}
      onFlushRepositoryFileStagingReady={onFlushRepositoryFileStagingReady}
    />
  );
}
