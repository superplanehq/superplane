import { useCommitCanvasRepositoryFiles } from "@/hooks/useCanvasData";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { useMemo, useRef, useState } from "react";

import { buildWorkflowFilesEditorResult } from "./lib/build-workflow-files-editor-result";
import { useWorkflowFilesEditorLifecycle } from "./useWorkflowFilesEditorLifecycle";
import { useWorkflowFilesPendingState } from "./useWorkflowFilesPendingState";
import { useWorkflowFilesPublish } from "./useWorkflowFilesPublish";
import { useWorkflowFilesTabState } from "./useWorkflowFilesTabState";
import {
  useWorkflowRepositoryFilesCatalog,
  useWorkflowRepositoryPathLists,
  useWorkflowRepositorySelectedFileQuery,
} from "./useWorkflowRepositoryFilesCatalog";
import type { WorkflowFile, WorkflowFilesHeaderActionsState } from "./workflow-files-types";

type UseWorkflowRepositoryFilesEditorOptions = {
  canvasId?: string;
  isEditing: boolean;
  canWrite: boolean;
  files: WorkflowFile[];
  activeBranch?: string | null;
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: WorkflowFilesHeaderActionsState | null) => void;
};

export function useWorkflowRepositoryFilesEditor({
  canvasId,
  isEditing,
  canWrite,
  files,
  activeBranch,
  headerActionsSlotId,
  onHeaderActionsChange,
}: UseWorkflowRepositoryFilesEditorOptions) {
  const leftOffset = useEffectiveLeftSidebarWidth();
  const canManageRepositoryFiles = canWrite && !!canvasId && isEditing;
  const catalog = useWorkflowRepositoryFilesCatalog(canvasId, files, activeBranch ?? undefined);
  const commitFiles = useCommitCanvasRepositoryFiles(canvasId ?? "");
  const [loadedContentByPath, setLoadedContentByPath] = useState<Record<string, string>>({});
  const [isDiffOpen, setIsDiffOpen] = useState(false);
  const [headerActionsHost, setHeaderActionsHost] = useState<HTMLElement | null>(null);
  const loadedContentByPathRef = useRef(loadedContentByPath);
  loadedContentByPathRef.current = loadedContentByPath;

  const bootstrapPaths = useWorkflowRepositoryPathLists(catalog.generatedPaths, catalog.repositoryPaths, []);
  const allPathsRef = useRef(bootstrapPaths.allPaths);
  const finalRepositoryPathsRef = useRef(bootstrapPaths.finalRepositoryPaths);
  const openFileRef = useRef<(path: string) => void>(() => {});
  const pending = useWorkflowFilesPendingState({
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
  const pathLists = useWorkflowRepositoryPathLists(catalog.generatedPaths, catalog.repositoryPaths, pendingChanges);
  allPathsRef.current = pathLists.allPaths;
  finalRepositoryPathsRef.current = pathLists.finalRepositoryPaths;
  const tabs = useWorkflowFilesTabState(catalog.generatedPaths[0] ?? null, pathLists.allPaths, catalog.generatedPaths);
  openFileRef.current = tabs.openFile;
  const selection = useWorkflowRepositorySelectedFileQuery(
    canvasId,
    tabs.selectedPath,
    catalog.repositoryPathSet,
    catalog.generatedFilesByPath,
  );

  useWorkflowFilesPublish({
    canManageRepositoryFiles,
    canPublishFiles:
      canManageRepositoryFiles && pendingChanges.length > 0 && !pathLists.commitPathError && !commitFiles.isPending,
    commitPathError: pathLists.commitPathError,
    headSha: catalog.headSha,
    branch: activeBranch ?? undefined,
    pendingChanges,
    setPendingChangesByPath: pending.setPendingChangesByPath,
    setLoadedContentByPath,
    discardAllChanges: pending.discardAllChanges,
    onHeaderActionsChange,
    commitFiles,
  });
  useWorkflowFilesEditorLifecycle({
    isEditing,
    resetPendingState: pending.resetPendingState,
    setIsDiffOpen,
    headerActionsSlotId,
    setHeaderActionsHost,
    selectedPath: tabs.selectedPath,
    selectedFileData: selection.selectedFileQuery.data,
    setLoadedContentByPath,
  });

  return buildWorkflowFilesEditorResult({
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
