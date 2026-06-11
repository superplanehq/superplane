import { describe, expect, it } from "vitest";
import { buildDraftDiffMap, hasDraftVersusLiveGraphDiff } from "./draftNodeDiff";

describe("hasDraftVersusLiveGraphDiff", () => {
  const node = (id: string, integrationId?: string | null, configuration: Record<string, unknown> = {}) => ({
    id,
    name: "N",
    type: "TYPE_ACTION",
    ref: "r",
    configuration,
    position: { x: 0, y: 0 },
    isCollapsed: false,
    integration: integrationId ? { id: integrationId, name: `integration-${integrationId}` } : undefined,
  });

  it("returns true when only edges differ", () => {
    const nodes = [node("a"), node("b")];
    const live = {
      spec: {
        nodes,
        edges: [{ sourceId: "a", targetId: "b", channel: "default" }],
      },
    };
    const draft = {
      spec: {
        nodes,
        edges: [] as { sourceId: string; targetId: string; channel: string }[],
      },
    };

    expect(hasDraftVersusLiveGraphDiff(live as never, draft as never)).toBe(true);
  });

  it("returns false when nodes and edges match", () => {
    const nodes = [node("a"), node("b")];
    const edges = [{ sourceId: "a", targetId: "b", channel: "default" }];
    const live = { spec: { nodes, edges } };
    const draft = { spec: { nodes, edges } };

    expect(hasDraftVersusLiveGraphDiff(live as never, draft as never)).toBe(false);
  });

  it("returns false when empty configuration is omitted in one version", () => {
    const liveNode = {
      id: "a",
      name: "N",
      type: "TYPE_ACTION",
      ref: "r",
      position: { x: 0, y: 0 },
      isCollapsed: false,
    };
    const draftNode = { ...liveNode, configuration: {} };
    const live = { spec: { nodes: [liveNode], edges: [] } };
    const draft = { spec: { nodes: [draftNode], edges: [] } };

    expect(hasDraftVersusLiveGraphDiff(live as never, draft as never)).toBe(false);
  });

  it("returns false when configuration keys are ordered differently", () => {
    const live = {
      spec: {
        nodes: [node("a", null, { a: 1, b: { c: 2, d: 3 } })],
        edges: [] as { sourceId: string; targetId: string; channel: string }[],
      },
    };
    const draft = {
      spec: {
        nodes: [node("a", null, { b: { d: 3, c: 2 }, a: 1 })],
        edges: [] as { sourceId: string; targetId: string; channel: string }[],
      },
    };

    expect(hasDraftVersusLiveGraphDiff(live as never, draft as never)).toBe(false);
  });

  it("returns true when only the integration binding changes", () => {
    const live = {
      spec: {
        nodes: [node("a", "github-1")],
        edges: [] as { sourceId: string; targetId: string; channel: string }[],
      },
    };
    const draft = {
      spec: {
        nodes: [node("a", "github-2")],
        edges: [] as { sourceId: string; targetId: string; channel: string }[],
      },
    };

    expect(hasDraftVersusLiveGraphDiff(live as never, draft as never)).toBe(true);
  });
});

describe("buildDraftDiffMap", () => {
  const node = (id: string, overrides: Record<string, unknown> = {}) => ({
    id,
    name: `Node ${id}`,
    type: "TYPE_ACTION",
    ref: "r",
    configuration: {},
    position: { x: 0, y: 0 },
    isCollapsed: false,
    ...overrides,
  });

  it("marks added, updated, and removed nodes", () => {
    const live = {
      spec: {
        nodes: [node("removed"), node("updated"), node("unchanged")],
      },
    };
    const draft = {
      spec: {
        nodes: [node("added"), node("updated", { configuration: { value: "new" } }), node("unchanged")],
      },
    };

    expect(buildDraftDiffMap(live as never, draft as never)).toEqual({
      statusMap: {
        added: "added",
        removed: "removed",
        updated: "updated",
      },
      removedNodes: [node("removed")],
    });
  });

  it("does not mark position-only changes as visual edits", () => {
    const live = { spec: { nodes: [node("a", { position: { x: 0, y: 0 } })] } };
    const draft = { spec: { nodes: [node("a", { position: { x: 100, y: 200 } })] } };

    expect(buildDraftDiffMap(live as never, draft as never)).toEqual({
      statusMap: {},
      removedNodes: [],
    });
  });

  it("does not mark empty or reordered configuration as visual edits", () => {
    const live = {
      spec: {
        nodes: [
          node("empty-live", { configuration: undefined }),
          node("ordered-live", { configuration: { a: 1, b: { c: 2, d: 3 } } }),
        ],
      },
    };
    const draft = {
      spec: {
        nodes: [
          node("empty-live", { configuration: {} }),
          node("ordered-live", { configuration: { b: { d: 3, c: 2 }, a: 1 } }),
        ],
      },
    };

    expect(buildDraftDiffMap(live as never, draft as never)).toEqual({
      statusMap: {},
      removedNodes: [],
    });
  });
});
