import type { CanvasesCanvasVersion } from "@/api-client";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";

type DraftBranchVersionLookup = {
  activeBranchMeta?: CanvasesCanvasVersion | null;
  activeBranch?: string | null;
  draftBranches: CanvasesCanvasVersion[];
};

export function resolveDraftVersionIdForBranch({
  activeBranchMeta,
  activeBranch,
  draftBranches,
  fallbackVersionId = "",
  latestDraftVersionId = "",
}: DraftBranchVersionLookup & {
  fallbackVersionId?: string;
  latestDraftVersionId?: string;
}): string {
  if (activeBranchMeta) {
    const versionId = draftVersionId(activeBranchMeta);
    if (versionId) {
      return versionId;
    }
  }

  if (activeBranch) {
    const branch = draftBranches.find((item) => draftBranchName(item) === activeBranch);
    const versionId = branch ? draftVersionId(branch) : "";
    if (versionId) {
      return versionId;
    }
  }

  if (latestDraftVersionId) {
    return latestDraftVersionId;
  }

  return fallbackVersionId;
}
