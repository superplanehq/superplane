import { FilePlus2 } from "lucide-react";
import { lazy, Suspense } from "react";
import { createPortal } from "react-dom";

import { WorkflowFilesFileEditor } from "./WorkflowFilesFileEditor";
import { WorkflowFilesFileList } from "./WorkflowFilesFileList";
import { WorkflowFilesTabBar } from "./WorkflowFilesTabBar";
import { WorkflowFilesDiffHeaderAction, WorkflowFilesIconButton } from "./WorkflowFilesUi";
import { useWorkflowRepositoryFilesEditor } from "./useWorkflowRepositoryFilesEditor";
import type { CanvasBranchStagingState } from "./useCanvasBranchStaging";
import type { WorkflowFile, WorkflowFilesHeaderActionsState } from "./workflow-files-types";

const WorkflowFilesDiffDialog = lazy(() =>
  import("./WorkflowFilesDiffDialog").then((module) => ({ default: module.WorkflowFilesDiffDialog })),
);

export function WorkflowFilesCanvasView({
  canvasId,
  isEditing,
  canWrite,
  activeBranch,
  branchStaging,
  files,
  headerActionsSlotId,
  onHeaderActionsChange,
}: {
  canvasId?: string;
  isEditing: boolean;
  canWrite: boolean;
  activeBranch?: string | null;
  branchStaging?: CanvasBranchStagingState;
  files: WorkflowFile[];
  headerActionsSlotId?: string;
  onHeaderActionsChange?: (actions: WorkflowFilesHeaderActionsState | null) => void;
}) {
  const editor = useWorkflowRepositoryFilesEditor({
    canvasId,
    isEditing,
    canWrite,
    activeBranch,
    branchStaging,
    files,
    headerActionsSlotId,
    onHeaderActionsChange,
  });

  return (
    <div
      className="absolute bottom-0 top-[5rem] z-10 grid min-h-0 grid-cols-[minmax(180px,260px)_minmax(0,1fr)] overflow-hidden bg-slate-50"
      style={{ left: editor.leftOffset, right: 0 }}
      data-testid="workflow-files-overlay"
    >
      <aside className="flex min-h-0 flex-col border-r border-slate-950/15 bg-white">
        {editor.canManageRepositoryFiles ? (
          <div className="flex h-7 shrink-0 items-center gap-1 border-b border-slate-950/10 px-2">
            <div className="ml-auto flex shrink-0 items-center">
              <WorkflowFilesIconButton
                label="New file"
                onClick={editor.startNewFile}
                className="size-6 hover:bg-transparent"
              >
                <FilePlus2 className="h-3.5 w-3.5" />
              </WorkflowFilesIconButton>
            </div>
          </div>
        ) : null}
        <WorkflowFilesFileList
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
        <WorkflowFilesTabBar
          openTabs={editor.openTabs}
          selectedPath={editor.selectedPath}
          pendingChangesByPath={editor.pendingChangesByPath}
          onOpenFile={editor.openFile}
          onCloseTab={editor.closeTab}
        />

        <WorkflowFilesFileEditor
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
          <WorkflowFilesDiffDialog
            changes={editor.pendingChanges}
            loadedContentByPath={editor.loadedContentByPath}
            open={editor.isDiffOpen}
            onOpenChange={editor.setIsDiffOpen}
          />
        </Suspense>
      ) : null}
      {editor.canManageRepositoryFiles && editor.headerActionsHost
        ? createPortal(
            <WorkflowFilesDiffHeaderAction
              hasPendingChanges={editor.pendingChanges.length > 0}
              onDiffOpen={() => editor.setIsDiffOpen(true)}
            />,
            editor.headerActionsHost,
          )
        : null}
    </div>
  );
}
