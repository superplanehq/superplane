import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, beforeEach, vi } from "vitest";
import type { SetURLSearchParams } from "react-router-dom";

import type { CanvasesCanvasVersion } from "@/api-client";

import {
  clearLastDraftBranch,
  pickDefaultDraftBranch,
  readLastDraftBranch,
  useActiveDraftBranch,
  writeLastDraftBranch,
} from "./useActiveDraftBranch";

const canvasId = "canvas-123";

function draft(branchName: string, overrides: Partial<CanvasesCanvasVersion> = {}): CanvasesCanvasVersion {
  return {
    metadata: {
      branchName,
      displayName: branchName,
      owner: { id: "user-1", name: "Ada Lovelace" },
      updatedAt: "2026-06-03T12:00:00.000Z",
      ...overrides.metadata,
    },
    ...overrides,
  };
}

describe("pickDefaultDraftBranch", () => {
  beforeEach(() => {
    clearLastDraftBranch(canvasId);
  });

  it("prefers the branch stored in localStorage when it still exists", () => {
    writeLastDraftBranch(canvasId, "drafts/stored");
    const branches = [draft("drafts/other"), draft("drafts/stored")];

    expect(pickDefaultDraftBranch(branches, canvasId)?.metadata?.branchName).toBe("drafts/stored");
  });

  it("falls back to the current user's most recently updated draft", () => {
    const branches = [
      draft("drafts/old", {
        metadata: { owner: { id: "user-1", name: "Ada" }, updatedAt: "2026-06-01T12:00:00.000Z" },
      }),
      draft("drafts/new", {
        metadata: { owner: { id: "user-1", name: "Ada" }, updatedAt: "2026-06-03T12:00:00.000Z" },
      }),
      draft("drafts/other-user", {
        metadata: { owner: { id: "user-2", name: "Grace" }, updatedAt: "2026-06-04T12:00:00.000Z" },
      }),
    ];

    expect(pickDefaultDraftBranch(branches, canvasId, "user-1")?.metadata?.branchName).toBe("drafts/new");
  });

  it("falls back to the most recently updated org draft", () => {
    const branches = [
      draft("drafts/old", { metadata: { updatedAt: "2026-06-01T12:00:00.000Z" } }),
      draft("drafts/new", { metadata: { updatedAt: "2026-06-03T12:00:00.000Z" } }),
    ];

    expect(pickDefaultDraftBranch(branches, canvasId)?.metadata?.branchName).toBe("drafts/new");
  });
});

describe("useActiveDraftBranch", () => {
  beforeEach(() => {
    clearLastDraftBranch(canvasId);
    localStorage.clear();
  });

  it("syncs active branch to URL and localStorage", () => {
    let branchParam: string | null = null;
    const searchParams = new URLSearchParams();
    const setSearchParams = ((updater) => {
      if (typeof updater !== "function") {
        return;
      }

      branchParam = new URLSearchParams(updater(searchParams) as URLSearchParams).get("branch");
    }) as SetURLSearchParams;

    const { result } = renderHook(() =>
      useActiveDraftBranch({
        canvasId,
        searchParams,
        setSearchParams,
        draftBranches: [draft("drafts/user-1")],
      }),
    );

    act(() => {
      result.current.activateBranch("drafts/user-1");
    });

    expect(result.current.activeBranch).toBe("drafts/user-1");
    expect(readLastDraftBranch(canvasId)).toBe("drafts/user-1");
    expect(branchParam).toBe("drafts/user-1");

    act(() => {
      result.current.exitToLive();
    });

    expect(result.current.activeBranch).toBeNull();
    expect(branchParam).toBeNull();
  });

  it("keeps a locally activated branch when URL sync briefly drops branch", () => {
    const searchParams = new URLSearchParams();
    const setSearchParams = vi.fn() as unknown as SetURLSearchParams;

    const { result, rerender } = renderHook(
      ({ params }) =>
        useActiveDraftBranch({
          canvasId,
          searchParams: params,
          setSearchParams,
          draftBranches: [draft("drafts/user-1")],
        }),
      { initialProps: { params: searchParams } },
    );

    act(() => {
      result.current.activateBranch("drafts/user-1");
    });

    expect(result.current.activeBranch).toBe("drafts/user-1");

    rerender({ params: new URLSearchParams() });

    expect(result.current.activeBranch).toBe("drafts/user-1");

    rerender({ params: new URLSearchParams("branch=drafts%2Fuser-1") });

    expect(result.current.activeBranch).toBe("drafts/user-1");
  });
});
