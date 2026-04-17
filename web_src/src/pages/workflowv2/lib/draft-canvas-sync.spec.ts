import { describe, expect, it } from "vitest";
import { shouldIgnoreIncomingLiveSpecWhileEditingDraft } from "./draft-canvas-sync";

describe("shouldIgnoreIncomingLiveSpecWhileEditingDraft", () => {
  const liveVersionSpec = {
    nodes: [{ id: "live-node" }],
    edges: [],
  };

  it("keeps the current draft graph when a late live spec arrives", () => {
    expect(
      shouldIgnoreIncomingLiveSpecWhileEditingDraft({
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
      shouldIgnoreIncomingLiveSpecWhileEditingDraft({
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
      shouldIgnoreIncomingLiveSpecWhileEditingDraft({
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
