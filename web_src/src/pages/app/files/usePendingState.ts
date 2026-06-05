import { showErrorToast } from "@/lib/toast";
import { useCallback, useState, type RefObject } from "react";

import { applyPendingContentUpdate, applyPendingDelete } from "./lib/files-pending-state";
import { getPathValidationError, nextUntitledPath, normalizeFilePath } from "./lib/files-paths";
import type { PendingFileChange } from "./types";

type UsePendingStateOptions = {
  generatedPathSet: Set<string>;
  generatedPaths: string[];
  finalRepositoryPathsRef: RefObject<string[]>;
  allPathsRef: RefObject<string[]>;
  loadedContentByPathRef: RefObject<Record<string, string>>;
  openFile: (path: string) => void;
};

export function usePendingState({
  generatedPathSet,
  generatedPaths,
  finalRepositoryPathsRef,
  allPathsRef,
  loadedContentByPathRef,
  openFile,
}: UsePendingStateOptions) {
  const [pendingChangesByPath, setPendingChangesByPath] = useState<Record<string, PendingFileChange>>({});
  const [newFilePath, setNewFilePath] = useState<string | null>(null);

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

    setPendingChangesByPath((current) => ({
      ...current,
      [path]: { type: "added", path, content: "" },
    }));
    setNewFilePath(null);
    openFile(path);
  }, [finalRepositoryPathsRef, generatedPaths, newFilePath, openFile]);

  const cancelNewFile = useCallback(() => {
    setNewFilePath(null);
  }, []);

  const updateSelectedContent = useCallback(
    (selectedPath: string | null, value: string) => {
      if (!selectedPath || generatedPathSet.has(selectedPath)) return;

      setPendingChangesByPath((current) =>
        applyPendingContentUpdate(current, selectedPath, value, loadedContentByPathRef.current[selectedPath]),
      );
    },
    [generatedPathSet, loadedContentByPathRef],
  );

  const deleteFile = useCallback(
    (path: string) => {
      if (generatedPathSet.has(path)) return;
      setPendingChangesByPath((current) => applyPendingDelete(current, path));
    },
    [generatedPathSet],
  );

  const discardAllChanges = useCallback(() => {
    setPendingChangesByPath({});
  }, []);

  const resetPendingState = useCallback(() => {
    setPendingChangesByPath({});
    setNewFilePath(null);
  }, []);

  return {
    pendingChangesByPath,
    setPendingChangesByPath,
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
