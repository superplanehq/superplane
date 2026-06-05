import { useCommitCanvasRepositoryFiles } from "@/hooks/useCanvasData";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { useMemo, useRef, useState } from "react";

import { buildFilesEditorResult } from "./lib/build-files-editor-result";
import { useEditorLifecycle } from "./useEditorLifecycle";
import { usePendingState } from "./usePendingState";
import { useFilesPublish } from "./useFilesPublish";
import { useFilesTabState } from "./useFilesTabState";
import { useCatalog, useRepositoryPathLists, useRepositorySelectedFileQuery } from "./useCatalog";
import type { AppFile, FilesHeaderActionsState } from "./types";

type UseEditorOptions = {
  canvasId?: string;
  isEditing: boolean;
  canWrite: boolean;
  files: AppFile[];
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: FilesHeaderActionsState | null) => void;
};

export function useEditor({
  canvasId,
  isEditing,
  canWrite,
  files,
  headerActionsSlotId,
  onHeaderActionsChange,
}: UseEditorOptions) {
  const leftOffset = useEffectiveLeftSidebarWidth();
  const canManageRepositoryFiles = canWrite && !!canvasId && isEditing;
  const catalog = useCatalog(canvasId, files);
  const commitFiles = useCommitCanvasRepositoryFiles(canvasId ?? "");
  const [loadedContentByPath, setLoadedContentByPath] = useState<Record<string, string>>({});
  const [isDiffOpen, setIsDiffOpen] = useState(false);
  const [headerActionsHost, setHeaderActionsHost] = useState<HTMLElement | null>(null);
  const loadedContentByPathRef = useRef(loadedContentByPath);
  loadedContentByPathRef.current = loadedContentByPath;

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
    openFile: (path) => openFileRef.current(path),
  });
  const pendingChanges = useMemo(
    () => Object.values(pending.pendingChangesByPath).sort((left, right) => left.path.localeCompare(right.path)),
    [pending.pendingChangesByPath],
  );
  const pathLists = useRepositoryPathLists(catalog.generatedPaths, catalog.repositoryPaths, pendingChanges);
  allPathsRef.current = pathLists.allPaths;
  finalRepositoryPathsRef.current = pathLists.finalRepositoryPaths;
  const tabs = useFilesTabState(catalog.generatedPaths[0] ?? null, pathLists.allPaths, catalog.generatedPaths);
  openFileRef.current = tabs.openFile;
  const selection = useRepositorySelectedFileQuery(
    canvasId,
    tabs.selectedPath,
    catalog.repositoryPathSet,
    catalog.generatedFilesByPath,
  );

  useFilesPublish({
    canManageRepositoryFiles,
    canPublishFiles:
      canManageRepositoryFiles && pendingChanges.length > 0 && !pathLists.commitPathError && !commitFiles.isPending,
    commitPathError: pathLists.commitPathError,
    headSha: catalog.headSha,
    pendingChanges,
    setPendingChangesByPath: pending.setPendingChangesByPath,
    setLoadedContentByPath,
    discardAllChanges: pending.discardAllChanges,
    onHeaderActionsChange,
    commitFiles,
  });
  useEditorLifecycle({
    isEditing,
    resetPendingState: pending.resetPendingState,
    setIsDiffOpen,
    headerActionsSlotId,
    setHeaderActionsHost,
    selectedPath: tabs.selectedPath,
    selectedFileData: selection.selectedFileQuery.data,
    setLoadedContentByPath,
  });

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
