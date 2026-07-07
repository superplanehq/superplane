import { useCallback, useEffect, useRef, type MutableRefObject } from "react";

import { useDiscardRepositoryFilePaths, useStageRepositoryFiles } from "@/hooks/useCanvasData";

import { matchesCommittedRepositoryFileContent } from "../lib/staging-content-match";
import { encodeRepositoryFileContent } from "./lib/repository-files";
import type { PendingFileChange } from "./types";

const REPOSITORY_FILE_STAGING_DEBOUNCE_MS = 500;

type UseRepositoryFileStagingOptions = {
  canvasId?: string;
  versionId?: string;
  enabled: boolean;
  pendingChanges: PendingFileChange[];
  onFlushReady?: (flush: (() => Promise<void>) | null) => void;
};

async function syncRepositoryFileStaging({
  canvasId,
  versionId,
  pendingChanges,
  stageFiles,
  discardPaths,
  stagedPathsRef,
  isLatestRun = () => true,
}: {
  canvasId: string;
  versionId: string;
  pendingChanges: PendingFileChange[];
  stageFiles: ReturnType<typeof useStageRepositoryFiles>;
  discardPaths: ReturnType<typeof useDiscardRepositoryFilePaths>;
  stagedPathsRef: MutableRefObject<Set<string>>;
  isLatestRun?: () => boolean;
}) {
  const currentPaths = new Set(pendingChanges.map((change) => change.path));
  const pathsToDiscard = new Set([...stagedPathsRef.current].filter((path) => !currentPaths.has(path)));
  const operations: Array<{ path: string; delete: true } | { path: string; content: string }> = [];
  const stillStagedPaths = new Set<string>();

  for (const change of pendingChanges) {
    if (change.type === "deleted") {
      operations.push({ path: change.path, delete: true });
      stillStagedPaths.add(change.path);
      continue;
    }

    const matchesCommitted = await matchesCommittedRepositoryFileContent(
      canvasId,
      versionId,
      change.path,
      change.content,
    );
    if (!isLatestRun()) {
      return;
    }
    if (matchesCommitted) {
      if (stagedPathsRef.current.has(change.path)) {
        pathsToDiscard.add(change.path);
      }
      continue;
    }

    operations.push({ path: change.path, content: encodeRepositoryFileContent(change.content) });
    stillStagedPaths.add(change.path);
  }

  if (operations.length > 0) {
    await stageFiles.mutateAsync(operations);
  }
  if (!isLatestRun()) {
    return;
  }
  if (pathsToDiscard.size > 0) {
    await discardPaths.mutateAsync([...pathsToDiscard]);
  }
  if (!isLatestRun()) {
    return;
  }

  stagedPathsRef.current = stillStagedPaths;
}

/**
 * Mirrors arbitrary (non-spec) Files tab edits into the draft version's staging
 * layer so the header switches to Reset/Commit and Commit can durably persist
 * them to git. Spec files (canvas.yaml/console.yaml) are handled separately by
 * useSpecFileAutosave. Edits are debounced; paths that are reverted back to the
 * committed content (no longer pending) are discarded from staging.
 */
export function useRepositoryFileStaging({
  canvasId,
  versionId,
  enabled,
  pendingChanges,
  onFlushReady,
}: UseRepositoryFileStagingOptions) {
  const stageFiles = useStageRepositoryFiles(canvasId ?? "");
  const discardPaths = useDiscardRepositoryFilePaths(canvasId ?? "");
  const stageFilesRef = useRef(stageFiles);
  stageFilesRef.current = stageFiles;
  const discardPathsRef = useRef(discardPaths);
  discardPathsRef.current = discardPaths;
  const stagedPathsRef = useRef<Set<string>>(new Set());
  const runGenerationRef = useRef(0);

  const flushPendingStaging = useCallback(async () => {
    if (!enabled || !canvasId || !versionId) {
      return;
    }

    runGenerationRef.current += 1;
    await syncRepositoryFileStaging({
      canvasId,
      versionId,
      pendingChanges,
      stageFiles: stageFilesRef.current,
      discardPaths: discardPathsRef.current,
      stagedPathsRef,
    });
  }, [canvasId, enabled, pendingChanges, versionId]);

  useEffect(() => {
    onFlushReady?.(enabled && canvasId && versionId ? flushPendingStaging : null);
    return () => onFlushReady?.(null);
  }, [canvasId, enabled, flushPendingStaging, onFlushReady, versionId]);

  // Forget what we previously staged when the version changes; staging is keyed
  // per draft version on the server.
  useEffect(() => {
    stagedPathsRef.current = new Set();
  }, [versionId]);

  useEffect(() => {
    if (!enabled || !canvasId || !versionId) {
      return;
    }

    const generation = ++runGenerationRef.current;
    const timer = setTimeout(() => {
      void (async () => {
        try {
          await syncRepositoryFileStaging({
            canvasId,
            versionId,
            pendingChanges,
            stageFiles: stageFilesRef.current,
            discardPaths: discardPathsRef.current,
            stagedPathsRef,
            isLatestRun: () => generation === runGenerationRef.current,
          });
        } catch {
          // Debounced staging failures surface on the next edit or explicit flush.
        }
      })();
    }, REPOSITORY_FILE_STAGING_DEBOUNCE_MS);

    return () => {
      clearTimeout(timer);
      if (generation === runGenerationRef.current) {
        runGenerationRef.current += 1;
      }
    };
  }, [enabled, canvasId, versionId, pendingChanges]);
}
