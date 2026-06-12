import { useEffect, useRef } from "react";

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
};

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
}: UseRepositoryFileStagingOptions) {
  const stageFiles = useStageRepositoryFiles(canvasId ?? "", versionId ?? "");
  const discardPaths = useDiscardRepositoryFilePaths(canvasId ?? "", versionId ?? "");
  const stageFilesRef = useRef(stageFiles);
  stageFilesRef.current = stageFiles;
  const discardPathsRef = useRef(discardPaths);
  discardPathsRef.current = discardPaths;
  const stagedPathsRef = useRef<Set<string>>(new Set());

  // Forget what we previously staged when the version changes; staging is keyed
  // per draft version on the server.
  useEffect(() => {
    stagedPathsRef.current = new Set();
  }, [versionId]);

  useEffect(() => {
    if (!enabled || !canvasId || !versionId) {
      return;
    }

    const timer = setTimeout(() => {
      void (async () => {
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
          await stageFilesRef.current.mutateAsync(operations);
        }
        if (pathsToDiscard.size > 0) {
          await discardPathsRef.current.mutateAsync([...pathsToDiscard]);
        }

        stagedPathsRef.current = stillStagedPaths;
      })();
    }, REPOSITORY_FILE_STAGING_DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [enabled, canvasId, versionId, pendingChanges]);
}
