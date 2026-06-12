import { useEffect, useState } from "react";

import { fetchRepositorySpecFileContent } from "../lib/repository-spec-files";
import { normalizeSpecFileContentForDiff } from "./lib/spec-yaml-normalize";
import type { StagedFileDiff } from "./types";

type UseStagedFileDiffsOptions = {
  canvasId?: string;
  versionId?: string;
  paths: string[];
  enabled: boolean;
};

// Reads a single side of the diff, treating "not found" as empty content so
// added files (no committed content) and staged deletions (no effective
// content) render correctly instead of failing the whole batch.
async function readSide(canvasId: string, path: string, versionId: string, stage: boolean): Promise<string> {
  try {
    return await fetchRepositorySpecFileContent(canvasId, path, versionId, stage);
  } catch {
    return "";
  }
}

/**
 * Loads committed-vs-effective diffs for paths whose edits live in the draft's
 * staging layer rather than in the in-session pending changes. This covers the
 * virtual spec files (canvas.yaml / console.yaml) and, after a page refresh,
 * repository files whose edits were persisted to staging in a prior session.
 * The committed side is the stage=false server read; the effective side
 * overlays the staged edits (stage=true). Only paths that actually differ are
 * returned.
 */
export function useStagedFileDiffs({
  canvasId,
  versionId,
  paths,
  enabled,
}: UseStagedFileDiffsOptions): StagedFileDiff[] {
  const [diffs, setDiffs] = useState<StagedFileDiff[]>([]);
  const pathsKey = paths.join("|");

  useEffect(() => {
    if (!enabled || !canvasId || !versionId || paths.length === 0) {
      setDiffs([]);
      return;
    }

    let cancelled = false;
    void Promise.all(
      paths.map(async (path) => {
        const [committed, effective] = await Promise.all([
          readSide(canvasId, path, versionId, false),
          readSide(canvasId, path, versionId, true),
        ]);
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
