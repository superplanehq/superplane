import { describe, expect, it } from "vitest";

import {
  isAwaitingStagedCanvasSpec,
  isViewingCurrentLiveCanvasVersion,
  resolveSelectedCanvasVersion,
  shouldReadStagedCanvasVersion,
} from "./live-edit-session";

describe("shouldReadStagedCanvasVersion", () => {
  it("reads staged content only while editing the live version", () => {
    expect(
      shouldReadStagedCanvasVersion({
        editSessionActive: true,
        activeCanvasVersionId: "live-version",
        effectiveLiveCanvasVersionId: "live-version",
        liveCanvasVersionId: "live-version",
      }),
    ).toBe(true);
  });

  it("reads committed content outside edit mode", () => {
    expect(
      shouldReadStagedCanvasVersion({
        editSessionActive: false,
        activeCanvasVersionId: "live-version",
        effectiveLiveCanvasVersionId: "live-version",
        liveCanvasVersionId: "live-version",
      }),
    ).toBe(false);
  });

  it("reads committed content when previewing a historical version in edit mode", () => {
    expect(
      shouldReadStagedCanvasVersion({
        editSessionActive: true,
        activeCanvasVersionId: "old-version",
        effectiveLiveCanvasVersionId: "live-version",
        liveCanvasVersionId: "live-version",
      }),
    ).toBe(false);
  });
});

describe("isAwaitingStagedCanvasSpec", () => {
  it("is true while entering edit before staged content is cached", () => {
    expect(
      isAwaitingStagedCanvasSpec({
        activeCanvasVersionId: "live-version",
        shouldReadStagedCanvasVersion: true,
        loadedStagedCanvasVersion: undefined,
        loadedStagedCanvasVersionLoading: false,
        loadedStagedCanvasVersionFetching: false,
        isEnteringEditSession: true,
      }),
    ).toBe(true);
  });

  it("ignores staged cache from a previous live version after commit", () => {
    expect(
      isAwaitingStagedCanvasSpec({
        activeCanvasVersionId: "new-live-version",
        shouldReadStagedCanvasVersion: true,
        loadedStagedCanvasVersion: {
          metadata: { id: "old-live-version" },
          spec: { nodes: [{ id: "stale-node" }], edges: [] },
        },
        loadedStagedCanvasVersionLoading: false,
        loadedStagedCanvasVersionFetching: true,
        isEnteringEditSession: false,
      }),
    ).toBe(true);
  });

  it("waits while a matched staged cache is refetching", () => {
    expect(
      isAwaitingStagedCanvasSpec({
        activeCanvasVersionId: "live-version",
        shouldReadStagedCanvasVersion: true,
        loadedStagedCanvasVersion: {
          metadata: { id: "live-version" },
          spec: { nodes: [{ id: "stale-node" }], edges: [] },
        },
        loadedStagedCanvasVersionLoading: false,
        loadedStagedCanvasVersionFetching: true,
        isEnteringEditSession: false,
      }),
    ).toBe(true);
  });

  it("uses a settled staged cache for the active live version", () => {
    expect(
      isAwaitingStagedCanvasSpec({
        activeCanvasVersionId: "live-version",
        shouldReadStagedCanvasVersion: true,
        loadedStagedCanvasVersion: {
          metadata: { id: "live-version" },
          spec: { nodes: [{ id: "staged-node" }], edges: [] },
        },
        loadedStagedCanvasVersionLoading: false,
        loadedStagedCanvasVersionFetching: false,
        isEnteringEditSession: false,
      }),
    ).toBe(false);
  });
});

describe("isViewingCurrentLiveCanvasVersion", () => {
  it("treats the active live version as current even when selected content is stale", () => {
    expect(
      isViewingCurrentLiveCanvasVersion({
        activeCanvasVersionId: "new-live-version",
        selectedCanvasVersion: {
          metadata: { id: "old-live-version" },
          spec: { nodes: [], edges: [] },
        },
        effectiveLiveCanvasVersionId: "new-live-version",
        liveCanvasVersionId: "new-live-version",
      }),
    ).toBe(true);
  });

  it("still reports historical previews as non-live", () => {
    expect(
      isViewingCurrentLiveCanvasVersion({
        activeCanvasVersionId: "old-version",
        selectedCanvasVersion: {
          metadata: { id: "old-version" },
          spec: { nodes: [], edges: [] },
        },
        effectiveLiveCanvasVersionId: "new-live-version",
        liveCanvasVersionId: "new-live-version",
      }),
    ).toBe(false);
  });
});

describe("resolveSelectedCanvasVersion", () => {
  const shellVersion = {
    metadata: { id: "live-version" },
    spec: { nodes: [{ id: "list-node" }], edges: [] },
  };
  const stagedVersion = {
    metadata: { id: "live-version" },
    spec: { nodes: [{ id: "staged-node" }], edges: [] },
  };

  it("returns staged content when available", () => {
    expect(
      resolveSelectedCanvasVersion({
        activeCanvasVersionId: "live-version",
        shouldReadStagedCanvasVersion: true,
        loadedStagedCanvasVersion: stagedVersion,
        loadedCommittedCanvasVersion: undefined,
        activeCanvasVersion: shellVersion,
        isAwaitingStagedSpec: false,
      }),
    ).toBe(stagedVersion);
  });

  it("strips shell spec while staged content is still loading", () => {
    expect(
      resolveSelectedCanvasVersion({
        activeCanvasVersionId: "live-version",
        shouldReadStagedCanvasVersion: true,
        loadedStagedCanvasVersion: undefined,
        loadedCommittedCanvasVersion: undefined,
        activeCanvasVersion: shellVersion,
        isAwaitingStagedSpec: true,
      }),
    ).toEqual({
      metadata: { id: "live-version" },
      spec: undefined,
    });
  });

  it("ignores staged cache from a previous live version after commit", () => {
    expect(
      resolveSelectedCanvasVersion({
        activeCanvasVersionId: "new-live-version",
        shouldReadStagedCanvasVersion: true,
        loadedStagedCanvasVersion: {
          metadata: { id: "old-live-version" },
          spec: { nodes: [{ id: "stale-node" }], edges: [] },
        },
        loadedCommittedCanvasVersion: undefined,
        activeCanvasVersion: {
          metadata: { id: "new-live-version" },
          spec: { nodes: [{ id: "live-node" }], edges: [] },
        },
        isAwaitingStagedSpec: false,
      }),
    ).toEqual({
      metadata: { id: "new-live-version" },
      spec: { nodes: [{ id: "live-node" }], edges: [] },
    });
  });
});
