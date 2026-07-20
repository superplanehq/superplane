import { describe, expect, it } from "vitest";
import {
  shouldApplyPreservedDraftSpec,
  shouldPreserveDraftSpec,
  shouldSkipDraftSpecSyncFromLoadedVersion,
} from "./draft-canvas-sync";

describe("shouldPreserveDraftSpec", () => {
  const liveVersionSpec = {
    nodes: [{ id: "live-node" }],
    edges: [],
  };

  it("keeps the current draft graph when a late live spec arrives", () => {
    expect(
      shouldPreserveDraftSpec({
        incomingSpec: liveVersionSpec,
        draftSpec: {
          nodes: [{ id: "draft-node" }],
          edges: [],
        },
        selectedDraftVersionSpec: {
          nodes: [{ id: "draft-node" }],
          edges: [],
        },
        liveVersionSpec,
      }),
    ).toBe(true);
  });

  it("accepts draft-originated updates while editing", () => {
    expect(
      shouldPreserveDraftSpec({
        incomingSpec: {
          nodes: [{ id: "draft-node" }],
          edges: [],
        },
        draftSpec: {
          nodes: [{ id: "draft-node" }],
          edges: [],
        },
        selectedDraftVersionSpec: {
          nodes: [{ id: "draft-node" }],
          edges: [],
        },
        liveVersionSpec,
      }),
    ).toBe(false);
  });

  it("falls back to the selected draft version when local draft state is not seeded yet", () => {
    expect(
      shouldPreserveDraftSpec({
        incomingSpec: liveVersionSpec,
        draftSpec: null,
        selectedDraftVersionSpec: {
          nodes: [{ id: "draft-node" }],
          edges: [],
        },
        liveVersionSpec,
      }),
    ).toBe(true);
  });

  it("does not preserve the draft when the selected draft version already matches the incoming spec", () => {
    expect(
      shouldPreserveDraftSpec({
        incomingSpec: liveVersionSpec,
        draftSpec: null,
        selectedDraftVersionSpec: liveVersionSpec,
        liveVersionSpec,
      }),
    ).toBe(false);
  });
});

describe("shouldApplyPreservedDraftSpec", () => {
  const committedSpec = {
    nodes: [{ id: "live-node" }],
    edges: [],
  };
  const stagedSpec = {
    nodes: [{ id: "staged-node" }],
    edges: [],
  };

  it("uses preserved draft spec when no staged version has loaded yet", () => {
    expect(shouldApplyPreservedDraftSpec(committedSpec, null)).toBe(true);
  });

  it("replaces stale committed cache once staged content arrives", () => {
    expect(shouldApplyPreservedDraftSpec(committedSpec, stagedSpec)).toBe(false);
  });

  it("keeps preserved draft spec when it already matches the staged version", () => {
    expect(shouldApplyPreservedDraftSpec(stagedSpec, stagedSpec)).toBe(true);
  });
});

describe("shouldSkipDraftSpecSyncFromLoadedVersion", () => {
  const localDraftSpec = {
    nodes: [{ id: "local-node" }],
    edges: [],
  };
  const loadedDraftSpec = {
    nodes: [{ id: "loaded-node" }],
    edges: [],
  };

  it("skips sync when local draft is ahead of the loaded version", () => {
    expect(shouldSkipDraftSpecSyncFromLoadedVersion(localDraftSpec, loadedDraftSpec)).toBe(true);
  });

  it("allows sync when local draft is not seeded yet", () => {
    expect(shouldSkipDraftSpecSyncFromLoadedVersion(null, loadedDraftSpec)).toBe(false);
  });

  it("allows sync when local draft already matches the loaded version", () => {
    expect(shouldSkipDraftSpecSyncFromLoadedVersion(localDraftSpec, localDraftSpec)).toBe(false);
  });
});
