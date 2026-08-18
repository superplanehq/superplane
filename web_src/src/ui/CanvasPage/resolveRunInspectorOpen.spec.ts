import { describe, expect, it } from "vitest";
import { resolveRunInspectorOpen } from "./resolveRunInspectorOpen";

describe("resolveRunInspectorOpen", () => {
  const base = {
    isRunInspectionMode: true,
    isEditing: false,
    hasCanvasId: true,
    hasRunOrLoading: true,
  };

  it("opens for non-factory run inspection without a selected node", () => {
    expect(resolveRunInspectorOpen(base)).toBe(true);
  });

  it("keeps factory run inspection closed until a node is selected", () => {
    expect(resolveRunInspectorOpen({ ...base, factoryEmbed: true })).toBe(false);
    expect(resolveRunInspectorOpen({ ...base, factoryEmbed: true, selectedNodeId: null })).toBe(false);
    expect(resolveRunInspectorOpen({ ...base, factoryEmbed: true, selectedNodeId: "node-1" })).toBe(true);
  });

  it("stays closed while editing or outside run inspection", () => {
    expect(resolveRunInspectorOpen({ ...base, isEditing: true })).toBe(false);
    expect(resolveRunInspectorOpen({ ...base, isRunInspectionMode: false })).toBe(false);
    expect(resolveRunInspectorOpen({ ...base, hasRunOrLoading: false })).toBe(false);
  });
});
