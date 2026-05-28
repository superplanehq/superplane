import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useCanvasRepository,
  useCanvasRepositoryFile,
  useCanvasRepositoryFiles,
  useCommitCanvasRepositoryFiles,
} from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Editor } from "@monaco-editor/react";
import { MultiFileDiff, Virtualizer } from "@pierre/diffs/react";
import { FileTree as TreesFileTree, useFileTree } from "@pierre/trees/react";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { FilePlus2, GitCompareArrows, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { CSSProperties, ReactNode } from "react";
import type { FileContents } from "@pierre/diffs/react";
import type { ContextMenuItem, ContextMenuOpenContext } from "@pierre/trees";

export type WorkflowFile = {
  path: string;
  content: string;
  language?: string;
  loading?: boolean;
  errorMessage?: string;
};

const repositoryFileTreeStyle = {
  height: "100%",
  colorScheme: "light",
  "--trees-bg-override": "#ffffff",
  "--trees-bg-muted-override": "#f1f5f9",
  "--trees-border-color-override": "rgba(15, 23, 42, 0.15)",
  "--trees-fg-override": "#334155",
  "--trees-fg-muted-override": "#64748b",
  "--trees-focus-ring-color-override": "#0f172a",
  "--trees-selected-bg-override": "#e2e8f0",
  "--trees-selected-fg-override": "#020617",
  "--trees-padding-inline-override": "0px",
  "--trees-item-margin-x-override": "0px",
  "--trees-border-radius-override": "0px",
  "--trees-scrollbar-gutter-override": "0px",
  "--trees-action-lane-width-override": "0px",
} as CSSProperties;

const fileEditorOptions = {
  minimap: { enabled: false },
  fontSize: 13,
  lineNumbers: "on" as const,
  wordWrap: "on" as const,
  folding: true,
  automaticLayout: true,
  scrollBeyondLastLine: false,
  renderWhitespace: "boundary" as const,
  smoothScrolling: true,
  tabSize: 2,
  insertSpaces: true,
  cursorBlinking: "smooth" as const,
  contextmenu: true,
  selectOnLineNumbers: true,
  renderLineHighlight: "line" as const,
};

export type WorkflowFilesHeaderActionsState = {
  hasPendingChanges: boolean;
  publishDisabled: boolean;
  publishDisabledTooltip?: string;
  discardDisabled: boolean;
  publishPending: boolean;
  onPublish: () => void | Promise<void>;
  onDiscardAll: () => void;
};

interface WorkflowFilesOverlayLayerProps {
  isFilesMode: boolean;
  isEditing?: boolean;
  canvasId?: string;
  canWrite?: boolean;
  files: WorkflowFile[];
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: WorkflowFilesHeaderActionsState | null) => void;
}

type PendingFileChange =
  | {
      type: "added";
      path: string;
      content: string;
    }
  | {
      type: "modified";
      path: string;
      content: string;
    }
  | {
      type: "deleted";
      path: string;
    };

export function WorkflowFilesOverlayLayer({
  isFilesMode,
  isEditing = false,
  canvasId,
  canWrite = false,
  files,
  headerActionsSlotId,
  onHeaderActionsChange,
}: WorkflowFilesOverlayLayerProps) {
  if (!isFilesMode) return null;

  return (
    <CanvasFilesView
      canvasId={canvasId}
      isEditing={isEditing}
      canWrite={canWrite}
      files={files}
      headerActionsSlotId={headerActionsSlotId}
      onHeaderActionsChange={onHeaderActionsChange}
    />
  );
}

function CanvasFilesView({
  canvasId,
  isEditing,
  canWrite,
  files,
  headerActionsSlotId,
  onHeaderActionsChange,
}: {
  canvasId?: string;
  isEditing: boolean;
  canWrite: boolean;
  files: WorkflowFile[];
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: WorkflowFilesHeaderActionsState | null) => void;
}) {
  const leftOffset = useEffectiveLeftSidebarWidth();
  const canUseRepository = !!canvasId;
  const canManageRepositoryFiles = canWrite && canUseRepository && isEditing;
  const repositoryQuery = useCanvasRepository(canvasId ?? "", canUseRepository);
  const filesQuery = useCanvasRepositoryFiles(canvasId ?? "", canUseRepository);
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

  useEffect(() => {
    if (isEditing) return;

    setPendingChangesByPath({});
    setNewFilePath(null);
    setIsDiffOpen(false);
  }, [isEditing]);

  useEffect(() => {
    if (!headerActionsSlotId) {
      setHeaderActionsHost(null);
      return;
    }

    setHeaderActionsHost(document.getElementById(headerActionsSlotId));
  }, [headerActionsSlotId]);

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

    if (!selectedPath || allPathSet.has(selectedPath)) return;
    setSelectedPath(null);
  }, [allPaths, selectedPath]);

  useEffect(() => {
    const data = selectedFileQuery.data;
    const path = data?.path;
    if (!path || path !== selectedPath) return;

    const content = data.content || "";
    setLoadedContentByPath((current) => {
      if (current[path] === content) return current;
      return { ...current, [path]: content };
    });
  }, [selectedFileQuery.data?.content, selectedFileQuery.data?.path, selectedPath]);

  const openFile = useCallback((path: string) => {
    setOpenTabs((current) => (current.includes(path) ? current : [...current, path]));
    setSelectedPath(path);
  }, []);

  const closeTab = useCallback(
    (path: string) => {
      setOpenTabs((current) => {
        const nextTabs = current.filter((tabPath) => tabPath !== path);
        if (selectedPath !== path) return nextTabs;

        const closedIndex = current.indexOf(path);
        const nextSelectedPath = nextTabs[Math.min(closedIndex, nextTabs.length - 1)] || null;
        setSelectedPath(nextSelectedPath);
        return nextTabs;
      });
    },
    [selectedPath],
  );

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

        const originalContent = loadedContentByPath[selectedPath];
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
    [generatedPathSet, loadedContentByPath, selectedPath],
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

          return { path: change.path, content: encodeBase64(change.content) };
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

  return (
    <div
      className="absolute bottom-0 top-[5rem] z-10 grid min-h-0 grid-cols-[minmax(180px,260px)_minmax(0,1fr)] overflow-hidden bg-slate-50"
      style={{ left: leftOffset, right: 0 }}
      data-testid="workflow-files-overlay"
    >
      <aside className="flex min-h-0 flex-col border-r border-slate-950/15 bg-white">
        {canManageRepositoryFiles ? (
          <div className="flex h-7 shrink-0 items-center gap-1 border-b border-slate-950/10 px-2">
            <div className="ml-auto flex shrink-0 items-center">
              <EditorIconButton label="New file" onClick={startNewFile} className="size-6 hover:bg-transparent">
                <FilePlus2 className="h-3.5 w-3.5" />
              </EditorIconButton>
            </div>
          </div>
        ) : null}
        <FileList
          paths={visiblePaths}
          selectedPath={selectedPath}
          loading={canUseRepository && (filesQuery.isLoading || repositoryQuery.isLoading)}
          errorMessage={
            filesQuery.error
              ? getApiErrorMessage(filesQuery.error, "Failed to load files.")
              : repositoryQuery.error
                ? getApiErrorMessage(repositoryQuery.error, "Failed to load repository.")
                : undefined
          }
          canWrite={canManageRepositoryFiles}
          newFilePath={newFilePath}
          readOnlyPaths={generatedPathSet}
          onDelete={deleteFile}
          onNewFileCancel={cancelNewFile}
          onNewFileCommit={createNewFile}
          onNewFilePathChange={setNewFilePath}
          onSelect={openFile}
        />
      </aside>

      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="flex h-7 shrink-0 items-center border-b border-slate-950/15 bg-white">
          <div className="flex min-w-0 flex-1 items-start self-stretch overflow-x-auto overflow-y-hidden">
            {openTabs.map((path) => {
              const change = pendingChangesByPath[path];
              const active = selectedPath === path;

              return (
                <div
                  key={path}
                  className={
                    active
                      ? "flex h-7 min-w-0 max-w-56 items-center border-x border-t border-slate-950/15 bg-slate-50 text-xs text-slate-950"
                      : "flex h-7 min-w-0 max-w-56 items-center border-r border-slate-950/10 text-xs text-slate-600 hover:bg-slate-50 hover:text-slate-950"
                  }
                >
                  <button
                    type="button"
                    className="flex h-full min-w-0 flex-1 items-center gap-1.5 px-2.5 text-left"
                    onClick={() => openFile(path)}
                  >
                    {change ? <span className="size-1.5 shrink-0 rounded-full bg-sky-500" aria-hidden /> : null}
                    <span className="min-w-0 truncate">{path}</span>
                  </button>
                  <button
                    type="button"
                    aria-label={`Close ${path}`}
                    className="mr-1 flex size-4 shrink-0 items-center justify-center rounded text-slate-500 hover:bg-slate-200 hover:text-slate-950"
                    onClick={() => closeTab(path)}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              );
            })}
          </div>
        </div>

        <FileEditor
          path={selectedPath}
          content={selectedContent}
          deleted={selectedIsDeleted}
          language={selectedGeneratedFile?.language}
          loading={
            !!selectedGeneratedFile?.loading ||
            (!!selectedPath && selectedPathExistsInRepository && !selectedContentLoaded && selectedFileQuery.isLoading)
          }
          errorMessage={
            selectedGeneratedFile?.errorMessage ||
            (selectedFileQuery.error ? getApiErrorMessage(selectedFileQuery.error, "Failed to load file.") : undefined)
          }
          disabled={
            !!selectedGeneratedFile ||
            !canManageRepositoryFiles ||
            !selectedPath ||
            selectedIsDeleted ||
            !selectedContentLoaded
          }
          onChange={updateSelectedContent}
        />
      </main>

      <DiffDialog
        changes={pendingChanges}
        loadedContentByPath={loadedContentByPath}
        open={isDiffOpen}
        onOpenChange={setIsDiffOpen}
      />
      {canManageRepositoryFiles && headerActionsHost
        ? createPortal(
            <FilesHeaderActions hasPendingChanges={pendingChanges.length > 0} onDiffOpen={() => setIsDiffOpen(true)} />,
            headerActionsHost,
          )
        : null}
    </div>
  );
}

function FilesHeaderActions({ hasPendingChanges, onDiffOpen }: { hasPendingChanges: boolean; onDiffOpen: () => void }) {
  if (!hasPendingChanges) {
    return null;
  }

  return (
    <Button type="button" variant="outline" size="sm" onClick={onDiffOpen}>
      <GitCompareArrows className="h-4 w-4" />
      Diff
    </Button>
  );
}

function EditorIconButton({
  label,
  disabled,
  onClick,
  className,
  children,
}: {
  label: string;
  disabled?: boolean;
  onClick: () => void;
  className?: string;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={label}
          disabled={disabled}
          onClick={onClick}
          className={`text-slate-600 hover:bg-slate-100 hover:text-slate-950 ${className ?? ""}`}
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom">{label}</TooltipContent>
    </Tooltip>
  );
}

function DiffDialog({
  changes,
  loadedContentByPath,
  open,
  onOpenChange,
}: {
  changes: PendingFileChange[];
  loadedContentByPath: Record<string, string>;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const diffFiles = useMemo(
    () =>
      changes.map((change) => ({
        path: change.path,
        ...buildDiffFile(change, loadedContentByPath),
      })),
    [changes, loadedContentByPath],
  );
  const diffOptions = useMemo(
    () => ({
      theme: "pierre-light" as const,
      themeType: "light" as const,
      diffStyle: "split" as const,
      stickyHeader: true,
    }),
    [],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="90vw" className="grid grid-rows-[auto_minmax(0,1fr)] gap-4 p-0">
        <DialogHeader className="border-b border-slate-950/15 px-5 py-4">
          <DialogTitle>Diff</DialogTitle>
        </DialogHeader>
        <div className="min-h-0 overflow-hidden">
          {diffFiles.length === 0 ? (
            <div className="flex h-full items-center justify-center text-sm text-slate-500">No changes</div>
          ) : (
            <Virtualizer className="h-full overflow-auto" contentClassName="min-w-0">
              <div className="space-y-4 p-4">
                {diffFiles.map(({ path, oldFile, newFile }) => (
                  <MultiFileDiff
                    key={path}
                    oldFile={oldFile}
                    newFile={newFile}
                    options={diffOptions}
                    className="overflow-hidden rounded border border-slate-950/15 bg-white"
                  />
                ))}
              </div>
            </Virtualizer>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function FileList({
  paths,
  selectedPath,
  loading,
  errorMessage,
  canWrite,
  newFilePath,
  readOnlyPaths,
  onDelete,
  onNewFileCancel,
  onNewFileCommit,
  onNewFilePathChange,
  onSelect,
}: {
  paths: string[];
  selectedPath: string | null;
  loading: boolean;
  errorMessage?: string;
  canWrite: boolean;
  newFilePath: string | null;
  readOnlyPaths: Set<string>;
  onDelete: (path: string) => void;
  onNewFileCancel: () => void;
  onNewFileCommit: () => void;
  onNewFilePathChange: (path: string) => void;
  onSelect: (path: string) => void;
}) {
  if (loading) {
    return <div className="flex-1 p-4 text-sm text-slate-500">Loading files...</div>;
  }

  if (errorMessage) {
    return <div className="flex-1 p-4 text-sm text-red-600">{errorMessage}</div>;
  }

  if (paths.length === 0 && newFilePath === null) {
    return <div className="flex-1 p-4 text-sm text-slate-500">No files</div>;
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {newFilePath !== null ? (
        <NewFileTreeInput
          path={newFilePath}
          onCancel={onNewFileCancel}
          onChange={onNewFilePathChange}
          onCommit={onNewFileCommit}
        />
      ) : null}
      <RepositoryFileTree
        paths={paths}
        selectedPath={selectedPath}
        canWrite={canWrite}
        readOnlyPaths={readOnlyPaths}
        onDelete={onDelete}
        onSelect={onSelect}
      />
    </div>
  );
}

function NewFileTreeInput({
  path,
  onCancel,
  onChange,
  onCommit,
}: {
  path: string;
  onCancel: () => void;
  onChange: (path: string) => void;
  onCommit: () => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  return (
    <div className="flex h-7 shrink-0 items-center px-2 text-xs text-slate-700">
      <Input
        ref={inputRef}
        value={path}
        onBlur={onCommit}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            onCommit();
            return;
          }

          if (event.key === "Escape") {
            event.preventDefault();
            onCancel();
          }
        }}
        className="h-6 rounded border-slate-300 px-2 text-xs shadow-none focus-visible:ring-1"
      />
    </div>
  );
}

function RepositoryFileTree({
  paths,
  selectedPath,
  canWrite,
  readOnlyPaths,
  onDelete,
  onSelect,
}: {
  paths: string[];
  selectedPath: string | null;
  canWrite: boolean;
  readOnlyPaths: Set<string>;
  onDelete: (path: string) => void;
  onSelect: (path: string) => void;
}) {
  const filePathSetRef = useRef(new Set(paths));
  const readOnlyPathSetRef = useRef(readOnlyPaths);
  const onSelectRef = useRef(onSelect);
  const onDeleteRef = useRef(onDelete);

  useEffect(() => {
    filePathSetRef.current = new Set(paths);
  }, [paths]);

  useEffect(() => {
    readOnlyPathSetRef.current = readOnlyPaths;
  }, [readOnlyPaths]);

  useEffect(() => {
    onSelectRef.current = onSelect;
  }, [onSelect]);

  useEffect(() => {
    onDeleteRef.current = onDelete;
  }, [onDelete]);

  const { model } = useFileTree({
    paths,
    density: "compact",
    flattenEmptyDirectories: true,
    icons: { set: "minimal", colored: false },
    initialExpansion: "open",
    initialSelectedPaths: selectedPath ? [selectedPath] : [],
    unsafeCSS: `
      :host { color-scheme: light; }
      [data-file-tree-virtualized-scroll='true'] {
        scrollbar-gutter: auto;
        padding-inline-end: 0;
      }
    `,
    onSelectionChange: (selectedPaths) => {
      const path = selectedPaths.at(-1);
      if (!path || !filePathSetRef.current.has(path)) return;
      onSelectRef.current(path);
    },
  });

  useEffect(() => {
    model.resetPaths(paths);
  }, [model, paths]);

  useEffect(() => {
    const selectedPaths = model.getSelectedPaths();
    for (const path of selectedPaths) {
      if (path !== selectedPath) {
        model.getItem(path)?.deselect();
      }
    }

    if (!selectedPath || !filePathSetRef.current.has(selectedPath)) return;

    model.getItem(selectedPath)?.select();
    model.scrollToPath(selectedPath, { focus: false, offset: "nearest" });
  }, [model, selectedPath, paths]);

  return (
    <TreesFileTree
      model={model}
      className="h-full w-full bg-white text-xs text-slate-700"
      renderContextMenu={
        canWrite
          ? (item, context) => (
              <FileTreeContextMenu
                item={item}
                context={context}
                readOnlyPaths={readOnlyPathSetRef.current}
                onDelete={(path) => onDeleteRef.current(path)}
              />
            )
          : undefined
      }
      style={repositoryFileTreeStyle}
    />
  );
}

function FileTreeContextMenu({
  item,
  context,
  readOnlyPaths,
  onDelete,
}: {
  item: ContextMenuItem;
  context: ContextMenuOpenContext;
  readOnlyPaths: Set<string>;
  onDelete: (path: string) => void;
}) {
  const canDelete = item.kind === "file" && !readOnlyPaths.has(item.path);

  return (
    <div
      data-file-tree-context-menu-root="true"
      className="min-w-36 rounded border border-slate-950/15 bg-white p-1 text-xs text-slate-700 shadow-lg"
      role="menu"
    >
      <button
        type="button"
        role="menuitem"
        className="flex h-7 w-full items-center rounded px-2 text-left hover:bg-slate-100 disabled:cursor-not-allowed disabled:text-slate-400"
        disabled={!canDelete}
        onClick={() => {
          if (!canDelete) return;
          onDelete(item.path);
          context.close();
        }}
      >
        Delete
      </button>
    </div>
  );
}

function FileEditor({
  path,
  content,
  deleted,
  language,
  loading,
  errorMessage,
  disabled,
  onChange,
}: {
  path: string | null;
  content: string;
  deleted: boolean;
  language?: string;
  loading: boolean;
  errorMessage?: string;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  const suppressNextChangeRef = useRef(false);
  const previousPathRef = useRef<string | null>(path);

  useEffect(() => {
    if (previousPathRef.current === path) return;

    previousPathRef.current = path;
    suppressNextChangeRef.current = true;
  }, [path]);

  const handleChange = useCallback(
    (value: string | undefined) => {
      if (suppressNextChangeRef.current) {
        suppressNextChangeRef.current = false;
        return;
      }

      onChange(value ?? "");
    },
    [onChange],
  );

  if (!path) {
    return <div className="min-h-0 flex-1 bg-white" />;
  }

  if (loading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-500">Loading file...</div>
    );
  }

  if (errorMessage) {
    return <div className="p-4 text-sm text-red-600">{errorMessage}</div>;
  }

  if (deleted) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-500">
        File marked for deletion
      </div>
    );
  }

  return (
    <div className="min-h-0 flex-1 bg-white" data-testid="workflow-file-editor">
      <Editor
        key={path}
        height="100%"
        language={language ?? getMonacoLanguage(path)}
        value={content}
        theme="vs"
        onChange={handleChange}
        options={{
          ...fileEditorOptions,
          readOnly: disabled,
          domReadOnly: disabled,
        }}
      />
    </div>
  );
}

function getMonacoLanguage(path: string | null): string {
  if (!path) return "plaintext";

  const normalizedPath = path.toLowerCase();
  const extension = normalizedPath.split(".").pop();

  if (normalizedPath.endsWith("dockerfile") || normalizedPath.includes("/dockerfile")) return "dockerfile";
  if (normalizedPath.endsWith("makefile") || normalizedPath.includes("/makefile")) return "makefile";

  switch (extension) {
    case "css":
      return "css";
    case "go":
      return "go";
    case "html":
      return "html";
    case "js":
    case "mjs":
    case "cjs":
      return "javascript";
    case "json":
    case "jsonc":
      return "json";
    case "md":
    case "mdx":
      return "markdown";
    case "py":
      return "python";
    case "sh":
    case "bash":
    case "zsh":
      return "shell";
    case "ts":
      return "typescript";
    case "tsx":
      return "typescript";
    case "xml":
      return "xml";
    case "yaml":
    case "yml":
      return "yaml";
    default:
      return "plaintext";
  }
}

function nextUntitledPath(paths: Set<string>): string {
  if (!paths.has("untitled.txt")) return "untitled.txt";

  let index = 1;
  while (paths.has(`untitled-${index}.txt`)) {
    index += 1;
  }

  return `untitled-${index}.txt`;
}

function buildDiffFile(
  change: PendingFileChange,
  loadedContentByPath: Record<string, string>,
): { oldFile: FileContents; newFile: FileContents } {
  const previousContents = change.type === "added" ? "" : (loadedContentByPath[change.path] ?? "");
  const nextContents = change.type === "deleted" ? "" : change.content;

  return {
    oldFile: {
      name: change.path,
      contents: previousContents,
      cacheKey: `${change.path}:old:${previousContents}`,
    },
    newFile: {
      name: change.path,
      contents: nextContents,
      cacheKey: `${change.path}:new:${nextContents}`,
    },
  };
}

function buildRenderableTreePaths(repositoryPaths: string[], changes: PendingFileChange[]): string[] {
  const deletedPaths = new Set(changes.filter((change) => change.type === "deleted").map((change) => change.path));
  const paths = repositoryPaths.filter((path) => !deletedPaths.has(path));

  for (const change of changes) {
    if (change.type === "deleted") continue;
    if (getPathValidationError([...paths, change.path])) continue;
    paths.push(change.path);
  }

  return Array.from(new Set(paths)).sort();
}

function buildFinalRepositoryPaths(repositoryPaths: string[], changes: PendingFileChange[]): string[] {
  const paths = new Set(repositoryPaths);

  for (const change of changes) {
    if (change.type === "deleted") {
      paths.delete(change.path);
      continue;
    }

    paths.add(change.path);
  }

  return Array.from(paths).sort();
}

function getPathValidationError(paths: string[]): string | undefined {
  const filePaths = new Set<string>();
  const directoryPaths = new Set<string>();

  for (const path of paths) {
    const normalizedPath = normalizeFilePath(path);
    if (!normalizedPath || normalizedPath.endsWith("/")) {
      return "File path is required.";
    }

    const segments = normalizedPath.split("/");
    if (segments.some((segment) => segment === "" || segment === "." || segment === ".." || segment === ".git")) {
      return `Invalid file path "${path}".`;
    }

    for (let index = 1; index < segments.length; index += 1) {
      const directoryPath = segments.slice(0, index).join("/");
      if (filePaths.has(directoryPath)) {
        return `Path "${normalizedPath}" collides with existing file "${directoryPath}".`;
      }
      directoryPaths.add(directoryPath);
    }

    if (directoryPaths.has(normalizedPath)) {
      return `Path "${normalizedPath}" collides with an existing directory.`;
    }

    if (filePaths.has(normalizedPath)) {
      return `Path "${normalizedPath}" is already used.`;
    }

    filePaths.add(normalizedPath);
  }
}

function normalizeFilePath(path: string): string {
  return path.trim().replace(/\\/g, "/").replace(/^\/+/, "");
}

function encodeBase64(value: string): string {
  const bytes = new TextEncoder().encode(value);
  return bytesToBase64(bytes);
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  for (let index = 0; index < bytes.length; index += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(index, index + 0x8000));
  }
  return btoa(binary);
}
