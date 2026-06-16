import { useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import {
  fetchRepositoryFileContentCached,
  useCanvasRepositoryFileChanges,
  useCanvasVersionStaging,
} from "@/hooks/useCanvasData";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "../lib/workflow-spec-paths";
import { normalizeSpecFileContentForDiff } from "./lib/spec-yaml-normalize";
import type { FileDiffVersusLive, PendingFileChange } from "./types";

type UseFilesDiffOptions = {
  canvasId?: string;
  versionId?: string;
  canManageRepositoryFiles: boolean;
  isDiffOpen: boolean;
  pendingChanges: PendingFileChange[];
  pendingChangesByPath: Record<string, PendingFileChange>;
  // Whether the draft's spec files differ from live (canvas graph / console). The
  // backend file-changes signal excludes spec files, so these come from the canvas
  // tab's own draft-vs-live diff and let the Files diff include the spec files too.
  hasCanvasSpecDiffVersusLive: boolean;
  hasConsoleSpecDiffVersusLive: boolean;
};

// useFilesDiff orchestrates the Files-tab "diff vs live" view. It collects the set
// of files to diff — in-session pending edits, uncommitted staged paths, files
// committed to the draft branch that differ from live, and the spec files when the
// canvas graph / console differ from live — and computes each file's
// live-vs-effective-draft diff (lazily, only while the dialog is open). diffPaths is
// also used to decide whether the Diff button appears, so it reflects committed
// changes even when there are no uncommitted edits.
export function useFilesDiff({
  canvasId,
  versionId,
  canManageRepositoryFiles,
  isDiffOpen,
  pendingChanges,
  pendingChangesByPath,
  hasCanvasSpecDiffVersusLive,
  hasConsoleSpecDiffVersusLive,
}: UseFilesDiffOptions): { diffPaths: string[]; fileDiffsVersusLive: FileDiffVersusLive[] } {
  const enabled = canManageRepositoryFiles && !!versionId;
  const stagingQuery = useCanvasVersionStaging(canvasId ?? "", versionId, enabled);
  const fileChangesQuery = useCanvasRepositoryFileChanges(canvasId ?? "", versionId, enabled);
  const stagedPaths = stagingQuery.data?.stagedPaths;
  const committedChangedPaths = fileChangesQuery.data?.changedPaths;
  const diffPaths = useMemo(() => {
    const paths = new Set<string>();
    for (const change of pendingChanges) {
      paths.add(change.path);
    }
    for (const path of stagedPaths ?? []) {
      paths.add(path);
    }
    for (const path of committedChangedPaths ?? []) {
      paths.add(path);
    }
    if (hasCanvasSpecDiffVersusLive) {
      paths.add(CANVAS_YAML_PATH);
    }
    if (hasConsoleSpecDiffVersusLive) {
      paths.add(CONSOLE_YAML_PATH);
    }
    return Array.from(paths).sort();
  }, [pendingChanges, stagedPaths, committedChangedPaths, hasCanvasSpecDiffVersusLive, hasConsoleSpecDiffVersusLive]);

  const fileDiffsVersusLive = useFilesDiffVersusLive({
    canvasId,
    versionId,
    paths: diffPaths,
    pendingChangesByPath,
    enabled: isDiffOpen,
  });

  return { diffPaths, fileDiffsVersusLive };
}

type UseFilesDiffVersusLiveOptions = {
  canvasId?: string;
  versionId?: string;
  paths: string[];
  pendingChangesByPath: Record<string, PendingFileChange>;
  enabled: boolean;
};

/**
 * Computes per-file diffs of the effective draft against the live (main) version,
 * mirroring the canvas tab's "draft vs live" comparison. The baseline (old side)
 * is always the live content; the target (new side) is the effective draft:
 *   - an in-session pending edit when present (added / modified / deleted), else
 *   - the staged read (committed draft branch plus uncommitted staging).
 * Only paths that actually differ are returned, and spec files are normalized so
 * cosmetic YAML differences don't surface as changes.
 */
export function useFilesDiffVersusLive({
  canvasId,
  versionId,
  paths,
  pendingChangesByPath,
  enabled,
}: UseFilesDiffVersusLiveOptions): FileDiffVersusLive[] {
  const queryClient = useQueryClient();
  const [diffs, setDiffs] = useState<FileDiffVersusLive[]>([]);
  // Capture both the path set and each path's pending content so the diff
  // recomputes when an in-session edit changes without depending on a new object
  // identity every render.
  const pendingKey = paths
    .map((path) => {
      const change = pendingChangesByPath[path];
      if (!change) {
        return `${path}:`;
      }
      return `${path}:${change.type}:${change.type === "deleted" ? "" : change.content}`;
    })
    .join("|");

  useEffect(() => {
    if (!enabled || !canvasId || paths.length === 0) {
      setDiffs([]);
      return;
    }

    // Reads go through the React Query cache so identical reads are deduped.
    // "Not found" becomes empty content so added files (no live content) and
    // deletions (no draft content) render correctly instead of failing the batch.
    const readSide = async (path: string, readVersionId: string | undefined, stage: boolean): Promise<string> => {
      try {
        return await fetchRepositoryFileContentCached(queryClient, canvasId, path, readVersionId, stage);
      } catch {
        return "";
      }
    };

    // The live side is read with no version id, so the server serves the default
    // (main) branch / live version — the same baseline the canvas tab diffs against.
    const readDraft = async (path: string): Promise<string> => {
      const pending = pendingChangesByPath[path];
      if (pending) {
        return pending.type === "deleted" ? "" : pending.content;
      }
      return readSide(path, versionId, true);
    };

    let cancelled = false;
    void Promise.all(
      paths.map(async (path) => {
        const [live, draft] = await Promise.all([readSide(path, undefined, false), readDraft(path)]);
        return {
          path,
          liveContent: normalizeSpecFileContentForDiff(path, live),
          draftContent: normalizeSpecFileContentForDiff(path, draft),
        };
      }),
    ).then((results) => {
      if (cancelled) {
        return;
      }
      setDiffs(results.filter((diff) => diff.liveContent !== diff.draftContent));
    });

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, canvasId, versionId, pendingKey]);

  return diffs;
}
