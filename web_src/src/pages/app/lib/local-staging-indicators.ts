import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { hasDraftVersusLiveConsoleDiff } from "../draftConsoleDiff";
import { hasDraftVersusLiveGraphDiff } from "../draftNodeDiff";
import type { PendingFileChange } from "../files/types";

export function hasLocalCanvasGraphDiff(
  committedSpec: CanvasesCanvas["spec"] | undefined,
  effectiveSpec: CanvasesCanvas["spec"] | undefined,
): boolean {
  if (!committedSpec || !effectiveSpec) {
    return false;
  }

  return hasDraftVersusLiveGraphDiff(
    { spec: committedSpec } as CanvasesCanvasVersion,
    { spec: effectiveSpec } as CanvasesCanvasVersion,
  );
}

export function hasLocalConsoleDiff(
  committed: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] } | undefined,
  effective: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] } | undefined,
): boolean {
  if (!committed || !effective) {
    return false;
  }

  return hasDraftVersusLiveConsoleDiff(committed, effective);
}

export function hasLocalFilesStaging(
  pendingChanges: PendingFileChange[],
  committedContentByPath: Record<string, string>,
): boolean {
  for (const change of pendingChanges) {
    if (change.type === "deleted" || change.type === "added") {
      return true;
    }

    const committed = committedContentByPath[change.path];
    if (committed === undefined || change.content !== committed) {
      return true;
    }
  }

  return false;
}
