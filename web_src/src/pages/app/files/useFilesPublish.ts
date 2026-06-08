import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useEffect, useRef, type Dispatch, type SetStateAction } from "react";

import { encodeRepositoryFileContent } from "./lib/repository-files";
import { mergeLoadedContentAfterPublish } from "./lib/files-pending-state";
import type { PendingFileChange, FilesHeaderActionsState } from "./types";

type CommitFilesMutation = {
  mutateAsync: (request: {
    message: string;
    expectedHeadSha?: string;
    versionId?: string;
    operations: Array<{ path: string; delete: true } | { path: string; content: string }>;
  }) => Promise<unknown>;
  isPending: boolean;
};

type UseFilesPublishOptions = {
  canManageRepositoryFiles: boolean;
  canPublishFiles: boolean;
  commitPathError?: string;
  headSha?: string;
  versionId?: string;
  pendingChanges: PendingFileChange[];
  setPendingChangesByPath: (value: Record<string, PendingFileChange>) => void;
  setLoadedContentByPath: Dispatch<SetStateAction<Record<string, string>>>;
  discardAllChanges: () => void;
  onHeaderActionsChange?: (actions: FilesHeaderActionsState | null) => void;
  commitFiles: CommitFilesMutation;
};

export function useFilesPublish({
  canManageRepositoryFiles,
  canPublishFiles,
  commitPathError,
  headSha,
  versionId,
  pendingChanges,
  setPendingChangesByPath,
  setLoadedContentByPath,
  discardAllChanges,
  onHeaderActionsChange,
  commitFiles,
}: UseFilesPublishOptions) {
  const publishChanges = useCallback(async () => {
    if (pendingChanges.length === 0) {
      return;
    }

    if (commitPathError) {
      showErrorToast(commitPathError);
      return;
    }

    const hasSpecChanges = pendingChanges.some((change) => change.type !== "deleted");
    if (hasSpecChanges && !versionId) {
      showErrorToast("Select a draft version before saving canvas.yaml or console.yaml.");
      return;
    }

    try {
      await commitFiles.mutateAsync({
        message: "Update files",
        expectedHeadSha: headSha,
        versionId,
        operations: pendingChanges.map((change) => {
          if (change.type === "deleted") {
            return { path: change.path, delete: true };
          }

          return { path: change.path, content: encodeRepositoryFileContent(change.content) };
        }),
      });

      showSuccessToast("Files saved.");
      setPendingChangesByPath({});
      setLoadedContentByPath((current) => mergeLoadedContentAfterPublish(current, pendingChanges));
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to save files."));
    }
  }, [
    commitFiles,
    commitPathError,
    headSha,
    pendingChanges,
    setLoadedContentByPath,
    setPendingChangesByPath,
    versionId,
  ]);

  const publishChangesRef = useRef(publishChanges);
  publishChangesRef.current = publishChanges;
  const discardAllChangesRef = useRef(discardAllChanges);
  discardAllChangesRef.current = discardAllChanges;

  const publishFileChanges = useCallback(() => {
    void publishChangesRef.current();
  }, []);

  const discardAllFileChanges = useCallback(() => {
    discardAllChangesRef.current();
  }, []);

  const publishPending = commitFiles.isPending;

  useEffect(() => {
    if (!onHeaderActionsChange) {
      return;
    }

    if (!canManageRepositoryFiles) {
      onHeaderActionsChange(null);
      return;
    }

    onHeaderActionsChange({
      hasPendingChanges: pendingChanges.length > 0,
      publishDisabled: !canPublishFiles || publishPending,
      publishDisabledTooltip: commitPathError,
      discardDisabled: pendingChanges.length === 0,
      publishPending,
      onPublish: publishFileChanges,
      onDiscardAll: discardAllFileChanges,
    });
  }, [
    canManageRepositoryFiles,
    canPublishFiles,
    commitPathError,
    discardAllFileChanges,
    onHeaderActionsChange,
    pendingChanges.length,
    publishFileChanges,
    publishPending,
  ]);

  useEffect(() => {
    return () => onHeaderActionsChange?.(null);
  }, [onHeaderActionsChange]);
}

export function canPublishPendingFileChanges(pendingChanges: PendingFileChange[], commitPathError?: string): boolean {
  if (pendingChanges.length === 0) {
    return false;
  }

  return !commitPathError;
}
