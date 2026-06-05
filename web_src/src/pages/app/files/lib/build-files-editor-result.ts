import {
  getRepositoryFileListErrorMessage,
  getRepositoryFileListLoading,
  getSelectedFileViewState,
} from "./files-view-state";
import type { PendingFileChange, CanvasFile } from "../types";

type EditorViewParams = {
  catalog: {
    canUseRepository: boolean;
    repositoryQuery: Parameters<typeof getRepositoryFileListLoading>[1];
    repositoryReady: boolean;
    filesQuery: Parameters<typeof getRepositoryFileListLoading>[3];
    generatedPathSet: Set<string>;
  };
  pathLists: { visiblePaths: string[] };
  tabs: {
    selectedPath: string | null;
    openTabs: string[];
    openFile: (path: string) => void;
    closeTab: (path: string) => void;
  };
  pending: {
    pendingChangesByPath: Record<string, PendingFileChange>;
    newFilePath: string | null;
    startNewFile: () => void;
    createNewFile: () => void;
    cancelNewFile: () => void;
    updateSelectedContent: (selectedPath: string | null, value: string) => void;
    deleteFile: (path: string) => void;
    setNewFilePath: (path: string) => void;
  };
  pendingChanges: PendingFileChange[];
  selection: {
    selectedGeneratedFile?: CanvasFile;
    selectedPathExistsInRepository: boolean;
    selectedFileQuery: Parameters<typeof getSelectedFileViewState>[0]["selectedFileQuery"];
  };
  loadedContentByPath: Record<string, string>;
  canManageRepositoryFiles: boolean;
  leftOffset: number;
  isDiffOpen: boolean;
  setIsDiffOpen: (open: boolean) => void;
  headerActionsHost: HTMLElement | null;
};

export function buildFilesEditorResult({
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
}: EditorViewParams) {
  const selectedChange = tabs.selectedPath ? pending.pendingChangesByPath[tabs.selectedPath] : undefined;
  const editorView = getSelectedFileViewState({
    selectedPath: tabs.selectedPath,
    selectedGeneratedFile: selection.selectedGeneratedFile,
    selectedChange,
    loadedContentByPath,
    selectedPathExistsInRepository: selection.selectedPathExistsInRepository,
    selectedFileQuery: selection.selectedFileQuery,
    canManageRepositoryFiles,
  });

  return {
    leftOffset,
    canManageRepositoryFiles,
    generatedPathSet: catalog.generatedPathSet,
    visiblePaths: pathLists.visiblePaths,
    selectedPath: tabs.selectedPath,
    openTabs: tabs.openTabs,
    pendingChanges,
    pendingChangesByPath: pending.pendingChangesByPath,
    newFilePath: pending.newFilePath,
    isDiffOpen,
    setIsDiffOpen,
    headerActionsHost,
    loadedContentByPath,
    selectedContent: editorView.selectedContent,
    selectedIsDeleted: editorView.selectedIsDeleted,
    selectedGeneratedFile: selection.selectedGeneratedFile,
    editorLoading: editorView.editorLoading,
    editorErrorMessage: editorView.editorErrorMessage,
    editorDisabled: editorView.editorDisabled,
    fileListLoading: getRepositoryFileListLoading(
      catalog.canUseRepository,
      catalog.repositoryQuery,
      catalog.repositoryReady,
      catalog.filesQuery,
    ),
    fileListErrorMessage: getRepositoryFileListErrorMessage(catalog.repositoryQuery, catalog.filesQuery),
    startNewFile: pending.startNewFile,
    createNewFile: pending.createNewFile,
    cancelNewFile: pending.cancelNewFile,
    updateSelectedContent: (value: string) => pending.updateSelectedContent(tabs.selectedPath, value),
    deleteFile: pending.deleteFile,
    openFile: tabs.openFile,
    closeTab: tabs.closeTab,
    setNewFilePath: pending.setNewFilePath,
  };
}
