import { useMemo, useCallback, useState, type RefObject } from "react";

import { pendingChangesFromStaging } from "./lib/workflow-files-staging";
import { getPathValidationError, nextUntitledPath, normalizeFilePath } from "./lib/workflow-files-paths";
import type { CanvasBranchStagingState } from "./useCanvasBranchStaging";
import type { PendingFileChange } from "./workflow-files-types";
import { showErrorToast } from "@/lib/toast";

type UseWorkflowFilesStagingStateOptions = {
  branchStaging?: Pick<
    CanvasBranchStagingState,
    "stagingRecord" | "stageRepositoryFile" | "stageRepositoryFileDelete" | "unstageRepositoryFile"
  >;
  generatedPathSet: Set<string>;
  generatedPaths: string[];
  repositoryPathSet: Set<string>;
  finalRepositoryPathsRef: RefObject<string[]>;
  allPathsRef: RefObject<string[]>;
  loadedContentByPath: Record<string, string>;
  loadedContentByPathRef: RefObject<Record<string, string>>;
  openFile: (path: string) => void;
};

export function useWorkflowFilesStagingState({
  branchStaging,
  generatedPathSet,
  generatedPaths,
  repositoryPathSet,
  finalRepositoryPathsRef,
  allPathsRef,
  loadedContentByPath,
  loadedContentByPathRef,
  openFile,
}: UseWorkflowFilesStagingStateOptions) {
  const [localPendingChangesByPath, setLocalPendingChangesByPath] = useState<Record<string, PendingFileChange>>({});
  const [newFilePath, setNewFilePath] = useState<string | null>(null);

  const stagingPendingChangesByPath = useMemo(
    () => pendingChangesFromStaging(branchStaging?.stagingRecord, loadedContentByPath, repositoryPathSet),
    [branchStaging?.stagingRecord, loadedContentByPath, repositoryPathSet],
  );

  const pendingChangesByPath = branchStaging ? stagingPendingChangesByPath : localPendingChangesByPath;

  const startNewFile = useCallback(() => {
    setNewFilePath(nextUntitledPath(new Set(allPathsRef.current)));
  }, [allPathsRef]);

  const createNewFile = useCallback(() => {
    if (newFilePath === null) return;

    const path = normalizeFilePath(newFilePath);
    if (!path) {
      setNewFilePath(null);
      return;
    }

    const errorMessage = getPathValidationError([...generatedPaths, ...finalRepositoryPathsRef.current, path]);
    if (errorMessage) {
      showErrorToast(errorMessage);
      return;
    }

    if (branchStaging) {
      branchStaging.stageRepositoryFile(path, "");
    } else {
      setLocalPendingChangesByPath((current) => ({
        ...current,
        [path]: { type: "added", path, content: "" },
      }));
    }

    setNewFilePath(null);
    openFile(path);
  }, [branchStaging, finalRepositoryPathsRef, generatedPaths, newFilePath, openFile]);

  const cancelNewFile = useCallback(() => {
    setNewFilePath(null);
  }, []);

  const updateSelectedContent = useCallback(
    (selectedPath: string | null, value: string) => {
      if (!selectedPath || generatedPathSet.has(selectedPath)) return;

      const originalContent = loadedContentByPathRef.current[selectedPath];
      if (branchStaging) {
        if (originalContent === undefined) {
          branchStaging.stageRepositoryFile(selectedPath, value);
          return;
        }

        if (value === originalContent) {
          branchStaging.unstageRepositoryFile(selectedPath);
          return;
        }

        branchStaging.stageRepositoryFile(selectedPath, value);
        return;
      }

      if (originalContent === undefined) {
        return;
      }

      setLocalPendingChangesByPath((currentPending) => {
        const currentChange = currentPending[selectedPath];
        if (currentChange?.type === "added") {
          return { ...currentPending, [selectedPath]: { ...currentChange, content: value } };
        }

        if (value === originalContent) {
          const { [selectedPath]: _removed, ...remaining } = currentPending;
          return remaining;
        }

        return {
          ...currentPending,
          [selectedPath]: { type: "modified", path: selectedPath, content: value },
        };
      });
    },
    [branchStaging, generatedPathSet, loadedContentByPathRef],
  );

  const deleteFile = useCallback(
    (path: string) => {
      if (generatedPathSet.has(path)) return;

      if (branchStaging) {
        branchStaging.stageRepositoryFileDelete(path, repositoryPathSet.has(path));
        return;
      }

      setLocalPendingChangesByPath((currentPending) => {
        const currentChange = currentPending[path];
        if (currentChange?.type === "added") {
          const { [path]: _removed, ...remaining } = currentPending;
          return remaining;
        }

        return {
          ...currentPending,
          [path]: { type: "deleted", path },
        };
      });
    },
    [branchStaging, generatedPathSet, repositoryPathSet],
  );

  const discardAllChanges = useCallback(() => {
    setLocalPendingChangesByPath({});
  }, []);

  const resetPendingState = useCallback(() => {
    setLocalPendingChangesByPath({});
    setNewFilePath(null);
  }, []);

  return {
    pendingChangesByPath,
    setPendingChangesByPath: setLocalPendingChangesByPath,
    newFilePath,
    setNewFilePath,
    startNewFile,
    createNewFile,
    cancelNewFile,
    updateSelectedContent,
    deleteFile,
    discardAllChanges,
    resetPendingState,
  };
}
