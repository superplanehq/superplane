import { getApiErrorMessage } from "@/lib/errors";

import type { PendingFileChange, CanvasFile } from "../types";

type RepositoryQueryLike = {
  isLoading: boolean;
  error: unknown;
  data?: {
    status?: {
      state?: string;
      error?: string;
    };
  };
};

type FilesQueryLike = {
  isLoading: boolean;
  error: unknown;
};

type SelectedFileQueryLike = {
  isLoading: boolean;
  error: unknown;
};

export function getRepositoryFileListLoading(
  canUseRepository: boolean,
  repositoryQuery: RepositoryQueryLike,
  repositoryReady: boolean,
  filesQuery: FilesQueryLike,
): boolean {
  return (
    canUseRepository &&
    (repositoryQuery.isLoading ||
      (!repositoryReady && repositoryQuery.data?.status?.state === "STATE_PENDING") ||
      filesQuery.isLoading)
  );
}

export function getRepositoryFileListErrorMessage(
  repositoryQuery: RepositoryQueryLike,
  filesQuery: FilesQueryLike,
): string | undefined {
  if (filesQuery.error) {
    return getApiErrorMessage(filesQuery.error, "Failed to load files.");
  }

  if (repositoryQuery.error) {
    return getApiErrorMessage(repositoryQuery.error, "Failed to load repository.");
  }

  if (repositoryQuery.data?.status?.state === "STATE_ERROR") {
    return repositoryQuery.data.status.error || "Repository failed to provision.";
  }

  return undefined;
}

function resolveSelectedFileContent(
  selectedPath: string | null,
  selectedGeneratedFile: CanvasFile | undefined,
  selectedChange: PendingFileChange | undefined,
  loadedContentByPath: Record<string, string>,
): string {
  if (selectedGeneratedFile) {
    return selectedGeneratedFile.content;
  }

  if (selectedChange?.type === "added" || selectedChange?.type === "modified") {
    return selectedChange.content;
  }

  if (!selectedPath) {
    return "";
  }

  return loadedContentByPath[selectedPath] ?? "";
}

function isSelectedFileContentLoaded(
  selectedPath: string | null,
  selectedGeneratedFile: CanvasFile | undefined,
  selectedPathExistsInRepository: boolean,
  loadedContentByPath: Record<string, string>,
): boolean {
  if (selectedGeneratedFile || !selectedPath || !selectedPathExistsInRepository) {
    return true;
  }

  return loadedContentByPath[selectedPath] !== undefined;
}

export function getSelectedFileViewState({
  selectedPath,
  selectedGeneratedFile,
  selectedChange,
  loadedContentByPath,
  selectedPathExistsInRepository,
  selectedFileQuery,
  canManageRepositoryFiles,
}: {
  selectedPath: string | null;
  selectedGeneratedFile?: CanvasFile;
  selectedChange?: PendingFileChange;
  loadedContentByPath: Record<string, string>;
  selectedPathExistsInRepository: boolean;
  selectedFileQuery: SelectedFileQueryLike;
  canManageRepositoryFiles: boolean;
}) {
  const selectedIsDeleted = selectedChange?.type === "deleted";
  const selectedContent = resolveSelectedFileContent(
    selectedPath,
    selectedGeneratedFile,
    selectedChange,
    loadedContentByPath,
  );
  const selectedContentLoaded = isSelectedFileContentLoaded(
    selectedPath,
    selectedGeneratedFile,
    selectedPathExistsInRepository,
    loadedContentByPath,
  );
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

  return {
    selectedContent,
    selectedIsDeleted,
    editorLoading,
    editorErrorMessage,
    editorDisabled,
  };
}
