import { describe, expect, it } from "vitest";

import type { CanvasesStaging } from "@/api-client";

import {
  isAwaitingCanvasStaging,
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

describe("isAwaitingCanvasStaging", () => {
  it("is true while entering edit before staging is cached", () => {
    expect(
      isAwaitingCanvasStaging({
        shouldReadStagedCanvasVersion: true,
        stagingLoading: false,
        stagingFetching: false,
        isEnteringEditSession: true,
        staging: undefined,
      }),
    ).toBe(true);
  });

  it("waits while staging is refetching", () => {
    expect(
      isAwaitingCanvasStaging({
        shouldReadStagedCanvasVersion: true,
        stagingLoading: false,
        stagingFetching: true,
        isEnteringEditSession: false,
        staging: {
          hasStaging: true,
          stagedPaths: ["canvas.yaml"],
          spec: { nodes: [{ id: "stale-node" }], edges: [] },
        },
      }),
    ).toBe(true);
  });

  it("uses settled staging for the active live version", () => {
    expect(
      isAwaitingCanvasStaging({
        shouldReadStagedCanvasVersion: true,
        stagingLoading: false,
        stagingFetching: false,
        isEnteringEditSession: false,
        staging: {
          hasStaging: true,
          stagedPaths: ["canvas.yaml"],
          spec: { nodes: [{ id: "staged-node" }], edges: [] },
        },
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
  const staging: CanvasesStaging = {
    hasStaging: true,
    stagedPaths: ["canvas.yaml"],
    spec: { nodes: [{ id: "staged-node" }], edges: [] },
  };

  it("returns staged content when available", () => {
    expect(
      resolveSelectedCanvasVersion({
        activeCanvasVersionId: "live-version",
        shouldReadStagedCanvasVersion: true,
        staging,
        loadedCommittedCanvasVersion: undefined,
        activeCanvasVersion: shellVersion,
        awaitingCanvasStaging: false,
      }),
    ).toEqual({
      metadata: { id: "live-version" },
      spec: staging.spec,
    });
  });

  it("strips shell spec while staging is still loading", () => {
    expect(
      resolveSelectedCanvasVersion({
        activeCanvasVersionId: "live-version",
        shouldReadStagedCanvasVersion: true,
        staging: undefined,
        loadedCommittedCanvasVersion: undefined,
        activeCanvasVersion: shellVersion,
        awaitingCanvasStaging: true,
      }),
    ).toEqual({
      metadata: { id: "live-version" },
      spec: undefined,
    });
  });

  it("falls back to the active shell while staging settles without blocking", () => {
    expect(
      resolveSelectedCanvasVersion({
        activeCanvasVersionId: "new-live-version",
        shouldReadStagedCanvasVersion: true,
        staging: undefined,
        loadedCommittedCanvasVersion: undefined,
        activeCanvasVersion: {
          metadata: { id: "new-live-version" },
          spec: { nodes: [{ id: "live-node" }], edges: [] },
        },
        awaitingCanvasStaging: false,
      }),
    ).toEqual({
      metadata: { id: "new-live-version" },
      spec: { nodes: [{ id: "live-node" }], edges: [] },
    });
  });
});
