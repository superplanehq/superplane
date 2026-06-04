import type { CanvasStagingRecord } from "@/lib/canvas-staging";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/lib/canvas-staging";

import type { PendingFileChange } from "../workflow-files-types";

const RESERVED_STAGING_PATHS = new Set([CANVAS_YAML_PATH, CONSOLE_YAML_PATH]);

export function hasStagedRepositoryFileChanges(staging: CanvasStagingRecord | null | undefined): boolean {
  if (!staging) {
    return false;
  }

  if ((staging.deletedPaths ?? []).length > 0) {
    return true;
  }

  return Object.keys(staging.files).some((path) => !RESERVED_STAGING_PATHS.has(path));
}

export function pendingChangesFromStaging(
  stagingRecord: CanvasStagingRecord | null | undefined,
  baselineByPath: Record<string, string>,
  repositoryPathSet?: Set<string>,
): Record<string, PendingFileChange> {
  if (!stagingRecord) {
    return {};
  }

  const pending: Record<string, PendingFileChange> = {};

  for (const path of stagingRecord.deletedPaths ?? []) {
    if (path in baselineByPath || repositoryPathSet?.has(path)) {
      pending[path] = { type: "deleted", path };
    }
  }

  for (const [path, content] of Object.entries(stagingRecord.files)) {
    if (RESERVED_STAGING_PATHS.has(path)) {
      continue;
    }

    const baseline = baselineByPath[path];
    if (baseline === undefined) {
      if (repositoryPathSet?.has(path)) {
        pending[path] = { type: "modified", path, content };
        continue;
      }

      pending[path] = { type: "added", path, content };
      continue;
    }

    if (content !== baseline) {
      pending[path] = { type: "modified", path, content };
    }
  }

  return pending;
}
