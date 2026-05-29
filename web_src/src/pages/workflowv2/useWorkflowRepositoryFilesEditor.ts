import {
  useCanvasRepository,
  useCanvasRepositoryFile,
  useCanvasRepositoryFiles,
  useCommitCanvasRepositoryFiles,
} from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { encodeRepositoryFileContent } from "./lib/canvas-repository-files";
import {
  buildFinalRepositoryPaths,
  buildRenderableTreePaths,
  getPathValidationError,
  nextUntitledPath,
  normalizeFilePath,
} from "./lib/workflow-files-paths";
import type { PendingFileChange, WorkflowFile, WorkflowFilesHeaderActionsState } from "./workflow-files-types";

type UseWorkflowRepositoryFilesEditorOptions = {
  canvasId?: string;
  isEditing: boolean;
  canWrite: boolean;
  files: WorkflowFile[];
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: WorkflowFilesHeaderActionsState | null) => void;
};

export function useWorkflowRepositoryFilesEditor({
  canvasId,
  isEditing,
  canWrite,
  files,
  headerActionsSlotId,
  onHeaderActionsChange,
}: UseWorkflowRepositoryFilesEditorOptions) {
  const leftOffset = useEffectiveLeftSidebarWidth();
  const canUseRepository = !!canvasId;
  const canManageRepositoryFiles = canWrite && canUseRepository && isEditing;
  const repositoryQuery = useCanvasRepository(canvasId ?? "", canUseRepository);
  const repositoryReady = repositoryQuery.data?.status?.state === "STATE_READY";
  const filesQuery = useCanvasRepositoryFiles(canvasId ?? "", canUseRepository && repositoryReady);
  const commitFiles = useCommitCanvasRepositoryFiles(canvasId ?? "");
  const generatedPaths = useMemo(() => files.map((file) => file.path), [files]);
  const generatedPathSet = useMemo(() => new Set(generatedPaths), [generatedPaths]);
  const generatedFilesByPath = useMemo(() => {
    const generatedFiles = new Map<string, WorkflowFile>();
    for (const file of files) {
      generatedFiles.set(file.path, file);
    }
    return generatedFiles;
  }, [files]);
  const initialPath = generatedPaths[0] ?? null;
  const hasAutoOpenedInitialFileRef = useRef(Boolean(initialPath));
  const headSha = repositoryQuery.data?.status?.headSha;
  const repositoryPaths = useMemo(
    () =>
      (filesQuery.data?.files || [])
        .map((file) => file.path)
        .filter((path): path is string => !!path && !generatedPathSet.has(path))
        .sort(),
    [filesQuery.data?.files, generatedPathSet],
  );
  const repositoryPathSet = useMemo(() => new Set(repositoryPaths), [repositoryPaths]);
  const [loadedContentByPath, setLoadedContentByPath] = useState<Record<string, string>>({});
  const [pendingChangesByPath, setPendingChangesByPath] = useState<Record<string, PendingFileChange>>({});
  const [openTabs, setOpenTabs] = useState<string[]>(() => (initialPath ? [initialPath] : []));
  const [selectedPath, setSelectedPath] = useState<string | null>(() => initialPath);
  const [newFilePath, setNewFilePath] = useState<string | null>(null);
  const [isDiffOpen, setIsDiffOpen] = useState(false);
  const [headerActionsHost, setHeaderActionsHost] = useState<HTMLElement | null>(null);
  const loadedContentByPathRef = useRef(loadedContentByPath);
  loadedContentByPathRef.current = loadedContentByPath;
  const selectedGeneratedFile = selectedPath ? generatedFilesByPath.get(selectedPath) : undefined;
  const selectedPathExistsInRepository = selectedPath ? repositoryPathSet.has(selectedPath) : false;
  const selectedFileQuery = useCanvasRepositoryFile(
    canvasId ?? "",
    selectedPath,
    !!selectedPath && selectedPathExistsInRepository && !selectedGeneratedFile,
  );
  const pendingChanges = useMemo(
    () => Object.values(pendingChangesByPath).sort((left, right) => left.path.localeCompare(right.path)),
    [pendingChangesByPath],
  );
  const repositoryAndPendingPaths = useMemo(() => {
    return Array.from(
      new Set([
        ...repositoryPaths,
        ...pendingChanges.filter((change) => change.type === "added").map((change) => change.path),
      ]),
    ).sort();
  }, [pendingChanges, repositoryPaths]);
  const allPaths = useMemo(
    () => Array.from(new Set([...generatedPaths, ...repositoryAndPendingPaths])).sort(),
    [generatedPaths, repositoryAndPendingPaths],
  );
  const visiblePaths = useMemo(() => {
    return Array.from(
      new Set([...generatedPaths, ...buildRenderableTreePaths(repositoryPaths, pendingChanges)]),
    ).sort();
  }, [generatedPaths, pendingChanges, repositoryPaths]);
  const finalRepositoryPaths = useMemo(
    () => buildFinalRepositoryPaths(repositoryPaths, pendingChanges),
    [pendingChanges, repositoryPaths],
  );
  const commitPathError = useMemo(
    () => getPathValidationError([...generatedPaths, ...finalRepositoryPaths]),
    [finalRepositoryPaths, generatedPaths],
  );
  const selectedChange = selectedPath ? pendingChangesByPath[selectedPath] : undefined;
  const selectedIsDeleted = selectedChange?.type === "deleted";
  const selectedContent = selectedGeneratedFile
    ? selectedGeneratedFile.content
    : selectedChange?.type === "added" || selectedChange?.type === "modified"
      ? selectedChange.content
      : selectedPath
        ? (loadedContentByPath[selectedPath] ?? "")
        : "";
  const selectedContentLoaded =
    !!selectedGeneratedFile ||
    !selectedPath ||
    !selectedPathExistsInRepository ||
    loadedContentByPath[selectedPath] !== undefined;
  const canPublishFiles =
    canManageRepositoryFiles && pendingChanges.length > 0 && !commitPathError && !commitFiles.isPending;

  const fileListLoading =
    canUseRepository &&
    (repositoryQuery.isLoading ||
      (!repositoryReady && repositoryQuery.data?.status?.state === "STATE_PENDING") ||
      filesQuery.isLoading);
  const fileListErrorMessage = filesQuery.error
    ? getApiErrorMessage(filesQuery.error, "Failed to load files.")
    : repositoryQuery.error
      ? getApiErrorMessage(repositoryQuery.error, "Failed to load repository.")
      : repositoryQuery.data?.status?.state === "STATE_ERROR"
        ? repositoryQuery.data.status.error || "Repository failed to provision."
        : undefined;
  const editorLoading =
    !!selectedGeneratedFile?.loading ||
    (!!selectedPath && selectedPathExistsInRepository && !selectedContentLoaded && selectedFileQuery.isLoading);
  const editorErrorMessage =
    selectedGeneratedFile?.errorMessage ||
    (selectedFileQuery.error ? getApiErrorMessage(selectedFileQuery.error, "Failed to load file.") : undefined);
  const editorDisabled =
    !!selectedGeneratedFile ||
    !canManageRepositoryFiles ||
    !selectedPath ||
    selectedIsDeleted ||
    !selectedContentLoaded;

  useEffect(() => {
    if (isEditing) return;

    setPendingChangesByPath({});
    setNewFilePath(null);
    setIsDiffOpen(false);
  }, [isEditing]);

  useEffect(() => {
    if (!headerActionsSlotId || !isEditing) {
      setHeaderActionsHost(null);
      return;
    }

    setHeaderActionsHost(document.getElementById(headerActionsSlotId));
  }, [headerActionsSlotId, isEditing]);

  useEffect(() => {
    if (hasAutoOpenedInitialFileRef.current) return;

    const nextInitialPath = generatedPaths[0];
    if (!nextInitialPath) return;

    hasAutoOpenedInitialFileRef.current = true;
    setOpenTabs([nextInitialPath]);
    setSelectedPath(nextInitialPath);
  }, [generatedPaths]);

  useEffect(() => {
    const allPathSet = new Set(allPaths);
    setOpenTabs((current) => current.filter((path) => allPathSet.has(path)));
    setSelectedPath((current) => (current && allPathSet.has(current) ? current : null));
  }, [allPaths]);

  useEffect(() => {
    const data = selectedFileQuery.data;
    const path = data?.path;
    if (!path || path !== selectedPath) return;

    const content = data.content || "";
    setLoadedContentByPath((current) => {
      if (current[path] === content) return current;
      return { ...current, [path]: content };
    });
  }, [selectedFileQuery.data, selectedPath]);

  const openFile = useCallback((path: string) => {
    setOpenTabs((current) => (current.includes(path) ? current : [...current, path]));
    setSelectedPath(path);
  }, []);

  const closeTab = useCallback((path: string) => {
    setOpenTabs((current) => {
      const nextTabs = current.filter((tabPath) => tabPath !== path);
      setSelectedPath((selected) => {
        if (selected !== path) return selected;
        const closedIndex = current.indexOf(path);
        return nextTabs[Math.min(closedIndex, nextTabs.length - 1)] ?? null;
      });
      return nextTabs;
    });
  }, []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (!event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || event.key.toLowerCase() !== "w") return;
      if (!selectedPath) return;

      event.preventDefault();
      closeTab(selectedPath);
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [closeTab, selectedPath]);

  const startNewFile = useCallback(() => {
    const path = nextUntitledPath(new Set(allPaths));
    setNewFilePath(path);
  }, [allPaths]);

  const createNewFile = useCallback(() => {
    if (newFilePath === null) return;

    const path = normalizeFilePath(newFilePath);
    if (!path) {
      setNewFilePath(null);
      return;
    }

    const errorMessage = getPathValidationError([...generatedPaths, ...finalRepositoryPaths, path]);
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
  }, [finalRepositoryPaths, generatedPaths, newFilePath, openFile]);

  const cancelNewFile = useCallback(() => {
    setNewFilePath(null);
  }, []);

  const updateSelectedContent = useCallback(
    (value: string) => {
      if (!selectedPath || generatedPathSet.has(selectedPath)) return;

      setPendingChangesByPath((current) => {
        const currentChange = current[selectedPath];
        if (currentChange?.type === "added") {
          return { ...current, [selectedPath]: { ...currentChange, content: value } };
        }

        const originalContent = loadedContentByPathRef.current[selectedPath];
        if (originalContent === undefined) return current;

        if (value === originalContent) {
          const { [selectedPath]: _removed, ...remaining } = current;
          return remaining;
        }

        return {
          ...current,
          [selectedPath]: { type: "modified", path: selectedPath, content: value },
        };
      });
    },
    [generatedPathSet, selectedPath],
  );

  const deleteFile = useCallback(
    (path: string) => {
      if (generatedPathSet.has(path)) return;

      setPendingChangesByPath((current) => {
        const currentChange = current[path];
        if (currentChange?.type === "added") {
          const { [path]: _removed, ...remaining } = current;
          return remaining;
        }

        return {
          ...current,
          [path]: { type: "deleted", path },
        };
      });
    },
    [generatedPathSet],
  );

  const discardAllChanges = useCallback(() => {
    setPendingChangesByPath({});
  }, []);

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
        operations: pendingChanges.map((change) => {
          if (change.type === "deleted") {
            return { path: change.path, delete: true };
          }

          return { path: change.path, content: encodeRepositoryFileContent(change.content) };
        }),
      });

      showSuccessToast("Files published.");
      setPendingChangesByPath({});
      setLoadedContentByPath((current) => {
        const next = { ...current };
        for (const change of pendingChanges) {
          if (change.type === "deleted") {
            delete next[change.path];
            continue;
          }

          next[change.path] = change.content;
        }

        return next;
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to publish files."));
    }
  }, [commitFiles, commitPathError, headSha, pendingChanges]);

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

  return {
    leftOffset,
    canManageRepositoryFiles,
    generatedPathSet,
    visiblePaths,
    selectedPath,
    openTabs,
    pendingChanges,
    pendingChangesByPath,
    newFilePath,
    isDiffOpen,
    setIsDiffOpen,
    headerActionsHost,
    loadedContentByPath,
    selectedContent,
    selectedIsDeleted,
    selectedGeneratedFile,
    editorLoading,
    editorErrorMessage,
    editorDisabled,
    fileListLoading,
    fileListErrorMessage,
    startNewFile,
    createNewFile,
    cancelNewFile,
    updateSelectedContent,
    deleteFile,
    openFile,
    closeTab,
    setNewFilePath,
  };
}
