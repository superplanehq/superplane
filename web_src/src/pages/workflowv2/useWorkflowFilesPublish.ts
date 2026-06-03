import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useEffect, useRef, type Dispatch, type SetStateAction } from "react";

import { encodeRepositoryFileContent } from "./lib/canvas-repository-files";
import { mergeLoadedContentAfterPublish } from "./lib/workflow-files-pending-state";
import type { PendingFileChange, WorkflowFilesHeaderActionsState } from "./workflow-files-types";

type CommitFilesMutation = {
  mutateAsync: (request: {
    message: string;
    expectedHeadSha?: string;
    branch?: string;
    operations: Array<{ path: string; delete: true } | { path: string; content: string }>;
  }) => Promise<unknown>;
  isPending: boolean;
};

type UseWorkflowFilesPublishOptions = {
  canManageRepositoryFiles: boolean;
  canPublishFiles: boolean;
  commitPathError?: string;
  headSha?: string;
  branch?: string;
  pendingChanges: PendingFileChange[];
  setPendingChangesByPath: (value: Record<string, PendingFileChange>) => void;
  setLoadedContentByPath: Dispatch<SetStateAction<Record<string, string>>>;
  discardAllChanges: () => void;
  onHeaderActionsChange?: (actions: WorkflowFilesHeaderActionsState | null) => void;
  commitFiles: CommitFilesMutation;
};

export function useWorkflowFilesPublish({
  canManageRepositoryFiles,
  canPublishFiles,
  commitPathError,
  headSha,
  branch,
  pendingChanges,
  setPendingChangesByPath,
  setLoadedContentByPath,
  discardAllChanges,
  onHeaderActionsChange,
  commitFiles,
}: UseWorkflowFilesPublishOptions) {
  const publishChanges = useCallback(async () => {
    if (commitPathError) {
      showErrorToast(commitPathError);
      return;
    }

    if (pendingChanges.length === 0) {
      return;
    }

    try {
      await commitFiles.mutateAsync({
        message: "Update files",
        expectedHeadSha: headSha,
        branch,
        operations: pendingChanges.map((change) => {
          if (change.type === "deleted") {
            return { path: change.path, delete: true };
          }

          return { path: change.path, content: encodeRepositoryFileContent(change.content) };
        }),
      });

      showSuccessToast("Files published.");
      setPendingChangesByPath({});
      setLoadedContentByPath((current) => mergeLoadedContentAfterPublish(current, pendingChanges));
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to publish files."));
    }
  }, [commitFiles, commitPathError, headSha, branch, pendingChanges, setLoadedContentByPath, setPendingChangesByPath]);

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
      publishDisabled: !canPublishFiles,
      publishDisabledTooltip: commitPathError,
      discardDisabled: pendingChanges.length === 0,
      publishPending: commitFiles.isPending,
      onPublish: publishFileChanges,
      onDiscardAll: discardAllFileChanges,
    });
  }, [
    canManageRepositoryFiles,
    canPublishFiles,
    commitFiles.isPending,
    commitPathError,
    discardAllFileChanges,
    onHeaderActionsChange,
    pendingChanges.length,
    publishFileChanges,
  ]);

  useEffect(() => {
    return () => onHeaderActionsChange?.(null);
  }, [onHeaderActionsChange]);
}
