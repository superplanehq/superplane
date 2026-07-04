import { FilePlus2 } from "lucide-react";
import { lazy, Suspense } from "react";
import { createPortal } from "react-dom";

import { FileEditor } from "./FileEditor";
import { FileList } from "./FileList";
import { TabBar } from "./TabBar";
import { DiffHeaderAction, IconButton } from "./FilesUi";
import { useEditor } from "./useEditor";
import type { AppFile } from "./types";

const DiffDialog = lazy(() => import("./DiffDialog").then((module) => ({ default: module.DiffDialog })));

export function FilesView({
  canvasId,
  versionId,
  isEditing,
  canWrite,
  files,
  headerActionsSlotId,
  stagingResetNonce,
  suspendRepositoryFileStaging,
  onSpecFileChange,
  onLocalFilesStagingChange,
  onFlushRepositoryFileStagingReady,
}: {
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
}) {
  const editor = useEditor({
    canvasId,
    versionId,
    isEditing,
    canWrite,
    files,
    headerActionsSlotId,
    stagingResetNonce,
    suspendRepositoryFileStaging,
    onSpecFileChange,
    onLocalFilesStagingChange,
    onFlushRepositoryFileStagingReady,
  });

  return (
    <div
      className="absolute bottom-0 top-[5rem] z-10 grid min-h-0 grid-cols-[minmax(180px,260px)_minmax(0,1fr)] overflow-hidden bg-slate-50"
      style={{ left: editor.leftOffset, right: 0 }}
      data-testid="files-overlay"
    >
      <aside className="flex min-h-0 flex-col border-r border-slate-950/15 bg-white">
        {editor.canManageRepositoryFiles ? (
          <div className="flex h-7 shrink-0 items-center gap-1 border-b border-slate-950/10 px-2">
            <div className="ml-auto flex shrink-0 items-center">
              <IconButton label="New file" onClick={editor.startNewFile} className="size-6 hover:bg-transparent">
                <FilePlus2 className="h-3.5 w-3.5" />
              </IconButton>
            </div>
          </div>
        ) : null}
        <FileList
          paths={editor.visiblePaths}
          selectedPath={editor.selectedPath}
          loading={editor.fileListLoading}
          errorMessage={editor.fileListErrorMessage}
          canWrite={editor.canManageRepositoryFiles}
          newFilePath={editor.newFilePath}
          readOnlyPaths={editor.generatedPathSet}
          onDelete={editor.deleteFile}
          onNewFileCancel={editor.cancelNewFile}
          onNewFileCommit={editor.createNewFile}
          onNewFilePathChange={editor.setNewFilePath}
          onSelect={editor.openFile}
        />
      </aside>

      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <TabBar
          openTabs={editor.openTabs}
          selectedPath={editor.selectedPath}
          pendingChangesByPath={editor.pendingChangesByPath}
          specDraftByPath={editor.specDraftByPath}
          onOpenFile={editor.openFile}
          onCloseTab={editor.closeTab}
        />

        <FileEditor
          path={editor.selectedPath}
          content={editor.selectedContent}
          deleted={editor.selectedIsDeleted}
          language={editor.selectedGeneratedFile?.language}
          loading={editor.editorLoading}
          errorMessage={editor.editorErrorMessage}
          disabled={editor.editorDisabled}
          onChange={editor.updateSelectedContent}
        />
      </main>

      {editor.isDiffOpen ? (
        <Suspense fallback={null}>
          <DiffDialog
            changes={editor.pendingChanges}
            committedContentByPath={editor.committedContentByPath}
            loadedContentByPath={editor.loadedContentByPath}
            stagedFileDiffs={editor.stagedFileDiffs}
            open={editor.isDiffOpen}
            onOpenChange={editor.setIsDiffOpen}
          />
        </Suspense>
      ) : null}
      {editor.canManageRepositoryFiles && editor.headerActionsHost
        ? createPortal(
            <DiffHeaderAction
              hasPendingChanges={editor.pendingChanges.length > 0 || editor.stagedDiffPaths.length > 0}
              onDiffOpen={() => editor.setIsDiffOpen(true)}
            />,
            editor.headerActionsHost,
          )
        : null}
    </div>
  );
}
