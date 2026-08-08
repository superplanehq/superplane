import type { Node } from "@xyflow/react";
import { describe, expect, it } from "vitest";
import { isConnectableCanvasNode, isValidCanvasConnection } from "./connectionValidity";

function makeNode(overrides: Partial<Node> = {}): Node {
  return {
    id: "node-1",
    position: { x: 0, y: 0 },
    data: {},
    ...overrides,
  };
}

const connection = { source: "node-1", target: "node-2", sourceHandle: "default", targetHandle: null };

describe("isValidCanvasConnection", () => {
  it("allows a connection between two regular nodes", () => {
    const nodes = [makeNode({ id: "node-1" }), makeNode({ id: "node-2" })];

    expect(isValidCanvasConnection(nodes, connection)).toBe(true);
  });

  it("rejects a connection targeting a non-connectable node", () => {
    const nodes = [makeNode({ id: "node-1" }), makeNode({ id: "node-2", connectable: false })];

    expect(isValidCanvasConnection(nodes, connection)).toBe(false);
  });

  it("rejects a connection targeting a removed ghost node", () => {
    const nodes = [makeNode({ id: "node-1" }), makeNode({ id: "node-2", data: { _draftDiffStatus: "removed" } })];

    expect(isValidCanvasConnection(nodes, connection)).toBe(false);
  });

  it("rejects a connection starting from a removed ghost node", () => {
    const nodes = [makeNode({ id: "node-1", data: { _draftDiffStatus: "removed" } }), makeNode({ id: "node-2" })];

    expect(isValidCanvasConnection(nodes, connection)).toBe(false);
  });

  it("rejects a connection whose endpoint is not on the canvas", () => {
    const nodes = [makeNode({ id: "node-1" })];

    expect(isValidCanvasConnection(nodes, connection)).toBe(false);
  });
});

describe("isConnectableCanvasNode", () => {
  it("allows a regular node", () => {
    expect(isConnectableCanvasNode([makeNode()], "node-1")).toBe(true);
  });

  it("rejects a node marked not connectable", () => {
    expect(isConnectableCanvasNode([makeNode({ connectable: false })], "node-1")).toBe(false);
  });

  it("rejects a removed ghost node", () => {
    expect(isConnectableCanvasNode([makeNode({ data: { _draftDiffStatus: "removed" } })], "node-1")).toBe(false);
  });

  it("rejects a node that is not on the canvas", () => {
    expect(isConnectableCanvasNode([], "node-1")).toBe(false);
  });
});
