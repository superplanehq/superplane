import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { fetchRepositoryFileContentCached } from "@/hooks/useCanvasData";
import { normalizeSpecFileContentForDiff } from "./lib/spec-yaml-normalize";
import type { StagedFileDiff } from "./types";

type UseStagedFileDiffsOptions = {
  canvasId?: string;
  versionId?: string;
  paths: string[];
  enabled: boolean;
};

/**
 * Loads committed-vs-effective diffs for paths whose edits live in the draft's
 * staging layer rather than in the in-session pending changes. This covers the
 * virtual spec files (canvas.yaml / console.yaml) and, after a page refresh,
 * repository files whose edits were persisted to staging in a prior session.
 * The committed side is the version or live read (stage=false); the effective
 * side is the staging read (stage=true). Only paths that actually differ are
 * returned.
 */
export function useStagedFileDiffs({
  canvasId,
  versionId,
  paths,
  enabled,
}: UseStagedFileDiffsOptions): StagedFileDiff[] {
  const queryClient = useQueryClient();
  const [diffs, setDiffs] = useState<StagedFileDiff[]>([]);
  const pathsKey = paths.join("|");

  useEffect(() => {
    if (!enabled || !canvasId || !versionId || paths.length === 0) {
      setDiffs([]);
      return;
    }

    // Reads a single side of the diff through the React Query cache so identical
    // reads are reused/deduped. "Not found" is treated as empty content so added
    // files (no committed content) and staged deletions (no effective content)
    // render correctly instead of failing the whole batch.
    const readSide = async (path: string, stage: boolean): Promise<string> => {
      try {
        return await fetchRepositoryFileContentCached(queryClient, canvasId, path, versionId, stage);
      } catch {
        return "";
      }
    };

    let cancelled = false;
    void Promise.all(
      paths.map(async (path) => {
        const [committed, effective] = await Promise.all([readSide(path, false), readSide(path, true)]);
        return {
          path,
          committedContent: normalizeSpecFileContentForDiff(path, committed),
          effectiveContent: normalizeSpecFileContentForDiff(path, effective),
        };
      }),
    ).then((results) => {
      if (cancelled) {
        return;
      }
      setDiffs(results.filter((diff) => diff.committedContent !== diff.effectiveContent));
    });

    return () => {
      cancelled = true;
    };
    // pathsKey captures the path list so we refetch when the set of changed
    // paths changes without depending on a new array identity each render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, canvasId, versionId, pathsKey]);

  return diffs;
}
