import type { CanvasesCanvasVersion } from "@/api-client";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { SetURLSearchParams } from "react-router-dom";
import { draftBranchName, draftUpdatedAt, draftVersionId } from "@/lib/draftVersion";

const lastDraftBranchStorageKey = (canvasId: string) => `superplane:lastDraftBranch:${canvasId}`;

export function readLastDraftBranch(canvasId: string): string | null {
  if (typeof window === "undefined") {
    return null;
  }

  return window.localStorage.getItem(lastDraftBranchStorageKey(canvasId));
}

export function writeLastDraftBranch(canvasId: string, branch: string): void {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(lastDraftBranchStorageKey(canvasId), branch);
}

export function clearLastDraftBranch(canvasId: string): void {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.removeItem(lastDraftBranchStorageKey(canvasId));
}

/**
 * Selects the draft to open when entering edit mode: always the most recently
 * updated draft (the "latest", first on the list). Stored-branch and owner
 * preferences were intentionally dropped so the Edit button is predictable.
 */
export function pickDefaultDraftBranch(branches: CanvasesCanvasVersion[]): CanvasesCanvasVersion | null {
  if (branches.length === 0) {
    return null;
  }

  return (
    branches
      .slice()
      .sort((left, right) => Date.parse(draftUpdatedAt(right) || "") - Date.parse(draftUpdatedAt(left) || ""))[0] ??
    null
  );
}

type UseActiveDraftBranchOptions = {
  canvasId: string | undefined;
  searchParams: URLSearchParams;
  setSearchParams: SetURLSearchParams;
  draftBranches: CanvasesCanvasVersion[];
};

export function useActiveDraftBranch({
  canvasId,
  searchParams,
  setSearchParams,
  draftBranches,
}: UseActiveDraftBranchOptions) {
  const branchFromUrl = searchParams.get("branch");
  const [activeBranch, setActiveBranch] = useState<string | null>(branchFromUrl);
  const activeBranchRef = useRef(activeBranch);
  activeBranchRef.current = activeBranch;
  const ignoreMissingUrlBranchRef = useRef(false);

  useEffect(() => {
    const urlBranch = searchParams.get("branch");
    if (urlBranch && urlBranch === activeBranchRef.current) {
      ignoreMissingUrlBranchRef.current = false;
    }
    // A stale search-param write can briefly drop `branch` while local edit mode is active.
    if (!urlBranch && activeBranchRef.current && ignoreMissingUrlBranchRef.current) {
      return;
    }
    if (urlBranch !== activeBranchRef.current) {
      setActiveBranch(urlBranch);
    }
  }, [searchParams]);

  const activeBranchMeta = useMemo(
    () => draftBranches.find((branch) => draftBranchName(branch) === activeBranch) ?? null,
    [activeBranch, draftBranches],
  );

  const syncBranchToUrl = useCallback(
    (branch: string | null) => {
      setSearchParams(
        (current) => {
          const next = new URLSearchParams(current);
          if (branch) {
            next.set("branch", branch);
          } else {
            next.delete("branch");
          }
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const activateBranch = useCallback(
    (branch: string) => {
      if (!canvasId) {
        return;
      }

      ignoreMissingUrlBranchRef.current = true;
      setActiveBranch(branch);
      writeLastDraftBranch(canvasId, branch);
      syncBranchToUrl(branch);
    },
    [canvasId, syncBranchToUrl],
  );

  const exitToLive = useCallback(() => {
    ignoreMissingUrlBranchRef.current = false;
    setActiveBranch(null);
    syncBranchToUrl(null);
  }, [syncBranchToUrl]);

  return {
    activeBranch,
    activeBranchRef,
    activeBranchMeta,
    activateBranch,
    exitToLive,
    pickDefaultDraftBranch: useCallback(() => pickDefaultDraftBranch(draftBranches), [draftBranches]),
  };
}

export type ActiveDraftBranchState = ReturnType<typeof useActiveDraftBranch>;

export { draftBranchName, draftVersionId };
