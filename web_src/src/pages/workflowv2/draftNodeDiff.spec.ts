import { describe, expect, it } from "vitest";
import { hasDraftVersusLiveGraphDiff } from "./draftNodeDiff";

describe("hasDraftVersusLiveGraphDiff", () => {
  const node = (id: string, integrationId?: string | null) => ({
    id,
    name: "N",
    type: "TYPE_COMPONENT",
    ref: "r",
    configuration: {},
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

  it("returns true when only the readme changed", () => {
    const nodes = [node("a"), node("b")];
    const edges = [{ sourceId: "a", targetId: "b", channel: "default" }];
    const live = { spec: { nodes, edges, readme: "# Live" } };
    const draft = { spec: { nodes, edges, readme: "# Draft" } };

    expect(hasDraftVersusLiveGraphDiff(live as never, draft as never)).toBe(true);
  });

  it("treats missing readme the same as an empty readme", () => {
    const nodes = [node("a")];
    const edges = [] as { sourceId: string; targetId: string; channel: string }[];
    const live = { spec: { nodes, edges } };
    const draft = { spec: { nodes, edges, readme: "" } };

    expect(hasDraftVersusLiveGraphDiff(live as never, draft as never)).toBe(false);
  });
});
