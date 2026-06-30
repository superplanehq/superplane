import type { CanvasesCanvasBranch } from "@/api-client";

export const CANVAS_MAIN_BRANCH = "main";

export function branchName(branch: CanvasesCanvasBranch): string {
  return branch.name ?? "";
}

export function branchHeadVersionId(branch: CanvasesCanvasBranch): string {
  return branch.headVersionId ?? "";
}

export function pickDefaultCanvasBranch(branches: CanvasesCanvasBranch[]): CanvasesCanvasBranch | null {
  if (branches.length === 0) {
    return null;
  }

  const main = branches.find((branch) => branchName(branch) === CANVAS_MAIN_BRANCH);
  return main ?? branches[0];
}

export function sortCanvasBranches(branches: CanvasesCanvasBranch[]): CanvasesCanvasBranch[] {
  return branches.slice().sort((left, right) => {
    const leftName = branchName(left);
    const rightName = branchName(right);
    if (leftName === CANVAS_MAIN_BRANCH) {
      return -1;
    }
    if (rightName === CANVAS_MAIN_BRANCH) {
      return 1;
    }
    return leftName.localeCompare(rightName);
  });
}

export function shortCommitSha(sha?: string): string | undefined {
  if (!sha) {
    return undefined;
  }
  const trimmed = sha.trim();
  if (trimmed.length <= 7) {
    return trimmed;
  }
  return trimmed.slice(0, 7);
}
