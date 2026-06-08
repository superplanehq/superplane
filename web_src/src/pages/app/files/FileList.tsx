import { Input } from "@/components/ui/input";
import { FileTree as TreesFileTree, useFileTree } from "@pierre/trees/react";
import { useEffect, useRef } from "react";
import type { ContextMenuItem, ContextMenuOpenContext } from "@pierre/trees";

import { repositoryFileTreeStyle } from "./types";

export function FileList({
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
  const skipBlurCommitRef = useRef(false);

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  const handleBlur = () => {
    if (skipBlurCommitRef.current) {
      skipBlurCommitRef.current = false;
      return;
    }

    onCommit();
  };

  return (
    <div className="flex h-7 shrink-0 items-center px-2 text-xs text-slate-700">
      <Input
        ref={inputRef}
        value={path}
        onBlur={handleBlur}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            onCommit();
            return;
          }

          if (event.key === "Escape") {
            event.preventDefault();
            skipBlurCommitRef.current = true;
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
