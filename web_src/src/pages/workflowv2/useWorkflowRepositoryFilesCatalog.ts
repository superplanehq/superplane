import { useCanvasRepository, useCanvasRepositoryFile, useCanvasRepositoryFiles } from "@/hooks/useCanvasData";
import { useMemo } from "react";

import {
  buildFinalRepositoryPaths,
  buildRenderableTreePaths,
  getPathValidationError,
} from "./lib/workflow-files-paths";
import type { PendingFileChange, WorkflowFile } from "./workflow-files-types";

export function useWorkflowRepositoryFilesCatalog(
  canvasId: string | undefined,
  files: WorkflowFile[],
  branch?: string,
) {
  const canUseRepository = !!canvasId;
  const repositoryQuery = useCanvasRepository(canvasId ?? "", canUseRepository);
  const repositoryReady = repositoryQuery.data?.status?.state === "STATE_READY";
  const filesQuery = useCanvasRepositoryFiles(canvasId ?? "", canUseRepository && repositoryReady, branch);
  const generatedPaths = useMemo(() => files.map((file) => file.path), [files]);
  const generatedPathSet = useMemo(() => new Set(generatedPaths), [generatedPaths]);
  const generatedFilesByPath = useMemo(() => {
    const generatedFiles = new Map<string, WorkflowFile>();
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
    repositoryReady,
    filesQuery,
    headSha: repositoryQuery.data?.status?.headSha,
    generatedPaths,
    generatedPathSet,
    generatedFilesByPath,
    repositoryPaths,
    repositoryPathSet,
  };
}

export function useWorkflowRepositoryPathLists(
  generatedPaths: string[],
  repositoryPaths: string[],
  pendingChanges: PendingFileChange[],
) {
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

  return { allPaths, visiblePaths, finalRepositoryPaths, commitPathError };
}

export function useWorkflowRepositorySelectedFileQuery(
  canvasId: string | undefined,
  selectedPath: string | null,
  repositoryPathSet: Set<string>,
  generatedFilesByPath: Map<string, WorkflowFile>,
  branch?: string,
) {
  const selectedGeneratedFile = selectedPath ? generatedFilesByPath.get(selectedPath) : undefined;
  const selectedPathExistsInRepository = selectedPath ? repositoryPathSet.has(selectedPath) : false;
  const selectedFileQuery = useCanvasRepositoryFile(
    canvasId ?? "",
    selectedPath,
    !!selectedPath && selectedPathExistsInRepository && !selectedGeneratedFile,
    branch,
  );

  return { selectedGeneratedFile, selectedPathExistsInRepository, selectedFileQuery };
}
