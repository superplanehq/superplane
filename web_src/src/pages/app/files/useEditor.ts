import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { useEffect, useMemo, useRef, useState } from "react";

import { buildFilesEditorResult } from "./lib/build-files-editor-result";
import { hasLocalFilesStaging as computeLocalFilesStaging } from "../lib/local-staging-indicators";
import { useEditorLifecycle } from "./useEditorLifecycle";
import { usePendingState } from "./usePendingState";
import { useRepositoryFileStaging } from "./useRepositoryFileStaging";
import { useFilesTabState } from "./useFilesTabState";
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
}: UseEditorOptions) {
  const leftOffset = useEffectiveLeftSidebarWidth();
  const canManageRepositoryFiles = canWrite && !!canvasId && isEditing;
  const catalog = useCatalog(canvasId, files);
  const [loadedContentByPath, setLoadedContentByPath] = useState<Record<string, string>>({});
  const [committedContentByPath, setCommittedContentByPath] = useState<Record<string, string>>({});
  const [isDiffOpen, setIsDiffOpen] = useState(false);
  const [headerActionsHost, setHeaderActionsHost] = useState<HTMLElement | null>(null);
  const loadedContentByPathRef = useRef(loadedContentByPath);
  loadedContentByPathRef.current = loadedContentByPath;
  const committedContentByPathRef = useRef(committedContentByPath);
  committedContentByPathRef.current = committedContentByPath;

  const bootstrapPaths = useRepositoryPathLists(catalog.generatedPaths, catalog.repositoryPaths, []);
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
  const pathLists = useRepositoryPathLists(catalog.generatedPaths, catalog.repositoryPaths, pendingChanges);
  allPathsRef.current = pathLists.allPaths;
  finalRepositoryPathsRef.current = pathLists.finalRepositoryPaths;

  // Mirror non-spec file edits into the draft staging layer (debounced) so the
  // header switches to Reset/Commit and Commit can persist them to git.
  useRepositoryFileStaging({
    canvasId,
    versionId,
    enabled: canManageRepositoryFiles && !!versionId && !suspendRepositoryFileStaging,
    pendingChanges,
  });
  const tabs = useFilesTabState(pathLists.allPaths, catalog.generatedPaths, catalog.filesQuery.isLoading);
  openFileRef.current = tabs.openFile;

  const selection = useRepositorySelectedFileQuery(
    canvasId,
    tabs.selectedPath,
    catalog.repositoryPathSet,
    catalog.generatedFilesByPath,
    versionId,
  );

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

  const { reconcilePendingWithCommitted } = pending;

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

  return buildFilesEditorResult({
    catalog,
    pathLists,
    tabs,
    pending,
    pendingChanges,
    selection,
    loadedContentByPath,
    canManageRepositoryFiles,
    leftOffset,
    isDiffOpen,
    setIsDiffOpen,
    headerActionsHost,
  });
}
