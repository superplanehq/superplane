import { describe, expect, it } from "vitest";
import {
  buildConflictMarkerYAML,
  buildNodeMap,
  cloneJSON,
  concatenateBothNodes,
  concatenateConflictBlockLines,
  isPlainObject,
  localResolutionLabel,
  normalizeForCompare,
  parseNodeYAML,
  prettyYAML,
  pruneEdgesByNodes,
  upsertNode,
} from "./conflictResolverUtils";

describe("concatenateConflictBlockLines", () => {
  it("returns incoming lines when current is empty", () => {
    const result = concatenateConflictBlockLines([], ["name: test"]);
    expect(result).toEqual(["name: test"]);
  });

  it("returns current lines when incoming is empty", () => {
    const result = concatenateConflictBlockLines(["name: test"], []);
    expect(result).toEqual(["name: test"]);
  });

  it("concatenates both sides when both have content", () => {
    const currentLines = ["name: current-name"];
    const incomingLines = ["name: incoming-name"];
    const result = concatenateConflictBlockLines(currentLines, incomingLines);
    expect(result).toEqual(["name: current-name", "name: incoming-name"]);
  });

  it("preserves all lines including duplicates", () => {
    const currentLines = ["configuration:", "  key1: value1", "  key2: value2"];
    const incomingLines = ["configuration:", "  key2: updated", "  key3: value3"];
    const result = concatenateConflictBlockLines(currentLines, incomingLines);
    expect(result).toEqual([
      "configuration:",
      "  key1: value1",
      "  key2: value2",
      "configuration:",
      "  key2: updated",
      "  key3: value3",
    ]);
  });
});

describe("concatenateBothNodes", () => {
  it("returns 'null' when both nodes are undefined", () => {
    expect(concatenateBothNodes(undefined, undefined)).toBe("null\n");
  });

  it("returns incoming YAML when current is undefined", () => {
    const result = concatenateBothNodes(undefined, { id: "a", name: "test" });
    expect(result).toContain("name: test");
    expect(result.endsWith("\n")).toBe(true);
  });

  it("returns current YAML when incoming is undefined", () => {
    const result = concatenateBothNodes({ id: "a", name: "test" }, undefined);
    expect(result).toContain("name: test");
    expect(result.endsWith("\n")).toBe(true);
  });

  it("concatenates both node YAMLs when both exist", () => {
    const result = concatenateBothNodes({ id: "a", name: "current-name" }, { id: "a", name: "incoming-name" });
    expect(result).toContain("name: current-name");
    expect(result).toContain("name: incoming-name");
    const nameMatches = result.match(/name:/g);
    expect(nameMatches?.length).toBe(2);
  });

  it("includes all fields from both nodes", () => {
    const result = concatenateBothNodes({ id: "a", timeout: 30 }, { id: "a", retries: 5 });
    expect(result).toContain("timeout: 30");
    expect(result).toContain("retries: 5");
  });
});

describe("parseNodeYAML", () => {
  it("returns null node for empty input", () => {
    const result = parseNodeYAML("", "node-1");
    expect(result).toEqual({ node: null });
  });

  it("returns null node for whitespace-only input", () => {
    const result = parseNodeYAML("   \n  ", "node-1");
    expect(result).toEqual({ node: null });
  });

  it("rejects input with conflict markers", () => {
    const input = "<<<<<<< current\nname: a\n=======\nname: b\n>>>>>>> incoming";
    const result = parseNodeYAML(input, "node-1");
    expect(result.error).toBe("Resolve conflict markers before applying YAML.");
    expect(result.node).toBeNull();
  });

  it("parses valid YAML object and sets id", () => {
    const result = parseNodeYAML("name: test\ntype: component", "node-1");
    expect(result.error).toBeUndefined();
    expect(result.node).toEqual({ name: "test", type: "component", id: "node-1" });
  });

  it("rejects array YAML", () => {
    const result = parseNodeYAML("- item1\n- item2", "node-1");
    expect(result.error).toBe("Final Result must be a YAML object or null.");
  });

  it("returns null node for 'null' YAML", () => {
    const result = parseNodeYAML("null", "node-1");
    expect(result).toEqual({ node: null });
  });

  it("returns error for invalid YAML syntax", () => {
    const result = parseNodeYAML("key: [invalid", "node-1");
    expect(result.error).toBe("Invalid YAML format.");
  });
});

describe("buildConflictMarkerYAML", () => {
  it("produces conflict markers for differing fields", () => {
    const current = { id: "node-1", name: "Current Name" };
    const incoming = { id: "node-1", name: "Incoming Name" };
    const result = buildConflictMarkerYAML(current, incoming, "Current", "Incoming");
    expect(result).toContain("<<<<<<< Current");
    expect(result).toContain("=======");
    expect(result).toContain(">>>>>>> Incoming");
    expect(result).toContain("Current Name");
    expect(result).toContain("Incoming Name");
  });

  it("does not produce conflict markers for equal fields", () => {
    const current = { id: "node-1", name: "Same" };
    const incoming = { id: "node-1", name: "Same" };
    const result = buildConflictMarkerYAML(current, incoming, "Current", "Incoming");
    expect(result).not.toContain("<<<<<<<");
    expect(result).not.toContain("=======");
    expect(result).not.toContain(">>>>>>>");
  });

  it("marks absent keys with comments", () => {
    const current = { id: "node-1" };
    const incoming = { id: "node-1", extra: "value" };
    const result = buildConflictMarkerYAML(current, incoming, "Current", "Incoming");
    expect(result).toContain("# extra is absent");
    expect(result).toContain("extra: value");
  });
});

describe("upsertNode", () => {
  it("adds a new node when not found", () => {
    const nodes = [{ id: "a", name: "A" }];
    const result = upsertNode(nodes, "b", { id: "b", name: "B" });
    expect(result).toHaveLength(2);
    expect(result[1]).toEqual({ id: "b", name: "B" });
  });

  it("replaces an existing node", () => {
    const nodes = [{ id: "a", name: "A" }];
    const result = upsertNode(nodes, "a", { id: "a", name: "Updated A" });
    expect(result).toHaveLength(1);
    expect(result[0]).toEqual({ id: "a", name: "Updated A" });
  });

  it("removes a node when value is null", () => {
    const nodes = [
      { id: "a", name: "A" },
      { id: "b", name: "B" },
    ];
    const result = upsertNode(nodes, "a", null);
    expect(result).toHaveLength(1);
    expect(result[0]).toEqual({ id: "b", name: "B" });
  });

  it("returns unchanged array when removing non-existent node", () => {
    const nodes = [{ id: "a", name: "A" }];
    const result = upsertNode(nodes, "z", null);
    expect(result).toBe(nodes);
  });
});

describe("buildNodeMap", () => {
  it("builds a map from node array", () => {
    const nodes = [
      { id: "a", name: "A" },
      { id: "b", name: "B" },
    ];
    const map = buildNodeMap(nodes);
    expect(map.size).toBe(2);
    expect(map.get("a")).toEqual({ id: "a", name: "A" });
    expect(map.get("b")).toEqual({ id: "b", name: "B" });
  });

  it("skips nodes without id", () => {
    const nodes = [{ name: "No ID" }, { id: "a", name: "A" }];
    const map = buildNodeMap(nodes);
    expect(map.size).toBe(1);
  });
});

describe("pruneEdgesByNodes", () => {
  it("keeps edges whose source and target exist", () => {
    const edges = [{ sourceId: "a", targetId: "b" }];
    const nodes = [{ id: "a" }, { id: "b" }];
    expect(pruneEdgesByNodes(edges, nodes)).toHaveLength(1);
  });

  it("removes edges with missing source", () => {
    const edges = [{ sourceId: "missing", targetId: "b" }];
    const nodes = [{ id: "a" }, { id: "b" }];
    expect(pruneEdgesByNodes(edges, nodes)).toHaveLength(0);
  });

  it("removes edges with missing target", () => {
    const edges = [{ sourceId: "a", targetId: "missing" }];
    const nodes = [{ id: "a" }, { id: "b" }];
    expect(pruneEdgesByNodes(edges, nodes)).toHaveLength(0);
  });
});

describe("localResolutionLabel", () => {
  it("returns 'excluded' when finalNode is undefined", () => {
    expect(localResolutionLabel({ id: "a" }, { id: "a" }, undefined)).toBe("excluded");
  });

  it("returns 'current' when final matches current", () => {
    const node = { id: "a", name: "test" };
    expect(localResolutionLabel(node, { id: "a", name: "other" }, { ...node })).toBe("current");
  });

  it("returns 'incoming' when final matches incoming", () => {
    const node = { id: "a", name: "test" };
    expect(localResolutionLabel({ id: "a", name: "other" }, node, { ...node })).toBe("incoming");
  });

  it("returns 'custom' when final matches neither", () => {
    expect(
      localResolutionLabel({ id: "a", name: "current" }, { id: "a", name: "incoming" }, { id: "a", name: "custom" }),
    ).toBe("custom");
  });
});

describe("isPlainObject", () => {
  it("returns true for plain objects", () => {
    expect(isPlainObject({})).toBe(true);
    expect(isPlainObject({ a: 1 })).toBe(true);
  });

  it("returns false for arrays", () => {
    expect(isPlainObject([])).toBe(false);
  });

  it("returns false for null", () => {
    expect(isPlainObject(null)).toBe(false);
  });

  it("returns false for primitives", () => {
    expect(isPlainObject("string")).toBe(false);
    expect(isPlainObject(42)).toBe(false);
    expect(isPlainObject(undefined)).toBe(false);
  });
});

describe("normalizeForCompare", () => {
  it("sorts object keys", () => {
    const result = normalizeForCompare({ b: 2, a: 1 });
    expect(Object.keys(result as Record<string, unknown>)).toEqual(["a", "b"]);
  });

  it("handles arrays", () => {
    const result = normalizeForCompare([{ b: 2, a: 1 }]);
    expect(result).toEqual([{ a: 1, b: 2 }]);
  });

  it("returns primitives unchanged", () => {
    expect(normalizeForCompare("hello")).toBe("hello");
    expect(normalizeForCompare(42)).toBe(42);
    expect(normalizeForCompare(null)).toBe(null);
  });
});

describe("cloneJSON", () => {
  it("creates a deep copy", () => {
    const original = { a: { b: 1 } };
    const clone = cloneJSON(original);
    expect(clone).toEqual(original);
    expect(clone).not.toBe(original);
    expect(clone.a).not.toBe(original.a);
  });
});

describe("prettyYAML", () => {
  it("produces sorted YAML output", () => {
    const result = prettyYAML({ b: 2, a: 1 });
    const lines = result.trim().split("\n");
    expect(lines[0]).toBe("a: 1");
    expect(lines[1]).toBe("b: 2");
  });

  it("handles null and undefined", () => {
    expect(prettyYAML(null)).toBe("null\n");
    expect(prettyYAML(undefined)).toBe("null\n");
  });
});
