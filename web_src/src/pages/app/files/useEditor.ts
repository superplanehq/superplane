import { useCanvasVersionStaging } from "@/hooks/useCanvasData";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { useMemo, useRef, useState } from "react";

import { buildFilesEditorResult } from "./lib/build-files-editor-result";
import { useEditorCommittedContent } from "./useEditorCommittedContent";
import { useEditorLifecycle } from "./useEditorLifecycle";
import { useEditorStagingSync } from "./useEditorStagingSync";
import { usePendingState } from "./usePendingState";
import { useFilesTabState } from "./useFilesTabState";
import { useStagedFileDiffs } from "./useStagedFileDiffs";
import { useCatalog, useRepositoryPathLists, useRepositorySelectedFileQuery } from "./useCatalog";
import type { AppFile } from "./types";

type UseEditorOptions = {
  canvasId?: string;
  versionId?: string;
  isEditing: boolean;
  canWrite: boolean;
  files: AppFile[];
  headerActionsSlotId?: string;
  stagingResetNonce?: number;
  suspendRepositoryFileStaging?: boolean;
  onSpecFileChange?: (path: string, content: string) => void;
  onLocalFilesStagingChange?: (hasStaging: boolean) => void;
  onFlushRepositoryFileStagingReady?: (flush: (() => Promise<void>) | null) => void;
};

export function useEditor({
  canvasId,
  versionId,
  isEditing,
  canWrite,
  files,
  headerActionsSlotId,
  stagingResetNonce = 0,
  suspendRepositoryFileStaging = false,
  onSpecFileChange,
  onLocalFilesStagingChange,
  onFlushRepositoryFileStagingReady,
}: UseEditorOptions) {
  const leftOffset = useEffectiveLeftSidebarWidth();
  const canManageRepositoryFiles = canWrite && !!canvasId && isEditing;
  const catalog = useCatalog(canvasId, files);
  const stagingQuery = useCanvasVersionStaging(canvasId ?? "", versionId, canManageRepositoryFiles && !!versionId);
  const stagedRepositoryPaths = useMemo(() => {
    const stagedPaths = stagingQuery.data?.stagedPaths ?? [];
    return stagedPaths.filter((path) => !catalog.generatedPathSet.has(path));
  }, [catalog.generatedPathSet, stagingQuery.data?.stagedPaths]);
  const [loadedContentByPath, setLoadedContentByPath] = useState<Record<string, string>>({});
  const { committedContentByPath, setCommittedContentByPath, committedContentByPathRef } = useEditorCommittedContent();
  const [isDiffOpen, setIsDiffOpen] = useState(false);
  const [headerActionsHost, setHeaderActionsHost] = useState<HTMLElement | null>(null);
  const loadedContentByPathRef = useRef(loadedContentByPath);
  loadedContentByPathRef.current = loadedContentByPath;

  const bootstrapPaths = useRepositoryPathLists(
    catalog.generatedPaths,
    catalog.repositoryPaths,
    [],
    stagedRepositoryPaths,
  );
  const allPathsRef = useRef(bootstrapPaths.allPaths);
  const finalRepositoryPathsRef = useRef(bootstrapPaths.finalRepositoryPaths);
  const openFileRef = useRef<(path: string) => void>(() => {});
  const pending = usePendingState({
    generatedPathSet: catalog.generatedPathSet,
    generatedPaths: catalog.generatedPaths,
    finalRepositoryPathsRef,
    allPathsRef,
    loadedContentByPathRef,
    committedContentByPathRef,
    openFile: (path) => openFileRef.current(path),
    versionId,
    onSpecFileChange,
  });
  const pendingChanges = useMemo(
    () => Object.values(pending.pendingChangesByPath).sort((left, right) => left.path.localeCompare(right.path)),
    [pending.pendingChangesByPath],
  );
  const pathLists = useRepositoryPathLists(
    catalog.generatedPaths,
    catalog.repositoryPaths,
    pendingChanges,
    stagedRepositoryPaths,
  );
  allPathsRef.current = pathLists.allPaths;
  finalRepositoryPathsRef.current = pathLists.finalRepositoryPaths;
  const effectiveRepositoryPathSet = useMemo(
    () => new Set(pathLists.finalRepositoryPaths),
    [pathLists.finalRepositoryPaths],
  );

  const tabs = useFilesTabState(pathLists.allPaths, catalog.generatedPaths, catalog.filesQuery.isLoading);
  openFileRef.current = tabs.openFile;

  const selection = useRepositorySelectedFileQuery({
    canvasId,
    selectedPath: tabs.selectedPath,
    repositoryPathSet: effectiveRepositoryPathSet,
    generatedFilesByPath: catalog.generatedFilesByPath,
    versionId,
    stage: isEditing,
  });

  useEditorLifecycle({
    canvasId,
    versionId,
    isEditing,
    resetPendingState: pending.resetPendingState,
    setIsDiffOpen,
    headerActionsSlotId,
    setHeaderActionsHost,
    selectedPath: tabs.selectedPath,
    selectedFileData: selection.selectedFileQuery.data,
    setLoadedContentByPath,
    setCommittedContentByPath,
    stagingResetNonce,
  });

  useEditorStagingSync({
    canvasId,
    versionId,
    canManageRepositoryFiles,
    suspendRepositoryFileStaging,
    pendingChanges,
    committedContentByPath,
    reconcilePendingWithCommitted: pending.reconcilePendingWithCommitted,
    onLocalFilesStagingChange,
    onFlushRepositoryFileStagingReady,
  });

  // Some changes live in the draft's staging layer rather than in the
  // in-session pendingChanges: the virtual spec files (canvas.yaml /
  // console.yaml), and—after a page refresh—repository files whose staged edits
  // outlived the session. Detect them from the server staging state and surface
  // them in the Diff dialog. Paths still covered by a pending change are
  // excluded so they aren't diffed twice (the pending change wins, as it
  // reflects the freshest in-editor content).
  const stagedDiffPaths = useMemo(() => {
    const stagedPaths = stagingQuery.data?.stagedPaths ?? [];
    return stagedPaths.filter((path) => !pending.pendingChangesByPath[path]);
  }, [stagingQuery.data?.stagedPaths, pending.pendingChangesByPath]);
  const stagedFileDiffs = useStagedFileDiffs({
    canvasId,
    versionId,
    paths: stagedDiffPaths,
    enabled: isDiffOpen,
  });

  return buildFilesEditorResult({
    catalog,
    pathLists,
    tabs,
    pending,
    pendingChanges,
    selection,
    loadedContentByPath,
    committedContentByPath,
    stagedDiffPaths,
    stagedFileDiffs,
    canManageRepositoryFiles,
    leftOffset,
    isDiffOpen,
    setIsDiffOpen,
    headerActionsHost,
  });
}
