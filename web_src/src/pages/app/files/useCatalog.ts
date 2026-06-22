import { useCanvasRepository, useCanvasRepositoryFile, useCanvasRepositoryFiles } from "@/hooks/useCanvasData";
import { useMemo } from "react";

import { buildFinalRepositoryPaths, buildRenderableTreePaths, getPathValidationError } from "./lib/files-paths";
import type { PendingFileChange, AppFile } from "./types";

export function useCatalog(canvasId: string | undefined, files: AppFile[]) {
  const canUseRepository = !!canvasId;
  const repositoryQuery = useCanvasRepository(canvasId ?? "", canUseRepository);
  const filesQuery = useCanvasRepositoryFiles(canvasId ?? "", canUseRepository);
  const generatedPaths = useMemo(() => files.map((file) => file.path), [files]);
  const generatedPathSet = useMemo(() => new Set(generatedPaths), [generatedPaths]);
  const generatedFilesByPath = useMemo(() => {
    const generatedFiles = new Map<string, AppFile>();
    for (const file of files) {
      generatedFiles.set(file.path, file);
    }
    return generatedFiles;
  }, [files]);
  const repositoryPaths = useMemo(
    () =>
      (filesQuery.data?.files || [])
        .map((file) => file.path)
        .filter((path): path is string => !!path && !generatedPathSet.has(path))
        .sort(),
    [filesQuery.data?.files, generatedPathSet],
  );
  const repositoryPathSet = useMemo(() => new Set(repositoryPaths), [repositoryPaths]);

  return {
    canUseRepository,
    repositoryQuery,
    repositoryReady: repositoryQuery.data?.status?.state === "STATE_READY",
    filesQuery,
    headSha: repositoryQuery.data?.status?.headSha,
    generatedPaths,
    generatedPathSet,
    generatedFilesByPath,
    repositoryPaths,
    repositoryPathSet,
  };
}

export function useRepositoryPathLists(
  generatedPaths: string[],
  repositoryPaths: string[],
  pendingChanges: PendingFileChange[],
  stagedRepositoryPaths: string[] = [],
) {
  const repositoryAndPendingPaths = useMemo(() => {
    return Array.from(
      new Set([
        ...repositoryPaths,
        ...stagedRepositoryPaths,
        ...pendingChanges.filter((change) => change.type === "added").map((change) => change.path),
      ]),
    ).sort();
  }, [pendingChanges, repositoryPaths, stagedRepositoryPaths]);
  const allPaths = useMemo(
    () => Array.from(new Set([...generatedPaths, ...repositoryAndPendingPaths])).sort(),
    [generatedPaths, repositoryAndPendingPaths],
  );
  const visiblePaths = useMemo(() => {
    return Array.from(
      new Set([...generatedPaths, ...buildRenderableTreePaths(repositoryAndPendingPaths, pendingChanges)]),
    ).sort();
  }, [generatedPaths, pendingChanges, repositoryAndPendingPaths]);
  const finalRepositoryPaths = useMemo(
    () => buildFinalRepositoryPaths(repositoryAndPendingPaths, pendingChanges),
    [pendingChanges, repositoryAndPendingPaths],
  );
  const commitPathError = useMemo(
    () => getPathValidationError([...generatedPaths, ...finalRepositoryPaths]),
    [finalRepositoryPaths, generatedPaths],
  );

  return { allPaths, visiblePaths, finalRepositoryPaths, commitPathError };
}

type UseRepositorySelectedFileQueryOptions = {
  canvasId?: string;
  selectedPath: string | null;
  repositoryPathSet: Set<string>;
  generatedFilesByPath: Map<string, AppFile>;
  versionId?: string;
  stage?: boolean;
};

export function useRepositorySelectedFileQuery({
  canvasId,
  selectedPath,
  repositoryPathSet,
  generatedFilesByPath,
  versionId,
  stage = false,
}: UseRepositorySelectedFileQueryOptions) {
  const selectedGeneratedFile = selectedPath ? generatedFilesByPath.get(selectedPath) : undefined;
  const selectedPathExistsInRepository = selectedPath ? repositoryPathSet.has(selectedPath) : false;
  const selectedFileQuery = useCanvasRepositoryFile(
    canvasId ?? "",
    selectedPath,
    !!selectedPath && selectedPathExistsInRepository && !selectedGeneratedFile,
    versionId,
    stage,
  );

  return { selectedGeneratedFile, selectedPathExistsInRepository, selectedFileQuery };
}
