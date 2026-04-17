import { describe, expect, it } from "vitest";
import { shouldPreserveDraftSpec } from "./draft-canvas-sync";

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
});
