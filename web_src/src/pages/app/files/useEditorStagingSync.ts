import { useEffect, useMemo } from "react";

import { hasLocalFilesStaging as computeLocalFilesStaging } from "../lib/local-staging-indicators";
import { useRepositoryFileStaging } from "./useRepositoryFileStaging";
import type { PendingFileChange } from "./types";

type UseEditorStagingSyncOptions = {
  canvasId?: string;
  versionId?: string;
  canManageRepositoryFiles: boolean;
  suspendRepositoryFileStaging: boolean;
  pendingChanges: PendingFileChange[];
  committedContentByPath: Record<string, string>;
  reconcilePendingWithCommitted: (committed: Record<string, string>) => void;
  onLocalFilesStagingChange?: (hasStaging: boolean) => void;
  onFlushRepositoryFileStagingReady?: (flush: (() => Promise<void>) | null) => void;
};

export function useEditorStagingSync({
  canvasId,
  versionId,
  canManageRepositoryFiles,
  suspendRepositoryFileStaging,
  pendingChanges,
  committedContentByPath,
  reconcilePendingWithCommitted,
  onLocalFilesStagingChange,
  onFlushRepositoryFileStagingReady,
}: UseEditorStagingSyncOptions) {
  useRepositoryFileStaging({
    canvasId,
    versionId,
    enabled: canManageRepositoryFiles && !!versionId && !suspendRepositoryFileStaging,
    pendingChanges,
    onFlushReady: onFlushRepositoryFileStagingReady,
  });

  useEffect(() => {
    reconcilePendingWithCommitted(committedContentByPath);
  }, [committedContentByPath, reconcilePendingWithCommitted]);

  const hasLocalFilesStaging = useMemo(
    () => computeLocalFilesStaging(pendingChanges, committedContentByPath),
    [pendingChanges, committedContentByPath],
  );

  useEffect(() => {
    onLocalFilesStagingChange?.(hasLocalFilesStaging);
  }, [hasLocalFilesStaging, onLocalFilesStagingChange]);
}
