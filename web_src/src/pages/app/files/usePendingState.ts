import { showErrorToast } from "@/lib/toast";
import { useCallback, useEffect, useState, type RefObject } from "react";

import { isWorkflowSpecPath } from "../lib/workflow-spec-paths";
import { applyPendingContentUpdate, applyPendingDelete } from "./lib/files-pending-state";
import { getPathValidationError, nextUntitledPath, normalizeFilePath } from "./lib/files-paths";
import type { PendingFileChange } from "./types";

type UsePendingStateOptions = {
  generatedPathSet: Set<string>;
  generatedPaths: string[];
  finalRepositoryPathsRef: RefObject<string[]>;
  allPathsRef: RefObject<string[]>;
  loadedContentByPathRef: RefObject<Record<string, string>>;
  committedContentByPathRef: RefObject<Record<string, string>>;
  openFile: (path: string) => void;
  versionId?: string;
  onSpecFileChange?: (path: string, content: string) => void;
};

export function usePendingState({
  generatedPathSet,
  generatedPaths,
  finalRepositoryPathsRef,
  allPathsRef,
  loadedContentByPathRef,
  committedContentByPathRef,
  openFile,
  versionId,
  onSpecFileChange,
}: UsePendingStateOptions) {
  const [pendingChangesByPath, setPendingChangesByPath] = useState<Record<string, PendingFileChange>>({});
  const [specDraftByPath, setSpecDraftByPath] = useState<Record<string, string>>({});
  const [newFilePath, setNewFilePath] = useState<string | null>(null);

  // Spec drafts are scoped to a canvas version; drop them when the version
  // changes so a switch never shows stale edits from a previous version.
  useEffect(() => {
    setSpecDraftByPath({});
  }, [versionId]);

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
      if (!selectedPath) return;

      // Spec files (canvas.yaml / console.yaml) are not part of the normal
      // pending/publish flow. They are materialized into the live canvas /
      // console state and auto-saved immediately. We keep a local draft so the
      // editor stays responsive while the debounced save runs.
      if (isWorkflowSpecPath(selectedPath)) {
        setSpecDraftByPath((current) => ({ ...current, [selectedPath]: value }));
        onSpecFileChange?.(selectedPath, value);
        return;
      }

      if (generatedPathSet.has(selectedPath)) return;

      // Prefer the committed (stage=false) baseline so reverting an edit back to
      // the original clears the pending change. loadedContentByPath holds staged
      // content for draft reads, so after autosave it no longer reflects the
      // original and would keep a phantom pending change (and Diff button) alive.
      const originalContent =
        committedContentByPathRef.current[selectedPath] ?? loadedContentByPathRef.current[selectedPath];
      setPendingChangesByPath((current) => applyPendingContentUpdate(current, selectedPath, value, originalContent));
    },
    [committedContentByPathRef, generatedPathSet, loadedContentByPathRef, onSpecFileChange],
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
    setSpecDraftByPath({});
    setNewFilePath(null);
  }, []);

  const reconcilePendingWithCommitted = useCallback((committedContentByPath: Record<string, string>) => {
    setPendingChangesByPath((current) => {
      let changed = false;
      const next: Record<string, PendingFileChange> = { ...current };

      for (const [path, change] of Object.entries(current)) {
        if (change.type !== "modified") {
          continue;
        }

        const committed = committedContentByPath[path];
        if (committed !== undefined && change.content === committed) {
          delete next[path];
          changed = true;
        }
      }

      return changed ? next : current;
    });
  }, []);

  return {
    pendingChangesByPath,
    setPendingChangesByPath,
    specDraftByPath,
    newFilePath,
    setNewFilePath,
    startNewFile,
    createNewFile,
    cancelNewFile,
    updateSelectedContent,
    deleteFile,
    discardAllChanges,
    resetPendingState,
    reconcilePendingWithCommitted,
  };
}
