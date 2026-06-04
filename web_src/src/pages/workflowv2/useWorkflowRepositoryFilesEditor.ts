import { useCommitCanvasRepositoryFiles } from "@/hooks/useCanvasData";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { useMemo, useRef, useState, useEffect } from "react";

import { fetchCanvasRepositoryFileContent } from "./lib/canvas-repository-files";

import { buildWorkflowFilesEditorResult } from "./lib/build-workflow-files-editor-result";
import { useWorkflowFilesEditorLifecycle } from "./useWorkflowFilesEditorLifecycle";
import { useWorkflowFilesStagingState } from "./useWorkflowFilesStagingState";
import { useWorkflowFilesPublish } from "./useWorkflowFilesPublish";
import { useWorkflowFilesTabState } from "./useWorkflowFilesTabState";
import {
  useWorkflowRepositoryFilesCatalog,
  useWorkflowRepositoryPathLists,
  useWorkflowRepositorySelectedFileQuery,
} from "./useWorkflowRepositoryFilesCatalog";
import type { CanvasBranchStagingState } from "./useCanvasBranchStaging";
import type { WorkflowFile, WorkflowFilesHeaderActionsState } from "./workflow-files-types";

type UseWorkflowRepositoryFilesEditorOptions = {
  canvasId?: string;
  isEditing: boolean;
  canWrite: boolean;
  files: WorkflowFile[];
  activeBranch?: string | null;
  branchTipSha?: string;
  branchStaging?: CanvasBranchStagingState;
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: WorkflowFilesHeaderActionsState | null) => void;
};

export function useWorkflowRepositoryFilesEditor({
  canvasId,
  isEditing,
  canWrite,
  files,
  activeBranch,
  branchTipSha,
  branchStaging,
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
  const useBranchStaging = !!branchStaging && !!activeBranch;

  const bootstrapPaths = useWorkflowRepositoryPathLists(catalog.generatedPaths, catalog.repositoryPaths, []);
  const allPathsRef = useRef(bootstrapPaths.allPaths);
  const finalRepositoryPathsRef = useRef(bootstrapPaths.finalRepositoryPaths);
  const openFileRef = useRef<(path: string) => void>(() => {});
  const pending = useWorkflowFilesStagingState({
    branchStaging: useBranchStaging ? branchStaging : undefined,
    generatedPathSet: catalog.generatedPathSet,
    generatedPaths: catalog.generatedPaths,
    repositoryPathSet: catalog.repositoryPathSet,
    finalRepositoryPathsRef,
    allPathsRef,
    loadedContentByPath,
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
    useBranchStaging ? (activeBranch ?? undefined) : undefined,
  );
  const branchHeadSha = useBranchStaging ? branchTipSha : catalog.headSha;

  useEffect(() => {
    if (!useBranchStaging || !canvasId || !activeBranch || !branchHeadSha) {
      return;
    }

    const paths = Object.keys(loadedContentByPathRef.current).filter((path) => catalog.repositoryPathSet.has(path));
    if (paths.length === 0) {
      return;
    }

    let cancelled = false;

    void Promise.all(
      paths.map(async (path) => {
        const content = await fetchCanvasRepositoryFileContent(canvasId, path, activeBranch).catch(() => "");
        return [path, content] as const;
      }),
    ).then((results) => {
      if (cancelled) {
        return;
      }

      setLoadedContentByPath((current) => {
        const next = { ...current };
        let changed = false;

        for (const [path, content] of results) {
          if (next[path] === content) {
            continue;
          }

          next[path] = content;
          changed = true;
        }

        return changed ? next : current;
      });
    });

    return () => {
      cancelled = true;
    };
  }, [activeBranch, branchHeadSha, canvasId, catalog.repositoryPathSet, useBranchStaging]);

  useEffect(() => {
    if (!useBranchStaging || !canvasId || !activeBranch || !branchStaging?.stagingRecord) {
      return;
    }

    const stagedPaths = [
      ...Object.keys(branchStaging.stagingRecord.files),
      ...(branchStaging.stagingRecord.deletedPaths ?? []),
    ].filter((path) => catalog.repositoryPathSet.has(path) && loadedContentByPath[path] === undefined);

    if (stagedPaths.length === 0) {
      return;
    }

    let cancelled = false;

    void Promise.all(
      stagedPaths.map(async (path) => {
        const content = await fetchCanvasRepositoryFileContent(canvasId, path, activeBranch).catch(() => "");
        return [path, content] as const;
      }),
    ).then((results) => {
      if (cancelled) {
        return;
      }

      setLoadedContentByPath((current) => {
        const next = { ...current };
        let changed = false;

        for (const [path, content] of results) {
          if (next[path] !== undefined) {
            continue;
          }

          next[path] = content;
          changed = true;
        }

        return changed ? next : current;
      });
    });

    return () => {
      cancelled = true;
    };
  }, [
    activeBranch,
    branchStaging?.stagingRecord,
    canvasId,
    catalog.repositoryPathSet,
    loadedContentByPath,
    useBranchStaging,
  ]);

  useWorkflowFilesPublish({
    canManageRepositoryFiles: canManageRepositoryFiles && !useBranchStaging,
    canPublishFiles:
      canManageRepositoryFiles &&
      !useBranchStaging &&
      pendingChanges.length > 0 &&
      !pathLists.commitPathError &&
      !commitFiles.isPending,
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
    branchBaselineFiles: useBranchStaging ? branchStaging?.baselineFiles : undefined,
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
