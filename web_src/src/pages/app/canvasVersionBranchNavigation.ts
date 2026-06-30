import type { CanvasesCanvasVersion } from "@/api-client";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";
import { isDraftVersion } from "./lib/canvas-versions";
import { clearComponentSidebarSearchParams } from "./viewState";

export function resolveBranchNameForVersion(
  versionID: string,
  version: CanvasesCanvasVersion,
  draftBranches: CanvasesCanvasVersion[],
): string {
  const matchingBranch = draftBranches.find((branch) => draftVersionId(branch) === versionID);
  if (matchingBranch) {
    return draftBranchName(matchingBranch);
  }

  if (isDraftVersion(version)) {
    return draftBranchName(version);
  }

  return version.metadata?.branchName ?? "";
}

export function applyVersionSelectionSearchParams(
  current: URLSearchParams,
  options: { isCurrentLive: boolean; versionID: string; branchName: string; preserveBranchOnLive?: boolean },
): URLSearchParams {
  const next = new URLSearchParams(current);
  if (next.get("view") === "versions") {
    next.delete("view");
  }
  if (options.isCurrentLive && !options.preserveBranchOnLive) {
    next.delete("version");
    next.delete("branch");
  } else if (options.isCurrentLive && options.preserveBranchOnLive) {
    next.delete("version");
    if (options.branchName) {
      next.set("branch", options.branchName);
    } else {
      next.delete("branch");
    }
  } else {
    if (options.branchName) {
      next.delete("view");
      next.delete("run");
    }
    next.set("version", options.versionID);
    if (options.branchName) {
      next.set("branch", options.branchName);
    } else {
      next.delete("branch");
    }
  }

  return clearComponentSidebarSearchParams(next);
}
