import { describe, expect, it, vi } from "vitest";
import { getConfigSummary, getNodeHoverMetadataItems, listNodeNeighbors } from "./nodeChipHover";
import type { SuperplaneComponentsNode } from "@/api-client";

vi.mock("@/pages/app/mappers", () => ({
  getComponentBaseMapper: (name: string) => ({
    props: () => {
      if (name === "github.createIssue") {
        return {
          metadata: [{ icon: "book", label: "acme/widgets" }],
        };
      }
      if (name === "http") {
        return {
          metadata: [{ icon: "link", label: "GET https://example.com" }],
        };
      }
      return { metadata: [] };
    },
  }),
  getTriggerRenderer: (name: string) => ({
    getTriggerProps: () => {
      if (name === "github.onPullRequest") {
        return {
          metadata: [
            { icon: "book", label: "acme/widgets" },
            { icon: "funnel", label: "opened, synchronize" },
          ],
        };
      }
      return { metadata: [] };
    },
  }),
}));

const nodes: SuperplaneComponentsNode[] = [
  { id: "a", name: "Webhook Trigger" },
  { id: "b", name: "Call Target API" },
  { id: "c", name: "Check API Result" },
  { id: "d", name: "Notify Success" },
  { id: "e", name: "Notify Failure" },
  { id: "f" },
];

describe("getConfigSummary", () => {
  it("returns a one-line http summary", () => {
    expect(getConfigSummary("http", { method: "GET", url: "https://example.com" })).toBe("GET https://example.com");
  });

  it("returns null for unknown components or empty summaries", () => {
    expect(getConfigSummary("unknown", { foo: "bar" })).toBeNull();
    expect(getConfigSummary("if", { expression: "" })).toBeNull();
    expect(getConfigSummary("http", undefined)).toBeNull();
  });
});

describe("getNodeHoverMetadataItems", () => {
  it("uses mapper metadata for integration components", () => {
    expect(
      getNodeHoverMetadataItems(
        {
          id: "gh-1",
          name: "Create Issue",
          type: "TYPE_ACTION",
          component: "github.createIssue",
          metadata: { repository: { name: "acme/widgets" } },
        },
        "github.createIssue",
      ),
    ).toEqual([{ icon: "book", label: "acme/widgets" }]);
  });

  it("uses trigger mapper metadata", () => {
    expect(
      getNodeHoverMetadataItems(
        {
          id: "gh-pr",
          name: "On PR",
          type: "TYPE_TRIGGER",
          component: "github.onPullRequest",
          metadata: { repository: { name: "acme/widgets" } },
        },
        "github.onPullRequest",
      ),
    ).toEqual([
      { icon: "book", label: "acme/widgets" },
      { icon: "funnel", label: "opened, synchronize" },
    ]);
  });

  it("falls back to built-in summarizers when mapper metadata is empty", () => {
    expect(
      getNodeHoverMetadataItems(
        {
          id: "wait-1",
          name: "Wait",
          type: "TYPE_ACTION",
          component: "wait",
          configuration: { duration: "30" },
        },
        "wait",
      ),
    ).toEqual([{ icon: "info", label: "Wait: 30" }]);
  });
});

describe("listNodeNeighbors", () => {
  it("lists upstream then downstream with names", () => {
    const result = listNodeNeighbors(
      "b",
      [
        { sourceId: "a", targetId: "b" },
        { sourceId: "b", targetId: "c" },
      ],
      nodes,
    );

    expect(result).toEqual({
      items: [
        { id: "a", label: "Webhook Trigger", direction: "upstream" },
        { id: "c", label: "Check API Result", direction: "downstream" },
      ],
      overflow: 0,
    });
  });

  it("dedupes repeated edges and falls back to id for unnamed nodes", () => {
    const result = listNodeNeighbors(
      "b",
      [
        { sourceId: "f", targetId: "b" },
        { sourceId: "f", targetId: "b" },
        { sourceId: "b", targetId: "c" },
      ],
      nodes,
    );

    expect(result.items).toEqual([
      { id: "f", label: "f", direction: "upstream" },
      { id: "c", label: "Check API Result", direction: "downstream" },
    ]);
  });

  it("caps visible neighbors and reports overflow", () => {
    const result = listNodeNeighbors(
      "b",
      [
        { sourceId: "a", targetId: "b" },
        { sourceId: "b", targetId: "c" },
        { sourceId: "b", targetId: "d" },
        { sourceId: "b", targetId: "e" },
        { sourceId: "b", targetId: "f" },
      ],
      nodes,
      4,
    );

    expect(result.items).toHaveLength(4);
    expect(result.overflow).toBe(1);
    expect(result.items.map((item) => item.id)).toEqual(["a", "c", "d", "e"]);
  });

  it("returns an empty list when there are no edges", () => {
    expect(listNodeNeighbors("b", [], nodes)).toEqual({ items: [], overflow: 0 });
    expect(listNodeNeighbors(undefined, [{ sourceId: "a", targetId: "b" }], nodes)).toEqual({
      items: [],
      overflow: 0,
    });
  });
});
