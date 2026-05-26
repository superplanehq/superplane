import { Editor } from "@monaco-editor/react";
import { FileTree as TreesFileTree, useFileTree } from "@pierre/trees/react";
import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";

export type WorkflowFile = {
  path: string;
  content: string;
  language?: string;
  loading?: boolean;
  errorMessage?: string;
};

interface WorkflowFilesOverlayLayerProps {
  isFilesMode: boolean;
  files: WorkflowFile[];
}

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

export function WorkflowFilesOverlayLayer({ isFilesMode, files }: WorkflowFilesOverlayLayerProps) {
  if (!isFilesMode) return null;

  return <CanvasYamlFilesView files={files} />;
}

function CanvasYamlFilesView({ files }: { files: WorkflowFile[] }) {
  const filePaths = useMemo(() => files.map((file) => file.path), [files]);
  const [selectedPath, setSelectedPath] = useState<string | null>(() => filePaths[0] ?? null);
  const [openTabs, setOpenTabs] = useState<string[]>(() => (filePaths[0] ? [filePaths[0]] : []));
  const selectedFile = files.find((file) => file.path === selectedPath) ?? null;

  useEffect(() => {
    const filePathSet = new Set(filePaths);

    setOpenTabs((current) => {
      const nextTabs = current.filter((path) => filePathSet.has(path));
      return nextTabs.length === current.length ? current : nextTabs;
    });
    setSelectedPath((current) => (current && filePathSet.has(current) ? current : null));
  }, [filePaths]);

  const openFile = (path: string) => {
    setSelectedPath(path);
    setOpenTabs((current) => (current.includes(path) ? current : [...current, path]));
  };

  const closeTab = (path: string) => {
    setOpenTabs((current) => {
      const nextTabs = current.filter((tabPath) => tabPath !== path);
      if (selectedPath !== path) return nextTabs;

      const closedIndex = current.indexOf(path);
      setSelectedPath(nextTabs[Math.min(closedIndex, nextTabs.length - 1)] ?? null);
      return nextTabs;
    });
  };

  return (
    <div
      className="absolute inset-x-0 bottom-0 top-[5rem] z-10 grid min-h-0 grid-cols-[minmax(180px,260px)_minmax(0,1fr)] overflow-hidden bg-slate-50"
      data-testid="workflow-files-overlay"
    >
      <aside className="flex min-h-0 flex-col border-r border-slate-950/15 bg-white">
        <div className="flex h-7 shrink-0 items-center border-b border-slate-950/10 px-2 text-xs font-medium text-slate-500">
          Files
        </div>
        <FileList paths={filePaths} selectedPath={selectedPath} onSelect={openFile} />
      </aside>

      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="flex h-7 shrink-0 items-center border-b border-slate-950/15 bg-white">
          <div className="flex min-w-0 flex-1 items-start self-stretch overflow-x-auto overflow-y-hidden">
            {openTabs.map((path) => (
              <FileTab key={path} path={path} active={selectedPath === path} onSelect={openFile} onClose={closeTab} />
            ))}
          </div>
        </div>

        <FileEditor file={selectedFile} />
      </main>
    </div>
  );
}

function FileList({
  paths,
  selectedPath,
  onSelect,
}: {
  paths: string[];
  selectedPath: string | null;
  onSelect: (path: string) => void;
}) {
  if (paths.length === 0) {
    return <div className="flex-1 p-4 text-sm text-slate-500">No files</div>;
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <RepositoryFileTree paths={paths} selectedPath={selectedPath} onSelect={onSelect} />
    </div>
  );
}

function RepositoryFileTree({
  paths,
  selectedPath,
  onSelect,
}: {
  paths: string[];
  selectedPath: string | null;
  onSelect: (path: string) => void;
}) {
  const filePathSetRef = useRef(new Set(paths));
  const onSelectRef = useRef(onSelect);

  useEffect(() => {
    filePathSetRef.current = new Set(paths);
  }, [paths]);

  useEffect(() => {
    onSelectRef.current = onSelect;
  }, [onSelect]);

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
      style={repositoryFileTreeStyle}
    />
  );
}

function FileTab({
  path,
  active,
  onSelect,
  onClose,
}: {
  path: string;
  active: boolean;
  onSelect: (path: string) => void;
  onClose: (path: string) => void;
}) {
  return (
    <div
      className={
        active
          ? "flex h-7 min-w-0 max-w-56 items-center border-x border-t border-slate-950/15 bg-slate-50 text-xs text-slate-950"
          : "flex h-7 min-w-0 max-w-56 items-center border-r border-slate-950/10 text-xs text-slate-600 hover:bg-slate-50 hover:text-slate-950"
      }
    >
      <button
        type="button"
        className="flex h-full min-w-0 flex-1 items-center px-2.5 text-left"
        onClick={() => onSelect(path)}
      >
        <span className="min-w-0 truncate">{path}</span>
      </button>
      <button
        type="button"
        aria-label={`Close ${path}`}
        className="mr-1 flex size-4 shrink-0 items-center justify-center rounded text-slate-500 hover:bg-slate-200 hover:text-slate-950"
        onClick={() => onClose(path)}
      >
        <span aria-hidden>x</span>
      </button>
    </div>
  );
}

function FileEditor({ file }: { file: WorkflowFile | null }) {
  if (!file) {
    return <div className="min-h-0 flex-1 bg-white" />;
  }

  if (file.loading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-500">Loading file...</div>
    );
  }

  if (file.errorMessage) {
    return <div className="p-4 text-sm text-red-600">{file.errorMessage}</div>;
  }

  return (
    <div className="min-h-0 flex-1 bg-white" data-testid="workflow-file-editor">
      <Editor
        height="100%"
        language={file.language ?? getMonacoLanguage(file.path)}
        value={file.content}
        theme="vs"
        options={{
          ...fileEditorOptions,
          readOnly: true,
          domReadOnly: true,
        }}
      />
    </div>
  );
}

function getMonacoLanguage(path: string): string {
  const extension = path.toLowerCase().split(".").pop();

  if (extension === "yaml" || extension === "yml") return "yaml";

  return "plaintext";
}
